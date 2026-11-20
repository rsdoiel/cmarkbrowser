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

{app_name} [OPTIONS] [URL]

{app_name} -dump [-style NAME] [-width N] URL

# DESCRIPTION

{app_name} fetches a page as Markdown, shows it in the terminal and follows
its links. It is modelled on Dave Winer's BonBon: HTML in a page is stripped,
a page cannot carry its own styling, and a site's table of contents is read
from /.well-known/toc.opml.

In the browser, Tab and Shift-Tab move between links (the focused link's
address is shown at the bottom) and Enter follows it. Or type a link's number
and press Enter. b and f go back and forward; Esc clears a typed number or the
focus; arrows, PgUp, PgDn, space, g and G scroll; q quits.

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

-v, -version
: Show the version.

# EXIT STATUS

0
: Success.

2
: Bad command line.

65
: The page is not Markdown, or its front matter is not valid.

66
: The page is not there (404 or 410).

69
: The server cannot be reached.

70
: A bug.
`
