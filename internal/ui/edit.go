package ui

import (
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/rsdoiel/cmarkbrowser/internal/edit"
	"github.com/rsdoiel/cmarkbrowser/internal/local"
)

// editDoneMsg is the end of an editor session. addr is the page that was being
// edited in place ("" for a copy of a remote page), offset the scroll position
// to come back to.
type editDoneMsg struct {
	err    error
	addr   string
	copied string // the path of a copy of a remote page
	offset int
}

// runEditor starts the editor with the terminal handed over. A variable so
// tests can run a stand-in editor without a terminal.
var runEditor = func(c *exec.Cmd, fn tea.ExecCallback) tea.Cmd { return tea.ExecProcess(c, fn) }

// editPage hands the page being shown to $VISUAL or $EDITOR. A local file (a
// page of a local site, the bookmarks file, the history file) is edited in
// place and shown again afterwards. A remote page cannot be saved back, so its
// source is saved as a new file in the Downloads folder and that copy is
// edited; the person keeps or deletes it. A reader view has no source of its
// own and is refused.
func (m Model) editPage() (tea.Model, tea.Cmd) {
	if m.view == nil {
		m.status = "no page to edit"
		return m, nil
	}
	if m.view.Page.Reader {
		m.status = "a reader view has no Markdown source to edit"
		return m, nil
	}
	addr := m.view.Page.URL
	file, inPlace := local.PathOf(addr)
	if !inPlace {
		dir, err := edit.DownloadsDir()
		if err != nil {
			m.status = "cannot save a copy to edit: " + err.Error()
			return m, nil
		}
		if file, err = edit.SaveCopy(dir, addr, m.view.Page.Raw); err != nil {
			m.status = "cannot save a copy to edit: " + err.Error()
			return m, nil
		}
	}
	cmd, err := edit.Command(os.Getenv, file)
	if err != nil {
		if !inPlace {
			os.Remove(file) // nothing was edited; do not leave a copy behind
		}
		m.status = err.Error()
		return m, nil
	}
	m.log.Event("edit", "file", file, "in_place", inPlace, "editor", cmd.Args[0])
	done := editDoneMsg{offset: m.vp.YOffset()}
	if inPlace {
		done.addr = addr
	} else {
		done.copied = file
	}
	return m, runEditor(cmd, func(err error) tea.Msg {
		d := done
		d.err = err
		return d
	})
}

// editDone shows the page again after an edit in place, at the same place in
// it and in the same history entry, or says where the copy of a remote page is.
func (m Model) editDone(msg editDoneMsg) (tea.Model, tea.Cmd) {
	m.log.Event("edit.done", "addr", msg.addr, "copy", msg.copied, "err", msg.err)
	if msg.addr == "" {
		m.status = "copy saved at " + msg.copied
		if msg.err != nil {
			m.status = "editor: " + msg.err.Error() + "; copy kept at " + msg.copied
		}
		return m, nil
	}
	m.restoreAddr, m.restoreOffset = msg.addr, msg.offset
	m.busy, m.status = true, "reloading "+shown(msg.addr)
	if msg.err != nil {
		m.status = "editor: " + msg.err.Error() + "; reloading " + shown(msg.addr)
	}
	return m, m.load(msg.addr, m.idx, "edit")
}
