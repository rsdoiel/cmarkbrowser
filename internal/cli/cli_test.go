package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func run(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestStandardOptions(t *testing.T) {
	for _, opt := range []string{"-h", "-help", "--help", "-l", "-license", "--license", "-v", "-version", "--version"} {
		code, out, errs := run(opt)
		if code != 0 || out == "" || errs != "" {
			t.Errorf("%s: code %d, stdout %q, stderr %q", opt, code, out, errs)
		}
	}
}

func TestHelpFillsPlaceholders(t *testing.T) {
	_, out, _ := run("-h")
	if strings.Contains(out, "{app_name}") || strings.Contains(out, "{version}") {
		t.Errorf("help still has placeholders:\n%s", out)
	}
}

func TestUnknownOptionIsUsage(t *testing.T) {
	code, out, errs := run("-nope")
	if code != ExitUsage || out != "" || !strings.HasPrefix(errs, "cmarkbrowser: ") || strings.Count(errs, "\n") != 1 {
		t.Errorf("code %d, stdout %q, stderr %q", code, out, errs)
	}
}

func TestTooManyArgumentsIsUsage(t *testing.T) {
	if code, _, _ := run("https://a.example/", "https://b.example/"); code != ExitUsage {
		t.Errorf("code %d, want %d", code, ExitUsage)
	}
}

func TestDumpPrintsRenderedPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		w.Write([]byte("---\ntitle: T\n---\n# Heading\n\nSee [x](y.md) <script>bad()</script>\n"))
	}))
	defer srv.Close()
	code, out, errs := run("-dump", "-style", "notty", srv.URL+"/p.md")
	if code != 0 || errs != "" {
		t.Fatalf("code %d, stderr %q", code, errs)
	}
	if !strings.Contains(out, "Heading") || strings.Contains(out, "bad()") || !strings.Contains(out, "[1]") {
		t.Errorf("out %q", out)
	}
	if !strings.Contains(out, srv.URL+"/y.md") {
		t.Errorf("link list missing: %q", out)
	}
}

func TestDumpWithoutURLIsUsage(t *testing.T) {
	if code, _, _ := run("-dump"); code != ExitUsage {
		t.Errorf("code %d", code)
	}
}

func TestFetchFailureClasses(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL + "/x.md"
	code404, _, _ := run("-dump", url)
	srv.Close()
	codeDown, _, _ := run("-dump", url)
	html := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
	}))
	defer html.Close()
	codeHTML, _, _ := run("-dump", html.URL+"/x")
	if code404 != 66 || codeDown != 69 || codeHTML != 65 {
		t.Errorf("404 %d, down %d, html %d; want 66, 69, 65", code404, codeDown, codeHTML)
	}
}

func TestNoURLIsUsage(t *testing.T) {
	if code, _, errs := run(); code != ExitUsage || !strings.Contains(errs, "URL is needed") {
		t.Errorf("code %d, stderr %q", code, errs)
	}
}

func TestBrowserNeedsATerminal(t *testing.T) {
	code, _, errs := run("https://example.org/")
	if code != ExitUsage || !strings.Contains(errs, "needs a terminal") {
		t.Errorf("code %d, stderr %q", code, errs)
	}
}

func TestExitCodeBySentinelNotMessage(t *testing.T) {
	if got := ExitCode(fmt.Errorf("%w: x", ErrUsage)); got != ExitUsage {
		t.Errorf("wrapped usage: %d", got)
	}
	if got := ExitCode(errors.New("usage")); got != ExitInternal {
		t.Errorf("message text must not classify: %d", got)
	}
	if got := ExitCode(nil); got != ExitOK {
		t.Errorf("nil: %d", got)
	}
}

func TestDumpWarnsAboutMissingMacroKeys(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		w.Write([]byte("---\ntitle: T\n---\n# {{page.title}} {{page.nope}}\n"))
	}))
	defer srv.Close()
	var out, errb bytes.Buffer
	code := Run([]string{"-dump", "-style", "notty", srv.URL + "/p.md"}, &out, &errb)
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(errb.String(), "cmarkbrowser: warning:") || !strings.Contains(errb.String(), "page.nope") {
		t.Errorf("stderr %q", errb.String())
	}
	if strings.Contains(out.String(), "warning") {
		t.Errorf("warning on stdout: %q", out.String())
	}
}

// readDebug returns the events in a debug log file.
func readDebug(t *testing.T, path string) []map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no debug log: %s", err)
	}
	var evs []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		var e map[string]any
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("not JSON: %q", line)
		}
		evs = append(evs, e)
	}
	return evs
}

func TestDebugFlagWritesASessionLog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		w.Write([]byte("---\ntitle: Secret Title\n---\n# {{page.title}}\n\nSecret body text [x](y.md).\n"))
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "session.jsonl")
	code, out, errs := run("-dump", "-style", "notty", "-debug", path, srv.URL+"/p.md")
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if !strings.Contains(out, "Secret body text") {
		t.Errorf("the page was not printed: %q", out)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("log mode %v, want 0600: it records the addresses visited", info.Mode().Perm())
	}
	evs := readDebug(t, path)
	names := []string{}
	for _, e := range evs {
		names = append(names, e["event"].(string))
	}
	if names[0] != "start" || names[len(names)-1] != "exit" {
		t.Errorf("events %v: want start first and exit last", names)
	}
	start := evs[0]
	if start["mode"] != "dump" || start["version"] == nil || start["schema"] != float64(1) {
		t.Errorf("start event %v", start)
	}
	if evs[len(evs)-1]["code"] != float64(0) {
		t.Errorf("exit event %v", evs[len(evs)-1])
	}
	has := func(n string) bool {
		for _, x := range names {
			if x == n {
				return true
			}
		}
		return false
	}
	if !has("fetch.try") || !has("fetch.result") || !has("open") {
		t.Errorf("events %v", names)
	}
	b, _ := os.ReadFile(path)
	if strings.Contains(string(b), "Secret") {
		t.Errorf("the log holds page content:\n%s", b)
	}
}

func TestDebugLogRecordsTheExitCodeOfAFailure(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "s.jsonl")
	code, _, _ := run("-dump", "-debug", path, srv.URL+"/nope.md")
	if code != ExitNoInput {
		t.Fatalf("exit %d", code)
	}
	evs := readDebug(t, path)
	if last := evs[len(evs)-1]; last["event"] != "exit" || last["code"] != float64(ExitNoInput) {
		t.Errorf("last event %v", last)
	}
}

// A log that cannot be created is a failure to create output (DR-0003: 73),
// and nothing is fetched or printed.
func TestDebugFileThatCannotBeCreatedExits73(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer srv.Close()
	code, out, errs := run("-dump", "-debug", filepath.Join(t.TempDir(), "no", "such", "dir", "log.jsonl"), srv.URL+"/p.md")
	if code != ExitCantCreate || ExitCantCreate != 73 {
		t.Errorf("exit %d, want 73", code)
	}
	if out != "" || hit {
		t.Errorf("work was done before the log could be created: out %q hit %v", out, hit)
	}
	if !strings.HasPrefix(errs, "cmarkbrowser: ") || strings.Count(errs, "\n") != 1 {
		t.Errorf("stderr %q: want one line prefixed with the program name", errs)
	}
}

func TestDebugFlagNeedsAFileName(t *testing.T) {
	code, _, _ := run("-dump", "-debug")
	if code != ExitUsage {
		t.Errorf("exit %d, want 2", code)
	}
}

func TestHelpDocumentsDebugAndExit73(t *testing.T) {
	_, out, _ := run("-h")
	for _, want := range []string{"-debug FILE", "73"} {
		if !strings.Contains(out, want) {
			t.Errorf("help lacks %q", want)
		}
	}
}

func TestDumpOfAnHTMLPageUsesReaderViewAndSaysSo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>T</title></head><body><nav>menu</nav><article><h1>Headline</h1><p>` +
			strings.Repeat("Reader body text. ", 20) + `</p></article></body></html>`))
	}))
	defer srv.Close()
	code, out, errs := run("-dump", "-style", "notty", srv.URL+"/post")
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if !strings.Contains(out, "Reader body text") || strings.Contains(out, "menu") {
		t.Errorf("stdout:\n%s", out)
	}
	if !strings.Contains(errs, "cmarkbrowser: note:") || !strings.Contains(errs, "reader view") {
		t.Errorf("stderr %q: want a note that this is a reader view", errs)
	}
	if strings.Contains(out, "reader view") {
		t.Errorf("the note is on stdout: %q", out)
	}
}
