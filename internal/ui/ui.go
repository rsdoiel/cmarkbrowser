// Package ui is the cmarkbrowser terminal interface: a scrolling page with
// numbered links, history, and a status line.
package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/rsdoiel/cmarkbrowser/internal/browse"
	"github.com/rsdoiel/cmarkbrowser/internal/render"
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
}

/** Model is the browser. Create it with New and give it to tea.NewProgram.
 *
 * Keys: a number then Enter follows that link; Esc clears the number;
 * b and f go back and forward; q quits; the rest scroll (arrows, PgUp, PgDn,
 * space, g, G).
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
	num    string
	status string
	busy   bool
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
	return Model{r: r, start: start, idx: -1, w: 80, h: 24, vp: viewport.New(viewport.WithWidth(80), viewport.WithHeight(22))}
}

// width is the column count pages are rendered to.
func (m Model) width() int { return min(max(m.w, 20), maxWidth) }

// Init loads the start page.
func (m Model) Init() tea.Cmd { return m.load(m.start, -1) }

// load fetches addr; idx is the history index to move to, or -1 to add an entry.
func (m Model) load(addr string, idx int) tea.Cmd {
	r, width := m.r, m.width()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		v, err := browse.Open(ctx, addr, r, width)
		return loadMsg{v: v, err: err, addr: addr, idx: idx}
	}
}

// Update handles keys, resizes and finished loads.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.vp.SetWidth(m.w)
		m.vp.SetHeight(max(m.h-2, 1))
		if m.view != nil {
			if err := m.view.Rerender(m.r, m.width()); err != nil {
				m.status = err.Error()
			}
			m.vp.SetContent(m.view.Text)
		}
		return m, nil
	case loadMsg:
		return m.loaded(msg), nil
	case tea.KeyPressMsg:
		return m.key(msg)
	}
	return m, nil
}

// loaded installs a finished load, or shows its error and keeps the old page.
func (m Model) loaded(msg loadMsg) Model {
	m.busy = false
	if msg.err != nil {
		m.status = msg.err.Error()
		return m
	}
	m.status = ""
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
	m.vp.SetContent(msg.v.Text)
	m.vp.GotoTop()
	return m
}

// key handles one key press.
func (m Model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch {
	case s == "ctrl+c" || (s == "q" && m.num == ""):
		return m, tea.Quit
	case len(s) == 1 && s[0] >= '0' && s[0] <= '9':
		if len(m.num) < 4 {
			m.num += s
		}
		m.status = ""
		return m, nil
	case s == "esc":
		m.num, m.status = "", ""
		return m, nil
	case s == "backspace" && m.num != "":
		m.num = m.num[:len(m.num)-1]
		return m, nil
	case s == "enter" && m.num != "":
		return m.follow()
	case s == "b" || s == "left":
		return m.step(-1)
	case s == "f" || s == "right":
		return m.step(1)
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

// follow opens the link whose number has been typed.
func (m Model) follow() (tea.Model, tea.Cmd) {
	n, _ := strconv.Atoi(m.num)
	m.num = ""
	if m.view != nil {
		for _, l := range m.view.Links {
			if l.N == n {
				m.busy, m.status = true, "loading "+l.URL
				return m, m.load(l.URL, -1)
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
	return m, m.load(m.hist[to], to)
}

// View draws the title line, the page and the status line.
func (m Model) View() tea.View {
	title := "cmarkbrowser"
	if m.view != nil {
		title = m.view.Page.Title()
		if title == "" {
			title = m.view.Page.URL
		}
		title += "  " + m.view.Page.URL
	}
	foot := "number+Enter follow  b back  f forward  q quit"
	switch {
	case m.status != "":
		foot = m.status
	case m.num != "":
		foot = "link " + m.num + "  (Enter to follow, Esc to cancel)"
	}
	if m.view != nil && m.status == "" && m.num == "" {
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
