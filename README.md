
# cmarkbrowser

A terminal browser for the Markdown web, after Dave Winer's
[BonBon](https://github.com/scripting/bonBon/). It is a concept experiment
built on the Charm libraries (bubbletea v2, rendering with glamour); nothing browses yet.

BonBon's idea is that a web page can be a Markdown file. HTML in a page is
stripped, a page cannot bring its own styling or scripts, and the reader
chooses how text looks. A site's table of contents is an OPML file at
`/.well-known/toc.opml`, and a page is asked for with
`Accept: text/markdown, text/x-opml`. cmarkbrowser is a second, terminal
sized reader for the same pages, and a way to test what
[antennaApp](https://github.com/rsdoiel/antennaApp) publishes for them.

## Status

A working spike. `cmarkbrowser URL` opens a page in a terminal viewer; type a
link's number and press Enter to follow it, `b` and `f` go back and forward,
`q` quits. `cmarkbrowser -dump URL` prints the rendered page and its links.
Pages are fetched with BonBon's `Accept` header, front matter and HTML are
stripped, and Markdown is rendered with glamour. No `toc.opml`, search or
local-directory sites yet; see TODO.md.

### Authors

- Doiel, R. S.

## Software Requirements

- Go >= 1.27.3

## Related resources

- [BonBon charter](https://github.com/scripting/bonBon/blob/main/docs/charter.md)
- [Textcasting](https://textcasting.org/)
- [Getting Help, Reporting bugs](https://github.com/rsdoiel/cmarkbrowser/issues)
- [LICENSE](https://www.gnu.org/licenses/agpl-3.0.en.html)
