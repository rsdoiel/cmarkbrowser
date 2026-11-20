package edit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestVisualWinsOverEditorAndArgumentsAreKept(t *testing.T) {
	c, err := Command(env(map[string]string{"VISUAL": "code -w", "EDITOR": "vi"}), "/x/a.md")
	if err != nil || len(c.Args) != 3 || c.Args[0] != "code" || c.Args[1] != "-w" || c.Args[2] != "/x/a.md" {
		t.Errorf("%v %v", c, err)
	}
	c, err = Command(env(map[string]string{"EDITOR": "  nano "}), "/x/a.md")
	if err != nil || c.Args[0] != "nano" || c.Args[len(c.Args)-1] != "/x/a.md" {
		t.Errorf("%v %v", c, err)
	}
}

func TestNoEditorIsAnErrorNotAGuess(t *testing.T) {
	for _, e := range []map[string]string{{}, {"VISUAL": "  ", "EDITOR": ""}} {
		if _, err := Command(env(e), "/x/a.md"); !errors.Is(err, ErrNoEditor) {
			t.Errorf("%v: %v", e, err)
		}
	}
}

func TestAFileNameWithSpacesStaysOneArgument(t *testing.T) {
	c, _ := Command(env(map[string]string{"EDITOR": "vi"}), "/my notes/a b.md")
	if c.Args[len(c.Args)-1] != "/my notes/a b.md" {
		t.Errorf("%v", c.Args)
	}
}

func TestACopyIsNamedFromTheAddressAndNeverReplacesAFile(t *testing.T) {
	dir := t.TempDir()
	p1, err := SaveCopy(dir, "https://x.org/blog/2026/About.md?x=1#top", "one")
	if err != nil || filepath.Base(p1) != "About.md" { // case kept
		t.Fatalf("%q %v", p1, err)
	}
	p2, _ := SaveCopy(dir, "https://x.org/blog/2026/About.md", "two")
	p3, _ := SaveCopy(dir, "https://x.org/other/About.md", "three")
	if filepath.Base(p2) != "About-1.md" || filepath.Base(p3) != "About-2.md" {
		t.Errorf("%q %q", p2, p3)
	}
	if b, _ := os.ReadFile(p1); string(b) != "one" {
		t.Errorf("the first copy was overwritten: %q", b)
	}
}

func TestCopyNamesAreSafe(t *testing.T) {
	for addr, want := range map[string]string{
		"https://x.org/":              "index.md",
		"https://x.org":               "index.md",
		"https://x.org/dir/":          "dir.md",
		"https://x.org/a/b.html":      "b.md",
		"https://x.org/a/my notes.md": "my_notes.md",
		"https://x.org/a/.hidden":     "hidden.md",
		"https://x.org/a/notes.txt":   "notes.txt",
	} {
		got := copyName(addr)
		if strings.ContainsAny(got, "/\\") || strings.HasPrefix(got, ".") {
			t.Errorf("%s: unsafe name %q", addr, got)
		}
		if got != want {
			t.Errorf("%s: %q, want %q", addr, got, want)
		}
	}
}

func TestATraversalInTheAddressCannotLeaveTheFolder(t *testing.T) {
	for _, addr := range []string{"https://x.org/a/..%2F..%2Fevil.md", "https://x.org/../../evil.md", "https://x.org/a/..", "https://x.org/a/%00.md"} {
		got := copyName(addr)
		if strings.ContainsAny(got, "/\\") || strings.HasPrefix(got, ".") || got == "" {
			t.Errorf("%s: %q", addr, got)
		}
	}
}

func TestACopyInAMissingFolderIsAnError(t *testing.T) {
	if _, err := SaveCopy(filepath.Join(t.TempDir(), "nope"), "https://x.org/a.md", "x"); err == nil {
		t.Error("expected an error")
	}
}

func TestDownloadsDirNeedsTheFolderToExist(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if _, err := DownloadsDir(); err == nil {
		t.Error("no Downloads folder yet")
	}
	os.Mkdir(filepath.Join(home, "Downloads"), 0o755)
	if d, err := DownloadsDir(); err != nil || d != filepath.Join(home, "Downloads") {
		t.Errorf("%q %v", d, err)
	}
}
