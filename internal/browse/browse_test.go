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
