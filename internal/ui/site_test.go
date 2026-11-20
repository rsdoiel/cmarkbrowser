package ui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/rsdoiel/cmarkbrowser/internal/local"
	"github.com/rsdoiel/cmarkbrowser/internal/render"
)

// siteDir makes a local site under a temp folder: the site root is its "site"
// folder; "outside.md" sits beside it.
func siteDir(t *testing.T) (root string) {
	t.Helper()
	base := t.TempDir()
	files := map[string]string{
		"site/index.md":  "# Home\n\n[about](about.md) [root-relative](/blog/a.md) [up](../outside.md) [abs](" + "ABS" + ") [file](file:///etc/hostname) [web](WEB) [html](blog/a.html)\n",
		"site/about.md":  "---\nurlNext: blog/a.md\n---\n# About\n\n[home](/)\n",
		"site/blog/a.md": "# Post A\n\n[b](b.md)\n",
		"site/blog/b.md": "# Post B\n",
		"site/.well-known/toc.opml": `<opml version="2.0"><body><outline text="Pages">` +
			`<outline text="h" url="https://pub.example/index.md"/>` +
			`<outline text="a" url="https://pub.example/blog/a.md"/>` +
			`<outline text="b" url="https://pub.example/blog/b.md"/>` +
			`</outline></body></opml>`,
		"outside.md": "# SECRET OUTSIDE\n",
	}
	for name, body := range files {
		p := filepath.Join(base, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	real, _ := filepath.EvalSymlinks(filepath.Join(base, "site"))
	return real
}

// openSite starts the browser on a local site the way the command line does.
func openSite(t *testing.T, root string, extra ...func(Model) Model) (Model, string) {
	t.Helper()
	hist := filepath.Join(t.TempDir(), "history.md")
	bm := filepath.Join(t.TempDir(), "bookmarks.md")
	site, start, err := local.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	m := New(start, render.Glamour{Style: "notty"}).WithLocalStart().WithSite(site).WithHistory(hist).WithBookmarks(bm)
	for _, f := range extra {
		m = f(m)
	}
	m = drive(t, m, m.Init())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return next.(Model), hist
}

func TestALocalSitePageOpensAndShowsAPathNotAFileAddress(t *testing.T) {
	root := siteDir(t)
	m, _ := openSite(t, root)
	out := m.View().Content
	if !strings.Contains(out, "Home") || !strings.Contains(out, filepath.Join(root, "index.md")) || strings.Contains(strings.SplitN(out, "\n", 2)[0], "file://") {
		t.Errorf("title line:\n%s", strings.SplitN(out, "\n", 2)[0])
	}
}

func TestRelativeRootRelativeAndHTMLLinksStayInsideTheSite(t *testing.T) {
	root := siteDir(t)
	m, _ := openSite(t, root)
	m = press(t, m, "1", "enter") // about.md
	if !strings.Contains(m.View().Content, "About") {
		t.Fatalf("relative link:\n%s", m.View().Content)
	}
	m = press(t, m, "b", "2", "enter") // /blog/a.md: the site root, not the file system root
	if !strings.Contains(m.View().Content, "Post A") {
		t.Fatalf("root-relative link:\n%s", m.View().Content)
	}
	m = press(t, m, "b", "7", "enter") // blog/a.html has no file: its .md copy
	if !strings.Contains(m.View().Content, "Post A") {
		t.Errorf("the .html link did not reach its .md:\n%s", m.View().Content)
	}
	m = press(t, m, "1", "enter") // from a.md, b.md, and then back through the session history
	m = press(t, m, "b")
	if !strings.Contains(m.View().Content, "Post A") {
		t.Errorf("b should go back inside a local site:\n%s", m.View().Content)
	}
}

func TestALinkOutOfTheSiteRootIsRefusedAndReadsNothing(t *testing.T) {
	root := siteDir(t)
	m, _ := openSite(t, root)
	m = press(t, m, "3", "enter") // ../outside.md
	out := m.View().Content
	if strings.Contains(out, "SECRET OUTSIDE") || !strings.Contains(out, "Home") {
		t.Fatalf("left the site root:\n%s", out)
	}
	if !strings.Contains(m.status, "outside the site root") || !strings.Contains(m.status, "outside.md") {
		t.Errorf("status %q", m.status)
	}
}

func TestASymlinkOutOfTheSiteIsRefused(t *testing.T) {
	root := siteDir(t)
	os.Symlink(filepath.Join(filepath.Dir(root), "outside.md"), filepath.Join(root, "sneaky.md"))
	os.WriteFile(filepath.Join(root, "index.md"), []byte("# Home\n\n[s](sneaky.md)\n"), 0o644)
	m, _ := openSite(t, root)
	m = press(t, m, "1", "enter")
	if strings.Contains(m.View().Content, "SECRET OUTSIDE") || !strings.Contains(m.status, "outside the site root") {
		t.Errorf("status %q\n%s", m.status, m.View().Content)
	}
}

func TestAFileAddressInALocalPageIsNeverFollowed(t *testing.T) {
	root := siteDir(t)
	m, _ := openSite(t, root)
	m = press(t, m, "5", "enter")
	if !strings.Contains(m.status, "file: address is not followed") || !strings.Contains(m.View().Content, "Home") {
		t.Errorf("status %q", m.status)
	}
	// the same address, even inside the root, is refused: no protocol, only paths
	abs := local.Address(filepath.Join(root, "about.md"))
	os.WriteFile(filepath.Join(root, "index.md"), []byte("# Home\n\n["+"x"+"]("+abs+")\n"), 0o644)
	m, _ = openSite(t, root)
	m = press(t, m, "1", "enter")
	if !strings.Contains(m.status, "file: address is not followed") || strings.Contains(m.View().Content, "About") {
		t.Errorf("status %q\n%s", m.status, m.View().Content)
	}
}

func TestALocalPageIsNeverRecordedInHistoryNorBookmarked(t *testing.T) {
	root := siteDir(t)
	m, hist := openSite(t, root)
	m = press(t, m, "1", "enter", "m")
	if _, err := os.Stat(hist); err == nil {
		b, _ := os.ReadFile(hist)
		t.Errorf("history was written for a local site:\n%s", b)
	}
	if !strings.Contains(m.status, "not bookmarked") {
		t.Errorf("status %q", m.status)
	}
	if _, err := os.Stat(m.bookmarksFile()); err == nil {
		t.Error("a bookmarks file was created")
	}
}

func TestBracketKeysUseTheTableOfContentsUnderTheSiteRoot(t *testing.T) {
	root := siteDir(t)
	m, _ := openSite(t, root)
	m = press(t, m, "1", "enter") // about.md, which names urlNext: blog/a.md
	m = press(t, m, "]")
	if !strings.Contains(m.View().Content, "Post A") {
		t.Fatalf("urlNext as a path:\n%s", m.View().Content)
	}
	// blog/a.md is in toc.opml (as https://pub.example/blog/a.md) between the home page and b
	m = press(t, m, "]")
	if !strings.Contains(m.View().Content, "Post B") {
		t.Fatalf("] by the outline:\n%s", m.View().Content)
	}
	m = press(t, m, "[", "[")
	if !strings.Contains(m.View().Content, "Home") {
		t.Errorf("[ [ by the outline:\n%s", m.View().Content)
	}
}

func TestAWebLinkFromALocalPageGoesToTheNetwork(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		fmt.Fprint(w, "# From The Web\n")
	}))
	t.Cleanup(srv.Close)
	root := siteDir(t)
	idx, _ := os.ReadFile(filepath.Join(root, "index.md"))
	os.WriteFile(filepath.Join(root, "index.md"), []byte(strings.ReplaceAll(string(idx), "WEB", srv.URL+"/w.md")), 0o644)
	m, hist := openSite(t, root)
	m = press(t, m, "6", "enter")
	if !strings.Contains(m.View().Content, "From The Web") {
		t.Fatalf("%s", m.View().Content)
	}
	// the web page is recorded, as any web page is; the local one before it was not
	b, _ := os.ReadFile(hist)
	if !strings.Contains(string(b), srv.URL) || strings.Contains(string(b), root) {
		t.Errorf("history:\n%s", b)
	}
}

func TestABookmarkedLocalPathIsStillRefusedFromTheBookmarksFile(t *testing.T) {
	root := siteDir(t)
	bm := filepath.Join(t.TempDir(), "bookmarks.md")
	os.WriteFile(bm, []byte("# Bookmarks\n\n- [mine]("+filepath.Join(root, "about.md")+")\n- [url]("+local.Address(filepath.Join(root, "about.md"))+")\n"), 0o644)
	site, start, _ := local.Open(root)
	m := New(start, render.Glamour{Style: "notty"}).WithLocalStart().WithSite(site).WithBookmarks(bm)
	m = drive(t, m, m.Init())
	m = press(t, m, "B", "1", "enter")
	out := m.View().Content
	if strings.Contains(out, "# About") || strings.Contains(out, "About\n") && !strings.Contains(out, "Bookmarks") {
		t.Errorf("the bookmarks file reached a local page:\n%s", out)
	}
	if !strings.Contains(out, "Bookmarks") || !strings.Contains(m.status, "local files") {
		t.Errorf("status %q, view:\n%s", m.status, out)
	}
}
