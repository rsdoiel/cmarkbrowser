// Package search finds Go regular expression matches in the text a page is
// shown as, and marks them. It works on the rendered text, the way a person
// reads it: a paragraph that the terminal wrapped over several lines is one
// run of words, so a phrase is found even when the wrap splits it.
package search

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Highlight styles. Backgrounds only, so the colours and weights the renderer
// chose for the text underneath are left alone.
const (
	MatchOn    = "\x1b[48;5;238m"
	MatchOff   = "\x1b[49m"
	CurrentOn  = "\x1b[48;5;214;38;5;16m"
	CurrentOff = "\x1b[49;39m"
)

// sentinel is the zero-width mark the browser puts before link markers; it is
// not text and is never counted.
const sentinel = "⁠"

// escLen returns the length of the escape sequence that starts s (s[0] is
// ESC): a CSI sequence (colours, styles), an OSC sequence (hyperlinks) up to
// BEL or ESC \, or a two-byte escape. A truncated sequence runs to the end.
func escLen(s string) int {
	if len(s) < 2 {
		return len(s)
	}
	switch s[1] {
	case '[':
		for i := 2; i < len(s); i++ {
			if s[i] >= 0x40 && s[i] <= 0x7e {
				return i + 1
			}
		}
		return len(s)
	case ']':
		for i := 2; i < len(s); i++ {
			if s[i] == 0x07 {
				return i + 1
			}
			if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}
		return len(s)
	}
	return 2
}

/** StripANSI returns s as plain text: colour and style codes, OSC hyperlinks
 * and the zero-width link sentinel removed.
 *
 * Example:
 *   search.StripANSI("\x1b[1mbold\x1b[0m") // "bold"
 */
func StripANSI(s string) string {
	if !strings.ContainsAny(s, "\x1b\u2060") {
		return s
	}
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); {
		switch {
		case s[i] == 0x1b:
			i += escLen(s[i:])
		case strings.HasPrefix(s[i:], sentinel):
			i += len(sentinel)
		default:
			j := i + 1
			for j < len(s) && s[j] != 0x1b && s[j] != 0xe2 {
				j++
			}
			sb.WriteString(s[i:j])
			i = j
		}
	}
	return sb.String()
}

/** Compile turns what a person typed into a Go regexp. A pattern with no
 * upper-case letter ignores case, the way less and vim do with smart case; an
 * upper-case letter makes it case sensitive. A class such as \S or \W does not
 * count as an upper-case letter. Write (?i) or (?-i) in the pattern to decide
 * for yourself.
 *
 * Parameters:
 *   pattern (string) — a Go regexp (RE2) as typed
 *
 * Returns:
 *   *regexp.Regexp — the compiled pattern
 *   error — an empty or invalid pattern
 *
 * Example:
 *   re, err := search.Compile("yap (state|sate)") // matches "Yap State"
 */
func Compile(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, fmt.Errorf("empty pattern")
	}
	if !hasUpper(pattern) {
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

// hasUpper reports whether the pattern holds an upper-case letter that is
// literal text: escapes (\S, \pL, \p{Lu}) and group names are skipped.
func hasUpper(p string) bool {
	rs := []rune(p)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case r == '\\' && i+1 < len(rs):
			i++
			if (rs[i] == 'p' || rs[i] == 'P') && i+1 < len(rs) && rs[i+1] == '{' {
				for i < len(rs) && rs[i] != '}' {
					i++
				}
			}
		case r == '(' && i+2 < len(rs) && rs[i+1] == '?' && (rs[i+2] == 'P' || rs[i+2] == '<'):
			for i < len(rs) && rs[i] != '>' {
				i++
			}
		case unicode.IsUpper(r):
			return true
		}
	}
	return false
}

// Range is part of a match on one line of the shown text: columns Start to
// Stop (exclusive), counted in characters, not terminal cells.
type Range struct{ Line, Start, Stop int }

// Match is one match. It has one Range for each line it covers, usually one.
type Match struct{ Ranges []Range }

// Line returns the line the match starts on.
func (m Match) Line() int { return m.Ranges[0].Line }

// seg is the text of one shown line inside the flowed text: n characters that
// start at character col of the line and at character flowStart of the flow.
type seg struct{ flowStart, line, col, n int }

// Index is the searchable form of a page's shown text.
type Index struct {
	flow string
	segs []seg // in flow order; the gaps between them are joining spaces and paragraph breaks
}

/** NewIndex prepares shown text (with its colour codes) for searching. Lines
 * of one paragraph are joined with a single space, so a phrase that wrapped is
 * still found; a blank line becomes a newline, so a match never runs from one
 * paragraph into the next and (?m)^ anchors at the start of a paragraph.
 *
 * Parameters:
 *   display (string) — the text as shown, lines separated by newlines
 *
 * Returns:
 *   *Index — ready for Find
 *
 * Example:
 *   ix := search.NewIndex(view.Display(-1))
 */
func NewIndex(display string) *Index {
	ix := &Index{}
	var sb strings.Builder
	flowRunes := 0
	inText := false
	for li, line := range strings.Split(StripANSI(display), "\n") {
		runes := []rune(line)
		lead, end := 0, len(runes)
		for lead < end && unicode.IsSpace(runes[lead]) {
			lead++
		}
		for end > lead && unicode.IsSpace(runes[end-1]) {
			end--
		}
		if lead == end { // a blank line ends the paragraph
			if inText {
				sb.WriteByte('\n')
				flowRunes++
				inText = false
			}
			continue
		}
		if inText {
			sb.WriteByte(' ')
			flowRunes++
		}
		sb.WriteString(string(runes[lead:end]))
		ix.segs = append(ix.segs, seg{flowStart: flowRunes, line: li, col: lead, n: end - lead})
		flowRunes += end - lead
		inText = true
	}
	ix.flow = sb.String()
	return ix
}

/** Find returns the matches of re, in the order they appear. A match that is
 * empty, or that covers only joining spaces, is left out. Matching is over the
 * flowed text described at NewIndex.
 *
 * Parameters:
 *   re (*regexp.Regexp) — from Compile
 *
 * Returns:
 *   []Match — the matches
 *
 * Example:
 *   for _, m := range ix.Find(re) { fmt.Println(m.Line()) }
 */
func (ix *Index) Find(re *regexp.Regexp) []Match {
	var out []Match
	prevByte, prevRune := 0, 0 // matches come in order, so count runes only between them
	runeAt := func(b int) int {
		prevRune += utf8.RuneCountInString(ix.flow[prevByte:b])
		prevByte = b
		return prevRune
	}
	for _, loc := range re.FindAllStringIndex(ix.flow, -1) {
		if loc[1] <= loc[0] {
			continue
		}
		from, to := runeAt(loc[0]), runeAt(loc[1])
		var m Match
		// first segment that ends after from
		i := sort.Search(len(ix.segs), func(i int) bool { return ix.segs[i].flowStart+ix.segs[i].n > from })
		for ; i < len(ix.segs) && ix.segs[i].flowStart < to; i++ {
			sg := ix.segs[i]
			a, b := max(from, sg.flowStart), min(to, sg.flowStart+sg.n)
			if a < b {
				m.Ranges = append(m.Ranges, Range{sg.line, sg.col + a - sg.flowStart, sg.col + b - sg.flowStart})
			}
		}
		if len(m.Ranges) > 0 {
			out = append(out, m)
		}
	}
	return out
}

type markSpan struct {
	start, stop int
	current     bool
}

/** Marked is a page's text with its matches marked, ready to show with any
 * match as the current one. Moving to another match re-marks only the lines
 * of the old and new match, not the whole page, so n and N stay instant on a
 * page of ninety thousand lines.
 */
type Marked struct {
	raw     []string // the lines as given
	base    []string // lines with their matches in the ordinary style
	matches []Match
	spans   map[int][]markSpan // ordinary spans of each line that has matches
	last    int                // current match of the previous Render, or -1
	out     []string           // the previous Render's lines, reused
}

/** NewMarked marks every match of display in the ordinary style.
 *
 * Parameters:
 *   display (string) — the shown text, with its colour codes
 *   matches ([]Match) — from Find on an Index of the same text
 *
 * Returns:
 *   *Marked — call Render(cur) to get the text with match cur current
 *
 * Example:
 *   mk := search.NewMarked(view.Display(-1), matches)
 *   screen := mk.Render(0)
 */
func NewMarked(display string, matches []Match) *Marked {
	m := &Marked{matches: matches, spans: map[int][]markSpan{}, last: -1}
	if len(matches) == 0 {
		m.raw = []string{display}
		m.base = m.raw
		return m
	}
	m.raw = strings.Split(display, "\n")
	m.base = append([]string(nil), m.raw...)
	for _, mt := range matches {
		for _, r := range mt.Ranges {
			m.spans[r.Line] = append(m.spans[r.Line], markSpan{r.Start, r.Stop, false})
		}
	}
	for li, sp := range m.spans {
		if li >= 0 && li < len(m.base) {
			m.base[li] = markLine(m.raw[li], sp)
		}
	}
	m.out = append([]string(nil), m.base...)
	return m
}

// Render returns the text with match cur (or none, for -1) in the current style.
func (m *Marked) Render(cur int) string {
	if len(m.matches) == 0 {
		return m.raw[0]
	}
	m.remark(m.last, false)
	m.remark(cur, true)
	m.last = cur
	return strings.Join(m.out, "\n")
}

// remark puts the lines of match i back to ordinary style, or to current style.
func (m *Marked) remark(i int, current bool) {
	if i < 0 || i >= len(m.matches) {
		return
	}
	for _, r := range m.matches[i].Ranges {
		if r.Line < 0 || r.Line >= len(m.out) {
			continue
		}
		if !current {
			m.out[r.Line] = m.base[r.Line]
			continue
		}
		sp := append([]markSpan(nil), m.spans[r.Line]...)
		for k := range sp {
			if sp[k].start == r.Start && sp[k].stop == r.Stop {
				sp[k].current = true
			}
		}
		m.out[r.Line] = markLine(m.raw[r.Line], sp)
	}
}

/** Highlight returns display with every match marked: the current one (index
 * cur, or -1 for none) in its own style, the others in the ordinary one. The
 * visible text is unchanged: colour codes and hyperlinks in display are copied
 * through and are not counted as columns. For a page shown while the current
 * match moves, use NewMarked and Render, which do not re-mark the whole page.
 *
 * Parameters:
 *   display (string) — the shown text, with its colour codes
 *   matches ([]Match) — from Find on an Index of the same text
 *   cur (int) — index of the current match, or -1
 *
 * Returns:
 *   string — display with the matches marked
 *
 * Example:
 *   screen := search.Highlight(view.Display(-1), matches, 3)
 */
func Highlight(display string, matches []Match, cur int) string {
	return NewMarked(display, matches).Render(cur)
}

// markLine inserts the highlight codes into one line of styled text. Codes and
// hyperlinks are copied through without counting as columns.
func markLine(line string, spans []markSpan) string {
	starts := map[int]markSpan{}
	stops := map[int]bool{}
	for _, s := range spans {
		starts[s.start] = s
		stops[s.stop] = true
	}
	var sb strings.Builder
	var open *markSpan
	closeOpen := func() {
		if open != nil {
			if open.current {
				sb.WriteString(CurrentOff)
			} else {
				sb.WriteString(MatchOff)
			}
			open = nil
		}
	}
	col := 0
	for i := 0; i < len(line); {
		// A match ends where its last character ends, before any escape code
		// that follows it.
		if stops[col] {
			closeOpen()
		}
		if line[i] == 0x1b {
			n := escLen(line[i:])
			sb.WriteString(line[i : i+n])
			i += n
			continue
		}
		if strings.HasPrefix(line[i:], sentinel) {
			sb.WriteString(sentinel)
			i += len(sentinel)
			continue
		}
		if s, ok := starts[col]; ok {
			open = &s
			if s.current {
				sb.WriteString(CurrentOn)
			} else {
				sb.WriteString(MatchOn)
			}
		}
		_, size := utf8.DecodeRuneInString(line[i:])
		sb.WriteString(line[i : i+size])
		i += size
		col++
	}
	closeOpen()
	return sb.String()
}
