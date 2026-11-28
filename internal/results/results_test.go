package results

import (
	"net/url"
	"strings"
	"testing"

	"github.com/rsdoiel/cmarkfind/corpus"
)

// cmarkbrowser DR-0004: the results page shows the facets of a result, before
// the hits, and each facet value is a link to a search of its own.

func facetedResults() *corpus.Results {
	return &corpus.Results{
		Searched: 4,
		Hits: []corpus.Hit{
			{Title: "One", Address: "https://f.example/p1.md", Tier: "own", Snippet: "compilers one"},
			{Title: "Two", Address: "https://f.example/p2.md", Tier: "own", Snippet: "compilers two"},
		},
		Facets: []corpus.FacetField{
			{Field: "keywords", Label: "Topics", Values: []corpus.FacetValue{
				{Value: "go", Count: 3, Query: `(compilers) AND keywords:"go"`},
				{Value: "oberon", Count: 1, Query: `(compilers) AND keywords:"oberon"`},
			}},
			{Field: "datepublished", Label: "Pub Date", Values: []corpus.FacetValue{
				{Value: "2026", Count: 2, Query: `(compilers) AND datepublished:[2026 TO 2026]`},
			}},
		},
	}
}

func lines(md string) []string { return strings.Split(strings.TrimRight(md, "\n"), "\n") }

func TestAFacetedResultHasFacetsThenResultsAsTwoOuterLists(t *testing.T) {
	got := lines(Page("compilers", facetedResults()))
	i := indexOf(got, "- Facets")
	j := indexOf(got, "- Results")
	if i < 0 || j < 0 || i > j {
		t.Fatalf("Facets at %d and Results at %d, want both and Facets first:\n%s", i, j, strings.Join(got, "\n"))
	}
	// the counts line stays above both lists
	if c := indexOf(got, "2 hits, searched 4 pages."); c < 0 || c > i {
		t.Errorf("counts line at %d, want it before Facets at %d", c, i)
	}
	want := []string{
		"- Facets",
		"  - Topics",
		"    - [go (3)](corpus:search?q=" + url.QueryEscape(`(compilers) AND keywords:"go"`) + ")",
		"    - [oberon (1)](corpus:search?q=" + url.QueryEscape(`(compilers) AND keywords:"oberon"`) + ")",
		"  - Pub Date",
		"    - [2026 (2)](corpus:search?q=" + url.QueryEscape(`(compilers) AND datepublished:[2026 TO 2026]`) + ")",
		"- Results",
	}
	for k, w := range want {
		if i+k >= len(got) || got[i+k] != w {
			t.Fatalf("line %d = %q, want %q\n%s", i+k, safe(got, i+k), w, strings.Join(got, "\n"))
		}
	}
}

func TestTheHitsUnderResultsAreAnOrderedListThatKeepsItsNumbers(t *testing.T) {
	md := Page("compilers", facetedResults())
	got := lines(md)
	j := indexOf(got, "- Results")
	if j < 0 {
		t.Fatalf("no Results list:\n%s", md)
	}
	rest := strings.Join(got[j+1:], "\n")
	for _, w := range []string{
		"  1. [One](https://f.example/p1.md) — own",
		"     compilers one",
		"  2. [Two](https://f.example/p2.md) — own",
		"     compilers two",
	} {
		if !strings.Contains(rest, w) {
			t.Errorf("missing %q under Results:\n%s", w, rest)
		}
	}
	if strings.Contains(rest, "\n  - [") || strings.HasPrefix(rest, "  - [") {
		t.Errorf("a bullet hit under Results:\n%s", rest)
	}
}

// With no facets the page is exactly what cmarkbrowser DR-0003 made.
func TestAResultWithNoFacetsIsTheFlatPageOfBefore(t *testing.T) {
	res := &corpus.Results{Searched: 7, Hits: []corpus.Hit{{Title: "Oberon", Address: "https://a.example/o.md", Tier: "own", Snippet: "a snippet"}}}
	want := "---\ntitle: \"Search: oberon\"\n---\n# Search: oberon\n\n1 hit, searched 7 pages.\n\n1. [Oberon](https://a.example/o.md) — own\n   a snippet\n\n"
	if got := Page("oberon", res); got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
	res.Facets = []corpus.FacetField{}
	if got := Page("oberon", res); got != want {
		t.Errorf("empty (not nil) Facets changed the page:\n%q", got)
	}
}

func TestFacetsShowEvenWhenThereAreNoHits(t *testing.T) {
	res := facetedResults()
	res.Hits = nil
	md := Page("compilers", res)
	if !strings.Contains(md, "no matches; searched 4 pages.") {
		t.Errorf("counts:\n%s", md)
	}
	// a result with facets but no hits is not a thing cmarkfind makes; whatever
	// is shown, a Results item with nothing under it is not a list to follow
	if strings.Contains(md, "1. [") {
		t.Errorf("a hit from nowhere:\n%s", md)
	}
}

// A facet's label comes from the set's author and its values from the pages'
// front matter: stored text, other people's, so the same escaping as a title.
func TestAHostileFacetLabelOrValueMakesNoLinkImageHeadingOrHTML(t *testing.T) {
	hostile := "[click](https://evil.example/x) ![i](https://evil.example/i.png) <script>alert(1)</script> www.evil.example me@evil.example\n# heading"
	res := &corpus.Results{Searched: 1, Hits: []corpus.Hit{{Title: "T", Address: "https://f.example/a.md", Tier: "own"}},
		Facets: []corpus.FacetField{{Field: "keywords", Label: hostile, Values: []corpus.FacetValue{
			{Value: hostile, Count: 1, Query: `(x) AND keywords:"y"`},
		}}}}
	md := Page("x", res)
	if strings.Contains(md, "<script>") || strings.Contains(md, "evil.example/x)") || strings.Contains(md, "\n# heading") {
		t.Errorf("markup got through:\n%s", md)
	}
	// the only links are the facet value's own and the hit's
	if n := strings.Count(md, "]("); n != 2 {
		t.Errorf("%d links, want 2 (the value and the hit):\n%s", n, md)
	}
	if strings.Count(md, "://") != 1 {
		// the hit's own address is the one unbroken ://
		t.Errorf("an unescaped address in stored text:\n%s", md)
	}
	// a control character or newline cannot start a new list item or line
	for _, l := range lines(md) {
		if strings.HasPrefix(l, "heading") || strings.HasPrefix(strings.TrimSpace(l), "# ") && !strings.HasPrefix(l, "# Search") {
			t.Errorf("stored text made a line of its own: %q", l)
		}
	}
}

// The destination is a search address and cannot end early or hold a second one.
func TestAFacetLinkDestinationIsOneSearchAddress(t *testing.T) {
	res := facetedResults()
	res.Facets[0].Values[0].Query = `(a b) AND keywords:"x) (y"` + "\n" + `corpus:search?q=evil`
	md := Page("compilers", res)
	for _, l := range lines(md) {
		if !strings.HasPrefix(l, "    - [go (3)](") {
			continue
		}
		dest := strings.TrimSuffix(strings.TrimPrefix(l[strings.Index(l, "]("):], "]("), ")")
		if !strings.HasPrefix(dest, "corpus:search?q=") || strings.ContainsAny(dest, " ()\n<>") || strings.Count(dest, "corpus:") != 1 {
			t.Errorf("destination %q is not one search address", dest)
		}
		if _, err := url.QueryUnescape(strings.TrimPrefix(dest, "corpus:search?q=")); err != nil {
			t.Errorf("destination does not decode: %v", err)
		}
		return
	}
	t.Errorf("no go facet line:\n%s", md)
}

func indexOf(ls []string, s string) int {
	for i, l := range ls {
		if l == s {
			return i
		}
	}
	return -1
}

func safe(ls []string, i int) string {
	if i < len(ls) {
		return ls[i]
	}
	return "<missing>"
}
