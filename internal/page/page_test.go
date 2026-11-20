package page

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseFrontMatter(t *testing.T) {
	src := "---\ntitle: Hello\nauthor: Dave\nurlNext: /two.md\n---\n# Body\n\ntext\n"
	p, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if p.Title() != "Hello" || p.Meta["author"] != "Dave" || p.URLNext() != "/two.md" {
		t.Errorf("meta %#v", p.Meta)
	}
	if strings.Contains(p.Body, "title:") || !strings.HasPrefix(p.Body, "# Body") {
		t.Errorf("body %q", p.Body)
	}
}

func TestParseNoFrontMatter(t *testing.T) {
	p, err := Parse([]byte("# Just text\n"))
	if err != nil || p.Body != "# Just text\n" || len(p.Meta) != 0 {
		t.Errorf("%#v %v", p, err)
	}
}

func TestParseBadYAMLIsDataError(t *testing.T) {
	_, err := Parse([]byte("---\ntitle: [unclosed\n---\nbody\n"))
	if err == nil || !strings.Contains(err.Error(), "front matter") {
		t.Errorf("err %v", err)
	}
}

func TestParseUnterminatedFrontMatterIsBody(t *testing.T) {
	src := "---\nnot closed\n"
	p, err := Parse([]byte(src))
	if err != nil || p.Body != src {
		t.Errorf("%#v %v", p, err)
	}
}

func TestIsMarkdown(t *testing.T) {
	cases := []struct {
		ct, path string
		want     bool
	}{
		{"text/markdown; charset=utf-8", "/a", true},
		{"text/plain", "/a.md", true},
		{"text/plain; charset=utf-8", "/a.markdown", true},
		{"application/octet-stream", "/a.md", true},
		{"text/plain", "/a.txt", false},
		{"text/html", "/a.md", false},
	}
	for _, c := range cases {
		if got := IsMarkdown(c.ct, c.path); got != c.want {
			t.Errorf("%q %q: %v", c.ct, c.path, got)
		}
	}
}

func TestFetchSendsBonBonAcceptAndReturnsMarkdown(t *testing.T) {
	var accept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Write([]byte("---\ntitle: T\n---\nhi\n"))
	}))
	defer srv.Close()
	p, err := Fetch(context.Background(), srv.URL+"/x")
	if err != nil {
		t.Fatal(err)
	}
	if accept != "text/markdown, text/x-opml" {
		t.Errorf("Accept %q", accept)
	}
	if p.Title() != "T" || p.Body != "hi\n" {
		t.Errorf("%#v", p)
	}
}

func TestFetchHTMLIsNotAPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<p>hi</p>"))
	}))
	defer srv.Close()
	_, err := Fetch(context.Background(), srv.URL+"/x")
	if err == nil || !strings.Contains(err.Error(), "not Markdown") {
		t.Errorf("err %v", err)
	}
}

func TestFetchMissingPage(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	_, err := Fetch(context.Background(), srv.URL+"/x")
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("err %v", err)
	}
}
