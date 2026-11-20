# cmarkbrowser and BonBon

[BonBon](https://github.com/scripting/bonBon/) is Dave Winer's browser for the
Markdown web: a web page can be a Markdown file, and the reader decides how it
looks. cmarkbrowser follows the same rules in a terminal. This page says what is
the same, what is different, and how this site is published for both kinds of
reader.

## What is the same

- A page is asked for with `Accept: text/markdown, text/x-opml`. It is a page when
  the server says `text/markdown`, or `text/plain` or a generic type and the
  address ends in `.md` or `.markdown`.
- Front matter is read and removed. A `{{page.title}}` macro is filled in from it.
- HTML in a page is stripped. A page cannot bring styling or scripts.
- The site's table of contents is `/.well-known/toc.opml`. A page with no
  `urlPrev` or `urlNext` takes its neighbours from it. `include` outlines join
  other outlines of the same site.
- `Content-Location` names where a page's content really is.
- `bonbon.json`: `urlMirrorOf` names the HTML site a Markdown site mirrors, and a
  link into that HTML site opens the Markdown copy when it exists.
- XML and text open as source, images and other files as a note.

## What is different

- **Terminal, not a window.** Links are numbered and focused with Tab. A
  picture is a note with its address, not an image.
- **HTML links stay in the terminal.** BonBon sends a clicked HTML link to your
  system browser. cmarkbrowser shows it in reader view.
- **The next and previous page by calendar.** antennaApp writes its posts oldest
  first, as BonBon reads them. cmarkbrowser also reads Dave Winer's newest-first
  files by looking at the numbers in their calendar outlines.
- **Local sites.** `cmarkbrowser ./` browses a folder with no server, within a
  site root. The [User Manual](user_manual.md) has the rules.
- **Editing.** `e` opens the page in your editor.
- **No search server or live reload.** `urlSearch` and `urlUpdateSocket` in
  `bonbon.json` need a server program, and cmarkbrowser uses neither yet.

## How this documentation is published for both

The pages here are written once, in Markdown. The build makes:

- a `.html` page beside each `.md` page, for an ordinary browser. Links between
  pages are written as `.md` links in the source, because the Markdown is the
  truth, and become `.html` links in the HTML;
- the Markdown pages themselves, unchanged, at their `.md` addresses. A host that
  serves `.md` as `text/markdown` or `text/plain` works for BonBon;
- `index.md`, a copy of the README, so the site's address opens a page;
- `/.well-known/toc.opml`, the table of contents, listing the pages in order;
- a `<link rel="alternate" type="text/markdown">` in each HTML page's head, so a
  reader that finds the HTML page can find its Markdown;
- an empty `.nojekyll` file, so GitHub Pages serves the `.well-known` folder.

Check it before you publish. `cmarkbrowser ./` in the built folder reads the
site as a BonBon browser would, and `]` and `[` walk the table of contents.

The table of contents is at the folder of this site. BonBon asks only the
address root of a host, so a site published in a sub-folder (as this one is, at
`/cmarkbrowser/`) is found by BonBon when the host's own
`/.well-known/toc.opml` has an `include` outline pointing at it. cmarkbrowser
looks in the folders above a page as well.
