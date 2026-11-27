package cli

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rsdoiel/cmarkbrowser/internal/config"
	"github.com/rsdoiel/cmarkfind/corpus"
	"github.com/rsdoiel/markdownweb/toc"
)

// makeCorpus fills a corpus from a tiny local site whose pages mention word.
func makeCorpus(t *testing.T, word string) string {
	t.Helper()
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a.md"), []byte("# Title about "+word+"\n\nText with "+word+" in it.\n"), 0o644)
	os.MkdirAll(filepath.Join(root, ".well-known"), 0o755)
	os.WriteFile(filepath.Join(root, ".well-known", "toc.opml"),
		[]byte(`<opml version="2.0"><head/><body><outline text="a" url="https://pub.example/a.md"/></body></opml>`), 0o644)
	path := filepath.Join(t.TempDir(), "corpus.db")
	db, err := corpus.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.FillLocal(context.Background(), root, toc.Limits{}); err != nil {
		t.Fatal(err)
	}
	return path
}

// noConfig points the configuration directory at an empty folder.
func noConfig(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CMARKFIND_DB", "")
}

func TestFindPrintsTheResultsPageAndExitsZero(t *testing.T) {
	noConfig(t)
	db := makeCorpus(t, "badgers")
	code, out, errs := run("-corpus", db, "-find", "badgers")
	if code != 0 || errs != "" {
		t.Fatalf("code %d err %q", code, errs)
	}
	for _, want := range []string{"# Search: badgers", "1 hit, searched 1 page", "[Title about badgers](https://pub.example/a.md)"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestFindWithNoMatchExitsOneAndStillPrintsThePage(t *testing.T) {
	noConfig(t)
	code, out, _ := run("-corpus", makeCorpus(t, "badgers"), "-find", "zebra")
	if code != 1 || !strings.Contains(out, "no matches") {
		t.Errorf("code %d out %q", code, out)
	}
}

func TestFindExitCodes(t *testing.T) {
	noConfig(t)
	good := makeCorpus(t, "badgers")
	notDB := filepath.Join(t.TempDir(), "x.db")
	os.WriteFile(notDB, []byte(strings.Repeat("not a database ", 500)), 0o644)
	newer := makeCorpus(t, "badgers")
	raw, _ := sql.Open("sqlite", newer)
	raw.Exec(`UPDATE meta SET value = '999' WHERE key = 'schema_version'`)
	raw.Close()
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"missing corpus", []string{"-corpus", filepath.Join(t.TempDir(), "none.db"), "-find", "x"}, 66},
		{"not a corpus", []string{"-corpus", notDB, "-find", "x"}, 65},
		{"newer schema", []string{"-corpus", newer, "-find", "x"}, 65},
		{"with a URL", []string{"-corpus", good, "-find", "x", "https://example.org/"}, 2},
		{"with -no-corpus", []string{"-no-corpus", "-find", "x"}, 2},
		{"with -dump", []string{"-corpus", good, "-dump", "-find", "x"}, 2},
		{"corpus and no-corpus", []string{"-corpus", good, "-no-corpus", "-find", "x"}, 2},
		{"empty query", []string{"-corpus", good, "-find", " "}, 2},
	}
	for _, c := range cases {
		if code, out, errs := run(c.args...); code != c.want {
			t.Errorf("%s: exit %d, want %d\nout %q\nerr %q", c.name, code, c.want, out, errs)
		}
	}
}

// The corpus is named by -corpus, else the configuration file, else
// CMARKFIND_DB, else cmarkfind's default (cmarkbrowser DR-0003 2).
func TestWhichCorpusFileIsUsed(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	env, yamlDB, flagDB := makeCorpus(t, "fromenv"), makeCorpus(t, "fromyaml"), makeCorpus(t, "fromflag")
	t.Setenv("CMARKFIND_DB", env)
	has := func(args []string, word string) bool {
		code, _, _ := run(append(args, "-find", word)...)
		return code == 0
	}
	if !has(nil, "fromenv") {
		t.Error("CMARKFIND_DB should be used when nothing else names a file")
	}
	os.MkdirAll(filepath.Join(cfgDir, "cmarkbrowser"), 0o755)
	os.WriteFile(filepath.Join(cfgDir, "cmarkbrowser", config.FileName), []byte("corpus: "+yamlDB+"\n"), 0o644)
	if !has(nil, "fromyaml") || has(nil, "fromenv") {
		t.Error("the configuration file beats CMARKFIND_DB")
	}
	if !has([]string{"-corpus", flagDB}, "fromflag") || has([]string{"-corpus", flagDB}, "fromyaml") {
		t.Error("-corpus beats the configuration file")
	}
}

func TestCorpusPathFallsBackToCmarkfindsDefault(t *testing.T) {
	t.Setenv("CMARKFIND_DB", "")
	if got := corpusPath("", false, config.Config{}); got != corpus.DefaultPath() {
		t.Errorf("got %q", got)
	}
	if got := corpusPath("x.db", true, config.Config{}); got != "" {
		t.Errorf("-no-corpus gave %q", got)
	}
}
