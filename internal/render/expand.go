package render

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// macroPattern matches {{page.KEY}}, where KEY is letters, digits, underscore,
// hyphen and dots (a dot reaches into a nested front matter table).
var macroPattern = regexp.MustCompile(`\{\{page\.([A-Za-z0-9_.-]+)\}\}`)

/** Expand replaces {{page.KEY}} macros in a Markdown body with values from the
 * page's front matter, the way BonBon pages use them (a story starts with
 * `# {{page.title}}`). It expands only in text: never in code spans, fenced
 * or indented code blocks. It makes one pass, so a value that itself holds a
 * macro is shown, not expanded. A value is escaped so it cannot become
 * markup, a link or HTML. A key that is absent, or holds a table, leaves the
 * macro visible in the text and is reported. A dotted key reaches into a
 * nested table (`{{page.source.title}}`); a list shows its items joined with
 * commas.
 *
 * Parameters:
 *   src (string) — the Markdown body
 *   meta (map[string]any) — the page's front matter
 *
 * Returns:
 *   string — the body with macros expanded, or src itself when it has none
 *   []string — the keys that could not be expanded, in order, without repeats
 *
 * Example:
 *   md, missing := render.Expand("# {{page.title}}", map[string]any{"title": "Hi"}) // "# Hi", nil
 */
func Expand(src string, meta map[string]any) (string, []string) {
	if !macroPattern.MatchString(src) {
		return src, nil
	}
	b := []byte(src)
	doc := goldmark.New().Parser().Parse(text.NewReader(b))
	var edits []edit
	var missing []string
	seen := map[string]bool{}
	done := map[ast.Node]bool{}

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.CodeSpan:
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			if done[v] {
				return ast.WalkContinue, nil
			}
			// goldmark splits a run of text at characters such as _ and *, so
			// join the touching text nodes and look for macros in the run.
			start, stop := v.Segment.Start, v.Segment.Stop
			for s := v.NextSibling(); s != nil; s = s.NextSibling() {
				t, ok := s.(*ast.Text)
				if !ok || t.Segment.Start != stop {
					break
				}
				done[t] = true
				stop = t.Segment.Stop
			}
			for _, m := range macroPattern.FindAllSubmatchIndex(b[start:stop], -1) {
				key := string(b[start+m[2] : start+m[3]])
				val, ok := lookupMeta(meta, key)
				if !ok {
					if !seen[key] {
						seen[key] = true
						missing = append(missing, key)
					}
					continue
				}
				edits = append(edits, edit{start + m[0], start + m[1], escapeValue(val)})
			}
		}
		return ast.WalkContinue, nil
	})

	return applyEdits(b, edits), missing
}

// lookupMeta finds a dotted key in front matter and returns its text. ok is
// false for an absent key, a null, or a table.
func lookupMeta(meta map[string]any, key string) (string, bool) {
	var cur any = meta
	for _, part := range strings.Split(key, ".") {
		m, isMap := cur.(map[string]any)
		if !isMap {
			return "", false
		}
		cur, isMap = m[part]
		if !isMap {
			return "", false
		}
	}
	switch v := cur.(type) {
	case nil, map[string]any:
		return "", false
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			switch item.(type) {
			case nil, map[string]any, []any:
				continue
			}
			parts = append(parts, fmt.Sprint(item))
		}
		return strings.Join(parts, ", "), true
	default:
		return fmt.Sprint(v), true
	}
}

// escapeValue makes a front matter value inert as Markdown: one line, with
// every character that could start markup, a link, HTML or a macro escaped.
func escapeValue(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '\\', '`', '*', '_', '[', ']', '<', '>', '&', '~', '|', '{', '}', '!':
			sb.WriteByte('\\')
		}
		sb.WriteRune(r)
	}
	return sb.String()
}
