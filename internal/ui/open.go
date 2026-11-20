package ui

import (
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/rsdoiel/cmarkbrowser/internal/history"
)

/** WithHistory sets the Markdown file that every page visited is recorded in,
 * newest first, and that H opens. With no file nothing is recorded and H says
 * how to turn it on.
 *
 * Parameters:
 *   path (string) — the history file; "" for none
 *
 * Returns:
 *   Model — the same browser with the file set
 *
 * Example:
 *   m := ui.New(addr, r).WithHistory("history.md")
 */
func (m Model) WithHistory(path string) Model {
	m.histPath = path
	return m
}

// normalizeAddr turns what was typed into an address to fetch: an http or https
// address as it is, a bare host name (a dot, no spaces, no scheme) with https://
// added. Anything else, a file name or another scheme included, gives "": a
// typed address never reads a local file or runs a script.
func normalizeAddr(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, " \t") {
		return ""
	}
	if !strings.Contains(s, "://") {
		host := strings.SplitN(s, "/", 2)[0]
		bare := !strings.Contains(s, "/")
		switch {
		case strings.HasPrefix(s, "/") || !strings.Contains(host, "."):
			return "" // a path, or a word: not a host name
		case bare && isFileName(host):
			return "" // notes.md is a file, though .md is also a domain
		}
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	return u.String()
}

// openAddrPrompt starts typing an address to open.
func (m Model) openAddrPrompt() Model {
	m.addrPrompt, m.addr, m.status = true, "", ""
	return m
}

// addrKey handles a key while an address is being typed. Every key goes to the
// prompt, so q does not quit and digits are not link numbers.
func (m Model) addrKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch {
	case s == "ctrl+c":
		m.log.Event("quit", "key", s)
		return m, tea.Quit
	case s == "esc":
		m.addrPrompt, m.addr, m.status = false, "", ""
	case s == "enter":
		typed := m.addr
		m.addrPrompt, m.addr = false, ""
		if strings.TrimSpace(typed) == "" {
			return m, nil
		}
		addr := normalizeAddr(typed)
		m.log.Event("open", "typed", typed, "addr", addr)
		if addr == "" {
			m.status = "not an address: " + typed + " (it starts with http:// or https://)"
			return m, nil
		}
		m.busy, m.status = true, "loading "+addr
		return m, m.load(addr, -1, "typed")
	case s == "backspace":
		if m.addr == "" {
			m.addrPrompt = false
			return m, nil
		}
		_, size := utf8.DecodeLastRuneInString(m.addr)
		m.addr = m.addr[:len(m.addr)-size]
	case s == "ctrl+u":
		m.addr = ""
	case msg.Text != "":
		m.addr += msg.Text
	}
	return m, nil
}

// record adds the page just shown to the history file, when there is one. A
// failure is shown in the status line and logged; it never stops the page.
func (m Model) record() Model {
	if m.histPath == "" || m.view == nil {
		return m
	}
	addr := m.view.Page.URL
	if isFileAddr(addr) {
		return m
	}
	if err := history.Record(m.histPath, m.view.Page.Title(), addr, time.Now()); err != nil {
		m.log.Event("history", "ok", false, "err", err)
		m.status = "history: " + err.Error()
		return m
	}
	m.log.Event("history", "ok", true, "addr", addr)
	return m
}

// openHistory shows the history file as a page.
func (m Model) openHistory() (tea.Model, tea.Cmd) {
	if m.histPath == "" {
		m.status = "no history file; start with -history FILE to keep one"
		return m, nil
	}
	if _, err := os.Stat(m.histPath); err != nil {
		m.status = "no history yet (" + m.histPath + ")"
		return m, nil
	}
	m.busy, m.status = true, "loading "+m.histPath
	return m, m.loadFile(m.histPath, -1, "history")
}

// isFileName reports whether a bare name ends like a document file.
func isFileName(s string) bool {
	switch strings.ToLower(s[strings.LastIndex(s, ".")+1:]) {
	case "md", "markdown", "txt", "html", "htm", "opml", "xml":
		return true
	}
	return false
}
