package ui

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/rsdoiel/cmarkbrowser/internal/browse"
	"github.com/rsdoiel/cmarkbrowser/internal/results"
	"github.com/rsdoiel/cmarkfind/corpus"
	"github.com/rsdoiel/markdownweb/debuglog"
	"github.com/rsdoiel/markdownweb/fetch"
	"github.com/rsdoiel/markdownweb/toc"
)

// corpusScheme starts the address of a search results page (cmarkbrowser
// DR-0003 4); nothing fetches it, the page is made from the corpus.
const corpusScheme = "corpus:search?q="

// maxHits is the most hits a results page shows.
const maxHits = 50

// fillTimeout bounds one F: a batch of pages at the fill's pause.
const fillTimeout = 10 * time.Minute

func isCorpusAddr(addr string) bool { return strings.HasPrefix(addr, "corpus:") }

/** WithCorpus returns the browser with a corpus to search (S) and fill (F).
 * Without it both keys say search is off.
 *
 * Parameters:
 *   path (string)      — the corpus file (cmarkfind's); a search never creates it, a fill does
 *   userAgent (string) — the User-Agent a fill sends
 *
 * Returns:
 *   Model — the browser
 *
 * Example:
 *   m := ui.New(start, r).WithCorpus(corpus.PathFromEnv(), "cmarkbrowser/0.0.4")
 */
func (m Model) WithCorpus(path, userAgent string) Model {
	m.corpusPath, m.corpusUA = path, userAgent
	m.fillBatch, m.fillPause = 25, time.Second
	return m
}

// searchOff is what S and F say when no corpus is configured.
const searchOff = "search is off (no corpus; -no-corpus was given)"

// openFindPrompt starts typing a query, when there is a corpus to search.
func (m Model) openFindPrompt() Model {
	if m.corpusPath == "" {
		m.status = searchOff
		return m
	}
	if _, err := os.Stat(m.corpusPath); err != nil {
		m.status = "no corpus at " + m.corpusPath + "; F fills it from the site being read"
		return m
	}
	m.findPrompt, m.find, m.status = true, "", ""
	return m
}

// findKey handles a key while a query is being typed.
func (m Model) findKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch {
	case s == "ctrl+c":
		m.log.Event("quit", "key", s)
		return m, tea.Quit
	case s == "esc":
		m.findPrompt, m.find, m.status = false, "", ""
	case s == "enter":
		q := strings.TrimSpace(m.find)
		m.findPrompt, m.find = false, ""
		if q == "" {
			return m, nil
		}
		m.log.Event("search.corpus", "query", q)
		m.busy, m.status = true, "searching"
		return m, m.load(corpusScheme+url.QueryEscape(q), -1, "search")
	case s == "backspace":
		if m.find == "" {
			m.findPrompt = false
			return m, nil
		}
		_, size := utf8.DecodeLastRuneInString(m.find)
		m.find = m.find[:len(m.find)-size]
	case s == "ctrl+u":
		m.find = ""
	case msg.Text != "":
		m.find += msg.Text
	}
	return m, nil
}

// searchPage makes the command that runs the query an address names and shows
// the results as a page. Going back to a results page runs it again.
func (m Model) searchPage(addr string, idx int) tea.Cmd {
	r, width, log, path := m.r, m.width(), m.log, m.corpusPath
	log.Event("load.start", "addr", addr, "idx", idx, "why", "search")
	return func() tea.Msg {
		t0 := time.Now()
		fail := func(err error) tea.Msg { return loadMsg{err: err, addr: addr, idx: idx, took: time.Since(t0)} }
		q, err := url.QueryUnescape(strings.TrimPrefix(addr, corpusScheme))
		if err != nil || path == "" {
			return fail(fmt.Errorf("not a search address: %s", addr))
		}
		if _, err := os.Stat(path); err != nil {
			return fail(fmt.Errorf("no corpus at %s", path))
		}
		db, err := corpus.Open(path)
		if err != nil {
			return fail(err)
		}
		defer db.Close()
		ctx := debuglog.WithContext(context.Background(), log)
		res, err := db.Search(ctx, q, maxHits)
		if err != nil {
			return fail(err)
		}
		v, err := browse.OpenMarkdown(ctx, addr, results.Page(q, res), r, width)
		return loadMsg{v: v, err: err, addr: addr, idx: idx, took: time.Since(t0)}
	}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// fillMsg is the answer of a fill started with F.
type fillMsg struct {
	rep   *corpus.FillReport
	err   error
	where string // the site filled, for the status line
	local bool
}

// startFill fills the corpus from the site being read: one batch of pages of
// the table of contents that lists the page, or the whole local site.
func (m Model) startFill() (tea.Model, tea.Cmd) {
	switch {
	case m.corpusPath == "":
		m.status = searchOff
		return m, nil
	case m.filling:
		m.status = "a fill is already running"
		return m, nil
	case m.view == nil:
		m.status = "no page to fill from"
		return m, nil
	}
	path, ua, batch, pause, log := m.corpusPath, m.corpusUA, m.fillBatch, m.fillPause, m.log
	addr, finder, site := m.view.Page.URL, m.toc, m.view.Site
	if site == nil && (isFileAddr(addr) || isCorpusAddr(addr)) {
		m.status = "nothing to fill from this page"
		return m, nil
	}
	m.filling, m.status = true, "filling the corpus…"
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(debuglog.WithContext(context.Background(), log), fillTimeout)
		defer cancel()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fillMsg{err: err}
		}
		db, err := corpus.Open(path)
		if err != nil {
			return fillMsg{err: err}
		}
		defer db.Close()
		if site != nil {
			rep, err := db.FillLocal(ctx, site.Root, toc.Limits{})
			return fillMsg{rep: rep, err: err, where: site.Root, local: true}
		}
		res := finder.Lookup(ctx, addr)
		if !res.Found || res.TOC == "" {
			return fillMsg{err: fmt.Errorf("no table of contents lists this page, so there is no site to fill")}
		}
		f := fetch.New(fetch.Options{UserAgent: ua, Pause: pause, Robots: true})
		rep, err := db.FillRemoteWith(ctx, f, res.TOC, toc.Limits{}, corpus.FillOptions{Batch: batch})
		return fillMsg{rep: rep, err: err, where: res.TOC}
	}
}

// fillDone says what a fill did and what is left.
func (m Model) fillDone(msg fillMsg) Model {
	m.filling = false
	if msg.rep == nil {
		m.status = "fill: " + msg.err.Error()
		return m
	}
	r := msg.rep
	done := r.Added + r.Updated + r.Unchanged
	var s string
	switch {
	case msg.local:
		s = fmt.Sprintf("filled %d %s from %s", done, plural(done, "page", "pages"), msg.where)
	case r.Remaining > 0:
		s = fmt.Sprintf("filled %d; %d %s not fetched yet; F for more", done, r.Remaining, plural(r.Remaining, "page", "pages"))
	default:
		s = fmt.Sprintf("filled %d; all %d pages fetched", done, r.Listed)
	}
	if r.Failed > 0 {
		s += fmt.Sprintf("; %d failed", r.Failed)
	}
	if msg.err != nil && r.Failed == 0 {
		s += "; " + msg.err.Error()
	}
	m.status = s
	return m
}
