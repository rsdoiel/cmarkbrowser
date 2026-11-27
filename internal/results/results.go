// Package results makes the Markdown page that shows a search of the corpus
// (cmarkbrowser DR-0003 3): the same page in the browser and in -find.
package results

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rsdoiel/cmarkfind/corpus"
)

/** Page makes the Markdown of a search results page. Everything that
 * came out of the corpus (titles, snippets, addresses) is other people's text:
 * it is escaped so it can make no link, image, heading or HTML of its own, and
 * only the hit itself is a link.
 *
 * Parameters:
 *   q (string)             — the query, as typed
 *   res (*corpus.Results)  — the hits and counts
 *
 * Returns:
 *   string — Markdown with front matter
 *
 * Example:
 *   md := results.Page("oberon", res)
 */
func Page(q string, res *corpus.Results) string {
	var b strings.Builder
	title, _ := json.Marshal("Search: " + oneLine(q))
	fmt.Fprintf(&b, "---\ntitle: %s\n---\n# Search: %s\n\n", title, esc(oneLine(q)))
	counts := fmt.Sprintf("%d %s, searched %d %s", len(res.Hits), plural(len(res.Hits), "hit", "hits"), res.Searched, plural(res.Searched, "page", "pages"))
	if len(res.Hits) == 0 {
		counts = fmt.Sprintf("no matches; searched %d %s", res.Searched, plural(res.Searched, "page", "pages"))
	}
	if res.Pending > 0 {
		counts += fmt.Sprintf("; %d listed %s not fetched yet", res.Pending, plural(res.Pending, "page", "pages"))
	}
	fmt.Fprintf(&b, "%s.\n\n", counts)
	for i, h := range res.Hits {
		t := oneLine(h.Title)
		if t == "" {
			t = oneLine(h.Address)
		}
		fmt.Fprintf(&b, "%d. [%s](%s) — %s\n", i+1, esc(t), destination(h.Address), esc(h.Tier))
		if sn := oneLine(h.Snippet); sn != "" {
			fmt.Fprintf(&b, "   %s\n", esc(sn))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// oneLine puts stored text on one line, without control characters.
func oneLine(s string) string {
	return strings.Join(strings.FieldsFunc(s, func(r rune) bool { return r < ' ' || r == 0x7f }), " ")
}

// esc makes stored text plain text: a backslash before each character that
// Markdown reads as structure (the glamour renderer shows a backslash before
// some other punctuation as it is, so only these are escaped), and a
// zero-width space inside what a reader would turn into an automatic link
// (an address with ://, one starting www., an e-mail address).
func esc(s string) string {
	s = strings.NewReplacer("://", ":\u200b//", "www.", "www\u200b.", "@", "\u200b@").Replace(s)
	var b strings.Builder
	for _, r := range s {
		if r < 0x80 && strings.ContainsRune("\\`*_[]()<>!#&|~{}", r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// destination makes a link destination that cannot end early.
func destination(addr string) string {
	return strings.NewReplacer(" ", "%20", "(", "%28", ")", "%29", "<", "%3C", ">", "%3E", "\\", "%5C", "\n", "", "\r", "").Replace(addr)
}
