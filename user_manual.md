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
and press Enter. `b` goes back and `f` goes forward. Esc undoes one thing: a typed
number, then the focus, then a search.

A page is Markdown with optional front matter. HTML inside a page is removed, as
a BonBon browser does, and a page cannot change how it looks.

## Opening an address

Press `o`, type an address and press Enter. An address starts with `http://` or
`https://`; a bare host such as `example.org/a.md` gets `https://`. A file name
is not accepted here (see Local sites). `Esc` cancels.

## The next and previous page

`]` opens the next page and `[` the previous one. A page can name them in its own
front matter as `urlNext` and `urlPrev`. When it names none, cmarkbrowser asks the
site's table of contents, `/.well-known/toc.opml`: it reads the site root's file
first, then the folders between the root and the page, and follows the outlines
it includes from the same site. A site with no table of contents says so in the
status line.

## Searching

Press `/`, type a regular expression and press Enter. Every match is marked and
the page scrolls to the first one. `n` and `N` go to the next and previous match.
A pattern with no capital letters ignores case. `/` then Enter repeats the last
search.

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

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success. |
| 2 | Bad command line. |
| 65 | The page is not text, or its front matter is not valid. |
| 66 | The page, path or index.md is not there. |
| 69 | The server cannot be reached. |
| 77 | The operating system refused access to the path. |

## When something looks wrong

`-debug FILE` appends a log of the session to `FILE`, one JSON object per line:
every key, every address tried with its status and time, and the exit code. It
holds addresses and counts, never page content. Send it with a bug report to
the [issue tracker](https://github.com/rsdoiel/cmarkbrowser/issues).
