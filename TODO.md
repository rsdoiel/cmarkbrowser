
# TODO

Ideas, not quite a roadmap. The design brief is
`agents/projects/cmarkbrowser/design/cmarkbrowser-design.md` in the Laboratory
workspace.

## Scaffold (2026-11-19)

- [x] go.mod (Go 1.27.3), codemeta.json (CodeMeta 3.0), version.go, LICENSE, .gitignore, Makefile (build, test, generate, install, clean).
- [x] `cmd/cmarkbrowser` over `internal/cli`: `-h`, `-l`, `-v`, usage errors exit 2, tests.
- [ ] No Charm dependency yet: add bubbletea v2, bubbles v2 and lipgloss v2 plus glamour (versions as in rptop and knowledge for the Charm ones) with the first code that imports them, so `go mod tidy` keeps them.

Not applicable yet, because this is a concept experiment with nothing to release: release.bash and release.ps1, publish scripts, website.mak, page.tmpl, install notes, man pages. Take them from knowledge when a first release is planned.

## Browsing

- [ ] Fetch a page with `Accept: text/markdown, text/x-opml`. A page is `text/markdown`, or `text/plain` or a generic binary type with a `.md` or `.markdown` path. HTML and XML show as source; images are not shown at first.
- [ ] Parse front matter (`title`, `description`, `author`, `pubDate`, `link`, `urlPrev`, `urlNext`, `rightMarginImage`) and strip it before display.
- [ ] Render Markdown to the terminal with glamour first (author, 2026-11-19). Switch to goldmark's AST with our own renderer only if glamour proves too constrained. Check early what glamour does with raw HTML, since BonBon strips it, and whether links can be numbered or focused.
- [ ] Scrolling viewport, numbered or focusable links, follow a link, back and forward, history.
- [ ] Read `/.well-known/toc.opml` once per site (the `opml` module) for prev and next; `<outline type="include">` joins outlines.
- [ ] Read `/.well-known/bonbon.json` (`urlSearch`, `urlMirrorOf`, `cname`, `urlUpdateSocket`); search by `?q=`.
- [ ] Open a local directory or file as a site, so antenna's output can be checked without a server.
- [ ] Exit codes per DR-0003: unreachable server 69, bad Markdown or OPML 65, missing page 66.

## Open questions

- [ ] Is a bookmarks file wanted, and where does it live? (No global settings files in the workspace; configuration may have to be a file in the working directory.)
- [ ] Images: show nothing, show alt text, or use a terminal graphics protocol later.
- [ ] Should it be a TUI only, or also print a rendered page to standard output for use in pipes (`cmarkbrowser -dump URL`)?
