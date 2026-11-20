package ui

import (
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/rsdoiel/cmarkbrowser/internal/browse"
	"github.com/rsdoiel/cmarkbrowser/internal/local"
)

/** WithSite says the start argument opened a local site: pages inside the
 * site root are read as paths, and links in them that stay inside the root are
 * followed (cmarkbrowser DR-0001). Use it with WithLocalStart. The root is set
 * only here, from the command line; nothing in a page can set or widen it.
 *
 * Parameters:
 *   s (*local.Site) — the site root, from local.Open
 *
 * Returns:
 *   Model — the same browser with the site set
 *
 * Example:
 *   site, start, _ := local.Open("./")
 *   m := ui.New(start, r).WithLocalStart().WithSite(site)
 */
func (m Model) WithSite(s *local.Site) Model {
	m.site = s
	return m
}

// shown is an address as the person sees it: a local page as its path, never
// as a file: address (DR-0001); anything else as it is.
func shown(addr string) string {
	if p, ok := local.PathOf(addr); ok {
		return p
	}
	return addr
}

// followLink opens a link of the page being shown, by number or focus.
func (m Model) followLink(l browse.Link) (tea.Model, tea.Cmd) {
	return m.followResolved(l, "link")
}

// followResolved opens a link that is resolved. A file: address in a link is
// never followed; a path link in a local page is followed only when it lies
// inside the site root, symlinks and ".." resolved; everything else is a web
// address, loaded as before.
func (m Model) followResolved(l browse.Link, why string) (tea.Model, tea.Cmd) {
	if !isFileAddr(l.URL) {
		m.busy, m.status = true, "loading "+l.URL
		return m, m.load(l.URL, -1, why)
	}
	if l.Path == "" || m.view == nil || m.view.Site == nil {
		m.status = "a file: address is not followed (local files are opened only from a path on the command line)"
		m.log.Event("refused", "link", l.URL, "why", "file address")
		return m, nil
	}
	real, err := m.view.Site.Inside(l.Path)
	if err != nil {
		m.status = "outside the site root, not opened: " + l.Path
		m.log.Event("refused", "path", l.Path, "root", m.view.Site.Root, "why", "outside the site root")
		return m, nil
	}
	return m.openLocal(real, why)
}

// openLocal loads a file inside the site root as a local page.
func (m Model) openLocal(real, why string) (tea.Model, tea.Cmd) {
	addr := local.Address(filepath.Clean(real))
	m.localOK[addr], m.rooted[addr] = true, true
	m.busy, m.status = true, "loading "+real
	return m, m.load(addr, -1, why)
}
