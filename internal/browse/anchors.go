package browse

import (
	"regexp"
	"strings"

	"github.com/rsdoiel/cmarkbrowser/internal/search"
	"github.com/rsdoiel/markdownweb/slug"
)

var (
	headingRE  = regexp.MustCompile(`^ {0,3}#{1,6}[ \t]+(.*?)(?:[ \t]+#+)?[ \t]*$`)
	mdLinkRE   = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)
	fenceRE    = regexp.MustCompile("^ {0,3}(```|~~~)")
	markerStrp = regexp.MustCompile(`\s*\x{2060}\\\[\d+\\\]`) // the link marker Prepare adds
	markupStrp = strings.NewReplacer("*", "", "`", "", "\u2060", "")

	// setextRE is the underline of a Setext heading; a lone `---` after a
	// paragraph line is one too, as in CommonMark.
	setextRE = regexp.MustCompile(`^ {0,3}(?:=+|-+)[ \t]*$`)
	// notParaRE starts a line that cannot be a one-line paragraph: a quote,
	// list item, table row, ATX heading or indented code.
	notParaRE = regexp.MustCompile(`^(?: {4}|\t|\s*(?:>|[-*+][ \t]|\d+[.)][ \t]|\||#))`)
	// emphRE is `_emphasis_` and `__strong__`: an underscore run that opens at
	// a word edge and closes at one, so snake_case keeps its underscores.
	emphRE = regexp.MustCompile(`(^|\W)_+([^_\s](?:[^_]*[^_\s])?)_+(\W|$)`)
)

// anchor is a heading of the page: its slug (markdownweb's rule) and the line
// of the rendered text it is on.
type anchor struct {
	slug string
	line int
}

// headingTexts returns the plain text of each ATX and Setext heading of
// Markdown src, in order, leaving out lines inside a code fence. A Setext
// heading is one paragraph line over an `===` or `---` underline.
func headingTexts(src string) []string {
	var out []string
	fence := ""
	prev, prevBlank := "", true // the line above, and whether the one above it was blank
	prevIsPara := false
	for _, line := range strings.Split(src, "\n") {
		if m := fenceRE.FindStringSubmatch(line); m != nil {
			switch fence {
			case "":
				fence = m[1]
			case m[1]:
				fence = ""
			}
			prevIsPara = false
			prevBlank = true
			continue
		}
		if fence != "" {
			continue
		}
		if m := headingRE.FindStringSubmatch(line); m != nil {
			out = append(out, plainHeading(m[1]))
			prevIsPara = false
			prevBlank = true
			continue
		}
		if prevIsPara && setextRE.MatchString(line) {
			out = append(out, plainHeading(prev))
			prevIsPara = false
			prevBlank = true
			continue
		}
		blank := strings.TrimSpace(line) == ""
		prevIsPara = !blank && prevBlank && !notParaRE.MatchString(line)
		prev, prevBlank = line, blank
	}
	return out
}

// plainHeading is a heading's text with its Markdown taken out: link markers,
// links (their text stays), `*`, backticks and `_emphasis_`.
func plainHeading(s string) string {
	t := mdLinkRE.ReplaceAllString(markerStrp.ReplaceAllString(s, ""), "$1")
	t = strings.TrimSpace(markupStrp.Replace(t))
	for {
		u := emphRE.ReplaceAllString(t, "$1$2$3")
		if u == t {
			return t
		}
		t = u
	}
}

// findAnchors slugs the headings of the cleaned Markdown and finds each on a
// line of the rendered text, searching forward so that a repeated heading
// finds its own line. A heading whose line is not found is left out.
func findAnchors(clean, text string) []anchor {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		lines[i] = strings.Join(strings.Fields(markupStrp.Replace(strings.TrimLeft(search.StripANSI(l), "# \t"))), " ")
	}
	slugs := slug.New()
	var out []anchor
	cur := 0
	for _, h := range headingTexts(clean) {
		sl := slugs.Next(h)
		words := strings.Fields(h)
		if sl == "" || len(words) == 0 {
			continue
		}
		// A heading that fits on one line is the whole line, which a body
		// line that merely starts with its words is not. A long heading
		// wraps, so then its first one to three words will do.
		found := -1
		whole := strings.Join(words, " ")
		for i := cur; i < len(lines) && found < 0; i++ {
			if lines[i] == whole {
				found = i
			}
		}
		for n := min(3, len(words)); n >= 1 && found < 0; n-- {
			want := strings.Join(words[:n], " ")
			for i := cur; i < len(lines); i++ {
				if strings.HasPrefix(lines[i], want) {
					found = i
					break
				}
			}
		}
		if found >= 0 {
			out = append(out, anchor{sl, found})
			cur = found + 1
		}
	}
	return out
}

/** LineOfFragment returns the line of Display's output where the heading a
 * fragment names starts. The fragment is a heading's slug, as markdownweb's
 * slug package makes it; a leading "#" is ignored and the match is exact.
 *
 * Parameters:
 *   fragment (string) — the part of a link after "#"
 *
 * Returns:
 *   int — the zero-based line
 *   bool — false when no heading has that slug, or its line was not found
 *
 * Example:
 *   line, ok := v.LineOfFragment("soil--seeds")
 */
func (v *View) LineOfFragment(fragment string) (int, bool) {
	fragment = strings.TrimPrefix(fragment, "#")
	for _, a := range v.anchors {
		if a.slug == fragment {
			return a.line, true
		}
	}
	return 0, false
}
