package cli

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
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
