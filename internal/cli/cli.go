package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"

	cmarkbrowser "github.com/rsdoiel/cmarkbrowser"
	"github.com/rsdoiel/cmarkbrowser/internal/browse"
	"github.com/rsdoiel/cmarkbrowser/internal/render"
	"github.com/rsdoiel/cmarkbrowser/internal/ui"
)

const appName = "cmarkbrowser"

/** Run parses the command line and runs the program. It never calls os.Exit.
 *
 * Parameters:
 *   args ([]string) — the command line without the program name
 *   stdout (io.Writer) — where results and help go
 *   stderr (io.Writer) — where errors and warnings go, one line each,
 *     prefixed with the program name
 *
 * Returns:
 *   int — the exit code (0, 2 or 70 so far)
 *
 * Example:
 *   code := cli.Run([]string{"-version"}, os.Stdout, os.Stderr) // 0
 */
func Run(args []string, stdout, stderr io.Writer) int {
	var showHelp, showLicense, showVersion, dump bool
	var style string
	var width int
	fs := flag.NewFlagSet(appName, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&showHelp, "h", false, "show help")
	fs.BoolVar(&showHelp, "help", false, "show help")
	fs.BoolVar(&showLicense, "l", false, "show the license")
	fs.BoolVar(&showLicense, "license", false, "show the license")
	fs.BoolVar(&showVersion, "v", false, "show the version")
	fs.BoolVar(&showVersion, "version", false, "show the version")
	fs.BoolVar(&dump, "dump", false, "print the rendered page and its links")
	fs.StringVar(&style, "style", "dark", "glamour style: dark, light, notty, ascii")
	fs.IntVar(&width, "width", 80, "columns to wrap to")
	if err := fs.Parse(args); err != nil {
		return fail(stderr, fmt.Errorf("%w: %s", ErrUsage, err))
	}
	switch {
	case showHelp:
		fmt.Fprint(stdout, cmarkbrowser.FmtHelp(HelpText, appName, cmarkbrowser.Version, cmarkbrowser.ReleaseDate, cmarkbrowser.ReleaseHash))
		return ExitOK
	case showLicense:
		fmt.Fprint(stdout, cmarkbrowser.LicenseText)
		return ExitOK
	case showVersion:
		fmt.Fprintf(stdout, "%s %s %s\n", appName, cmarkbrowser.Version, cmarkbrowser.ReleaseHash)
		return ExitOK
	}
	if fs.NArg() > 1 {
		return fail(stderr, fmt.Errorf("%w: at most one URL, got %d arguments", ErrUsage, fs.NArg()))
	}
	if dump {
		if fs.NArg() != 1 {
			return fail(stderr, fmt.Errorf("%w: -dump needs one URL", ErrUsage))
		}
		return runDump(fs.Arg(0), render.Glamour{Style: style}, width, stdout, stderr)
	}
	if fs.NArg() != 1 {
		return fail(stderr, fmt.Errorf("%w: a URL is needed (try -help)", ErrUsage))
	}
	if !isTerminal(stdout) {
		return fail(stderr, fmt.Errorf("%w: the browser needs a terminal; use -dump to print a page", ErrUsage))
	}
	if _, err := tea.NewProgram(ui.New(fs.Arg(0), render.Glamour{Style: style})).Run(); err != nil {
		return fail(stderr, err)
	}
	return ExitOK
}

// fail prints err as one line prefixed with the program name and returns its exit code.
func fail(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "%s: %s\n", appName, err)
	return ExitCode(err)
}

// runDump prints a rendered page, then its numbered links.
func runDump(addr string, r render.Renderer, width int, stdout, stderr io.Writer) int {
	v, err := browse.Open(context.Background(), addr, r, width)
	if err != nil {
		return fail(stderr, err)
	}
	fmt.Fprint(stdout, v.Text)
	if len(v.Links) > 0 {
		fmt.Fprintln(stdout, "\nLinks:")
		for _, l := range v.Links {
			fmt.Fprintf(stdout, "[%d] %s\n", l.N, l.URL)
		}
	}
	return ExitOK
}

// isTerminal reports whether w is a terminal.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
