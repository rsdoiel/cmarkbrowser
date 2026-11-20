
# TODO

Ideas, not quite a roadmap. The design brief is
`agents/projects/cmarkbrowser/design/cmarkbrowser-design.md` in the Laboratory
workspace.

## Scaffold (2026-11-19)

- [x] go.mod (Go 1.27.3), codemeta.json (CodeMeta 3.0), version.go, LICENSE, .gitignore, Makefile (build, test, generate, install, clean).
- [x] `cmd/cmarkbrowser` over `internal/cli`: `-h`, `-l`, `-v`, usage errors exit 2, tests.
- [ ] No Charm dependency yet: add bubbletea v2, bubbles v2 and lipgloss v2 plus glamour (versions as in rptop and knowledge for the Charm ones) with the first code that imports them, so `go mod tidy` keeps them.

Not applicable yet, because this is a concept experiment with nothing to release: release.bash and release.ps1, publish scripts, website.mak, page.tmpl, install notes, man pages. Take them from knowledge when a first release is planned.

## Spike (2026-11-19)

- [x] `internal/page`: fetch with `Accept: text/markdown, text/x-opml`, BonBon's page rule, front matter, size cap; errors wrap sentinels for exits 65, 66, 69.
- [x] `internal/render`: `Renderer` interface, glamour implementation, and `Prepare` (goldmark: strips HTML including script and style content, leaves code alone, numbers links ` [N]`, collects them).
- [x] `internal/browse`: fetch, prepare, render, resolve links against the page address, re-render on resize.
- [x] `internal/ui`: bubbletea viewport, follow a link by number, back and forward, status line, resize. Tested with synthetic messages; smoke-tested in a pty against https://dave.writersweb.org/ (open, follow link 1, back, quit; exit 0).
- [x] `-dump`, `-style`, `-width`.
- Findings: glamour keeps the text inside `<script>` (hence `Prepare`); it prints each link's URL after the link text, which duplicates our `[N]` and wraps badly for long URLs; it cannot report where a link landed, so links are followed by number, not focus.
- [ ] Try a custom glamour style that hides the URL after link text.
- [ ] Autolinks (`<https://...>`) and links whose text is only a code span get no `[N]` marker and, for autolinks, are not collected.
- [ ] Remove the `internal/render` limits: `Prepare` drops later edits that start inside an earlier removed range; check with a fuzz test.
- [x] Link focus, option A (2026-11-19): Tab and Shift-Tab move between links, the focused link's marker is shown in reverse video, the window scrolls to keep it in view, the address shows in the status line, Enter follows it; a typed number wins over focus; Tab starts from the first link in the window; it stops at the first and last link. Positions come from a zero-width sentinel (U+2060) before each `[N]` marker (`render.Sentinel`), found by `browse.View.LineOf`; focus state is in `internal/ui`, apart from the finder.
- [ ] Learn from using A, then decide keys with the author: arrows for focus as in lynx or for scrolling; a links list; `g` to type an address. Known limits: only the marker is highlighted, not the whole link text; a link with no found marker cannot be focused.
- [ ] Decide when to leave glamour (focusable links, layout glamour cannot do, tables) and move to an own renderer over a span-based intermediate form; see the design brief.

## Found against the live sites (2026-11-19)

https://rsdoiel.github.io/blog/index.md and https://rsdoiel.github.io/antenna/index.md already open, as do their post and collection `.md` files (GitHub Pages sends `text/markdown`). `/.well-known/toc.opml` there is 404.

- [x] cmarkbrowser tries the same address first (so a server that negotiates Markdown on `Accept`, as some CDNs do for agents, is respected), then the Markdown copy: `a.html` to `a.md`, a directory to its `index.md` (`page.Fetch`, 2026-11-19; checked on the live blog and antenna sites). Anything else (`.xml`, `.opml`) is not a page.
- [ ] Every link in those `.md` files points at `.html` (154 of 154 on the blog index; `.html`, `.xml` and `.opml` on the antenna front page), so following one is an error. Two fixes: antennaApp writes `.md` links into its Markdown copies (the real fix), and cmarkbrowser tries the `.md` sibling of a same-site `.html` link first, as BonBon's `urlMirrorOf` does.
- [ ] Following a link to a non-Markdown page: HTML opens in the system browser or says so, XML and OPML show as source text (BonBon's rules), images are noted. Today it is only an error line.
- [ ] glamour renders a table link as `text [N][M]` and adds a reference list under the table with its own numbers, so a page with prose links before a table shows two numbers for each table link (ours is the one that works). Seen on the antenna front page. A reason to move to our own renderer.
- [ ] Parse in `render.Prepare` with the extensions glamour uses (GFM, DefinitionList) so the two never disagree about what a link is; no failing case found yet.

## Found on Dave's site (2026-11-19)

- [ ] Pages there start with `# {{page.title}}` and `By {{page.author}} on {{page.pubDate}}.` (BonBon draws none of them itself), so cmarkbrowser shows literal braces on a story page such as https://dave.writersweb.org/scripting/2026/11/03/172109.md until it expands `{{page.NAME}}` from the front matter. Macros are not expanded inside code spans or code blocks (author, 2026-11-19: sample code must be able to show them), expansion is one pass with no re-expansion of a value, and nested keys are checked against a running BonBon first.
- [ ] Those pages give `urlPrev`, `urlNext` and (not in BonBon's list of eight) `urlParent` in front matter; use the first two for previous and next keys.
- [ ] The Markdown site mirrors an HTML site: `dave.writersweb.org/scripting/.well-known/bonbon.json` has `urlMirrorOf: http://scripting.com/`, `urlSearch` and `urlUpdateSocket`; `/.well-known/toc.opml` at the root has an `include` outline for it. scripting.com itself serves HTML only (no `index.md`, no `.well-known` files).

## Images (2026-11-19)

- [ ] A terminal cannot float or show an image: for a body picture show its alt text as a line (`[image: alt]`, with the address available to follow); for `rightMarginImage` show the alt text from `rightMarginImageAlt` (antennaApp's extension, DR-0009) when there is some and nothing when there is not (the image is treated as decorative). A terminal graphics protocol is later.

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
