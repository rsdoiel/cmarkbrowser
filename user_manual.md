# cmarkbrowser User Manual

cmarkbrowser reads the Markdown web in a terminal. This page is how to use it;
the [Command Reference](command_reference.md) lists every key and option. Run it
with an address or a path:

~~~shell
cmarkbrowser https://rsdoiel.github.io/blog/index.md
cmarkbrowser ./
~~~

## Reading

Arrow keys, PgUp, PgDn, space, `g` and `G` scroll. `q` quits.

Links are numbered, like `[3]`. Press Tab to move to the next link, Shift-Tab to
the previous one, and Enter to follow the focused link. Or type a link's number
and press Enter. `b` goes back and `f` goes forward, each to the place on the page where you left it. Esc undoes one thing: a typed
number, then the focus, then a search.

A page is Markdown with optional front matter. HTML inside a page is removed, as
a BonBon browser does, and a page cannot change how it looks.

Front matter is a YAML block at the very top of the file: a first line of `---`,
then the block, closed by a line of `---` or `...`. If the block is never closed
the whole file is shown as it is and the status line says the front matter is not
closed (`-dump` says so on standard error). A block that is not valid YAML, or is
not a list of keys and values, makes the page an error (exit 65).

In the text of a page, `{{page.NAME}}` is replaced by the front matter value NAME
(spaces inside the braces are allowed, and `{{page.source.title}}` reaches into a
table), except in code. A name the page does not have stays as written and is
reported. When the page has a `link` value, the expanded `title` is a link to it.
Includes (`@include-text-block`) are not expanded: they are a step the server or
the build does before it publishes a page, so a page that still holds one shows
it as text.

## Opening an address

Press `o`, type an address and press Enter. An address starts with `http://` or
`https://`; a bare host such as `example.org/a.md` gets `https://`. A file name
is not accepted here (see Local sites). `Esc` cancels.

## The next and previous page

`]` opens the next page and `[` the previous one. A page can name them in its own
front matter as `urlNext` and `urlPrev`. When it names none, cmarkbrowser asks the
site's table of contents, `/.well-known/toc.opml`: it reads the site root's file
first, then the folders between the root and the page, and follows the outlines
it includes from the same site. As in BonBon, the outline is read as one list,
top to bottom, a page with outlines under it before them: `]` is the line below
the page and `[` the line above. Groups do not stop the arrows and the ends
wrap, from the last line to the first. A site the outline includes is a list of
its own, so its last page wraps to its first. A neighbour with nothing at its address
(a non-2xx answer, or none) is passed over, as BonBon does: each address is asked
about once, the answer is remembered, and at most 12 candidates are tried per key
press. A site with no table of contents says so in the status line.

## Searching

Press `/`, type a regular expression and press Enter. Every match is marked and
the page scrolls to the first one. `n` and `N` go to the next and previous match.
A pattern with no capital letters ignores case. `/` then Enter repeats the last
search.

## Searching the sites you have read

cmarkbrowser can search a corpus of pages that [cmarkfind](https://github.com/rsdoiel/cmarkfind)
keeps in one SQLite file. Nothing is sent anywhere by a search.

- `S` asks for a query at the bottom of the screen; Enter runs it, `Esc` cancels.
  The query is words, "phrases", `AND`, `OR`, `NOT` and `word*`; anything else is
  searched as plain words. The results are a page like any other: numbered links,
  the matching text, and a count that says how many pages the sites list but have
  not been fetched yet. `b` comes back to the results and runs the query again.
- `F` fills the corpus from the site you are reading: one batch of 25 pages of the
  table of contents that lists the page (a local site is filled whole). Press `F`
  again for the next batch. Nothing is added to the corpus unless you press `F`.
- `cmarkbrowser -find QUERY` prints the results page's Markdown on standard output,
  with no browser, for scripts. It exits 0 with matches, 1 with none, 66 when there
  is no corpus and 65 when the file is not a corpus or is newer than this program
  reads.

The corpus file is named by `-corpus FILE`, else the `corpus` key of the
configuration file, else `CMARKFIND_DB`, else cmarkfind's own default. `-no-corpus`
turns `S` and `F` off. Fill a big site, or keep one up to date, with the
`cmarkfind` program.

## History and bookmarks

Every web page you visit is added to `history.md` in the folder you started in,
newest first, and `H` opens it as a page: click back to where you left off. Start
cmarkbrowser with no address and it opens the history. `-no-history` records
nothing and `-history FILE` uses another file.

Press `m` to add the page you are reading to `bookmarks.md`, and `B` to open it.
Both files are plain Markdown that you can read, edit and keep under version
control.

## Links that are not pages

- A link to an HTML page with a Markdown copy opens the copy.
- An HTML page with none opens in **reader view**: its text, links and picture
  descriptions, without scripts, styles or menus.
- XML (RSS, Atom, OPML) and text (JSON, YAML, CSV, code) open as **source**, in a
  block, up to 1 MB, with control characters removed.
- An image or any other file is a **note** with its type, size and address. It is
  not downloaded, and a terminal cannot show a picture.

## Local sites

Give a path instead of an address to browse a Markdown site on your own file
system, with no web server and no `file://` address:

~~~shell
cmarkbrowser ./          # this folder, starting at index.md
cmarkbrowser notes/a.md  # a file; its site is the folder above with a toc.opml, or its own
~~~

The folder you give is the **site root**. A link in a page is a path: `b.md` is
next to the page, `../c.md` is above it, and `/d.md` is relative to the site
root, as it is on the web. A link to `x.html` opens `x.md` when there is one, and a
link to a folder opens its `index.md`. A link to an `http://` or `https://`
address goes to the web.

Nothing a page says can reach outside the site root: a link that leaves it, by
`..` or through a symlink, is refused, and so is any `file:` address. `]` and `[`
read `.well-known/toc.opml` from the folder, so a site you have not published yet
works. Local pages are not written to the history and cannot be bookmarked.

## Editing

Press `e` to edit the page you are reading in `$VISUAL`, else `$EDITOR`. The
browser steps aside while the editor runs and shows the page again, at the same
place, when you quit it. A local file is edited in place. A page from the web
cannot be saved back, so its source is saved as a new file in your `Downloads`
folder and that copy is edited. Together with local sites this makes a preview
loop: edit, quit the editor, read the result.

## Printing a page

`cmarkbrowser -dump ADDRESS` writes the rendered page and a numbered list of its
links to standard output, and `-style notty` leaves out the colors:

~~~shell
cmarkbrowser -dump -style notty https://rsdoiel.github.io/blog/index.md | less
~~~

## Configuration

If `~/.config/cmarkbrowser/cmarkbrowser.yaml` exists (`$XDG_CONFIG_HOME` is used
when set) cmarkbrowser reads it before it fetches anything. It holds two
settings. `user-agent` is the User-Agent sent with every request,
`cmarkbrowser/VERSION` unless you name another, and `corpus` is the cmarkfind
corpus that `S` and `F` use:

~~~yaml
user-agent: cmarkbrowser/0.0.4 (+https://example.org/about)
corpus: ~/.local/share/cmarkfind/corpus.db
~~~

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success. |
| 2 | Bad command line. |
| 1 | `-find` found nothing. |
| 65 | The page is not text, or its front matter is not valid; or the corpus is not a corpus or is newer than this program reads. |
| 66 | The page, path or index.md is not there, or `-find` has no corpus to read. |
| 69 | The server cannot be reached. |
| 77 | The operating system refused access to the path. |
| 78 | The configuration file is there and wrong. |

## When something looks wrong

`-debug FILE` appends a log of the session to `FILE`, one JSON object per line:
every key, every address tried with its status and time, and the exit code. It
holds addresses and counts, never page content. Send it with a bug report to
the [issue tracker](https://github.com/rsdoiel/cmarkbrowser/issues).
