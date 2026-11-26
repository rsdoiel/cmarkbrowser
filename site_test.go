package cmarkbrowser_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/rsdoiel/cmarkbrowser"
	"github.com/rsdoiel/cmarkbrowser/internal/cli"
	"github.com/rsdoiel/markdownweb/local"
	"github.com/rsdoiel/markdownweb/toc"
)

// The documentation is a Markdown site (README.md and the pages beside it) that
// is published as HTML and as Markdown for BonBon browsers. These tests keep it
// that way: the pages link to each other as Markdown, the table of contents
// lists them all, the command reference is the program's own help, and the
// built site reads in cmarkbrowser itself.

// noPublish are the Markdown files that are not website pages.
var noPublish = map[string]bool{"TODO.md": true, "index.md": true, "cmarkbrowser.1.md": true}

// pageOrder is PAGE_ORDER from website.mak.
func pageOrder(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile("website.mak")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^PAGE_ORDER = (.*)$`).FindSubmatch(b)
	if m == nil {
		t.Fatal("website.mak has no PAGE_ORDER")
	}
	return strings.Fields(string(m[1]))
}

// publishedPages are the Markdown files in the folder that the site publishes.
func publishedPages(t *testing.T) []string {
	t.Helper()
	all, _ := filepath.Glob("*.md")
	var out []string
	for _, f := range all {
		if !noPublish[f] {
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out
}

func TestEveryPublishedPageIsInThePageOrder(t *testing.T) {
	order := append([]string(nil), pageOrder(t)...)
	sort.Strings(order)
	if got, want := strings.Join(order, " "), strings.Join(publishedPages(t), " "); got != want {
		t.Errorf("PAGE_ORDER in website.mak\n  %s\nbut the published pages are\n  %s", got, want)
	}
	if o := pageOrder(t); o[0] != "README.md" {
		t.Errorf("the home page must come first: %v", o)
	}
}

func TestPagesLinkToEachOtherAsMarkdown(t *testing.T) {
	link := regexp.MustCompile(`\]\(([^)\s]+)\)`)
	for _, page := range publishedPages(t) {
		b, err := os.ReadFile(page)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range link.FindAllStringSubmatch(string(b), -1) {
			dest := m[1]
			if strings.Contains(dest, "://") || strings.HasPrefix(dest, "#") || strings.HasPrefix(dest, "mailto:") {
				continue
			}
			file := strings.SplitN(dest, "#", 2)[0]
			if !strings.HasSuffix(file, ".md") {
				t.Errorf("%s links to %q: a link to another page is written as .md (the HTML build changes it)", page, dest)
				continue
			}
			if _, err := os.Stat(file); err != nil || noPublish[file] {
				t.Errorf("%s links to %q, which is not a published page", page, dest)
			}
		}
	}
}

func TestTheCommandReferenceIsTheProgramsOwnHelp(t *testing.T) {
	got, err := os.ReadFile("command_reference.md")
	if err != nil {
		t.Fatal(err)
	}
	help := cmarkbrowser.FmtHelp(cli.HelpText, "cmarkbrowser", cmarkbrowser.Version, cmarkbrowser.ReleaseDate, cmarkbrowser.ReleaseHash)
	lines := strings.Split(help, "\n")
	body := strings.Join(lines[3:], "\n")
	body = regexp.MustCompile(`(?m)^# `).ReplaceAllString(body, "## ")
	want := "# cmarkbrowser command reference\n\nThis page is written by `cmarkbrowser -help`.\n\n" + body
	if strings.TrimRight(string(got), "\n") != strings.TrimRight(want, "\n") {
		t.Error("command_reference.md is out of date with the program's help: run `make docs`")
	}
}

// bash runs a command and returns what it printed and its exit status.
func run(t *testing.T, dir string, name string, args ...string) (string, int) {
	t.Helper()
	c := exec.Command(name, args...)
	c.Dir = dir
	var out, errb bytes.Buffer
	c.Stdout, c.Stderr = &out, &errb
	err := c.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return out.String(), code
}

func needs(t *testing.T, tools ...string) {
	t.Helper()
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not installed", tool)
		}
	}
}

func TestTocScriptWritesAnOutlineTheReaderUnderstands(t *testing.T) {
	needs(t, "bash")
	pages := pageOrder(t)
	args := append([]string{"toc.bash", "https://example.org/docs/", "Docs & <More>"}, pages...)
	out, code := run(t, ".", "bash", args...)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	d, err := toc.Parse([]byte(out), "https://example.org/docs/.well-known/toc.opml")
	if err != nil {
		t.Fatalf("not an outline: %v\n%s", err, out)
	}
	entries := d.Entries()
	if len(entries) != len(pages) || entries[0].URL != "https://example.org/docs/index.md" {
		t.Fatalf("entries %v", entries)
	}
	for i, p := range pages[1:] {
		if entries[i+1].URL != "https://example.org/docs/"+p {
			t.Errorf("entry %d is %s, want %s", i+1, entries[i+1].URL, p)
		}
	}
	if !strings.Contains(out, "Docs &amp; &lt;More&gt;") {
		t.Errorf("the title was not escaped:\n%s", out)
	}
	if prev, next, ok := d.NeighboursByPath(pages[1]); !ok || prev != "https://example.org/docs/index.md" || next != "https://example.org/docs/"+pages[2] {
		t.Errorf("neighbours of %s: %q %q %v", pages[1], prev, next, ok)
	}
	if again, _ := run(t, ".", "bash", args...); again != out {
		t.Error("the same input gave a different table of contents")
	}
}

func TestTocScriptRefusesWhatItCannotDo(t *testing.T) {
	needs(t, "bash")
	if _, code := run(t, ".", "bash", "toc.bash", "https://x.org/docs", "T", "README.md"); code != 2 {
		t.Errorf("a base URL with no trailing slash: exit %d, want 2", code)
	}
	if _, code := run(t, ".", "bash", "toc.bash", "https://x.org/", "T", "no-such-page.md"); code != 66 {
		t.Errorf("a missing page: exit %d, want 66", code)
	}
	if _, code := run(t, ".", "bash", "toc.bash"); code != 2 {
		t.Errorf("no arguments: exit %d, want 2", code)
	}
}

// copySite copies the files the website build reads into a new folder.
func copySite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, f := range []string{"website.mak", "toc.bash", "page.tmpl", "links-to-html.lua", "add-col-scope.lua", "README.md", "TODO.md"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(dir, f), b, 0o755)
	}
	for _, p := range pageOrder(t) {
		if p == "README.md" {
			continue
		}
		b, _ := os.ReadFile(p)
		os.WriteFile(filepath.Join(dir, p), b, 0o644)
	}
	return dir
}

func TestTheBuiltSiteIsAMarkdownSiteThatReadsInCmarkbrowser(t *testing.T) {
	needs(t, "bash", "make")
	dir := copySite(t)
	if out, code := run(t, dir, "make", "-f", "website.mak"); code != 0 {
		t.Fatalf("make exit %d\n%s", code, out)
	}
	for _, f := range []string{"index.md", ".nojekyll", ".well-known/toc.opml"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("the build did not make %s", f)
		}
	}
	site, start, err := local.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(start) != "index.md" {
		t.Errorf("the site starts at %s", start)
	}
	// ] and [ walk the table of contents the build wrote, by the tail of each address
	order := pageOrder(t)
	f := toc.NewFinder()
	for i, p := range order[1 : len(order)-1] {
		r := f.LookupLocal(context.Background(), site.Root, filepath.Join(site.Root, p))
		wantPrev, wantNext := filepath.Join(site.Root, order[i]), filepath.Join(site.Root, order[i+2])
		if order[i] == "README.md" {
			wantPrev = filepath.Join(site.Root, "index.md")
		}
		if !r.Found || r.Prev != wantPrev || r.Next != wantNext {
			t.Errorf("%s: prev %s next %s found %v; want %s and %s", p, r.Prev, r.Next, r.Found, wantPrev, wantNext)
		}
	}
	if out, code := run(t, dir, "make", "-f", "website.mak", "check-order"); code != 0 {
		t.Errorf("check-order: %s", out)
	}
	// a page the order does not list stops the build
	os.WriteFile(filepath.Join(dir, "extra.md"), []byte("# Extra\n"), 0o644)
	if _, code := run(t, dir, "make", "-f", "website.mak", "check-order"); code == 0 {
		t.Error("an unlisted page did not stop the build")
	}
}

func TestHTMLLinksChangeOnlyForThePagesOwnMarkdown(t *testing.T) {
	needs(t, "pandoc")
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A\n\n[b](b.md#top) [home](README.md) [out](https://example.org/x.md) [abs](/y.md) [md](notes.md.txt)\n"), 0o644)
	lua, _ := os.ReadFile("links-to-html.lua")
	os.WriteFile(filepath.Join(dir, "links-to-html.lua"), lua, 0o644)
	out, code := run(t, dir, "pandoc", "-f", "markdown", "-t", "html5", "--lua-filter=links-to-html.lua", "a.md")
	if code != 0 {
		t.Fatalf("pandoc exit %d", code)
	}
	for _, want := range []string{`href="b.html#top"`, `href="index.html"`, `href="https://example.org/x.md"`, `href="/y.md"`, `href="notes.md.txt"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in\n%s", want, out)
		}
	}
}
