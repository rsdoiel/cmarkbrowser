# Changes

v0.0.0 is the prototype, a proof of concept: it shows that a terminal browser
can read the Markdown web and preview a Markdown site. From v0.0.1, if the work
continues, the development status becomes WIP.

## Unreleased

- A link to `page#heading` opens the page with that heading at the top. The heading's fragment is its slug by markdownweb's `slug` package (GitHub's rule, repeats numbered `-1`, `-2`). Links to a heading that is not found open at the top, as before.

## v0.0.4, 2026-11-26

- **Corpus search and fill** (cmarkbrowser DR-0003, with cmarkfind): `S` asks for a query and shows the results of a cmarkfind corpus as a page (numbered links, matching text, how many listed pages are not fetched yet; `b` runs the query again). `F` fills the corpus from the site being read, one batch of 25 pages per press (a local site whole); nothing is added unless `F` is pressed. `-corpus FILE`, the `corpus` key and `CMARKFIND_DB` name the file, cmarkfind's default otherwise; `-no-corpus` turns it off. `-find QUERY` prints the results page for scripts (exit 0 matches, 1 none, 66 no corpus, 65 not a corpus). Stored text is escaped so a result can make no link of its own. This adds cmarkfind v0.0.1 (and its pure-Go SQLite driver, no cgo) as a dependency.
- markdownweb is v0.0.5.
- `-help index` lists eleven help topics, and `-help TOPIC` shows one: reading, neighbours, search, corpus, bookmarks, local, editing, files, debug, config and exit-status. Each is a section 7 manual page, `cmarkbrowser-TOPIC.7.md`, written by `make build` and installed with the man pages. The `-help` page has a SEE ALSO that lists them. `-help` with a URL after it (it used to print the help) is now a usage error unless the word is a topic.
- `]` and `[` pass over a neighbour with nothing at its address (a non-2xx answer or none), as BonBon does (markdownweb DR-0004 Q1). Each address is asked about once and the answer is remembered; at most 12 candidates are tried per key press. Web pages only; a local site already skipped missing files.

## v0.0.3, 2026-11-26

- **Behaviour change:** `]` and `[` on a page that names no `urlPrev` and `urlNext` follow BonBon's rule (markdownweb v0.0.4, DR-0004): the outline is one flat list, previous is the line above, next the line below, and the ends wrap. Arrows no longer stop at the end of a group, cross-group steps are allowed, and a list with calendar outlines is no longer reversed (the line below is next, the older day in a newest-first file). A site included in the outline is a list of its own: its first and last pages wrap inside it. A page that names its neighbours in front matter is unchanged. A neighbour with nothing at its address is not skipped yet.
- markdownweb is v0.0.4 (it was v0.0.3): requests go through its polite fetcher. The User-Agent was Go's `Go-http-client/1.1`; it is now `cmarkbrowser/VERSION`, or the `user-agent` setting in `~/.config/cmarkbrowser/cmarkbrowser.yaml`. A missing file is fine; one that is there and wrong exits 78. The file is read before anything is fetched. New package `internal/config`; `gopkg.in/yaml.v3` is now a direct dependency.

- The page, toc, local, reader and debuglog packages now come from github.com/rsdoiel/markdownweb (v0.0.1), shared with cmarkfind and antennaApp. The table of contents is read through github.com/rsdoiel/opml. No change of behaviour.
- Front matter and `{{page.NAME}}` macros now come from github.com/rsdoiel/frontmatter (v0.0.1), the module antennaApp and others share. Changes you may notice:
  - a front matter block can also be closed by a `...` line;
  - a block that is never closed no longer passes silently: the whole file is shown and the status line (and `-dump`, on standard error) says it is not closed;
  - a page's line endings are kept as they were (a CRLF page is no longer rewritten);
  - a bare date in a macro shows as `2026-11-19`, not `2026-11-19 00:00:00 +0000 UTC`;
  - a macro value that begins like a heading or a list (`# `, `- `, `1. `) is shown as text, not as markup;
  - the expanded `title` is a link when the page has a `link` value (http, https, mailto or relative; anything else is ignored and noted).
- goldmark is now v1.8.7. A bare address followed by `)` in text (`(see https://example.org/x).`) now keeps the `)` out of the address, as GitHub does.
- Needs Go 1.27.4 (the frontmatter module's minimum).
- Includes are deliberately not supported: they are a server-side step.

## v0.0.0 — 2026-11-20 (prototype)

### Added

- **Reading the Markdown web.** `cmarkbrowser ADDRESS` fetches a page with BonBon's
  `Accept: text/markdown, text/x-opml`, removes the front matter and any HTML,
  fills in `{{page.NAME}}` macros, and shows it in a terminal viewer with
  numbered links. Tab and Shift-Tab focus a link, Enter or a typed number
  follows it, `b` and `f` go back and forward, `/` searches with `n` and `N`.
  `-dump` prints the rendered page and its links.
- **The next and previous page.** `]` and `[` use a page's `urlNext` and
  `urlPrev`, else the site's `/.well-known/toc.opml`: the site root first, then
  the folders down to the page, with same-site includes merged, and the
  direction of calendar outlines read from their names.
- **Links that are not pages.** An HTML page with no Markdown copy opens in
  reader view. XML and text open as source in a block (control characters
  removed, cut at 1 MB). An image or other file is a note, not downloaded.
  A link into the HTML site named by `urlMirrorOf` in `bonbon.json` opens the
  Markdown copy when it exists.
- **Bookmarks and history.** `m` adds the page to `bookmarks.md` and `B` opens
  it. Every web page visited is written to `history.md`, newest first, 500
  kept; `H` opens it, and with no address the browser starts from it.
  `-no-history` turns it off, `-history FILE` names another file.
- **`o` opens an address** without leaving the browser.
- **`e` edits the page** in `$VISUAL` or `$EDITOR`. A local file is edited in
  place and reloaded afterwards; a remote page is saved as a copy in the
  Downloads folder and the copy is edited.
- **Local sites.** `cmarkbrowser ./` browses a folder with no server and no
  `file://` address. Links are paths inside the site root (`/` is the root);
  anything that leaves it, by `..` or a symlink, and any `file:` address is
  refused. `]` and `[` read the folder's `.well-known/toc.opml`. Local pages are
  not recorded in the history and cannot be bookmarked (DR-0001).
- **Exit codes** follow workspace DR-0003: 2, 65, 66, 69, 70, 73 and 77.
- **Documentation** as a Markdown site for BonBon browsers and an HTML website:
  User Manual, Command Reference, BonBon, INSTALL and About, with
  `/.well-known/toc.opml` and an `index.md`.
- **A Makefile and release scripts** for Linux (x86_64, aarch64), macOS
  (x86_64, arm64) and Windows (x86_64, arm64).

### Known gaps

- There is no `-help index` and no per-topic help pages; the program has one
  help page and one man page.
- No configuration file, and `history.md` and `bookmarks.md` live in the folder
  cmarkbrowser is started from (TODO).
- Images are not shown, and `urlSearch` in `bonbon.json` is not used.
- Windows and macOS builds are cross-compiled but have not been run.
