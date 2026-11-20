package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

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
 * numbers each link by adding " [N]" after its text. Images are not links.
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
				edits = append(edits, edit{pos, pos, fmt.Sprintf(" \\[%d\\]", len(links))})
			}
		}
		return ast.WalkContinue, nil
	})

	if len(edits) == 0 {
		return src, links
	}
	// Apply from the end so earlier offsets stay valid; a later edit that
	// starts inside an earlier removed range is dropped.
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	out := b
	lastStart := len(b) + 1
	for _, e := range edits {
		if e.stop > lastStart {
			continue
		}
		out = append(append(append([]byte{}, out[:e.start]...), e.repl...), out[e.stop:]...)
		lastStart = e.start
	}
	return string(out), links
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
