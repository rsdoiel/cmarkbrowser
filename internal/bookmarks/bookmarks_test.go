package bookmarks

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var day = time.Date(2026, 11, 19, 20, 30, 0, 0, time.UTC)

func links(t *testing.T, src string) [][2]string {
	t.Helper()
	b := []byte(src)
	var out [][2]string
	ast.Walk(goldmark.New().Parser().Parse(text.NewReader(b)), func(n ast.Node, e bool) (ast.WalkStatus, error) {
		if l, ok := n.(*ast.Link); ok && e {
			out = append(out, [2]string{string(util.UnescapePunctuations(l.Text(b))), string(l.Destination)})
		}
		return ast.WalkContinue, nil
	})
	return out
}

func TestAddCreatesTheFileWithAHeadingAndTheEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bookmarks.md")
	added, err := Add(path, "A Page", "https://example.org/a", day)
	if err != nil || !added {
		t.Fatalf("added %v, err %v", added, err)
	}
	b, _ := os.ReadFile(path)
	src := string(b)
	if !strings.HasPrefix(src, "# Bookmarks\n\n") {
		t.Errorf("no heading:\n%s", src)
	}
	if !strings.Contains(src, "- [A Page](https://example.org/a)") || !strings.Contains(src, "2026-11-19") {
		t.Errorf("entry:\n%s", src)
	}
	if !strings.HasSuffix(src, "\n") {
		t.Error("no final newline")
	}
}

func TestAddAppendsAndDoesNotDuplicate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "b.md")
	Add(path, "One", "https://example.org/1", day)
	added, err := Add(path, "Two", "https://example.org/2", day)
	if err != nil || !added {
		t.Fatalf("second add: %v %v", added, err)
	}
	added, err = Add(path, "One again", "https://example.org/1", day)
	if err != nil || added {
		t.Errorf("a duplicate was added: %v %v", added, err)
	}
	b, _ := os.ReadFile(path)
	got := links(t, string(b))
	if len(got) != 2 || got[0][1] != "https://example.org/1" || got[1][1] != "https://example.org/2" {
		t.Errorf("links %v in:\n%s", got, b)
	}
}

func TestEntriesParseBackToTheSameTitleAndAddress(t *testing.T) {
	path := filepath.Join(t.TempDir(), "b.md")
	cases := []struct{ title, addr string }{
		{"Brackets [in] a title", "https://example.org/a"},
		{"Line\nbreak  and   spaces", "https://example.org/b"},
		{"Pipe | star * under_score `tick` <tag>", "https://example.org/c"},
		{"Parens (and) more", "https://example.org/wiki/Go_(language)"},
		{"Query", "https://example.org/q?a=1&b=2#frag"},
		{"", "https://example.org/untitled"},
	}
	for _, c := range cases {
		if _, err := Add(path, c.title, c.addr, day); err != nil {
			t.Fatalf("%q: %s", c.title, err)
		}
	}
	b, _ := os.ReadFile(path)
	got := links(t, string(b))
	if len(got) != len(cases) {
		t.Fatalf("%d links for %d entries:\n%s", len(got), len(cases), b)
	}
	for i, c := range cases {
		wantTitle := strings.Join(strings.Fields(c.title), " ")
		if wantTitle == "" {
			wantTitle = c.addr // an untitled page is listed by its address
		}
		if got[i][0] != wantTitle || got[i][1] != c.addr {
			t.Errorf("entry %d: got %q, want [%q](%s)", i, got[i], wantTitle, c.addr)
		}
	}
	if strings.Contains(string(b), "<tag>") {
		t.Errorf("raw HTML in a bookmark:\n%s", b)
	}
}

func TestAddToAFileWithNoFinalNewlineAndAnExistingHeading(t *testing.T) {
	path := filepath.Join(t.TempDir(), "b.md")
	os.WriteFile(path, []byte("# My own heading\n\n- [Old](https://old.example/)"), 0o644)
	if _, err := Add(path, "New", "https://new.example/", day); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(b), "# My own heading\n") || strings.Count(string(b), "# ") != 1 {
		t.Errorf("the author's heading was changed:\n%s", b)
	}
	if got := links(t, string(b)); len(got) != 2 {
		t.Errorf("links %v in:\n%s", got, b)
	}
}

func TestAddNeverTouchesWhatIsAlreadyThere(t *testing.T) {
	path := filepath.Join(t.TempDir(), "b.md")
	orig := "# Reading list\n\nSome notes I wrote.\n\n- [Old](https://old.example/) (2020-01-01)\n"
	os.WriteFile(path, []byte(orig), 0o644)
	Add(path, "New", "https://new.example/", day)
	b, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(b), orig) {
		t.Errorf("existing text changed:\n%s", b)
	}
}

func TestAddReportsAnUnwritableFile(t *testing.T) {
	_, err := Add(filepath.Join(t.TempDir(), "no", "such", "dir", "b.md"), "T", "https://x.org/", day)
	if err == nil {
		t.Fatal("no error for a directory that does not exist")
	}
	var pe *os.PathError
	if !errors.As(err, &pe) {
		t.Errorf("err %T %v: want the OS error so the caller can classify it", err, err)
	}
}

func TestOnlyWebAddressesAreBookmarked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "b.md")
	for _, a := range []string{"file:///etc/passwd", "javascript:alert(1)", "data:text/html,x", "", "/relative"} {
		added, err := Add(path, "T", a, day)
		if added || !errors.Is(err, ErrNotWeb) {
			t.Errorf("%q: added %v err %v, want ErrNotWeb", a, added, err)
		}
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("a file was created for nothing")
	}
}

func TestHas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "b.md")
	Add(path, "One", "https://example.org/1", day)
	if !Has(path, "https://example.org/1") || Has(path, "https://example.org/2") || Has(filepath.Join(t.TempDir(), "none.md"), "x") {
		t.Error("Has is wrong")
	}
}
