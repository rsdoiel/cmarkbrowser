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

# DESCRIPTION

{app_name} fetches a page as Markdown, shows it in the terminal and follows
its links. It is modelled on Dave Winer's BonBon: HTML in a page is stripped,
a page cannot carry its own styling, and a site's table of contents is read
from /.well-known/toc.opml.

This is a scaffold: only the options below work. Browsing is not built yet.

# OPTIONS

-h, -help
: Show this help.

-l, -license
: Show the license.

-v, -version
: Show the version.

# EXIT STATUS

0
: Success.

2
: Bad command line.

70
: A bug, or a request for something not built yet.
`
