package ui

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/rsdoiel/cmarkfind/corpus"
	"github.com/rsdoiel/markdownweb/toc"
)

// cmarkbrowser DR-0004 through the browser: a corpus whose set opted into
// facets, S, and following a facet link.

var facetPages = map[string]string{
	"p1.md": "---\ndatePublished: 2026-03-01\nkeywords: [Go, Oberon]\n---\n# One\n\ncompilers one\n",
	"p2.md": "---\ndatePublished: 2026-04-02\nkeywords: [go]\n---\n# Two\n\ncompilers two\n",
	"p3.md": "---\nkeywords: \"go, wasm\"\n---\n# Three\n\ncompilers three\n",
	"p4.md": "# Four\n\ncompilers four, no front matter\n",
}

// facetedCorpus fills a local site of facetPages, has the set facet on its
// keywords and publication date, and returns the corpus file.
func facetedCorpus(t *testing.T, optIn bool) string {
	t.Helper()
	root := t.TempDir()
	var lines strings.Builder
	for name, body := range facetPages {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		lines.WriteString(`<outline text="` + name + `" url="https://f.example/` + name + `"/>`)
	}
	os.MkdirAll(filepath.Join(root, ".well-known"), 0o755)
	os.WriteFile(filepath.Join(root, ".well-known", "toc.opml"),
		[]byte(`<opml version="2.0"><head/><body>`+lines.String()+`</body></opml>`), 0o644)
	path := corpusFile(t)
	db, err := corpus.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if _, err := db.FillLocal(ctx, root, toc.Limits{}); err != nil {
		t.Fatal(err)
	}
	if optIn {
		if err := db.AssignFields(ctx, root, []string{`datePublished=date,facet,"Pub Date"`, "keywords=facet,fielded,Topics"}); err != nil {
			t.Fatal(err)
		}
		if _, err := db.BuildSets(ctx, "", 0); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

// facetLink is the number of the link whose address holds want, or 0.
func facetLink(m Model, want string) int {
	for i, l := range m.view.Links {
		if strings.HasPrefix(l.URL, corpusScheme) && strings.Contains(l.URL, want) {
			return i + 1
		}
	}
	return 0
}

func searchFor(t *testing.T, m Model, q string) Model {
	t.Helper()
	m = press(t, m, "S")
	m = typeText(t, m, q)
	return press(t, m, "enter")
}

// numberKeys is the keys that type a link number.
func numberKeys(n int) []string {
	var ks []string
	for _, r := range strconv.Itoa(n) {
		ks = append(ks, string(r))
	}
	return ks
}

func TestAResultsPageShowsTheFacetsBeforeTheHitsAndTheirLinksComeFirst(t *testing.T) {
	srv, _ := tocUISite(t)
	m := withCorpus(openTOC(t, srv, "/p/b.md"), facetedCorpus(t, true))
	m = searchFor(t, m, "compilers")
	out := m.View().Content
	for _, want := range []string{"Facets", "Topics", "go (3)", "oberon (1)", "wasm (1)", "Pub Date", "2026 (2)", "Results", "4 hits"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if strings.Index(out, "Facets") > strings.Index(out, "Results") {
		t.Errorf("Facets after Results:\n%s", out)
	}
	// 4 facet values (go, oberon, wasm, 2026) then 4 hits: the facets take the first numbers
	if len(m.view.Links) != 8 {
		t.Fatalf("%d links, want 4 facet values and 4 hits:\n%+v", len(m.view.Links), m.view.Links)
	}
	for i, l := range m.view.Links {
		if i < 4 && !strings.HasPrefix(l.URL, corpusScheme) {
			t.Errorf("link %d is %q, want a facet link first", i+1, l.URL)
		}
		if i >= 4 && !strings.HasPrefix(l.URL, "https://f.example/") {
			t.Errorf("link %d is %q, want a hit after the facets", i+1, l.URL)
		}
	}
}

func TestFollowingAFacetRunsItsQueryAndBackReturnsToTheFirstPage(t *testing.T) {
	srv, _ := tocUISite(t)
	m := withCorpus(openTOC(t, srv, "/p/b.md"), facetedCorpus(t, true))
	m = searchFor(t, m, "compilers")
	n := facetLink(m, url.QueryEscape(`keywords:"go"`))
	if n == 0 {
		t.Fatalf("no link for the go facet: %+v", m.view.Links)
	}
	m = press(t, m, append(numberKeys(n), "enter")...)
	if !strings.HasPrefix(m.view.Page.URL, corpusScheme) || !strings.Contains(m.view.Page.URL, url.QueryEscape(`keywords:"go"`)) {
		t.Fatalf("followed to %q", m.view.Page.URL)
	}
	out := m.View().Content
	if !strings.Contains(out, "3 hits") {
		t.Errorf("the go facet counted 3:\n%s", out)
	}
	if strings.Contains(out, "Four") {
		t.Errorf("the page with no front matter has no topic:\n%s", out)
	}
	// narrowing again: the page has facets of its own
	if !strings.Contains(out, "Topics") {
		t.Errorf("a narrowed page has its own facets:\n%s", out)
	}
	m = press(t, m, "b")
	back := m.View().Content
	if !strings.Contains(back, "4 hits") || !strings.Contains(back, "oberon (1)") {
		t.Errorf("back should show the first page, run again:\n%s", back)
	}
}

func TestAYearFacetIsFollowedTheSameWay(t *testing.T) {
	srv, _ := tocUISite(t)
	m := withCorpus(openTOC(t, srv, "/p/b.md"), facetedCorpus(t, true))
	m = searchFor(t, m, "compilers")
	n := facetLink(m, url.QueryEscape("datepublished:"))
	if n == 0 {
		t.Fatalf("no link for the year facet: %+v", m.view.Links)
	}
	m = press(t, m, append(numberKeys(n), "enter")...)
	out := m.View().Content
	if !strings.Contains(out, "2 hits") || !strings.Contains(out, "One") || !strings.Contains(out, "Two") {
		t.Errorf("2026 has two pages:\n%s", out)
	}
}

// A set that opted into nothing is the page DR-0003 made.
func TestASetWithNoFacetsShowsTheFlatPage(t *testing.T) {
	srv, _ := tocUISite(t)
	m := withCorpus(openTOC(t, srv, "/p/b.md"), facetedCorpus(t, false))
	m = searchFor(t, m, "compilers")
	out := m.View().Content
	if strings.Contains(out, "Facets") || strings.Contains(out, "Results") {
		t.Errorf("facets without an opt-in:\n%s", out)
	}
	if len(m.view.Links) != 4 {
		t.Errorf("%d links, want one per hit", len(m.view.Links))
	}
}

func TestAFacetPageIsNotRecordedInHistory(t *testing.T) {
	srv, _ := tocUISite(t)
	hist := filepath.Join(t.TempDir(), "history.md")
	m := openTOC(t, srv, "/p/b.md").WithHistory(hist)
	m = withCorpus(m, facetedCorpus(t, true))
	m = searchFor(t, m, "compilers")
	n := facetLink(m, url.QueryEscape(`keywords:"go"`))
	if n == 0 {
		t.Fatal("no go facet link")
	}
	m = press(t, m, append(numberKeys(n), "enter")...)
	b, _ := os.ReadFile(hist)
	if strings.Contains(string(b), "corpus:") {
		t.Errorf("history has a results page:\n%s", b)
	}
}
