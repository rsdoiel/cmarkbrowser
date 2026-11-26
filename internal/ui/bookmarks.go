package ui

import (
	"errors"
	"fmt"
	"github.com/rsdoiel/markdownweb/local"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/rsdoiel/cmarkbrowser/internal/bookmarks"
	"github.com/rsdoiel/cmarkbrowser/internal/browse"
)

// defaultBookmarks is the bookmarks file when none is named: in the working
// directory, so there is no settings file anywhere else.
const defaultBookmarks = "bookmarks.md"

/** WithBookmarks sets the Markdown file that m adds to and B opens.
 *
 * Parameters:
 *   path (string) — the bookmarks file; "" means bookmarks.md in the working directory
 *
 * Returns:
 *   Model — the same browser with the file set
 *
 * Example:
 *   m := ui.New(addr, r).WithBookmarks("reading.md")
 */
func (m Model) WithBookmarks(path string) Model {
	m.bmPath = path
	return m
}

/** WithLocalStart says the start argument is a local file the person named, not
 * an address. A local file is only ever opened this way or through B; a link
 * can never open one.
 *
 * Returns:
 *   Model — the same browser, starting from the file
 *
 * Example:
 *   m := ui.New("notes.md", r).WithLocalStart()
 */
func (m Model) WithLocalStart() Model {
	m.localStart = true
	return m
}

func (m Model) bookmarksFile() string {
	if m.bmPath == "" {
		return defaultBookmarks
	}
	return m.bmPath
}

// isFileAddr reports whether addr is a file: address.
func isFileAddr(addr string) bool { return strings.HasPrefix(strings.ToLower(addr), "file:") }

// loadFile opens a local file the person named. Its address is remembered as
// allowed, so history can come back to it; nothing else adds to that set.
func (m Model) loadFile(path string, idx int, why string) tea.Cmd {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	addr := (&url.URL{Scheme: "file", Path: abs}).String()
	if m.localOK != nil {
		m.localOK[addr] = true
	}
	r, width, log := m.r, m.width(), m.log
	// A page opened inside the site root has a site: its links are paths. The
	// bookmarks and history files, and anything else, have none (DR-0001).
	var site *local.Site
	if m.site != nil && m.rooted[addr] {
		site = m.site
	}
	log.Event("load.start", "addr", addr, "idx", idx, "why", why, "rooted", site != nil)
	return func() tea.Msg {
		t0 := time.Now()
		var v *browse.View
		var err error
		if site != nil {
			v, err = browse.OpenLocal(contextWith(log), abs, site, r, width)
		} else {
			v, err = browse.OpenFile(contextWith(log), abs, r, width)
		}
		return loadMsg{v: v, err: err, addr: addr, idx: idx, took: time.Since(t0)}
	}
}

// refuseFile is the answer to a link or history entry that names a local file
// the person did not open: nothing is read.
func refuseFile(addr string, idx int) tea.Cmd {
	return func() tea.Msg {
		return loadMsg{err: fmt.Errorf("%s: local files are opened only from the command line or as bookmarks, never from a link", addr), addr: addr, idx: idx}
	}
}

// openFileAddr loads a file: address when the person opened it, else refuses.
func (m Model) openFileAddr(addr string, idx int, why string) tea.Cmd {
	if !m.localOK[addr] {
		m.log.Event("load.start", "addr", addr, "idx", idx, "why", why, "refused", true)
		return refuseFile(addr, idx)
	}
	u, err := url.Parse(addr)
	if err != nil {
		return refuseFile(addr, idx)
	}
	return m.loadFile(u.Path, idx, why)
}

// openBookmarks shows the bookmarks file as a page.
func (m Model) openBookmarks() (tea.Model, tea.Cmd) {
	if _, err := os.Stat(m.bookmarksFile()); err != nil {
		m.status = "no bookmarks yet; press m on a page to add one (" + m.bookmarksFile() + ")"
		return m, nil
	}
	m.busy, m.status = true, "loading "+m.bookmarksFile()
	return m, m.loadFile(m.bookmarksFile(), -1, "bookmarks")
}

// mark adds the page being shown to the bookmarks file.
func (m Model) mark() Model {
	if m.view == nil {
		m.status = "no page to bookmark"
		return m
	}
	addr := m.view.Page.URL
	if isFileAddr(addr) {
		m.status = "a local file is not bookmarked"
		return m
	}
	title := m.view.Page.Title()
	added, err := bookmarks.Add(m.bookmarksFile(), title, addr, time.Now())
	m.log.Event("bookmark", "url", addr, "file", m.bookmarksFile(), "added", added, "err", err)
	switch {
	case errors.Is(err, bookmarks.ErrNotWeb):
		m.status = "only web pages can be bookmarked"
	case err != nil:
		m.status = "bookmark failed: " + err.Error()
	case !added:
		m.status = "already bookmarked"
	default:
		if title == "" {
			title = addr
		}
		m.status = "bookmarked: " + title + "  (" + filepath.Base(m.bookmarksFile()) + ")"
	}
	return m
}
