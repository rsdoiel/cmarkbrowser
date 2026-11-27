package browse

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/rsdoiel/markdownweb/page"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rsdoiel/cmarkbrowser/internal/render"
	"github.com/rsdoiel/markdownweb/debuglog"
)

func site(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/docs/a.md", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Write([]byte("---\ntitle: Page A\n---\n# A\n\nSee [b](b.md) and [root](/r.md) <script>bad()</script>\n"))
	})
	return httptest.NewServer(mux)
}

func TestOpenPipeline(t *testing.T) {
	srv := site(t)
	defer srv.Close()
	v, err := Open(context.Background(), srv.URL+"/docs/a.md", render.Glamour{Style: "notty"}, 60)
	if err != nil {
		t.Fatal(err)
	}
	if v.Page.Title() != "Page A" {
		t.Errorf("title %q", v.Page.Title())
	}
	if strings.Contains(v.Text, "bad()") || strings.Contains(v.Text, "title:") {
		t.Errorf("text has stripped content: %q", v.Text)
	}
	if len(v.Links) != 2 {
		t.Fatalf("links %#v", v.Links)
	}
	if v.Links[0].URL != srv.URL+"/docs/b.md" || v.Links[1].URL != srv.URL+"/r.md" {
		t.Errorf("resolved %#v", v.Links)
	}
}

func TestResolve(t *testing.T) {
	cases := []struct{ base, dest, want string }{
		{"https://x.org/a/b.md", "c.md", "https://x.org/a/c.md"},
		{"https://x.org/a/b.md", "/c.md", "https://x.org/c.md"},
		{"https://x.org/a/b.md", "https://y.org/z.md", "https://y.org/z.md"},
		{"https://x.org/a/b.md", "c.md#top", "https://x.org/a/c.md#top"},
	}
	for _, c := range cases {
		if got, err := Resolve(c.base, c.dest); err != nil || got != c.want {
			t.Errorf("%v %v: %q %v", c.base, c.dest, got, err)
		}
	}
	if _, err := Resolve("https://x.org/", "%zz"); err == nil {
		t.Error("bad destination accepted")
	}
}

// pageServer serves one Markdown body at /p.md.
func pageServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestPositionsFindEachLinkLine(t *testing.T) {
	body := "# T\n\nfirst [one](a.md)\n\n" + strings.Repeat("filler line\n\n", 5) + "last [two](b.md)\n"
	srv := pageServer(t, body)
	v, err := Open(context.Background(), srv.URL+"/p.md", render.Glamour{Style: "notty"}, 60)
	if err != nil {
		t.Fatal(err)
	}
	l1, ok1 := v.LineOf(1)
	l2, ok2 := v.LineOf(2)
	if !ok1 || !ok2 || l2 <= l1 {
		t.Fatalf("lines %d %v, %d %v", l1, ok1, l2, ok2)
	}
	lines := strings.Split(v.Display(-1), "\n")
	if !strings.Contains(lines[l1], "one") || !strings.Contains(lines[l2], "two") {
		t.Errorf("line %d %q, line %d %q", l1, lines[l1], l2, lines[l2])
	}
}

func TestPositionsIgnoreLookalikeMarkersInPageText(t *testing.T) {
	body := "A footnote [1] and [2] in prose.\n\nThe [real link](a.md) is later.\n"
	srv := pageServer(t, body)
	v, err := Open(context.Background(), srv.URL+"/p.md", render.Glamour{Style: "notty"}, 80)
	if err != nil {
		t.Fatal(err)
	}
	l, ok := v.LineOf(1)
	if !ok {
		t.Fatal("link 1 not found")
	}
	if line := strings.Split(v.Display(-1), "\n")[l]; !strings.Contains(line, "real link") {
		t.Errorf("found the footnote, not the link: %q", line)
	}
}

func TestPositionsSurviveNarrowWrap(t *testing.T) {
	body := "Some words that make a long sentence before the [link with several words](a.md) and more after it.\n"
	srv := pageServer(t, body)
	v, err := Open(context.Background(), srv.URL+"/p.md", render.Glamour{Style: "notty"}, 24)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := v.LineOf(1); !ok {
		t.Errorf("not found in:\n%s", v.Display(-1))
	}
}

func TestDisplayHasNoSentinelAndHighlightsTheFocusedMarker(t *testing.T) {
	srv := pageServer(t, "[a](a.md) and [b](b.md)\n")
	v, err := Open(context.Background(), srv.URL+"/p.md", render.Glamour{Style: "notty"}, 60)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(v.Display(-1), render.Sentinel) || strings.Contains(v.Display(2), render.Sentinel) {
		t.Error("sentinel shown")
	}
	plain, hi := v.Display(-1), v.Display(2)
	if strings.Contains(plain, "\x1b[7m") {
		t.Error("highlight without focus")
	}
	if !strings.Contains(hi, "\x1b[7m[2]\x1b[27m") || strings.Contains(hi, "\x1b[7m[1]") {
		t.Errorf("highlight wrong: %q", hi)
	}
}

func TestRerenderRefindsPositions(t *testing.T) {
	srv := pageServer(t, strings.Repeat("words ", 40)+"[a](a.md)\n")
	v, _ := Open(context.Background(), srv.URL+"/p.md", render.Glamour{Style: "notty"}, 100)
	wide, _ := v.LineOf(1)
	if err := v.Rerender(render.Glamour{Style: "notty"}, 20); err != nil {
		t.Fatal(err)
	}
	narrow, ok := v.LineOf(1)
	if !ok || narrow <= wide {
		t.Errorf("wide %d, narrow %d, ok %v", wide, narrow, ok)
	}
}

func TestPositionsWithStyledOutput(t *testing.T) {
	srv := pageServer(t, "Intro text.\n\nA [styled link](a.md) here and [another](b.md).\n")
	for _, style := range []string{"dark", "light", "ascii"} {
		v, err := Open(context.Background(), srv.URL+"/p.md", render.Glamour{Style: style}, 40)
		if err != nil {
			t.Fatal(err)
		}
		for n := 1; n <= 2; n++ {
			if _, ok := v.LineOf(n); !ok {
				t.Errorf("%s: link %d not found in %q", style, n, v.Text)
			}
		}
		if strings.Contains(v.Display(1), render.Sentinel) {
			t.Errorf("%s: sentinel left in display", style)
		}
	}
}

func TestTableLinksAreFoundInRenderedText(t *testing.T) {
	body := "# Front\n\n| Collection | Feeds |\n| --- | --- |\n| [SoCal](socal.html) | [RSS](socal.xml), [OPML](socal.opml) |\n| [Pacific](pacific.html) | |\n"
	srv := pageServer(t, body)
	v, err := Open(context.Background(), srv.URL+"/p.md", render.Glamour{Style: "notty"}, 80)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Links) != 4 {
		t.Fatalf("links %#v", v.Links)
	}
	for n := 1; n <= 4; n++ {
		if _, ok := v.LineOf(n); !ok {
			t.Errorf("link %d not found in:\n%s", n, v.Display(-1))
		}
	}
}

func TestOpenExpandsPageMacrosAndReportsMissing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/s.md", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		w.Write([]byte("---\ntitle: Story One\nauthor: Dave\n---\n# {{page.title}}\n\nBy {{page.author}} on {{page.pubDate}}.\n\n`{{page.title}}`\n"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	v, err := Open(context.Background(), srv.URL+"/s.md", render.Glamour{Style: "notty"}, 70)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Story One", "By Dave", "{{page.pubDate}}", "{{page.title}}"} {
		if !strings.Contains(v.Text, want) {
			t.Errorf("%q not in %q", want, v.Text)
		}
	}
	if len(v.Missing) != 1 || v.Missing[0] != "pubDate" {
		t.Errorf("Missing = %v", v.Missing)
	}
}

func TestOpenLogsTimingsAndCounts(t *testing.T) {
	srv := site(t)
	defer srv.Close()
	var buf bytes.Buffer
	ctx := debuglog.WithContext(context.Background(), debuglog.New(&buf))
	v, err := Open(ctx, srv.URL+"/docs/a.md", render.Glamour{Style: "notty"}, 60)
	if err != nil {
		t.Fatal(err)
	}
	var open map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var e map[string]any
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("not JSON: %q", line)
		}
		if e["event"] == "open" {
			open = e
		}
	}
	if open == nil {
		t.Fatalf("no open event in:\n%s", buf.String())
	}
	for _, k := range []string{"url", "final", "width", "fetch_ms", "expand_ms", "prepare_ms", "render_ms", "total_ms", "body_bytes", "text_bytes", "lines", "links", "missing_keys"} {
		if _, ok := open[k]; !ok {
			t.Errorf("open event lacks %q: %v", k, open)
		}
	}
	if open["links"] != float64(len(v.Links)) || open["width"] != float64(60) {
		t.Errorf("counts wrong: %v", open)
	}
	if strings.Contains(buf.String(), "Page A") || strings.Contains(buf.String(), "bad()") {
		t.Errorf("log holds page content:\n%s", buf.String())
	}
}

func TestRerenderLogsItsWidthAndTime(t *testing.T) {
	srv := site(t)
	defer srv.Close()
	v, err := Open(context.Background(), srv.URL+"/docs/a.md", render.Glamour{Style: "notty"}, 60)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	v.SetLogger(debuglog.New(&buf))
	if err := v.Rerender(render.Glamour{Style: "notty"}, 40); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"event":"rerender"`) || !strings.Contains(buf.String(), `"width":40`) {
		t.Errorf("log %q", buf.String())
	}
}

func htmlSite(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/story":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><head><title>Story Title</title></head><body><nav>menu</nav><article><h1>Story Title</h1>
<p>` + strings.Repeat("Story words {{page.title}} here. ", 12) + `See <a href="/other">the other page</a>.</p></article></body></html>`))
		case "/image.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write([]byte("x"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestOpenShowsAnHTMLPageInReaderView(t *testing.T) {
	srv := htmlSite(t)
	v, err := Open(context.Background(), srv.URL+"/story", render.Glamour{Style: "notty"}, 80)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Page.Reader || v.Page.Title() != "Story Title" {
		t.Errorf("page %+v", v.Page)
	}
	if !strings.Contains(v.Text, "Story words") || strings.Contains(v.Text, "menu") {
		t.Errorf("text:\n%s", v.Text)
	}
	if len(v.Links) != 1 || v.Links[0].URL != srv.URL+"/other" {
		t.Errorf("links %#v", v.Links)
	}
	// the page's own {{...}} text is not a macro of ours
	if len(v.Missing) != 0 || !strings.Contains(v.Text, "{{page.title}}") {
		t.Errorf("macros were applied to an HTML page: missing %v", v.Missing)
	}
}

func TestAnImageIsANoteAndAMissingPageStillFails(t *testing.T) {
	srv := htmlSite(t)
	v, err := Open(context.Background(), srv.URL+"/image.png", render.Glamour{Style: "notty"}, 80)
	if err != nil || v.Page.Kind != page.KindImage {
		t.Errorf("an image is noted, not an error: %+v %v", v, err)
	}
	if _, err := Open(context.Background(), srv.URL+"/missing", render.Glamour{Style: "notty"}, 80); err == nil {
		t.Error("a missing page opened")
	}
}

func TestOpenLogsThatItUsedReaderView(t *testing.T) {
	srv := htmlSite(t)
	var buf bytes.Buffer
	ctx := debuglog.WithContext(context.Background(), debuglog.New(&buf))
	if _, err := Open(ctx, srv.URL+"/story", render.Glamour{Style: "notty"}, 80); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"reader":true`) {
		t.Errorf("open event does not say reader:\n%s", buf.String())
	}
}

func TestOpenFileShowsALocalMarkdownFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "list.md")
	os.WriteFile(path, []byte("---\ntitle: List\n---\n# Reading\n\n- [one](https://example.org/one)\n- [rel](other.md)\n"), 0o644)
	v, err := OpenFile(context.Background(), path, render.Glamour{Style: "notty"}, 80)
	if err != nil {
		t.Fatal(err)
	}
	if v.Page.Title() != "List" || !strings.Contains(v.Text, "Reading") {
		t.Errorf("view %+v\n%s", v.Page, v.Text)
	}
	if len(v.Links) != 2 || v.Links[0].URL != "https://example.org/one" {
		t.Errorf("links %#v", v.Links)
	}
	if !strings.HasPrefix(v.Links[1].URL, "file://") {
		t.Errorf("a relative link in a local file resolves against the file: %q", v.Links[1].URL)
	}
}

func TestOpenFileGetsTheSameMacrosAsAPage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.md")
	os.WriteFile(path, []byte("---\ntitle: T\n---\n# {{page.title}}\n"), 0o644)
	v, err := OpenFile(context.Background(), path, render.Glamour{Style: "notty"}, 80)
	if err != nil || !strings.Contains(v.Text, "T") || strings.Contains(v.Text, "{{") {
		t.Errorf("a local Markdown file gets the same macros as a page: %v\n%s", err, v.Text)
	}
}

// The case that was wrong: an HTML address answered with Markdown from another
// folder. Its relative links belong to the Markdown's folder, not the HTML's.
func TestRelativeLinksResolveAgainstTheContentLocation(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/sub/page.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Location", "/other/items.md")
		w.Write([]byte("# Items\n\n[next](next.md) and [up](../top.md)\n"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	v, err := Open(context.Background(), srv.URL+"/sub/page.html", render.Glamour{Style: "notty"}, 80)
	if err != nil {
		t.Fatal(err)
	}
	if v.Page.URL != srv.URL+"/other/items.md" {
		t.Errorf("page address %q", v.Page.URL)
	}
	if len(v.Links) != 2 || v.Links[0].URL != srv.URL+"/other/next.md" || v.Links[1].URL != srv.URL+"/top.md" {
		t.Errorf("links %#v: want them resolved against /other/, where the Markdown is", v.Links)
	}
}

func picturesSite(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	page := func(path, front string) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			w.Write([]byte("---\ntitle: P\n" + front + "---\nBody ![A cat](cat.png) here.\n\n![](decor.png)\n"))
		})
	}
	page("/docs/alt.md", "rightMarginImage: /m.png\nrightMarginImageAlt: A hill at dusk\n")
	page("/docs/noalt.md", "rightMarginImage: /m.png\n")
	return httptest.NewServer(mux)
}

// A body picture is a link to follow, shown as its alt text; the margin picture
// is shown the same way at the top when the page gives alt text for it, and not
// at all when it does not (decorative). No address is printed in the body.
func TestPicturesAreAltTextLinksAndTheMarginImageNeedsAltText(t *testing.T) {
	srv := picturesSite(t)
	defer srv.Close()
	open := func(path string) *View {
		v, err := Open(context.Background(), srv.URL+path, render.Glamour{Style: "notty"}, 70)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}

	v := open("/docs/alt.md")
	if len(v.Links) != 2 || v.Links[0].URL != srv.URL+"/m.png" || v.Links[1].URL != srv.URL+"/docs/cat.png" {
		t.Fatalf("links %#v", v.Links)
	}
	if !strings.Contains(v.Text, "[image: A hill at dusk]") || !strings.Contains(v.Text, "Body [image: A cat]") ||
		strings.Contains(v.Text, ".png") || strings.Contains(v.Text, "decor") {
		t.Errorf("text:\n%s", v.Text)
	}
	if strings.Index(v.Text, "A hill at dusk") > strings.Index(v.Text, "Body") {
		t.Errorf("the margin picture is not at the top:\n%s", v.Text)
	}

	v = open("/docs/noalt.md")
	if len(v.Links) != 1 || v.Links[0].URL != srv.URL+"/docs/cat.png" || strings.Contains(v.Text, "m.png") {
		t.Errorf("a margin picture with no alt text should show nothing: %#v\n%s", v.Links, v.Text)
	}
}

// --- F5: macros and front matter come from the frontmatter module ---

func openBody(t *testing.T, body string) *View {
	t.Helper()
	srv := pageServer(t, body)
	v, err := Open(context.Background(), srv.URL+"/p.md", render.Glamour{Style: "notty"}, 80)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestAMacroShowsABareDateAsADate(t *testing.T) {
	v := openBody(t, "---\ntitle: T\ndate: 2026-11-19\n---\nOn {{page.date}}.\n")
	if !strings.Contains(v.Text, "On 2026-11-19.") || strings.Contains(v.Text, "00:00:00") {
		t.Errorf("text %q", v.Text)
	}
}

func TestTheTitleIsALinkWhenThePageHasALink(t *testing.T) {
	v := openBody(t, "---\ntitle: Hello\nlink: https://x.example/story\n---\n# {{page.title}}\n")
	if len(v.Links) != 1 || v.Links[0].URL != "https://x.example/story" || v.Links[0].Text != "Hello" {
		t.Errorf("links %#v", v.Links)
	}
}

func TestAnUnusableLinkValueIsNotedAndTheTitleStaysText(t *testing.T) {
	v := openBody(t, "---\ntitle: Hello\nlink: \"javascript:alert(1)\"\n---\n# {{page.title}}\n")
	if len(v.Links) != 0 || !strings.Contains(v.Text, "Hello") {
		t.Errorf("links %#v, text %q", v.Links, v.Text)
	}
	if len(v.Notes) != 1 || !strings.Contains(v.Notes[0], "link") {
		t.Errorf("Notes = %v", v.Notes)
	}
}

func TestAValueThatWouldStartAHeadingIsShownAsText(t *testing.T) {
	v := openBody(t, "---\ntitle: \"# Not a heading\"\n---\n{{page.title}}\n")
	if !strings.Contains(v.Clean, `\# Not a heading`) {
		t.Errorf("clean %q", v.Clean)
	}
}

func TestAnUnclosedBlockIsShownAsTextAndNoted(t *testing.T) {
	v := openBody(t, "---\ntitle: T\nthis block never closes\n\n# A heading\n")
	if !strings.Contains(v.Text, "this block never closes") || !strings.Contains(v.Text, "A heading") {
		t.Errorf("the whole file should show: %q", v.Text)
	}
	if len(v.Notes) != 1 || !strings.Contains(v.Notes[0], "not closed") {
		t.Errorf("Notes = %v", v.Notes)
	}
}

func TestFragmentLines(t *testing.T) {
	src := "# Top\n\nintro\n\n## Soil & Seeds\n\ntext\n\n## Notes\n\none\n\n## Notes\n\ntwo\n\n```\n# not a heading\n```\n\n### The *big* [idea](x.md)\n"
	v, err := OpenMarkdown(context.Background(), "https://x.org/g.md", src, render.Glamour{Style: "notty"}, 60)
	if err != nil {
		t.Fatal(err)
	}
	var prev = -1
	for _, f := range []string{"top", "soil--seeds", "notes", "notes-1", "the-big-idea"} {
		l, ok := v.LineOfFragment(f)
		if !ok || l <= prev {
			t.Fatalf("%q: line %d ok=%v after %d", f, l, ok, prev)
		}
		prev = l
	}
	if _, ok := v.LineOfFragment("not-a-heading"); ok {
		t.Error("a heading inside a code fence was indexed")
	}
	if _, ok := v.LineOfFragment("nope"); ok {
		t.Error("found a fragment that is not there")
	}
}
