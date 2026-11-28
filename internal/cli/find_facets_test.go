package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rsdoiel/cmarkfind/corpus"
	"github.com/rsdoiel/markdownweb/toc"
)

// cmarkbrowser DR-0004 4: -find prints the same page, facets included, and
// exits 0 with hits and 1 with none.
func facetedCorpusFile(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	pages := map[string]string{
		"a.md": "---\nkeywords: [go]\n---\n# A\n\ncompilers a\n",
		"b.md": "---\nkeywords: [go, oberon]\n---\n# B\n\ncompilers b\n",
	}
	var lines strings.Builder
	for n, body := range pages {
		os.WriteFile(filepath.Join(root, n), []byte(body), 0o644)
		lines.WriteString(`<outline text="` + n + `" url="https://f.example/` + n + `"/>`)
	}
	os.MkdirAll(filepath.Join(root, ".well-known"), 0o755)
	os.WriteFile(filepath.Join(root, ".well-known", "toc.opml"), []byte(`<opml version="2.0"><head/><body>`+lines.String()+`</body></opml>`), 0o644)
	path := filepath.Join(t.TempDir(), "corpus.db")
	db, err := corpus.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if _, err := db.FillLocal(ctx, root, toc.Limits{}); err != nil {
		t.Fatal(err)
	}
	if err := db.AssignFields(ctx, root, []string{"keywords=facet,fielded,Topics"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.BuildSets(ctx, "", 0); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFindPrintsTheFacetsAndExitsZeroWithHitsAndOneWithNone(t *testing.T) {
	noConfig(t)
	db := facetedCorpusFile(t)
	code, out, errs := run("-corpus", db, "-find", "compilers")
	if code != 0 || errs != "" {
		t.Fatalf("code %d err %q", code, errs)
	}
	for _, want := range []string{"- Facets\n", "  - Topics\n", "    - [go (2)](corpus:search?q=", "    - [oberon (1)](corpus:search?q=", "- Results\n", "  1. [A](https://f.example/a.md)"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	code, out, _ = run("-corpus", db, "-find", "zebra")
	if code != 1 || !strings.Contains(out, "no matches") {
		t.Errorf("no hits: code %d\n%s", code, out)
	}
}
