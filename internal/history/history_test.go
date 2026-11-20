package history

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 20, 14, 3, 0, 0, time.UTC)

func lines(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "- [") {
			out = append(out, l)
		}
	}
	return out
}

func TestTheNewestVisitIsAtTheTop(t *testing.T) {
	p := filepath.Join(t.TempDir(), "history.md")
	for i, a := range []string{"https://x.org/1", "https://x.org/2", "https://x.org/3"} {
		if err := Record(p, "Page", a, t0.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	got := lines(t, p)
	if len(got) != 3 || !strings.Contains(got[0], "x.org/3") || !strings.Contains(got[2], "x.org/1") {
		t.Errorf("%v", got)
	}
	if !strings.Contains(got[0], "(2026-11-20 14:05)") {
		t.Errorf("stamp: %s", got[0])
	}
}

func TestAnAddressVisitedAgainMovesToTheTopWithoutDuplicating(t *testing.T) {
	p := filepath.Join(t.TempDir(), "history.md")
	Record(p, "One", "https://x.org/1", t0)
	Record(p, "Two", "https://x.org/2", t0)
	Record(p, "One again", "https://x.org/1", t0.Add(time.Hour))
	got := lines(t, p)
	if len(got) != 2 || !strings.Contains(got[0], "One again") || !strings.Contains(got[1], "Two") {
		t.Errorf("%v", got)
	}
}

func TestAnAddressThatIsOnlyAPrefixOfAnotherIsNotTheSameEntry(t *testing.T) {
	p := filepath.Join(t.TempDir(), "history.md")
	Record(p, "Long", "https://x.org/a/b", t0)
	Record(p, "Short", "https://x.org/a", t0)
	if got := lines(t, p); len(got) != 2 {
		t.Errorf("%v", got)
	}
}

func TestTitlesCannotBecomeMarkup(t *testing.T) {
	p := filepath.Join(t.TempDir(), "history.md")
	Record(p, "<script>alert(1)</script> [x](y)", "https://x.org/a", t0)
	got := lines(t, p)[0]
	if strings.Contains(got, "<script>") || strings.Contains(got, "[x](y)") {
		t.Errorf("unescaped: %s", got)
	}
}

func TestOnlyWebAddressesAreRecorded(t *testing.T) {
	p := filepath.Join(t.TempDir(), "history.md")
	for _, a := range []string{"file:///etc/passwd", "javascript:alert(1)", "notes.md", ""} {
		if err := Record(p, "t", a, t0); !errors.Is(err, ErrNotWeb) {
			t.Errorf("%q: %v", a, err)
		}
	}
	if _, err := os.Stat(p); err == nil {
		t.Error("a file was written")
	}
}

func TestTheListIsCapped(t *testing.T) {
	p := filepath.Join(t.TempDir(), "history.md")
	for i := 0; i < MaxEntries+5; i++ {
		Record(p, "p", "https://x.org/"+strings.Repeat("a", 1)+string(rune('0'+i%10))+"/"+time.Duration(i).String(), t0)
	}
	if got := lines(t, p); len(got) != MaxEntries {
		t.Errorf("%d entries", len(got))
	}
}

func TestOtherTextInTheFileIsNotKeptAndNoTempFileIsLeft(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "history.md")
	os.WriteFile(p, []byte("# Mine\n\nnotes I typed\n\n- [Old](https://x.org/old) (2026-01-01 00:00)\n"), 0o644)
	if err := Record(p, "New", "https://x.org/new", t0); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if strings.Contains(string(b), "notes I typed") || len(lines(t, p)) != 2 {
		t.Errorf("%s", b)
	}
	if ents, _ := os.ReadDir(dir); len(ents) != 1 {
		t.Errorf("leftover files: %v", ents)
	}
}

func TestAnUnwritableDirectoryIsAnError(t *testing.T) {
	if err := Record(filepath.Join(t.TempDir(), "no", "such", "history.md"), "t", "https://x.org/", t0); err == nil {
		t.Error("expected an error")
	}
}
