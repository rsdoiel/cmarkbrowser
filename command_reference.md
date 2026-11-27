# cmarkbrowser command reference

This page is written by `cmarkbrowser -help`.


## NAME

cmarkbrowser — a terminal browser for the Markdown web

## SYNOPSIS

cmarkbrowser [OPTIONS] [URL|PATH]

cmarkbrowser [-history FILE | -no-history]

cmarkbrowser -dump [-style NAME] [-width N] [-debug FILE] URL|PATH

cmarkbrowser -find QUERY [-corpus FILE]

## DESCRIPTION

cmarkbrowser fetches a page as Markdown, shows it in the terminal and follows
its links. It is modelled on Dave Winer's BonBon: HTML in a page is stripped,
a page cannot carry its own styling, and a site's table of contents is read
from /.well-known/toc.opml.

In the browser, Tab and Shift-Tab move between links (the focused link's
address is shown at the bottom) and Enter follows it. Or type a link's number
and press Enter. b and f go back and forward; ] and [ open the next and
previous page that the page names in its urlNext and urlPrev front matter, or,
when it names none, that the site's table of contents gives (the nearest
.well-known/toc.opml above the page, with the outlines it includes; read the
first time one of these keys needs it, and remembered). As in BonBon the
outline is one list: ] is the line below the page, [ the line above, the ends
wrap, and a site the outline includes is a list of its own; Esc
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

S searches a corpus of pages that cmarkfind keeps in one SQLite file: type a
query at the bottom of the screen and press Enter (Esc cancels). The results
are a page like any other, with numbered links, the matching text and a count
(including how many pages the sites list that are not fetched yet); b comes
back to it and runs the query again. The query is words, "phrases", AND, OR,
NOT and word*; anything else is searched as plain words. F fills the corpus from
the site being read: one batch of 25 pages of the table of contents that lists
the page (a local site is filled whole); press F again for the next batch.
Nothing is added to the corpus unless F is pressed. -corpus FILE names the
file (default: the corpus key of the configuration file, then CMARKFIND_DB,
then cmarkfind's own default) and -no-corpus turns S and F off.

o asks for an address to open, at the bottom of the screen: type it and press
Enter (Esc cancels). An address starts with http:// or https://; a bare host
name such as example.org/a.md gets https://. A file name is refused: typed
addresses are for the web, and local files are opened only from the command
line and through B and H. The page joins the session history, so b comes back.

Every page shown is recorded in a history file, history.md in the working
directory (or the file named by -history FILE), newest first: a list with a
link and the date and time, and an address visited again moves to the top. The
list keeps the 500 latest. H opens it as a page, and with no URL on the command
line cmarkbrowser starts from it, so you can click to where you left off. The file
is rewritten by cmarkbrowser; keep notes elsewhere. -no-history records nothing;
-dump never records. The session history (b and f) is kept in memory either way.

Given a path instead of an address, cmarkbrowser browses a site on your own file
system, with no web server and no protocol: cmarkbrowser ./ browses the current
folder, starting at its index.md; a Markdown file starts at that file. The
site root is the folder you gave (for a file, the nearest folder above it with
a .well-known/toc.opml, else its own folder). A link in a page is a path:
b.md is next to the page, ../c.md is above it, and /d.md is relative to the site
root, as on the web; a link to x.html opens x.md when there is one. Links to
http:// and https:// addresses go to the web as usual. A link that would leave
the site root, by .. or through a symlink, is refused, and so is any file:
address, in any page: nothing a page says can read a file outside the folder
you named. ] and [ use the site's .well-known/toc.opml read from the folder
(its addresses match by path, so a site not yet published works). The point is
to preview a site and edit it: local pages are never recorded in the history
file and cannot be bookmarked. A directory with no index.md is exit 66; a
file: address on the command line is a usage error.

e edits the page being shown in your text editor ($VISUAL, else $EDITOR; a
value such as "code -w" works; neither set is a message, not a guess). The
browser steps aside while the editor runs and shows the page again, at the same
place, when it exits. A local file (a page of a local site, the bookmarks file,
the history file) is edited in place. A remote page cannot be saved back, so
its source, front matter included, is saved as a new file in your Downloads
folder (never replacing one, a.md then a-1.md) and that copy is edited; keep it
or delete it. A reader view has no Markdown source and is refused.

Local files are otherwise opened only through B and H, never from a link.

A link to something that is not a page is shown the way BonBon shows it. XML
(RSS, Atom, OPML, any XML) and text (plain text, JSON, YAML, CSV, code) are
shown as source, in a block, up to 1 MB, with control characters removed so
nothing in a file can steer your terminal. An image or any other file (a PDF,
an archive) is noted with its type, size and address and is not downloaded; a
terminal cannot show a picture. These are ordinary pages: they go in the
history, can be bookmarked and, if they are source, e edits a copy of them.
In a local site the same rules go by the file's extension.

A link to an HTML page that has no Markdown copy is shown in reader view: its
text, links and pictures that have alt text, without scripts, styles or menus.

## OPTIONS

-h, -help [TOPIC]
: Show this help, or one topic's page. -help index lists the topics. Each is
  also a manual page of section 7, cmarkbrowser-TOPIC(7).

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

-corpus FILE
: The cmarkfind corpus S searches and F fills. Cannot be combined with -no-corpus.

-no-corpus
: Turn corpus search and fill off.

-find QUERY
: Search the corpus and print the results page's Markdown on standard output,
  with no URL and no browser. Exit 0 with matches, 1 with none, 66 when there is
  no corpus, 65 when the file is not a corpus or has a newer schema.

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

## CONFIGURATION

If ~/.config/cmarkbrowser/cmarkbrowser.yaml exists ($XDG_CONFIG_HOME is used when
set) it is read before anything is fetched. A missing file is fine; a file that
is there and wrong is an error (exit 78). Settings so far:

user-agent
: The User-Agent sent with every request, so a site can tell who is asking.
  The default is cmarkbrowser/VERSION. Example:

      user-agent: cmarkbrowser/0.0.2 (+https://example.org/about)

corpus
: The cmarkfind corpus file for S, F and -find. Example:

      corpus: ~/.local/share/cmarkfind/corpus.db

## EXIT STATUS

0
: Success.

1
: -find matched nothing.

2
: Bad command line.

65
: The page is not Markdown (and not HTML), its front matter is not valid, or
  the file named is not text; or the corpus is not a corpus or is newer than
  this program reads.

66
: The page is not there (404 or 410), the path named does not exist, the
  directory named has no index.md, or -find has no corpus to read.

69
: The server cannot be reached.

70
: A bug.

73
: The -debug file cannot be created.

75
: The corpus is locked by another program; try again.

77
: The operating system refused access to the path named.

78
: The configuration file is there and wrong.

## SEE ALSO

cmarkbrowser-reading(7), cmarkbrowser-neighbours(7), cmarkbrowser-search(7),
cmarkbrowser-corpus(7), cmarkbrowser-bookmarks(7), cmarkbrowser-local(7),
cmarkbrowser-editing(7), cmarkbrowser-files(7), cmarkbrowser-debug(7),
cmarkbrowser-config(7), cmarkbrowser-exit-status(7)

cmarkfind(1), BonBon (Dave Winer)
