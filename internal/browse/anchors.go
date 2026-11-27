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
)

// anchor is a heading of the page: its slug (markdownweb's rule) and the line
// of the rendered text it is on.
type anchor struct {
	slug string
	line int
}

// headingTexts returns the plain text of each ATX heading of Markdown src, in
// order, leaving out lines inside a code fence.
func headingTexts(src string) []string {
	var out []string
	fence := ""
	for _, line := range strings.Split(src, "\n") {
		if m := fenceRE.FindStringSubmatch(line); m != nil {
			switch fence {
			case "":
				fence = m[1]
			case m[1]:
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		if m := headingRE.FindStringSubmatch(line); m != nil {
			t := mdLinkRE.ReplaceAllString(markerStrp.ReplaceAllString(m[1], ""), "$1")
			out = append(out, strings.TrimSpace(markupStrp.Replace(t)))
		}
	}
	return out
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
		found := -1
		for n := min(3, len(words)); n >= 1 && found < 0; n-- { // a long heading may wrap
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
