package browse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rsdoiel/cmarkbrowser/internal/render"
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
