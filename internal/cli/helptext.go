package cli

/** HelpText is the cmarkbrowser(1) page shown by `cmarkbrowser -help`. The
 * placeholders {app_name}, {version}, {release_hash} and {release_date} are
 * filled by cmarkbrowser.FmtHelp.
 *
 * Example:
 *   fmt.Print(cmarkbrowser.FmtHelp(cli.HelpText, "cmarkbrowser", cmarkbrowser.Version, cmarkbrowser.ReleaseDate, cmarkbrowser.ReleaseHash))
 */
const HelpText = `%{app_name}(1) user manual | version {version} {release_hash}
% R. S. Doiel
% {release_date}

# NAME

{app_name} — a terminal browser for the Markdown web

# SYNOPSIS

{app_name} [OPTIONS] [URL|FILE]

{app_name} [-history FILE | -no-history]

{app_name} -dump [-style NAME] [-width N] [-debug FILE] URL|FILE

# DESCRIPTION

{app_name} fetches a page as Markdown, shows it in the terminal and follows
its links. It is modelled on Dave Winer's BonBon: HTML in a page is stripped,
a page cannot carry its own styling, and a site's table of contents is read
from /.well-known/toc.opml.

In the browser, Tab and Shift-Tab move between links (the focused link's
address is shown at the bottom) and Enter follows it. Or type a link's number
and press Enter. b and f go back and forward; ] and [ open the next and
previous page that the page names in its urlNext and urlPrev front matter, or,
when it names none, that the site's table of contents gives (the nearest
.well-known/toc.opml above the page, with the outlines it includes; read the
first time one of these keys needs it, and remembered); Esc
clears a typed number, then the focus, then a search; arrows, PgUp, PgDn, space,
g and G scroll; q quits.

/ searches the page. Type a Go regular expression (the RE2 syntax of the
regexp package) and press Enter: the page scrolls to the first match at or
below the top of the window, every match is marked and the current one is
highlighted. n goes to the next match and N to the previous one, wrapping at
either end. A pattern with no upper-case letter ignores case; an upper-case
letter makes it case sensitive; write (?i) or (?-i) to decide yourself. Text
is searched as you read it: the lines of one paragraph count as one run of
words, so a phrase is found even where the screen wrapped it, and a match never
runs from one paragraph into the next. / then Enter on an empty prompt repeats
the last search. A search belongs to the page it was run on.

m adds the page being shown to a bookmarks file and B opens it. The file is
ordinary Markdown (bookmarks.md in the working directory, or the file named by
-bookmarks FILE): a list with a link and the date for each page, created with a
"# Bookmarks" heading when it is missing and never rewritten, only added to. A
page already in it is not added twice. Open it with B, or give it as the
argument, and follow its links like any page; b comes back to it. Edit it by
hand if you like.

o asks for an address to open, at the bottom of the screen: type it and press
Enter (Esc cancels). An address starts with http:// or https://; a bare host
name such as example.org/a.md gets https://. A file name is refused: typed
addresses are for the web, and local files are opened only from the command
line and through B and H. The page joins the session history, so b comes back.

Every page shown is recorded in a history file, history.md in the working
directory (or the file named by -history FILE), newest first: a list with a
link and the date and time, and an address visited again moves to the top. The
list keeps the 500 latest. H opens it as a page, and with no URL on the command
line {app_name} starts from it, so you can click to where you left off. The file
is rewritten by {app_name}; keep notes elsewhere. -no-history records nothing;
-dump never records. The session history (b and f) is kept in memory either way.

Given a file name instead of an address, {app_name} shows that file. Local
files are opened only that way, and through B: a link in a page, whatever it
says, never opens one.

A link to an HTML page that has no Markdown copy is shown in reader view: its
text, links and pictures that have alt text, without scripts, styles or menus.

# OPTIONS

-h, -help
: Show this help.

-l, -license
: Show the license.

-dump
: Fetch the URL, print the rendered page and its numbered links, and exit.

-style NAME
: glamour style for -dump: dark (default), light, notty, ascii.

-width N
: Columns to wrap to (default 80).

-bookmarks FILE
: The Markdown file that m adds to and B opens (default bookmarks.md).

-history FILE
: Record every page shown in this Markdown file, newest first, and start from
  it when no URL is given. H opens it. The default is history.md in the working
  directory.

-no-history
: Record no history. Cannot be combined with -history.

-debug FILE
: Append a log of the session to FILE, one JSON object per line: every key
  pressed, every address tried (the same address, the page's declared Markdown
  alternate, the .md sibling) with its status, size, time and outcome, which of
  them produced the page, the time each step took, and the exit code. It holds
  addresses, key names, counts and times, never page content. The file is
  created with mode 0600. Use it to see why a page opened as it did, and to
  send a session to someone who can read it.

-v, -version
: Show the version.

# EXIT STATUS

0
: Success.

2
: Bad command line.

65
: The page is not Markdown (and not HTML), its front matter is not valid, or
  the file named is a directory or not text.

66
: The page is not there (404 or 410), or the file named does not exist.

69
: The server cannot be reached.

70
: A bug.

73
: The -debug file cannot be created.
`
