package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

/** Sentinel is a zero-width character placed before each link's [N] marker, so
 * the marker can be found in rendered text without being confused with page
 * text that happens to look like [3], such as a footnote. It is removed
 * before anything is shown.
 */
const Sentinel = "\u2060"

/** Link is a link found in a page, in reading order.
 *
 * Fields:
 *   N (int) — its number, from 1, shown after the link text as [N]
 *   Dest (string) — the destination as written, possibly relative
 *   Text (string) — the plain link text
 */
type Link struct {
	N    int
	Dest string
	Text string
}

// edit replaces src[start:stop] with repl.
type edit struct {
	start, stop int
	repl        string
}

/** Prepare makes a page body safe and followable before rendering. It removes
 * HTML the way BonBon does (block HTML, inline tags, comments, and the content
 * of script and style elements), leaves code spans and code blocks alone, and
 * numbers each link by adding " [N]" after its text (with a Sentinel before the bracket). Images are not links.
 *
 * Parameters:
 *   src (string) — the Markdown body
 *
 * Returns:
 *   string — the cleaned Markdown
 *   []Link — the links in reading order
 *
 * Example:
 *   md, links := render.Prepare("See [one](a.md).") // "See [one \\[1\\]](a.md).", one link
 */
func Prepare(src string) (string, []Link) {
	b := []byte(src)
	doc := goldmark.New().Parser().Parse(text.NewReader(b))
	var edits []edit
	var links []Link
	skip := map[ast.Node]bool{}

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || skip[n] {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.HTMLBlock:
			if lines := v.Lines(); lines.Len() > 0 {
				stop := lines.At(lines.Len() - 1).Stop
				if v.HasClosure() {
					stop = v.ClosureLine.Stop
				}
				edits = append(edits, edit{lines.At(0).Start, stop, ""})
			} else if v.HasClosure() {
				edits = append(edits, edit{v.ClosureLine.Start, v.ClosureLine.Stop, ""})
			}
		case *ast.RawHTML:
			segs := v.Segments
			if segs.Len() == 0 {
				break
			}
			start, stop := segs.At(0).Start, segs.At(segs.Len()-1).Stop
			if tag := rawTag(b[start:stop]); tag == "script" || tag == "style" {
				// Remove everything up to the matching close tag.
				for s := n.NextSibling(); s != nil; s = s.NextSibling() {
					skip[s] = true
					if rh, ok := s.(*ast.RawHTML); ok && rh.Segments.Len() > 0 {
						ss := rh.Segments
						if t := closeTag(b[ss.At(0).Start:ss.At(ss.Len()-1).Stop]); t == tag {
							stop = ss.At(ss.Len() - 1).Stop
							break
						}
					}
				}
			}
			edits = append(edits, edit{start, stop, ""})
		case *ast.Link:
			links = append(links, Link{N: len(links) + 1, Dest: string(v.Destination), Text: plainText(v, b)})
			if pos, ok := lastTextStop(v); ok {
				edits = append(edits, edit{pos, pos, fmt.Sprintf(" %s\\[%d\\]", Sentinel, len(links))})
			}
		}
		return ast.WalkContinue, nil
	})

	if len(edits) == 0 {
		return src, links
	}
	return applyEdits(b, edits), links
}

/** applyEdits applies edits to b in one pass over the text and returns the
 * result. It gives the answer the one-edit-at-a-time algorithm gave (apply from
 * the end; an edit whose end passes the start of the one after it is dropped)
 * without copying the whole text for every edit, which took 17 seconds on a
 * 4.7 MB page with 8,000 links.
 *
 * Parameters:
 *   b ([]byte) — the source
 *   edits ([]edit) — replacements of b[start:stop]; reordered
 *
 * Returns:
 *   string — b with the edits applied
 *
 * Example:
 *   out := applyEdits([]byte("abc"), []edit{{1, 2, "XY"}}) // "aXYc"
 */
func applyEdits(b []byte, edits []edit) string {
	if len(edits) == 0 {
		return string(b)
	}
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	// Decide which edits are kept, from the end, as before.
	kept := edits[:0:0]
	lastStart := len(b) + 1
	for _, e := range edits {
		if e.stop > lastStart {
			continue
		}
		kept = append(kept, e)
		lastStart = e.start
	}
	// kept runs from the end of the text to the start; write it forwards.
	var sb strings.Builder
	sb.Grow(len(b) + 16*len(kept))
	pos := 0
	for i := len(kept) - 1; i >= 0; i-- {
		e := kept[i]
		sb.Write(b[pos:e.start])
		sb.WriteString(e.repl)
		pos = e.stop
	}
	sb.Write(b[pos:])
	return sb.String()
}

// rawTag returns the lower-case name of an opening tag such as "<script src=x>", or "".
func rawTag(s []byte) string {
	t := strings.ToLower(string(s))
	if !strings.HasPrefix(t, "<") || strings.HasPrefix(t, "</") || strings.HasPrefix(t, "<!") {
		return ""
	}
	t = t[1:]
	end := strings.IndexAny(t, " \t\n/>")
	if end < 0 {
		return t
	}
	return t[:end]
}

// closeTag returns the lower-case name of a closing tag such as "</script>", or "".
func closeTag(s []byte) string {
	t := strings.ToLower(string(s))
	if !strings.HasPrefix(t, "</") {
		return ""
	}
	return strings.TrimSpace(strings.TrimSuffix(t[2:], ">"))
}

// lastTextStop returns the end offset of the last text segment inside n.
func lastTextStop(n ast.Node) (int, bool) {
	pos, found := 0, false
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if t, ok := c.(*ast.Text); ok && entering {
			pos, found = t.Segment.Stop, true
		}
		return ast.WalkContinue, nil
	})
	return pos, found
}

// plainText returns the text under n without markup.
func plainText(n ast.Node, src []byte) string {
	var sb strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if t, ok := c.(*ast.Text); ok && entering {
			sb.Write(t.Segment.Value(src))
		}
		return ast.WalkContinue, nil
	})
	return sb.String()
}
