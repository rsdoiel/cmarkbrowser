package cli

import (
	"context"
	"flag"
	"fmt"
	localpkg "github.com/rsdoiel/markdownweb/local"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"

	cmarkbrowser "github.com/rsdoiel/cmarkbrowser"
	"github.com/rsdoiel/cmarkbrowser/internal/browse"
	"github.com/rsdoiel/cmarkbrowser/internal/config"
	"github.com/rsdoiel/cmarkbrowser/internal/render"
	"github.com/rsdoiel/cmarkbrowser/internal/ui"
	"github.com/rsdoiel/markdownweb/debuglog"
	"github.com/rsdoiel/markdownweb/fetch"
	"github.com/rsdoiel/markdownweb/page"
)

const appName = "cmarkbrowser"

// defaultHistory is the history file when -history names none: in the working
// directory, like bookmarks.md, so there is no settings file anywhere else.
const defaultHistory = "history.md"

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
	var showHelp, showLicense, showVersion, dump, noHistory bool
	var style, debugPath, bookmarksPath, historyPath string
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
	fs.StringVar(&debugPath, "debug", "", "write a JSON-lines log of the session to this file")
	fs.StringVar(&bookmarksPath, "bookmarks", "", "the Markdown file m adds to and B opens (default bookmarks.md)")
	fs.StringVar(&historyPath, "history", "", "the Markdown file that visited pages are recorded in, newest first; H opens it (default history.md)")
	fs.BoolVar(&noHistory, "no-history", false, "record no history")
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
	if dump && fs.NArg() != 1 {
		return fail(stderr, fmt.Errorf("%w: -dump needs one URL", ErrUsage))
	}
	if noHistory && historyPath != "" {
		return fail(stderr, fmt.Errorf("%w: -history and -no-history together", ErrUsage))
	}
	// History is on by default, in history.md in the working directory like
	// bookmarks.md; -no-history turns it off. -dump never records.
	if historyPath == "" && !noHistory {
		historyPath = defaultHistory
	}
	if noHistory || dump {
		historyPath = ""
	}
	// With no URL the browser starts from the history file, when there is one.
	start := fs.Arg(0)
	fromHistory := false
	if !dump && fs.NArg() == 0 {
		if historyPath == "" {
			return fail(stderr, fmt.Errorf("%w: a URL is needed (try -help)", ErrUsage))
		}
		if _, err := os.Stat(historyPath); err != nil {
			return fail(stderr, fmt.Errorf("%w: a URL is needed: there is no history yet in %s (try -help)", ErrUsage, historyPath))
		}
		start, fromHistory = historyPath, true
	}
	if !dump && !isTerminal(stdout) {
		return fail(stderr, fmt.Errorf("%w: the browser needs a terminal; use -dump to print a page", ErrUsage))
	}
	// The configuration is read before anything is fetched: a file that is
	// present and wrong stops the run (exit 78).
	cfg, err := config.Load()
	if err != nil {
		return fail(stderr, err)
	}
	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = defaultUserAgent()
	}
	page.SetFetcher(fetch.New(fetch.Options{UserAgent: userAgent, Timeout: 30 * time.Second, MaxRedirects: 10}))
	// The log is created before anything is fetched, so a failure to create it
	// stops the run instead of silently losing the record the person asked for.
	var log *debuglog.Logger
	if debugPath != "" {
		f, err := os.OpenFile(debugPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
		if err != nil {
			return fail(stderr, fmt.Errorf("%w: %s", ErrCantCreate, err))
		}
		defer f.Close()
		log = debuglog.New(f)
		mode := "tui"
		if dump {
			mode = "dump"
		}
		log.Event("start", "schema", 1, "version", cmarkbrowser.Version, "hash", cmarkbrowser.ReleaseHash,
			"mode", mode, "user_agent", userAgent, "args", args, "goos", runtime.GOOS, "goarch", runtime.GOARCH, "pid", os.Getpid())
	}
	// A file: address is not a path, and there is no protocol for a local site
	// (DR-0001): the argument is a path, or an http:// or https:// address.
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(start)), "file:") {
		return fail(stderr, fmt.Errorf("%w: %s: give a path (./ for this folder), not a file: address", ErrUsage, start))
	}
	// An argument with no scheme is a path the person named; an address needs
	// http:// or https://. A page's own links never open files outside the site
	// root they name (page.Fetch refuses them).
	isPath := !strings.Contains(start, "://")
	var site *localpkg.Site
	if isPath && !fromHistory { // the history file is a page of its own, with no site root
		var err error
		site, start, err = localpkg.Open(start)
		if err != nil {
			code := fail(stderr, err)
			log.Event("exit", "code", code)
			return code
		}
	}
	var code int
	if dump {
		ctx := debuglog.WithContext(context.Background(), log)
		code = runDump(ctx, start, site, render.Glamour{Style: style}, width, stdout, stderr)
	} else {
		model := ui.New(start, render.Glamour{Style: style}).WithDebug(log).WithBookmarks(bookmarksPath).WithHistory(historyPath)
		if isPath {
			model = model.WithLocalStart()
			if site != nil {
				model = model.WithSite(site)
			}
		}
		if _, err := tea.NewProgram(model).Run(); err != nil {
			code = fail(stderr, err)
		}
	}
	log.Event("exit", "code", code)
	return code
}

// defaultUserAgent is the User-Agent when the configuration file names none.
func defaultUserAgent() string {
	return appName + "/" + cmarkbrowser.Version
}

// runDump prints a rendered page, then its numbered links. A local file is
// read only when the person named it on the command line.
func runDump(ctx context.Context, arg string, site *localpkg.Site, r render.Renderer, width int, stdout, stderr io.Writer) int {
	var v *browse.View
	var err error
	if site != nil {
		v, err = browse.OpenLocal(ctx, arg, site, r, width)
	} else {
		v, err = browse.Open(ctx, arg, r, width)
	}
	if err != nil {
		return fail(stderr, err)
	}
	if v.Page.Reader {
		fmt.Fprintf(stderr, "%s: note: %s: an HTML page, shown in reader view\n", appName, v.Page.URL)
	}
	for _, key := range v.Missing {
		fmt.Fprintf(stderr, "cmarkbrowser: warning: %s: no front matter value for {{page.%s}}\n", v.Page.URL, key)
	}
	for _, note := range v.Notes {
		fmt.Fprintf(stderr, "cmarkbrowser: warning: %s: %s\n", v.Page.URL, note)
	}
	fmt.Fprint(stdout, v.Display(-1))
	if len(v.Links) > 0 {
		fmt.Fprintln(stdout, "\nLinks:")
		for _, l := range v.Links {
			dest := l.URL
			if l.Path != "" {
				dest = l.Path // a local page's path link is shown as the path (DR-0001)
			}
			fmt.Fprintf(stdout, "[%d] %s\n", l.N, dest)
		}
	}
	return ExitOK
}

// isTerminal reports whether w is a terminal.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// fail prints err as one line prefixed with the program name and returns its exit code.
func fail(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "%s: %s\n", appName, err)
	return ExitCode(err)
}
