package cli

import (
	"bytes"
	"errors"
	"fmt"
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

func TestBrowsingNotBuiltIsInternal(t *testing.T) {
	code, _, errs := run("https://example.org/")
	if code != ExitInternal || !strings.Contains(errs, "not built yet") {
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
