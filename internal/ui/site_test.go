package ui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
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

// fakeEditor makes $EDITOR a script that appends a line to the file it is
// given, and runs it without a terminal, as tea.ExecProcess would with one.
func fakeEditor(t *testing.T, line string) (ran *[]string) {
	t.Helper()
	script := filepath.Join(t.TempDir(), "ed.sh")
	os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s\\n' \""+line+"\" >> \"$1\"\n"), 0o755)
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", script)
	var calls []string
	old := runEditor
	runEditor = func(c *exec.Cmd, fn tea.ExecCallback) tea.Cmd {
		return func() tea.Msg {
			calls = append(calls, c.Args[len(c.Args)-1])
			return fn(c.Run())
		}
	}
	t.Cleanup(func() { runEditor = old })
	return &calls
}

func TestEEditsALocalPageInPlaceAndShowsItAgain(t *testing.T) {
	root := siteDir(t)
	calls := fakeEditor(t, "Added by the editor.")
	m, _ := openSite(t, root)
	m = press(t, m, "1", "enter") // about.md
	m = press(t, m, "e")
	about := filepath.Join(root, "about.md")
	if len(*calls) != 1 || (*calls)[0] != about {
		t.Fatalf("editor ran on %v, want %s", *calls, about)
	}
	if out := m.View().Content; !strings.Contains(out, "Added by the editor.") || !strings.Contains(out, "About") {
		t.Errorf("the page was not reloaded after the edit:\n%s", out)
	}
	// the same history entry was reloaded, not a new visit: b goes straight home
	m = press(t, m, "b")
	if !strings.Contains(m.View().Content, "Home") {
		t.Errorf("history after the edit:\n%s", m.View().Content)
	}
	if b, _ := os.ReadFile(about); !strings.HasSuffix(string(b), "Added by the editor.\n") {
		t.Errorf("file: %q", b)
	}
}

func TestAnEditComesBackToTheSameScrollPosition(t *testing.T) {
	root := siteDir(t)
	os.WriteFile(filepath.Join(root, "index.md"), []byte("# Home\n\n"+strings.Repeat("line of text here\n\n", 80)), 0o644)
	fakeEditor(t, "tail line")
	m, _ := openSite(t, root)
	m = press(t, m, "pgdown", "pgdown")
	before := m.vp.YOffset()
	if before == 0 {
		t.Fatal("did not scroll")
	}
	m = press(t, m, "e")
	if m.vp.YOffset() != before {
		t.Errorf("offset %d after the edit, was %d", m.vp.YOffset(), before)
	}
}

func TestEWithNoEditorSaysSoAndChangesNothing(t *testing.T) {
	root := siteDir(t)
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	before, _ := os.ReadFile(filepath.Join(root, "index.md"))
	m, _ := openSite(t, root)
	m = press(t, m, "e")
	if !strings.Contains(m.status, "$EDITOR") || !strings.Contains(m.View().Content, "Home") {
		t.Errorf("status %q", m.status)
	}
	if after, _ := os.ReadFile(filepath.Join(root, "index.md")); string(after) != string(before) {
		t.Error("the file changed")
	}
}

func TestEOnARemotePageEditsACopyInDownloadsAndLeavesThePageAlone(t *testing.T) {
	home := t.TempDir()
	os.Mkdir(filepath.Join(home, "Downloads"), 0o755)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	calls := fakeEditor(t, "my edit")
	m, base := open(t) // http://127.0.0.1:port/a.md
	m = press(t, m, "e")
	want := filepath.Join(home, "Downloads", "a.md")
	if len(*calls) != 1 || (*calls)[0] != want {
		t.Fatalf("editor ran on %v, want %s", *calls, want)
	}
	b, err := os.ReadFile(want)
	if err != nil || !strings.Contains(string(b), "title: Page A") || !strings.HasSuffix(string(b), "my edit\n") {
		t.Errorf("the copy: %q %v", b, err) // the front matter is kept in the copy
	}
	if !strings.Contains(m.status, want) || !strings.Contains(m.View().Content, base+"/a.md") {
		t.Errorf("status %q", m.status)
	}
	// a second e makes a second copy, never replacing the first
	m = press(t, m, "e")
	if _, err := os.Stat(filepath.Join(home, "Downloads", "a-1.md")); err != nil {
		t.Errorf("no second copy: %v", err)
	}
}

func TestERemoteWithNoDownloadsFolderOrNoEditorLeavesNothingBehind(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	fakeEditor(t, "x")
	m, _ := open(t)
	m = press(t, m, "e")
	if !strings.Contains(m.status, "Downloads") {
		t.Errorf("status %q", m.status)
	}
	os.Mkdir(filepath.Join(home, "Downloads"), 0o755)
	t.Setenv("EDITOR", "")
	m = press(t, m, "e")
	if ents, _ := os.ReadDir(filepath.Join(home, "Downloads")); len(ents) != 0 || !strings.Contains(m.status, "$EDITOR") {
		t.Errorf("a copy was left behind: %v (%q)", ents, m.status)
	}
}

func TestEOnTheBookmarksAndHistoryFilesEditsThemInPlace(t *testing.T) {
	srv := site(t)
	bm := filepath.Join(t.TempDir(), "bookmarks.md")
	hist := filepath.Join(t.TempDir(), "history.md")
	calls := fakeEditor(t, "- [added by hand](https://x.org/z)")
	m := New(srv.URL+"/a.md", render.Glamour{Style: "notty"}).WithBookmarks(bm).WithHistory(hist)
	m = drive(t, m, m.Init())
	m = press(t, m, "m", "B", "e")
	if len(*calls) != 1 || (*calls)[0] != bm || !strings.Contains(m.View().Content, "added by hand") {
		t.Errorf("bookmarks: %v\n%s", *calls, m.View().Content)
	}
	// the history file is rewritten by the browser; a hand edit survives the next visit
	m = press(t, m, "H", "e")
	m = press(t, m, "1", "enter") // follows the first entry of the history page
	b, _ := os.ReadFile(hist)
	if !strings.Contains(string(b), "added by hand") {
		t.Errorf("the hand edit was lost:\n%s", b)
	}
}

func TestEOnAReaderViewIsRefused(t *testing.T) {
	fakeEditor(t, "x")
	m, _ := open(t)
	m.view.Page.Reader = true
	m = press(t, m, "e")
	if !strings.Contains(m.status, "reader view") {
		t.Errorf("status %q", m.status)
	}
}
