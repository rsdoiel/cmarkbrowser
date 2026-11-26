package ui

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/rsdoiel/cmarkbrowser/internal/search"
	"github.com/rsdoiel/markdownweb/debuglog"
)

// refresh redraws the page in the viewport with the focus and search marks.
func (m *Model) refresh() {
	if m.view != nil {
		m.vp.SetContent(m.content())
	}
}

// content is the page as shown: with the focused link marked and, while a
// search is active, its matches marked and the current one in its own style.
func (m *Model) content() string {
	if len(m.matches) == 0 {
		return m.view.Display(m.focus)
	}
	if m.marked == nil || m.markedFocus != m.focus {
		m.marked = search.NewMarked(m.view.Display(m.focus), m.matches)
		m.markedFocus = m.focus
	}
	return m.marked.Render(m.cur)
}

// clearSearch drops the active search and its marks. The last pattern is kept
// so that / then Enter can repeat it.
func (m *Model) clearSearch() {
	m.matches, m.marked, m.pattern, m.cur = nil, nil, "", 0
	m.refresh()
}

// openPrompt starts typing a search.
func (m Model) openPrompt() Model {
	if m.view == nil {
		m.status = "no page to search"
		return m
	}
	m.prompt, m.query, m.status = true, "", ""
	return m
}

// promptKey handles a key while a search is being typed. Every key goes to the
// prompt, so q does not quit and digits are not link numbers.
func (m Model) promptKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch {
	case s == "ctrl+c":
		m.log.Event("quit", "key", s)
		return m, tea.Quit
	case s == "esc":
		m.prompt, m.query, m.status = false, "", ""
	case s == "enter":
		pattern := m.query
		m.prompt, m.query, m.status = false, "", ""
		if pattern == "" {
			pattern = m.last
		}
		if pattern == "" {
			m.status = "no previous search"
			return m, nil
		}
		m.runSearch(pattern)
	case s == "backspace":
		if m.query == "" {
			m.prompt = false
			return m, nil
		}
		_, size := utf8.DecodeLastRuneInString(m.query)
		m.query = m.query[:len(m.query)-size]
	case s == "ctrl+u":
		m.query = ""
	case msg.Text != "":
		m.query += msg.Text
	}
	return m, nil
}

// runSearch finds pattern (a Go regexp, smart case) in the page shown and goes
// to the first match at or after the top of the window. A pattern that does not
// compile keeps the earlier search; one that matches nothing ends it.
func (m *Model) runSearch(pattern string) {
	if m.view == nil {
		return
	}
	t0 := time.Now()
	re, err := search.Compile(pattern)
	if err != nil {
		msg := strings.TrimPrefix(err.Error(), "error parsing regexp: ")
		m.status = "invalid regexp: " + msg
		m.log.Event("search", "pattern", pattern, "ok", false, "err", err)
		return
	}
	matches := search.NewIndex(m.view.Display(-1)).Find(re)
	m.last = pattern
	m.log.Event("search", "pattern", pattern, "ok", true, "matches", len(matches), "ms", time.Since(t0))
	if len(matches) == 0 {
		m.clearSearch()
		m.status = fmt.Sprintf("no match for /%s/", pattern)
		return
	}
	m.matches, m.marked, m.pattern = matches, nil, pattern
	m.cur = 0
	top := m.vp.YOffset()
	for i, mt := range matches {
		if mt.Line() >= top {
			m.cur = i
			break
		}
	}
	m.show("")
}

// reindex finds the active pattern again after the page was drawn at a new
// width, keeping the same match current when it still exists.
func (m *Model) reindex() {
	if m.pattern == "" {
		return
	}
	re, err := search.Compile(m.pattern)
	if err != nil {
		m.clearSearch()
		return
	}
	m.matches = search.NewIndex(m.view.Display(-1)).Find(re)
	m.marked = nil
	if len(m.matches) == 0 {
		m.clearSearch()
		return
	}
	m.cur = min(m.cur, len(m.matches)-1)
}

// nextMatch moves to the next (dir 1) or previous (dir -1) match, wrapping.
func (m Model) nextMatch(dir int) Model {
	if len(m.matches) == 0 {
		m.status = "no search; press / to search"
		return m
	}
	n, next, note := len(m.matches), m.cur+dir, ""
	switch {
	case next >= n:
		next, note = 0, " (wrapped)"
	case next < 0:
		next, note = n-1, " (wrapped)"
	}
	m.cur = next
	m.show(note)
	return m
}

// show scrolls the current match into view, redraws and says where we are.
func (m *Model) show(note string) {
	line := m.matches[m.cur].Line()
	if top, h := m.vp.YOffset(), m.vp.Height(); line < top || line >= top+h {
		m.vp.SetYOffset(max(line-h/3, 0))
	}
	m.refresh()
	m.status = fmt.Sprintf("match %d of %d%s  /%s", m.cur+1, len(m.matches), note, m.pattern)
	m.log.Event("match", "n", m.cur+1, "of", len(m.matches), "line", line)
}

// contextWith returns a background context that carries the debug logger.
func contextWith(l *debuglog.Logger) context.Context {
	return debuglog.WithContext(context.Background(), l)
}
