# cmarkbrowser

A terminal browser for the Markdown web, after Dave Winer's
[BonBon](https://github.com/scripting/bonBon/). It fetches a page as Markdown,
strips the HTML in it, follows its links and its site's table of contents, and
lets you choose how the text looks. It also browses a Markdown site on your own
file system, with no web server, to preview a site and edit it.

BonBon's idea is that a web page can be a Markdown file. HTML in a page is
stripped, a page cannot bring its own styling or scripts, and the reader
chooses how text looks. A site's table of contents is an OPML file at
`/.well-known/toc.opml`, and a page is asked for with
`Accept: text/markdown, text/x-opml`. cmarkbrowser is a second, terminal
sized reader for the same pages, and a way to test what
[antennaApp](https://github.com/rsdoiel/antennaApp) publishes for them.

## Status

v0.0.4 is a work in progress (WIP), and it works: you can read the Markdown web
with it today.

- Open a page: `cmarkbrowser https://rsdoiel.github.io/blog/index.md`.
- Follow links with Tab and Enter, or by number; back, forward, search, bookmarks,
  history, the next and previous page from the site's table of contents.
- Browse a folder as a site: `cmarkbrowser ./`.
- Edit the page you are reading in your own editor with `e`.
- Search the sites you have read with `S`, and add the site you are reading to the corpus with `F`; the corpus is [cmarkfind](https://github.com/rsdoiel/cmarkfind)'s (`cmarkbrowser -find QUERY` for scripts).

The [User Manual](user_manual.md) says how, the
[Command Reference](command_reference.md) lists every key and option, and
[BonBon](bonbon.md) says how cmarkbrowser and BonBon differ.

## This documentation is a Markdown site too

These pages are published twice from the same source: as an HTML website, and
as Markdown for BonBon browsers and cmarkbrowser. Every page has a `.md`
address beside its `.html` address, the site's home page is `index.md`, and the
table of contents is `/.well-known/toc.opml`. Read it in a terminal:

~~~shell
cmarkbrowser https://rsdoiel.github.io/cmarkbrowser/
~~~

### Authors

- Doiel, R. S.

## Software Requirements

- Go >= 1.27.3

## Related resources

- [INSTALL](INSTALL.md)
- [BonBon charter](https://github.com/scripting/bonBon/blob/main/docs/charter.md)
- [Textcasting](https://textcasting.org/)
- [Getting Help, Reporting bugs](https://github.com/rsdoiel/cmarkbrowser/issues)
- [LICENSE](https://www.gnu.org/licenses/agpl-3.0.en.html)
