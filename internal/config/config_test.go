package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// home points the configuration directory at a fresh temporary directory and
// returns the path of the file Load reads.
func home(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	return filepath.Join(dir, "cmarkbrowser", "cmarkbrowser.yaml")
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMissingFileIsNotAnError(t *testing.T) {
	home(t)
	c, err := Load()
	if err != nil || c.UserAgent != "" {
		t.Fatalf("got %+v, %v", c, err)
	}
}

func TestPathIsUnderXDGConfigHome(t *testing.T) {
	want := home(t)
	got, err := Path()
	if err != nil || got != want {
		t.Fatalf("Path() = %q, %v; want %q", got, err, want)
	}
}

func TestReadsUserAgent(t *testing.T) {
	p := home(t)
	write(t, p, "user-agent: my-reader/1.0 (+https://example.org)\n")
	c, err := Load()
	if err != nil || c.UserAgent != "my-reader/1.0 (+https://example.org)" {
		t.Fatalf("got %+v, %v", c, err)
	}
}

func TestEmptyFileIsFine(t *testing.T) {
	p := home(t)
	write(t, p, "")
	if c, err := Load(); err != nil || c.UserAgent != "" {
		t.Fatalf("got %+v, %v", c, err)
	}
}

func TestKeysForLaterSettingsAreNotAnError(t *testing.T) {
	p := home(t)
	write(t, p, "style: light\nwidth: 100\n")
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
}

func TestWrongFileIsErrConfigNamingThePath(t *testing.T) {
	for name, text := range map[string]string{
		"not YAML":      "user-agent: [unclosed\n",
		"not a mapping": "- a\n- b\n",
		"wrong type":    "user-agent: [a, b]\n",
		"control chars": "user-agent: \"a\\nb\"\n",
	} {
		p := home(t)
		write(t, p, text)
		_, err := Load()
		if !errors.Is(err, ErrConfig) {
			t.Errorf("%s: err %v", name, err)
			continue
		}
		if !contains(err.Error(), p) {
			t.Errorf("%s: error does not name the file: %v", name, err)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestUnreadableFileIsErrConfig(t *testing.T) {
	p := home(t)
	write(t, p, "x: 1\n")
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	if f, err := os.Open(p); err == nil {
		f.Close()
		t.Skip("running as a user who can read mode 0 files")
	}
	if _, err := Load(); !errors.Is(err, ErrConfig) {
		t.Fatalf("err %v", err)
	}
}

func TestReadsCorpus(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	os.MkdirAll(filepath.Join(dir, "cmarkbrowser"), 0o755)
	os.WriteFile(filepath.Join(dir, "cmarkbrowser", FileName), []byte("corpus: /data/c.db\n"), 0o644)
	c, err := Load()
	if err != nil || c.Corpus != "/data/c.db" {
		t.Errorf("%+v %v", c, err)
	}
}
