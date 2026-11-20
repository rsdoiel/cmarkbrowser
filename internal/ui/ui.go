// Package ui is the cmarkbrowser terminal interface: a scrolling page with
// numbered links, history, and a status line.
package ui

import (
	"context"
	"fmt"
	"github.com/rsdoiel/cmarkbrowser/internal/local"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/rsdoiel/cmarkbrowser/internal/browse"
	"github.com/rsdoiel/cmarkbrowser/internal/debuglog"
	"github.com/rsdoiel/cmarkbrowser/internal/render"
	"github.com/rsdoiel/cmarkbrowser/internal/search"
	"github.com/rsdoiel/cmarkbrowser/internal/toc"
)

// maxWidth keeps lines readable on a very wide terminal.
const maxWidth = 100

// fetchTimeout bounds one page load.
const fetchTimeout = 35 * time.Second

// loadMsg is the result of a page load.
type loadMsg struct {
	v    *browse.View
	err  error
	addr string
	idx  int // history index to move to, or -1 to add a new entry
	took time.Duration
}

/** Model is the browser. Create it with New and give it to tea.NewProgram.
 *
 * Keys: Tab and Shift-Tab move the focus between links and Enter follows the
 * focused link; a number then Enter follows that link instead; Esc clears the
 * number, or the focus; b and f go back and forward; q quits; the rest scroll
 * (arrows, PgUp, PgDn, space, g, G).
 *
 * Example:
 *   p := tea.NewProgram(ui.New("https://dave.writersweb.org/", render.Glamour{}))
 */
type Model struct {
	r      render.Renderer
	start  string
	view   *browse.View
	hist   []string
	idx    int
	vp     viewport.Model
	w, h   int
	num    string // link number being typed
	focus  int    // number of the focused link, 0 for none
	status string
	busy   bool
	log    *debuglog.Logger // nil unless -debug

	// search: a prompt being typed, and the matches of the last search
	prompt      bool
	query       string // pattern being typed
	last        string // last pattern run, for / then Enter
	pattern     string // pattern of the active search, "" for none
	matches     []search.Match
	cur         int            // index of the current match
	marked      *search.Marked // the page with the matches marked, for the focus in markedFocus
	markedFocus int

	toc *toc.Finder // the site's table of contents, read when ] or [ needs it

	// an address being typed (o), and the history file (H)
	addrPrompt bool
	addr       string
	histPath   string // "" records nothing

	// bookmarks and local files
	bmPath        string          // the bookmarks file, "" for bookmarks.md
	localStart    bool            // the start argument is a file the person named
	localOK       map[string]bool // file addresses the person opened; a link never adds to it
	restoreAddr   string          // after an edit, the page to put back at restoreOffset
	restoreOffset int
	site          *local.Site     // the site root the person opened a local site at (DR-0001), nil for none
	rooted        map[string]bool // local pages opened inside the site root, so their links are paths
}

/** New returns a browser that opens start when it runs.
 *
 * Parameters:
 *   start (string) — the first page address
 *   r (render.Renderer) — the renderer
 *
 * Returns:
 *   Model — ready for tea.NewProgram
 *
 * Example:
 *   m := ui.New("https://x.org/a.md", render.Glamour{Style: "dark"})
 */
func New(start string, r render.Renderer) Model {
	return Model{r: r, start: start, idx: -1, w: 80, h: 24, localOK: map[string]bool{}, rooted: map[string]bool{}, toc: toc.NewFinder(), vp: viewport.New(viewport.WithWidth(80), viewport.WithHeight(22))}
}

/** WithDebug returns the browser with a debug logger: every key, load, status
 * message, focus move, scroll and resize is written to it as a JSON line. A
 * nil logger logs nothing.
 *
 * Parameters:
 *   l (*debuglog.Logger) — where the events go
 *
 * Returns:
 *   Model — the same browser with logging on
 *
 * Example:
 *   m := ui.New(addr, r).WithDebug(debuglog.New(f))
 */
func (m Model) WithDebug(l *debuglog.Logger) Model {
	m.log = l
	return m
}

// width is the column count pages are rendered to.
func (m Model) width() int { return min(max(m.w, 20), maxWidth) }

// Init loads the start page.
func (m Model) Init() tea.Cmd {
	if m.localStart {
		if m.site != nil {
			abs, err := filepath.Abs(m.start)
			if err != nil {
				abs = m.start
			}
			if _, err := m.site.Inside(abs); err == nil {
				m.rooted[local.Address(abs)] = true
			}
		}
		return m.loadFile(m.start, -1, "start")
	}
	return m.load(m.start, -1, "start")
}

// load fetches addr; idx is the history index to move to, or -1 to add an
// entry; why says what asked for it (start, link, history, next, prev) for the
// debug log.
func (m Model) load(addr string, idx int, why string) tea.Cmd {
	if isFileAddr(addr) {
		return m.openFileAddr(addr, idx, why)
	}
	r, width, log := m.r, m.width(), m.log
	// A followed link into an HTML site that this site mirrors opens the
	// Markdown copy when it is there, else the link as written (BonBon's rule).
	var finder *toc.Finder
	if why == "link" && m.canLookUpTOC() {
		finder = m.toc
	}
	page := ""
	if m.view != nil {
		page = m.view.Page.URL
	}
	log.Event("load.start", "addr", addr, "idx", idx, "why", why)
	return func() tea.Msg {
		t0 := time.Now()
		ctx, cancel := context.WithTimeout(debuglog.WithContext(context.Background(), log), fetchTimeout)
		defer cancel()
		if finder != nil {
			for _, mir := range finder.Mirrors(ctx, page) {
				dest := mir.Translate(addr)
				if dest == "" {
					continue
				}
				if v, err := browse.Open(ctx, dest, r, width); err == nil {
					log.Event("mirror.follow", "link", addr, "mirror", dest, "ok", true)
					return loadMsg{v: v, addr: dest, idx: idx, took: time.Since(t0)}
				}
				log.Event("mirror.follow", "link", addr, "mirror", dest, "ok", false)
			}
		}
		v, err := browse.Open(ctx, addr, r, width)
		return loadMsg{v: v, err: err, addr: addr, idx: idx, took: time.Since(t0)}
	}
}

// Update handles keys, resizes and finished loads, and logs what happened.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	before := m.status
	switch v := msg.(type) {
	case tea.KeyPressMsg:
		m.log.Event("key", "key", v.String(), "num", m.num, "focus", m.focus)
	case tea.WindowSizeMsg:
		m.log.Event("resize", "w", v.Width, "h", v.Height)
	}
	next, cmd := m.update(msg)
	if nm, ok := next.(Model); ok && nm.status != "" && nm.status != before {
		m.log.Event("status", "text", nm.status)
	}
	return next, cmd
}

// update is Update without the logging.
func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.vp.SetWidth(m.w)
		m.vp.SetHeight(max(m.h-2, 1))
		if m.view != nil {
			if err := m.view.Rerender(m.r, m.width()); err != nil {
				m.status = err.Error()
			}
			m.reindex()
			m.refresh()
		}
		return m, nil
	case loadMsg:
		return m.loaded(msg), nil
	case editDoneMsg:
		return m.editDone(msg)
	case tocMsg:
		return m.tocAnswered(msg)
	case tea.KeyPressMsg:
		return m.key(msg)
	}
	return m, nil
}

// loaded installs a finished load, or shows its error and keeps the old page.
func (m Model) loaded(msg loadMsg) Model {
	m.busy = false
	if msg.err != nil {
		m.restoreAddr = ""
		m.log.Event("load.done", "addr", msg.addr, "ok", false, "err", msg.err, "ms", msg.took)
		m.status = msg.err.Error()
		return m
	}
	m.log.Event("load.done", "addr", msg.addr, "final", msg.v.Page.URL, "ok", true, "idx", msg.idx, "ms", msg.took,
		"links", len(msg.v.Links), "missing_keys", msg.v.Missing)
	m.status = ""
	if len(msg.v.Missing) > 0 {
		m.status = "no front matter value for {{page." + strings.Join(msg.v.Missing, "}}, {{page.") + "}}"
	}
	m.view = msg.v
	addr := msg.v.Page.URL
	if msg.idx < 0 {
		if m.idx+1 < len(m.hist) {
			m.hist = m.hist[:m.idx+1]
		}
		m.hist = append(m.hist, addr)
		m.idx = len(m.hist) - 1
	} else {
		m.idx = msg.idx
	}
	m.focus = 0
	// a search belongs to the page it was run on
	m.prompt, m.query, m.matches, m.marked, m.pattern, m.cur = false, "", nil, nil, "", 0
	m.vp.SetContent(msg.v.Display(0))
	m.vp.GotoTop()
	if m.restoreAddr != "" && m.restoreAddr == msg.v.Page.URL {
		m.vp.SetYOffset(m.restoreOffset) // back where the edit began, clamped to the new length
	}
	m.restoreAddr = ""
	return m.record()
}

// key handles one key press.
func (m Model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.addrPrompt {
		return m.addrKey(msg)
	}
	if m.prompt {
		return m.promptKey(msg)
	}
	s := msg.String()
	switch {
	case s == "ctrl+c" || (s == "q" && m.num == ""):
		m.log.Event("quit", "key", s)
		return m, tea.Quit
	case len(s) == 1 && s[0] >= '0' && s[0] <= '9':
		if len(m.num) < 4 {
			m.num += s
		}
		m.status = ""
		return m, nil
	case s == "esc":
		// Esc undoes one thing: a typed number, else the focus, else the search.
		switch {
		case m.num != "":
		case m.focus != 0:
			m.setFocus(0)
		case len(m.matches) > 0:
			m.clearSearch()
		}
		m.num, m.status = "", ""
		return m, nil
	case s == "m" && m.num == "":
		return m.mark(), nil
	case s == "e" && m.num == "":
		return m.editPage()
	case s == "o" && m.num == "":
		return m.openAddrPrompt(), nil
	case s == "H" && m.num == "":
		return m.openHistory()
	case s == "B" && m.num == "":
		return m.openBookmarks()
	case s == "/" && m.num == "":
		return m.openPrompt(), nil
	case s == "n" && m.num == "":
		return m.nextMatch(1), nil
	case s == "N" && m.num == "":
		return m.nextMatch(-1), nil
	case s == "tab":
		return m.moveFocus(1), nil
	case s == "shift+tab":
		return m.moveFocus(-1), nil
	case s == "backspace" && m.num != "":
		m.num = m.num[:len(m.num)-1]
		return m, nil
	case s == "enter" && m.num != "":
		return m.follow()
	case s == "enter" && m.focus != 0:
		m.num = strconv.Itoa(m.focus)
		return m.follow()
	case s == "]" && m.num == "":
		return m.sibling("urlNext", "no next page")
	case s == "[" && m.num == "":
		return m.sibling("urlPrev", "no previous page")
	case s == "b" || s == "left":
		return m.step(-1)
	case s == "f" || s == "right":
		return m.step(1)
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	m.log.Event("scroll", "key", s, "offset", m.vp.YOffset(), "percent", int(m.vp.ScrollPercent()*100))
	return m, cmd
}

// sibling loads the page named by a front matter key (urlNext or urlPrev) of
// the current page, as a new visit. When the page has no such key, or it is
// not an address, the status says so and the page stays.
func (m Model) sibling(key, none string) (tea.Model, tea.Cmd) {
	m.status = ""
	if m.view == nil || m.busy {
		return m, nil
	}
	dest := ""
	if v, ok := m.view.Page.Meta[key]; ok && v != nil {
		dest = strings.TrimSpace(fmt.Sprint(v))
	}
	if dest == "" {
		// The page names no such page: ask the site's table of contents
		// (antennaApp DR-0013 5), which is read now and not when the page loads.
		if m.canLookUpTOC() {
			dir := -1
			if key == "urlNext" {
				dir = 1
			}
			m.busy, m.status = true, "looking in the table of contents"
			return m, m.lookupTOC(dir)
		}
		m.status = none
		return m, nil
	}
	why := "prev"
	if key == "urlNext" {
		why = "next"
	}
	if m.view.Site != nil {
		pagePath, _ := local.PathOf(m.view.Page.URL)
		ll, err := m.view.Site.ResolveLink(pagePath, dest)
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		l := browse.Link{URL: ll.URL}
		if ll.Path != "" {
			l = browse.Link{URL: local.Address(ll.Path), Path: ll.Path}
		}
		return m.followResolved(l, why)
	}
	addr, err := browse.Resolve(m.view.Page.URL, dest)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.busy, m.status = true, "loading "+addr
	return m, m.load(addr, -1, why)
}

// focusable returns the links whose markers were found in the rendered text.
func (m Model) focusable() []browse.Link {
	var out []browse.Link
	if m.view == nil {
		return nil
	}
	for _, l := range m.view.Links {
		if _, ok := m.view.LineOf(l.N); ok {
			out = append(out, l)
		}
	}
	return out
}

// moveFocus moves the focus to the next (dir 1) or previous (dir -1) link. With
// no focus, it starts from the first link in the window going forward, and
// from the last link in the window going back. It stops at either end.
func (m Model) moveFocus(dir int) Model {
	links := m.focusable()
	if len(links) == 0 {
		m.status = "no links on this page"
		return m
	}
	m.status = ""
	at := -1
	for i, l := range links {
		if l.N == m.focus {
			at = i
		}
	}
	top, bottom := m.vp.YOffset(), m.vp.YOffset()+m.vp.Height()
	switch {
	case at < 0 && dir > 0:
		at = 0
		for i, l := range links {
			if line, _ := m.view.LineOf(l.N); line >= top {
				at = i
				break
			}
		}
	case at < 0:
		at = len(links) - 1
		for i := len(links) - 1; i >= 0; i-- {
			if line, _ := m.view.LineOf(links[i].N); line < bottom {
				at = i
				break
			}
		}
	case at+dir < 0:
		m.status = "first link"
		return m
	case at+dir >= len(links):
		m.status = "last link"
		return m
	default:
		at += dir
	}
	m.setFocus(links[at].N)
	return m
}

// setFocus focuses link n (0 for none), redraws, and scrolls it into view.
func (m *Model) setFocus(n int) {
	m.focus = n
	url := ""
	for _, l := range m.view.Links {
		if l.N == n {
			url = l.URL
		}
	}
	m.log.Event("focus", "n", n, "url", url)
	m.refresh()
	if n == 0 {
		return
	}
	line, ok := m.view.LineOf(n)
	if !ok {
		return
	}
	if top, h := m.vp.YOffset(), m.vp.Height(); line < top || line >= top+h {
		m.vp.SetYOffset(max(line-h/3, 0))
	}
}

// follow opens the link whose number has been typed.
func (m Model) follow() (tea.Model, tea.Cmd) {
	n, _ := strconv.Atoi(m.num)
	m.num = ""
	if m.view != nil {
		for _, l := range m.view.Links {
			if l.N == n {
				return m.followLink(l)
			}
		}
	}
	m.status = fmt.Sprintf("no link %d", n)
	return m, nil
}

// step moves through history by delta, doing nothing at either end.
func (m Model) step(delta int) (tea.Model, tea.Cmd) {
	to := m.idx + delta
	if to < 0 || to >= len(m.hist) {
		return m, nil
	}
	m.busy, m.status = true, "loading "+m.hist[to]
	return m, m.load(m.hist[to], to, "history")
}

// View draws the title line, the page and the status line.
func (m Model) View() tea.View {
	title := "cmarkbrowser"
	if m.view != nil {
		title = m.view.Page.Title()
		if title == "" {
			title = shown(m.view.Page.URL)
		}
		title += "  " + shown(m.view.Page.URL)
		if m.view.Page.Reader {
			title += "  [HTML, reader view]"
		}
	}
	foot := "Tab focus link  Enter follow  / search  m bookmark  B bookmarks  b back  f forward"
	if m.view != nil {
		if v := m.view.Page.URLPrev(); v != "" {
			foot += "  [ prev"
		}
		if v := m.view.Page.URLNext(); v != "" {
			foot += "  ] next"
		}
	}
	foot += "  q quit"
	if len(m.matches) > 0 {
		foot = fmt.Sprintf("/%s  %d of %d  n next  N prev  Esc clear", m.pattern, m.cur+1, len(m.matches))
	}
	switch {
	case m.addrPrompt:
		foot = "open: " + m.addr + "▏"
	case m.prompt:
		foot = "/" + m.query + "▏"
	case m.status != "":
		foot = m.status
	case m.num != "":
		foot = "link " + m.num + "  (Enter to follow, Esc to cancel)"
	}
	if m.focus != 0 && m.status == "" && m.num == "" && m.view != nil {
		for _, l := range m.view.Links {
			if l.N == m.focus {
				foot = "→ " + shown(l.URL)
			}
		}
	}
	if m.view != nil && m.status == "" && m.num == "" && !m.prompt && !m.addrPrompt {
		foot = fmt.Sprintf("%3.0f%%  %s", m.vp.ScrollPercent()*100, foot)
	}
	v := tea.NewView(clip(title, m.w) + "\n" + m.vp.View() + "\n" + clip(foot, m.w))
	v.AltScreen = true
	return v
}

// clip cuts s to at most w runes, so a long address cannot wrap the status lines.
func clip(s string, w int) string {
	if w <= 0 {
		return s
	}
	if r := []rune(s); len(r) > w {
		return strings.TrimSpace(string(r[:w-1])) + "…"
	}
	return s
}
