package render

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
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
			pos, ok := lastTextStop(v)
			from, marked := pos, false
			// A picture inside the link shows its alt text; the marker goes after
			// the last one when the link's text ends inside it.
			for _, img := range imagesIn(v) {
				s, e, alt, iok := imageSpan(b, img)
				if !iok {
					continue
				}
				skip[img] = true
				repl := ""
				if alt != "" {
					repl = `\[image: ` + escapeAlt(alt) + `\]`
				}
				if ok && !marked && lastStopOf(img) == pos {
					repl += fmt.Sprintf(" %s\\[%d\\]", Sentinel, len(links))
					marked, from = true, e
				}
				edits = append(edits, edit{s, e, repl})
			}
			if ok {
				if !marked {
					edits = append(edits, edit{pos, pos, fmt.Sprintf(" %s\\[%d\\]", Sentinel, len(links))})
				}
				// Glamour prints a link's URL after its text. The Links list and
				// the status line give the address, so hand glamour a bare "#",
				// which it prints nothing for. Link.Dest keeps the real one.
				if s, e, ok := destSpan(b, from, string(v.Destination)); ok {
					edits = append(edits, edit{s, e, "(#)"})
				}
			}
		case *ast.Image:
			// A picture with alt text is a link whose text is the alt text, so the
			// address can be followed. One with none is decorative: left to the
			// glamour style, which prints nothing for it.
			if s, e, alt, ok := imageSpan(b, v); ok && alt != "" {
				links = append(links, Link{N: len(links) + 1, Dest: string(v.Destination), Text: "image: " + alt})
				edits = append(edits, edit{s, e, fmt.Sprintf(`[\[image: %s\] %s\[%d\]](#)`, escapeAlt(alt), Sentinel, len(links))})
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

/** destSpan finds the "(destination "title")" part of an inline link whose text
 * ended at from. It reports the span only when the destination it reads equals
 * dest (what the parser found), so a reference-style link or anything unusual
 * is left alone.
 *
 * Parameters:
 *   b ([]byte) — the Markdown source
 *   from (int) — the end of the link's last text
 *   dest (string) — the destination the parser reported
 *
 * Returns:
 *   int, int — the offsets of the "(" and just after the ")"
 *   bool — true when the span was found and matches dest
 *
 * Example:
 *   from, to, ok := destSpan([]byte("[a](b.md) c"), 2, "b.md") // 3, 9, true
 */
func destSpan(b []byte, from int, dest string) (int, int, bool) {
	for at := from; at < len(b); {
		i := bytes.Index(b[at:], []byte("]("))
		if i < 0 {
			return 0, 0, false
		}
		open := at + i + 1
		if raw, end, ok := parseDest(b, open); ok && string(util.UnescapePunctuations(raw)) == dest {
			return open, end, true
		}
		at = open
	}
	return 0, 0, false
}

// parseDest reads "(destination "title")" with the "(" at b[open]. It returns
// the destination as written and the offset just after the ")".
func parseDest(b []byte, open int) ([]byte, int, bool) {
	j := open + 1
	for j < len(b) && (b[j] == ' ' || b[j] == '\t' || b[j] == '\n') {
		j++
	}
	var raw []byte
	if j < len(b) && b[j] == '<' {
		k := j + 1
		for k < len(b) && b[k] != '>' && b[k] != '\n' {
			if b[k] == '\\' && k+1 < len(b) {
				k++
			}
			k++
		}
		if k >= len(b) || b[k] != '>' {
			return nil, 0, false
		}
		raw = b[j+1 : k]
		j = k + 1
	} else {
		k, depth := j, 0
		for k < len(b) && b[k] != ' ' && b[k] != '\t' && b[k] != '\n' {
			if b[k] == '\\' && k+1 < len(b) {
				k += 2
				continue
			}
			if b[k] == '(' {
				depth++
			} else if b[k] == ')' {
				if depth == 0 {
					break
				}
				depth--
			}
			k++
		}
		raw = b[j:k]
		j = k
	}
	for j < len(b) && (b[j] == ' ' || b[j] == '\t' || b[j] == '\n') {
		j++
	}
	if j < len(b) && (b[j] == '"' || b[j] == '\'' || b[j] == '(') {
		closer := b[j]
		if closer == '(' {
			closer = ')'
		}
		j++
		for j < len(b) && b[j] != closer {
			if b[j] == '\\' && j+1 < len(b) {
				j++
			}
			j++
		}
		if j >= len(b) {
			return nil, 0, false
		}
		j++
		for j < len(b) && (b[j] == ' ' || b[j] == '\t' || b[j] == '\n') {
			j++
		}
	}
	if j >= len(b) || b[j] != ')' {
		return nil, 0, false
	}
	return raw, j + 1, true
}

// imagesIn returns the images under n, in reading order.
func imagesIn(n ast.Node) []*ast.Image {
	var out []*ast.Image
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if i, ok := c.(*ast.Image); ok && entering {
			out = append(out, i)
		}
		return ast.WalkContinue, nil
	})
	return out
}

// lastStopOf is lastTextStop without the found flag, 0 when n has no text.
func lastStopOf(n ast.Node) int {
	pos, _ := lastTextStop(n)
	return pos
}

/** imageSpan finds a picture's whole "![alt](destination)" in the source. It
 * needs some alt text to anchor on, and reports the span only when the
 * destination it reads equals the one the parser found (see destSpan), so a
 * picture with no alt text, or anything unusual, is left alone.
 *
 * Parameters:
 *   b ([]byte) — the Markdown source
 *   img (*ast.Image) — the picture node
 *
 * Returns:
 *   int, int — the offsets of the "!" and just after the ")"
 *   string — the alt text, markup removed and white space collapsed
 *   bool — true when the span was found
 *
 * Example:
 *   from, to, alt, ok := imageSpan([]byte("a ![cat](c.png) b"), img) // 2, 15, "cat", true
 */
func imageSpan(b []byte, img *ast.Image) (int, int, string, bool) {
	first, found := -1, false
	_ = ast.Walk(img, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if t, ok := c.(*ast.Text); ok && entering && !found {
			first, found = t.Segment.Start, true
		}
		return ast.WalkContinue, nil
	})
	if !found {
		return 0, 0, "", false
	}
	start := bytes.LastIndex(b[:first], []byte("!["))
	if start < 0 {
		return 0, 0, "", false
	}
	stop, _ := lastTextStop(img)
	_, to, ok := destSpan(b, stop, string(img.Destination))
	if !ok {
		return 0, 0, "", false
	}
	alt := strings.Join(strings.Fields(string(util.UnescapePunctuations([]byte(plainText(img, b))))), " ")
	return start, to, alt, true
}

// altEscaper makes alt text safe to put back into Markdown as plain text.
var altEscaper = strings.NewReplacer(
	`\`, `\\`, "[", `\[`, "]", `\]`, "*", `\*`, "_", `\_`, "`", "\\`",
	"<", `\<`, ">", `\>`, "&", `\&`, "!", `\!`, "|", `\|`, "#", `\#`,
)

func escapeAlt(s string) string { return altEscaper.Replace(s) }

/** MarginImage returns Markdown for the picture a page names in its front matter
 * (rightMarginImage), so it is shown like any other picture: as its alt text
 * (rightMarginImageAlt), a link to follow. A terminal cannot float it into the
 * margin. With no alt text the picture is decorative and nothing is shown.
 *
 * Parameters:
 *   alt (string) — the alt text from the front matter
 *   dest (string) — the picture's address from the front matter
 *
 * Returns:
 *   string — "![alt](<dest>)" and a blank line, or "" when there is nothing to
 *     show or the values cannot be quoted safely
 *
 * Example:
 *   md := render.MarginImage("A hill", "/hill.png") // "![A hill](</hill.png>)\n\n"
 */
func MarginImage(alt, dest string) string {
	alt = strings.Join(strings.Fields(alt), " ")
	dest = strings.TrimSpace(dest)
	if alt == "" || dest == "" || strings.ContainsAny(dest, "<>\n\r") {
		return ""
	}
	return "![" + escapeAlt(alt) + "](<" + dest + ">)\n\n"
}
