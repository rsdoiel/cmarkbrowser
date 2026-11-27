package ui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/rsdoiel/markdownweb/debuglog"
	"github.com/rsdoiel/markdownweb/local"
	"github.com/rsdoiel/markdownweb/toc"
)

// tocMsg is the answer of a table of contents lookup. page is the address the
// question was asked about, so an answer that arrives after the person has
// moved on can be dropped; dir is +1 for next and -1 for previous.
type tocMsg struct {
	page  string
	dir   int
	res   toc.Result
	local bool // the answer is for a local page: Prev and Next are paths
}

// tocTimeout bounds one lookup: a few probes and any includes.
const tocTimeout = 30 * time.Second

// canLookUpTOC reports whether the page being shown can have a table of
// contents: a page of the Markdown web, fetched from a web address.
func (m Model) canLookUpTOC() bool {
	if m.toc == nil || m.view == nil || m.view.Page.Reader {
		return false
	}
	return !isFileAddr(m.view.Page.URL) || m.view.Site != nil // a local page has a table of contents only inside a site root
}

// lookupTOC asks the site's table of contents what comes before (dir -1) or
// after (dir +1) the page being shown. The reading is done in the background
// and the answer comes back as a tocMsg.
func (m Model) lookupTOC(dir int) tea.Cmd {
	page, finder, probe, log, site := m.view.Page.URL, m.toc, m.probe, m.log, m.view.Site
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(debuglog.WithContext(context.Background(), log), tocTimeout)
		defer cancel()
		if site != nil {
			p, _ := local.PathOf(page)
			return tocMsg{page: page, dir: dir, res: finder.LookupLocal(ctx, site.Root, p), local: true}
		}
		res := finder.Lookup(ctx, page)
		if res.Found && probe != nil {
			// pass over a neighbour with nothing at its address (DR-0004 Q1)
			if dir < 0 {
				res.Prev = probe.firstThatAnswers(ctx, res.Before, page)
			} else {
				res.Next = probe.firstThatAnswers(ctx, res.After, page)
			}
		}
		return tocMsg{page: page, dir: dir, res: res}
	}
}

// tocAnswered acts on a lookup: it opens the neighbouring page as a new visit,
// or says why there is none. An answer about a page that is no longer shown is
// dropped.
func (m Model) tocAnswered(msg tocMsg) (tea.Model, tea.Cmd) {
	m.busy = false
	if m.view == nil || m.view.Page.URL != msg.page {
		return m, nil
	}
	target, none, why := msg.res.Next, "no next page", "next"
	if msg.dir < 0 {
		target, none, why = msg.res.Prev, "no previous page", "prev"
	}
	switch {
	case !msg.res.Found:
		m.status = none + " (no table of contents lists this page)"
		return m, nil
	case target == "":
		m.status = none
		return m, nil
	}
	if msg.local {
		// a path the outline named, kept inside the site root like any path link
		real, err := m.view.Site.Inside(target)
		if err != nil {
			m.status = "outside the site root, not opened: " + target
			return m, nil
		}
		return m.openLocal(real, why)
	}
	m.busy, m.status = true, "loading "+target
	return m, m.load(target, -1, why)
}
