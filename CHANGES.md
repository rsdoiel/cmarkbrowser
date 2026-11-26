# Changes

v0.0.0 is the prototype, a proof of concept: it shows that a terminal browser
can read the Markdown web and preview a Markdown site. From v0.0.1, if the work
continues, the development status becomes WIP.

## Unreleased

- markdownweb is v0.0.3: requests go through its polite fetcher. The User-Agent was Go's `Go-http-client/1.1`; it is now `cmarkbrowser/VERSION`, or the `user-agent` setting in `~/.config/cmarkbrowser/cmarkbrowser.yaml`. A missing file is fine; one that is there and wrong exits 78. The file is read before anything is fetched. New package `internal/config`; `gopkg.in/yaml.v3` is now a direct dependency.

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
