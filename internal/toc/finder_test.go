package toc

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/rsdoiel/cmarkbrowser/internal/debuglog"
)

// tocSite serves outlines at the paths given and counts requests per path.
type tocSite struct {
	*httptest.Server
	mu   sync.Mutex
	hits map[string]int
}

func newTOCSite(t *testing.T, files map[string]string) *tocSite {
	t.Helper()
	s := &tocSite{hits: map[string]int{}}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.hits[r.URL.Path]++
		s.mu.Unlock()
		body, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		body = strings.ReplaceAll(body, "{base}", "http://"+r.Host)
		w.Header().Set("Content-Type", "text/x-opml; charset=utf-8")
		w.Write([]byte(body))
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *tocSite) count(path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits[path]
}

func opmlWith(entries ...string) string {
	return `<?xml version="1.0" encoding="UTF-8"?><opml version="2.0"><head/><body><outline text="Pages">` +
		strings.Join(entries, "") + `</outline></body></opml>`
}

func entry(url string) string { return `<outline text="e" url="` + url + `"/>` }

func TestTheNearestTableOfContentsAboveThePageIsFoundAndMissesAreRemembered(t *testing.T) {
	site := newTOCSite(t, map[string]string{
		"/.well-known/toc.opml": opmlWith(entry("{base}/blog/2026/a.md"), entry("{base}/blog/2026/b.md"), entry("{base}/blog/2026/c.md")),
	})
	f := NewFinder()
	r := f.Lookup(context.Background(), site.URL+"/blog/2026/b.html")
	if !r.Found || r.Prev != site.URL+"/blog/2026/a.md" || r.Next != site.URL+"/blog/2026/c.md" {
		t.Fatalf("result %+v", r)
	}
	if r.TOC != site.URL+"/.well-known/toc.opml" {
		t.Errorf("toc %q", r.TOC)
	}
	// every folder above the page was probed, once
	for _, p := range []string{"/blog/2026/.well-known/toc.opml", "/blog/.well-known/toc.opml", "/.well-known/toc.opml"} {
		if site.count(p) != 1 {
			t.Errorf("%s fetched %d times, want 1", p, site.count(p))
		}
	}
	// the next question about the same site costs nothing: hits and misses are remembered
	f.Lookup(context.Background(), site.URL+"/blog/2026/c.html")
	for _, p := range []string{"/blog/2026/.well-known/toc.opml", "/blog/.well-known/toc.opml", "/.well-known/toc.opml"} {
		if site.count(p) != 1 {
			t.Errorf("after a second lookup %s was fetched %d times", p, site.count(p))
		}
	}
}

func TestADeeperTableOfContentsWinsWhenItListsThePage(t *testing.T) {
	site := newTOCSite(t, map[string]string{
		"/.well-known/toc.opml":   opmlWith(entry("{base}/s/a.md"), entry("{base}/s/b.md")),
		"/s/.well-known/toc.opml": opmlWith(entry("{base}/s/x.md"), entry("{base}/s/a.md"), entry("{base}/s/y.md")),
	})
	r := NewFinder().Lookup(context.Background(), site.URL+"/s/a.md")
	if r.TOC != site.URL+"/s/.well-known/toc.opml" || r.Prev != site.URL+"/s/x.md" || r.Next != site.URL+"/s/y.md" {
		t.Errorf("%+v", r)
	}
}

func TestATableOfContentsThatDoesNotListThePageIsPassedOver(t *testing.T) {
	site := newTOCSite(t, map[string]string{
		"/s/.well-known/toc.opml": opmlWith(entry("{base}/s/other.md")),
		"/.well-known/toc.opml":   opmlWith(entry("{base}/s/a.md"), entry("{base}/s/b.md")),
	})
	r := NewFinder().Lookup(context.Background(), site.URL+"/s/a.md")
	if !r.Found || r.TOC != site.URL+"/.well-known/toc.opml" || r.Next != site.URL+"/s/b.md" {
		t.Errorf("%+v", r)
	}
}

func TestNoTableOfContentsMeansNotFoundNotAnError(t *testing.T) {
	site := newTOCSite(t, nil)
	r := NewFinder().Lookup(context.Background(), site.URL+"/a/b/c.md")
	if r.Found || r.Prev != "" || r.Next != "" {
		t.Errorf("%+v", r)
	}
	if r.Err != nil {
		t.Errorf("a site with no table of contents is not an error: %v", r.Err)
	}
}

func TestASiteWhoseFileIsNotAnOutlineIsSkipped(t *testing.T) {
	site := newTOCSite(t, map[string]string{
		"/.well-known/toc.opml": "<html><body>Not here, a soft 404 page</body></html>",
	})
	r := NewFinder().Lookup(context.Background(), site.URL+"/a.md")
	if r.Found {
		t.Errorf("%+v", r)
	}
}

// Dave Winer's root file is one include pointing at the section's own file.
func TestIncludesOnTheSameHostAreMergedAndOthersAreNot(t *testing.T) {
	site := newTOCSite(t, map[string]string{
		"/.well-known/toc.opml": `<opml version="2.0"><body>
			<outline text="Scripting News" type="include" url="{base}/scripting/.well-known/toc.opml"/>
			<outline text="Elsewhere" type="include" url="http://other.example/x/toc.opml"/></body></opml>`,
		"/scripting/.well-known/toc.opml": `<opml version="2.0"><body><outline text="2026" type="calendarYear">
			<outline text="d2" type="calendarDay" url="{base}/scripting/2026/11/19/a.md"/>
			<outline text="d1" type="calendarDay" url="{base}/scripting/2026/11/18/b.md"/></outline></body></opml>`,
	})
	r := NewFinder().Lookup(context.Background(), site.URL+"/scripting/2026/11/19/a.md")
	// found in the section's own file, the nearest above the page
	if !r.Found || r.Prev != site.URL+"/scripting/2026/11/18/b.md" || r.Next != "" {
		t.Errorf("%+v", r)
	}
	// a root-only site: the page is known only through the include
	rootOnly := newTOCSite(t, map[string]string{
		"/.well-known/toc.opml": `<opml version="2.0"><body><outline text="S" type="include" url="{base}/inc.opml"/></body></opml>`,
		"/inc.opml":             opmlWith(entry("{base}/p/a.md"), entry("{base}/p/b.md")),
	})
	r = NewFinder().Lookup(context.Background(), rootOnly.URL+"/p/a.md")
	if !r.Found || r.Next != rootOnly.URL+"/p/b.md" {
		t.Errorf("include not merged: %+v", r)
	}
}

func TestIncludeLoopsAndDeepChainsDoNotRunAway(t *testing.T) {
	site := newTOCSite(t, map[string]string{
		"/.well-known/toc.opml": `<opml version="2.0"><body><outline text="me" type="include" url="{base}/.well-known/toc.opml"/>
			<outline text="a" type="include" url="{base}/a.opml"/></body></opml>`,
		"/a.opml": `<opml version="2.0"><body><outline text="b" type="include" url="{base}/b.opml"/></body></opml>`,
		"/b.opml": `<opml version="2.0"><body><outline text="a" type="include" url="{base}/a.opml"/></body></opml>`,
	})
	r := NewFinder().Lookup(context.Background(), site.URL+"/x.md")
	if r.Found {
		t.Errorf("%+v", r)
	}
	if site.count("/a.opml") > 1 || site.count("/b.opml") > 1 || site.count("/.well-known/toc.opml") > 1 {
		t.Errorf("an include was fetched more than once: %v", site.hits)
	}
}

func TestPagesOnAnotherSchemeOrNonWebAddressesGetNoLookup(t *testing.T) {
	f := NewFinder()
	for _, page := range []string{"file:///tmp/x.md", "ftp://x.example/a.md", "", "not an address"} {
		if r := f.Lookup(context.Background(), page); r.Found || r.Err != nil {
			t.Errorf("%q: %+v", page, r)
		}
	}
}

func TestLookupsAreLoggedWithoutContent(t *testing.T) {
	site := newTOCSite(t, map[string]string{"/.well-known/toc.opml": opmlWith(entry("{base}/a.md"), entry("{base}/b.md"))})
	var buf bytes.Buffer
	ctx := debuglog.WithContext(context.Background(), debuglog.New(&buf))
	NewFinder().Lookup(ctx, site.URL+"/a.md")
	for _, want := range []string{`"event":"toc.fetch"`, `"status":200`, `"entries":2`, `"event":"toc.lookup"`, `"found":true`} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("log lacks %s:\n%s", want, buf.String())
		}
	}
}

func TestAHugeFileIsRefused(t *testing.T) {
	big := newTOCSite(t, map[string]string{"/.well-known/toc.opml": opmlWith(strings.Repeat(entry("http://x.example/p.md"), 400000))})
	if r := NewFinder().Lookup(context.Background(), big.URL+"/p.md"); r.Found {
		t.Error("a file over the size cap was read")
	}
}
