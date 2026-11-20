
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

- [x] (2026-11-19) `render.Expand` expands `{{page.KEY}}` from the front matter before `Prepare`: text only (not code spans or blocks), one pass, values escaped, dotted keys reach nested tables (`source.title`), lists join with commas; a missing key (or a table) stays visible and is reported (`-dump` warns on stderr, the TUI shows it in the status line). Checked on https://dave.writersweb.org/scripting/2026/11/03/172109.md. Left: `link` turning the expanded title into a link (antennaApp's extension, DR-0009), and BonBon's own handling of nested keys, not checked against a running BonBon.
- [x] (2026-11-19) `n` and `p` open the page named by the current page's `urlNext` and `urlPrev` (resolved against its address, a new history entry; a status line says so when the key is absent; ignored while a link number is being typed; the footer hint lists them only when the page has them). `urlParent` is not used yet. Not yet from `/.well-known/toc.opml` when a page has no keys.
- [ ] The Markdown site mirrors an HTML site: `dave.writersweb.org/scripting/.well-known/bonbon.json` has `urlMirrorOf: http://scripting.com/`, `urlSearch` and `urlUpdateSocket`; `/.well-known/toc.opml` at the root has an `include` outline for it. scripting.com itself serves HTML only (no `index.md`, no `.well-known` files).

## Images (2026-11-19)

- [ ] A terminal cannot float or show an image: for a body picture show its alt text as a line (`[image: alt]`, with the address available to follow); for `rightMarginImage` show the alt text from `rightMarginImageAlt` (antennaApp's extension, DR-0009) when there is some and nothing when there is not (the image is treated as decorative). A terminal graphics protocol is later.

## Declared alternate (2026-11-19)

- [x] When an address answers HTML, `page.Fetch` reads the first 128 KB, finds `<link rel="alternate" type="text/markdown" href>` in the head and fetches that before guessing the `.md` sibling (order: same address, declared alternate, sibling). Same host only; `application/markdown`, a link in the body and an alternate stylesheet do not count; a missing alternate falls back to the sibling. Checked on a copy of the antenna site: `food.html` now opens `food_items.md` (the news), where the sibling `food.md` is the author's feed list.
- [x] (2026-11-19) Slow on a large collection: `pacific_items.md` (4.5 MB, 96,863 rendered lines, 8,151 links) took 20.1 s under `-dump`. The `-debug` log showed 17.3 s in `render.Prepare`, not glamour: it copied the whole page once per link edit (about 38 GB). `applyEdits` now builds the result in one pass (93 ms; `Expand` uses it too); a 3,000-round random test checks it against the old algorithm and a 4.7 MB / 8,000-link test bounds the time. Total is 2.9 s, of which glamour is 2.7 s.

## Debug log (2026-11-19)

- [x] `-debug FILE` appends a JSON-lines log of the session (mode 0600; exit 73 when it cannot be created, checked before anything is fetched). Events: `start` (schema 1, version, mode, args), `key`, `focus`, `scroll`, `resize`, `status`, `load.start` (with why: start, link, history, next, prev), `fetch.try` (role same, alternate or sibling; status, content type, bytes, outcome, ms), `fetch.alternate`, `fetch.sibling`, `fetch.result` (via), `open` (fetch, expand, prepare and render ms, sizes, lines, links, missing macro keys), `rerender`, `load.done`, `quit`, `exit`. `at_ms` is time since the start; `ms` is always a step's duration. No page content is ever logged. Works with `-dump` and the browser; `internal/debuglog`, a nil logger does nothing, the logger travels in the context.
- [ ] Maybe: a `-debug` marker key (say F2) that writes a `note` event so a session can be annotated while it runs; a `replay` tool that prints the log as a readable timeline.

## Search, reader view, bookmarks (2026-11-19; author asked for all three)

- [x] Search: `/` opens a prompt for a Go regexp (RE2), Enter runs it, `n` and `N` go to the next and previous match and wrap, Esc clears it (one thing at a time: number, then focus, then search). Smart case (no upper-case letter ignores case; `(?i)` and `(?-i)` override). Matches are marked and the current one is highlighted; the page scrolls to the first match at or below the top of the window; the footer says `match 2 of 14  /pattern`. The text is searched as read: the lines of one paragraph are one run of words (a phrase wrapped by the screen is found), blank lines are paragraph breaks and `(?m)^` anchors at a paragraph start. A bad pattern keeps the earlier search; no match ends it; a search belongs to its page and survives a resize. `internal/search` (pure: `Compile`, `NewIndex`, `Find`, `NewMarked`, `Highlight`); on the 97,000-line Pacific collection a search takes about 0.9 s and moving to the next match 33 ms. Events `search` and `match` in the debug log. Page keys moved from `n` and `p` to `]` and `[`.
- [x] Reader view: a link that answers HTML with no Markdown copy and no declared alternate is fetched in full (8 MB cap) and converted to Markdown by `internal/reader` (largest `<article>`, else `<main>`, else `<body>`; scripts, styles, menus, forms, hidden elements, unsafe addresses and images without alt text dropped; text escaped so page text cannot become markup; links and images made absolute). The title line says `[HTML, reader view]`; `-dump` notes it on stderr; `{{page.*}}` macros are not applied to it. A page whose text is built by JavaScript has none: that is reported as an error that says so. Checked on the Yap State news item linked from `pacific_items.md` (1 MB of Wix HTML became 1,236 bytes of Markdown, about 1 s).
- [x] Bookmarks (2026-11-19). `m` adds the page being shown to a Markdown file (`bookmarks.md` in the working directory, or `-bookmarks FILE`): `- [Title](address) (date)`, created with a `# Bookmarks` heading when missing, only ever appended to, no duplicates, titles and addresses escaped so they cannot become markup (`internal/bookmarks`). `B` opens the file as a page; its links work like any page's and `b` returns to it. The address saved is the page actually shown (for an antenna collection, the `_items.md` page, not the feed list; for an HTML page, its address). A file name as the start argument opens that local file (`-dump` too). Local files are only ever opened by the person: the start argument and `B`; `page.Fetch` refuses file addresses and the UI refuses a `file:` link or history entry it was not told to open, so a page from the network can never read a local file (tests for both). Exit codes: missing file 66, a directory or binary file 65, `-bookmarks` with no value 2. Events `bookmark` in the debug log.
- [x] (2026-11-19) Reader view bug: `<h2>` to `<h6>` became screenfuls of `#`. The heading level was computed as `atom.H2 - atom.H1`, but atom values are hashes, not 1 to 6, so `<h2>` gave 263,425 `#` marks on one line (the 8,820-byte front page of rsdoiel.github.io became 1,361,961 bytes and 17,971 lines; seen in `session-blog.jsonl` as `out_bytes`, and on the Pacific Island Times). Fixed with a lookup (`headingLevel`). The test that should have caught it used `Contains("## Sub")`, which a line of 263,000 `#` also satisfies; it now matches whole lines for all six levels, and a second test bounds the Markdown to 3 times the HTML for headings, nested lists and quotes, tables and inline markup. Checked on the front page (4,920 bytes, 105 lines) and a Pacific Island Times article (932 KB of HTML, 3,939 bytes).
- [ ] Reader view limits to look at: pages whose text is built by JavaScript show an error, not content; tables are plain pipe tables; no `<picture>` or `srcset`; a page behind a login or a consent wall is shown as that wall. A `[image: alt]` line for body pictures is still open (see Images).
- [ ] XML, OPML, JSON and plain text links: show as source text (BonBon's rule); images: say what they are.

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
