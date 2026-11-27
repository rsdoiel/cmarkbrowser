package ui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rsdoiel/cmarkbrowser/internal/render"
	"github.com/rsdoiel/cmarkfind/corpus"
	"github.com/rsdoiel/markdownweb/toc"
)

// type presses each rune of s as a key.
func typeText(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		m = press(t, m, string(r))
	}
	return m
}

func corpusFile(t *testing.T) string { return filepath.Join(t.TempDir(), "corpus.db") }

// withCorpus gives the model a corpus and makes fills quick (no pause).
func withCorpus(m Model, path string) Model {
	m = m.WithCorpus(path, "cmarkbrowser-test")
	m.fillPause = 0
	return m
}

func searchCorpus(t *testing.T, path, q string) *corpus.Results {
	t.Helper()
	db, err := corpus.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	r, err := db.Search(context.Background(), q, 50)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// cmarkbrowser DR-0003 5: F fills one batch of the site that lists the page,
// and F again goes on.
func TestFFillsOneBatchAndAgainGoesOn(t *testing.T) {
	srv, _ := tocUISite(t)
	db := corpusFile(t)
	m := withCorpus(openTOC(t, srv, "/p/b.md"), db)
	m.fillBatch = 3
	m = press(t, m, "F")
	if !strings.Contains(m.status, "filled 3") || !strings.Contains(m.status, "4 pages not fetched yet") || !strings.Contains(m.status, "F for more") {
		t.Fatalf("status %q", m.status)
	}
	if got := searchCorpus(t, db, "page"); got.Searched != 3 {
		t.Errorf("searched %d", got.Searched)
	}
	m = press(t, m, "F", "F")
	if !strings.Contains(m.status, "all 7 pages") {
		t.Errorf("status %q", m.status)
	}
	if got := searchCorpus(t, db, "post"); len(got.Hits) != 3 {
		t.Errorf("post hits %d", len(got.Hits))
	}
}

func TestFOnASiteWithNoTableOfContentsSaysSo(t *testing.T) {
	srv, _ := tocUISite(t)
	m := withCorpus(openTOC(t, srv, "/lone.md"), corpusFile(t))
	m = press(t, m, "F")
	if !strings.Contains(m.status, "table of contents") {
		t.Errorf("status %q", m.status)
	}
}

func TestFFillsALocalSiteFromItsRoot(t *testing.T) {
	root := siteDir(t)
	db := corpusFile(t)
	m, _ := openSite(t, root, func(m Model) Model { return withCorpus(m, db) })
	m = press(t, m, "F")
	if !strings.Contains(m.status, "filled") {
		t.Fatalf("status %q", m.status)
	}
	if got := searchCorpus(t, db, "post"); len(got.Hits) != 2 {
		t.Errorf("hits %d", len(got.Hits))
	}
}

func fillAll(t *testing.T) (Model, string) {
	t.Helper()
	srv, _ := tocUISite(t)
	db := corpusFile(t)
	m := withCorpus(openTOC(t, srv, "/p/b.md"), db)
	m = press(t, m, "F")
	return m, db
}

func TestSAsksAQueryAndShowsTheResultsAsAPage(t *testing.T) {
	m, _ := fillAll(t)
	m = press(t, m, "S")
	m = typeText(t, m, "post")
	m = press(t, m, "enter")
	out := m.View().Content
	for _, want := range []string{"New post", "Mid post", "Old post", "3 hits", "searched 7 pages"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if len(m.view.Links) != 3 {
		t.Errorf("links %d, want one per hit", len(m.view.Links))
	}
	// follow a hit, and back returns to the results, run again
	m = press(t, m, "1", "enter")
	if !strings.Contains(m.view.Page.URL, "/blog/") {
		t.Fatalf("followed to %q", m.view.Page.URL)
	}
	m = press(t, m, "b")
	if !strings.Contains(m.View().Content, "3 hits") {
		t.Errorf("back:\n%s", m.View().Content)
	}
}

func TestResultsSayWhatIsNotFetchedYet(t *testing.T) {
	srv, _ := tocUISite(t)
	db := corpusFile(t)
	m := withCorpus(openTOC(t, srv, "/p/b.md"), db)
	m.fillBatch = 3
	m = press(t, m, "F", "S")
	m = typeText(t, m, "page")
	m = press(t, m, "enter")
	if out := m.View().Content; !strings.Contains(out, "4 listed pages not fetched yet") {
		t.Errorf("results:\n%s", out)
	}
}

func TestNoMatchesAndBadSyntaxAreAPageNotAnError(t *testing.T) {
	m, _ := fillAll(t)
	m = press(t, m, "S")
	m = typeText(t, m, "zebra")
	m = press(t, m, "enter")
	if out := m.View().Content; !strings.Contains(out, "no matches") {
		t.Errorf("no match:\n%s", out)
	}
	m = press(t, m, "S")
	m = typeText(t, m, `"unclosed`)
	m = press(t, m, "enter")
	if m.status != "" || !strings.Contains(m.View().Content, "Search") {
		t.Errorf("bad syntax: status %q\n%s", m.status, m.View().Content)
	}
}

func TestEscapeInThePromptCancelsAndAnEmptyQueryDoesNothing(t *testing.T) {
	m, _ := fillAll(t)
	before := m.view.Page.URL
	m = press(t, m, "S", "x", "esc")
	if m.view.Page.URL != before || m.findPrompt {
		t.Error("esc should cancel")
	}
	m = press(t, m, "S", "enter")
	if m.view.Page.URL != before {
		t.Error("an empty query should do nothing")
	}
}

// Stored text is other people's: a snippet shows as text and makes no link.
func TestAHostileSnippetMakesNoLinkImageHeadingOrHTML(t *testing.T) {
	root := t.TempDir()
	body := "# Innocent\n\nzebra https://evil.example/bare www.evil.example me@evil.example [click](https://evil.example/x) ![img](https://evil.example/i.png) <script>alert(1)</script> <b>bold</b> it's, \"fine\": ok/yes\n\n# Big heading zebra\n"
	os.WriteFile(filepath.Join(root, "a.md"), []byte(body), 0o644)
	os.MkdirAll(filepath.Join(root, ".well-known"), 0o755)
	os.WriteFile(filepath.Join(root, ".well-known", "toc.opml"),
		[]byte(`<opml version="2.0"><head/><body><outline text="a" url="https://pub.example/a.md"/></body></opml>`), 0o644)
	db := corpusFile(t)
	cdb, err := corpus.Open(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cdb.FillLocal(context.Background(), root, toc.Limits{}); err != nil {
		t.Fatal(err)
	}
	cdb.Close()
	srv, _ := tocUISite(t)
	m := withCorpus(openTOC(t, srv, "/p/b.md"), db)
	m = press(t, m, "S")
	m = typeText(t, m, "zebra")
	m = press(t, m, "enter")
	if len(m.view.Links) != 1 || m.view.Links[0].URL != "https://pub.example/a.md" {
		t.Errorf("links %+v: only the hit may be a link", m.view.Links)
	}
	out := m.View().Content
	if strings.Contains(out, "\x1b]8;") {
		t.Errorf("the terminal was handed a hyperlink made from stored text:\n%q", out)
	}
	if strings.Contains(out, "<script>") && !strings.Contains(out, "alert") {
		t.Errorf("odd:\n%s", out)
	}
	if strings.Contains(out, `\'`) || strings.Contains(out, `\,`) || strings.Contains(out, `\"`) {
		t.Errorf("a backslash leaked into the text:\n%s", out)
	}
	if !strings.Contains(out, "evil.example") {
		t.Errorf("the text should still be shown, as text:\n%s", out)
	}
}

func TestSearchIsOffWithoutACorpusAndSaysWhatIsMissing(t *testing.T) {
	srv, _ := tocUISite(t)
	m := openTOC(t, srv, "/p/b.md") // no WithCorpus: -no-corpus
	m = press(t, m, "S")
	if m.findPrompt || !strings.Contains(m.status, "search is off") {
		t.Errorf("status %q", m.status)
	}
	m = press(t, m, "F")
	if !strings.Contains(m.status, "search is off") {
		t.Errorf("status %q", m.status)
	}
	missing := filepath.Join(t.TempDir(), "none.db")
	m = withCorpus(openTOC(t, srv, "/p/b.md"), missing)
	m = press(t, m, "S")
	if m.findPrompt || !strings.Contains(m.status, "no corpus at "+missing) || !strings.Contains(m.status, "F") {
		t.Errorf("status %q", m.status)
	}
	if _, err := os.Stat(missing); err == nil {
		t.Error("a search created the corpus")
	}
}

func TestAResultsPageIsNotRecordedInHistory(t *testing.T) {
	srv, _ := tocUISite(t)
	hist := filepath.Join(t.TempDir(), "history.md")
	db := corpusFile(t)
	m := New(srv.URL+"/p/b.md", render.Glamour{Style: "notty"}).WithHistory(hist)
	m = withCorpus(m, db)
	m = drive(t, m, m.Init())
	next, _ := m.Update(winSize())
	m = next.(Model)
	m = press(t, m, "F", "S")
	m = typeText(t, m, "post")
	m = press(t, m, "enter")
	b, _ := os.ReadFile(hist)
	if strings.Contains(string(b), "corpus:") {
		t.Errorf("history has the results page:\n%s", b)
	}
	if !strings.Contains(string(b), "/p/b.md") {
		t.Errorf("history lost the page:\n%s", b)
	}
}

func winSize() tea.WindowSizeMsg { return tea.WindowSizeMsg{Width: 100, Height: 24} }
