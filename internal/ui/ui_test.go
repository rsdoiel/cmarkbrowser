package ui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/rsdoiel/cmarkbrowser/internal/render"
)

func site(t *testing.T) *httptest.Server {
	t.Helper()
	pages := map[string]string{
		"/a.md":   "---\ntitle: Page A\n---\n# Alpha\n\nGo to [b](b.md) or [c](c.md).\n",
		"/b.md":   "---\ntitle: Page B\n---\n# Beta\n\n[home](a.md)\n",
		"/c.md":   "# Gamma\n",
		"/s/1.md": "---\ntitle: One\nurlNext: 2.md\n---\n# Story One\n",
		"/s/2.md": "---\ntitle: Two\nurlPrev: 1.md\nurlNext: /c.md\n---\n# Story Two\n",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := pages[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/markdown")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// drive runs a command and feeds its message back, the way the runtime would.
func drive(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		return m
	}
	msg := cmd()
	next, cmd2 := m.Update(msg)
	m = next.(Model)
	if _, isQuit := msg.(tea.QuitMsg); isQuit {
		return m
	}
	if cmd2 != nil {
		if _, ok := cmd2().(loadMsg); ok {
			return drive(t, m, cmd2)
		}
	}
	return m
}

func key(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	}
	r := []rune(s)[0]
	return tea.KeyPressMsg{Code: r, Text: s}
}

func press(t *testing.T, m Model, keys ...string) Model {
	t.Helper()
	for _, k := range keys {
		next, cmd := m.Update(key(k))
		m = drive(t, next.(Model), cmd)
	}
	return m
}

func open(t *testing.T) (Model, string) {
	t.Helper()
	srv := site(t)
	m := New(srv.URL+"/a.md", render.Glamour{Style: "notty"})
	m = drive(t, m, m.Init())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return next.(Model), srv.URL
}

func TestLoadsStartPage(t *testing.T) {
	m, _ := open(t)
	out := m.View().Content
	if !strings.Contains(out, "Alpha") || !strings.Contains(out, "Page A") {
		t.Errorf("view:\n%s", out)
	}
}

func TestFollowLinkByNumberAndBack(t *testing.T) {
	m, base := open(t)
	m = press(t, m, "1", "enter")
	if out := m.View().Content; !strings.Contains(out, "Beta") || !strings.Contains(out, base+"/b.md") {
		t.Fatalf("after following 1:\n%s", out)
	}
	m = press(t, m, "b")
	if out := m.View().Content; !strings.Contains(out, "Alpha") {
		t.Errorf("after back:\n%s", out)
	}
	m = press(t, m, "f")
	if out := m.View().Content; !strings.Contains(out, "Beta") {
		t.Errorf("after forward:\n%s", out)
	}
}

func TestBackAtStartDoesNothing(t *testing.T) {
	m, _ := open(t)
	m = press(t, m, "b")
	if !strings.Contains(m.View().Content, "Alpha") {
		t.Error("lost the page")
	}
}

func TestNewPageClearsForwardHistory(t *testing.T) {
	m, _ := open(t)
	m = press(t, m, "1", "enter", "b", "2", "enter") // a -> b, back, a -> c
	m = press(t, m, "f")
	if !strings.Contains(m.View().Content, "Gamma") {
		t.Errorf("forward should do nothing after a new visit:\n%s", m.View().Content)
	}
}

func TestUnknownLinkNumberShowsStatus(t *testing.T) {
	m, _ := open(t)
	m = press(t, m, "9", "enter")
	out := m.View().Content
	if !strings.Contains(out, "no link 9") || !strings.Contains(out, "Alpha") {
		t.Errorf("view:\n%s", out)
	}
}

func TestEscapeAndBackspaceEditTheNumber(t *testing.T) {
	m, _ := open(t)
	m = press(t, m, "1", "2", "backspace")
	if m.num != "1" {
		t.Errorf("num %q", m.num)
	}
	m = press(t, m, "esc")
	if m.num != "" {
		t.Errorf("num %q", m.num)
	}
}

func TestFetchErrorKeepsThePageAndShowsTheError(t *testing.T) {
	srv := site(t)
	m := New(srv.URL+"/a.md", render.Glamour{Style: "notty"})
	m = drive(t, m, m.Init())
	m.view.Links[0].URL = srv.URL + "/missing.md"
	m = press(t, m, "1", "enter")
	out := m.View().Content
	if !strings.Contains(out, "404") || !strings.Contains(out, "Alpha") {
		t.Errorf("view:\n%s", out)
	}
}

func TestResizeRerendersAtNewWidth(t *testing.T) {
	m, _ := open(t)
	before := m.view.Text
	next, _ := m.Update(tea.WindowSizeMsg{Width: 30, Height: 10})
	m = next.(Model)
	if m.view.Text == before {
		t.Error("text was not rendered again at the new width")
	}
}

func TestQuit(t *testing.T) {
	m, _ := open(t)
	_, cmd := m.Update(key("q"))
	if cmd == nil {
		t.Fatal("no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("q did not quit")
	}
}

// longSite serves /long.md with many lines between two links.
func longSite(t *testing.T) *httptest.Server {
	t.Helper()
	body := "# Top\n\n[first](a.md)\n\n" + strings.Repeat("filler paragraph\n\n", 60) + "[far](b.md)\n\n[last](c.md)\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		if r.URL.Path == "/long.md" {
			fmt.Fprint(w, body)
			return
		}
		fmt.Fprintf(w, "# Visited %s\n", r.URL.Path)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func openLong(t *testing.T) Model {
	t.Helper()
	srv := longSite(t)
	m := New(srv.URL+"/long.md", render.Glamour{Style: "notty"})
	m = drive(t, m, m.Init())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	return next.(Model)
}

func TestTabFocusesFirstLinkAndHighlightsIt(t *testing.T) {
	m, _ := open(t)
	m = press(t, m, "tab")
	if m.focus != 1 {
		t.Fatalf("focus %d", m.focus)
	}
	if out := m.View().Content; !strings.Contains(out, "\x1b[7m[1]\x1b[27m") {
		t.Errorf("no highlight:\n%q", out)
	}
	m = press(t, m, "tab")
	if m.focus != 2 || strings.Contains(m.View().Content, "\x1b[7m[1]") {
		t.Errorf("focus %d, still highlighted 1", m.focus)
	}
}

func TestShiftTabMovesBackAndStopsAtTheFirstLink(t *testing.T) {
	m, _ := open(t)
	m = press(t, m, "tab", "tab", "shift+tab")
	if m.focus != 1 {
		t.Errorf("focus %d", m.focus)
	}
	m = press(t, m, "shift+tab")
	if m.focus != 1 || !strings.Contains(m.status, "first link") {
		t.Errorf("focus %d, status %q", m.focus, m.status)
	}
}

func TestTabStopsAtTheLastLink(t *testing.T) {
	m, _ := open(t)
	m = press(t, m, "tab", "tab", "tab")
	if m.focus != 2 || !strings.Contains(m.status, "last link") {
		t.Errorf("focus %d, status %q", m.focus, m.status)
	}
}

func TestEnterFollowsTheFocusedLink(t *testing.T) {
	m, _ := open(t)
	m = press(t, m, "tab", "tab", "enter")
	if out := m.View().Content; !strings.Contains(out, "Gamma") {
		t.Errorf("did not follow link 2:\n%s", out)
	}
	if m.focus != 0 {
		t.Errorf("focus %d after loading a page", m.focus)
	}
}

func TestTypedNumberWinsOverFocusOnEnter(t *testing.T) {
	m, _ := open(t)
	m = press(t, m, "tab", "2", "enter")
	if !strings.Contains(m.View().Content, "Gamma") {
		t.Errorf("typed 2 was not followed:\n%s", m.View().Content)
	}
}

func TestFocusScrollsTheLinkIntoView(t *testing.T) {
	m := openLong(t)
	m = press(t, m, "tab", "tab")
	line, ok := m.view.LineOf(2)
	if !ok {
		t.Fatal("link 2 not found")
	}
	top := m.vp.YOffset()
	if line < top || line >= top+m.vp.Height() {
		t.Errorf("link line %d outside the window %d..%d", line, top, top+m.vp.Height())
	}
	if !strings.Contains(m.View().Content, "\x1b[7m[2]") {
		t.Error("focused link not on screen")
	}
}

func TestTabStartsFromTheTopOfTheWindowAfterScrolling(t *testing.T) {
	m := openLong(t)
	line, _ := m.view.LineOf(2)
	m.vp.SetYOffset(line - 2)
	m = press(t, m, "tab")
	if m.focus != 2 {
		t.Errorf("focus %d, want the first link in view (2)", m.focus)
	}
}

func TestResizeKeepsTheFocus(t *testing.T) {
	m, _ := open(t)
	m = press(t, m, "tab", "tab")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m = next.(Model)
	if m.focus != 2 || !strings.Contains(m.View().Content, "\x1b[7m[2]") {
		t.Errorf("focus %d lost after resize", m.focus)
	}
}

func TestNextAndPreviousFollowFrontMatter(t *testing.T) {
	srv := site(t)
	m := New(srv.URL+"/s/1.md", render.Glamour{Style: "notty"})
	m = drive(t, m, m.Init())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	m = press(t, m, "n") // relative urlNext
	if !strings.Contains(m.View().Content, "Story Two") {
		t.Fatalf("after n:\n%s", m.View().Content)
	}
	m = press(t, m, "p") // relative urlPrev
	if !strings.Contains(m.View().Content, "Story One") {
		t.Fatalf("after p:\n%s", m.View().Content)
	}
	m = press(t, m, "n", "n") // absolute-path urlNext on page two
	if !strings.Contains(m.View().Content, "Gamma") {
		t.Fatalf("after n n:\n%s", m.View().Content)
	}
	// they are visits: back returns to where we were
	m = press(t, m, "b")
	if !strings.Contains(m.View().Content, "Story Two") {
		t.Errorf("back after next:\n%s", m.View().Content)
	}
}

func TestNextWithoutKeyShowsStatusAndKeepsPage(t *testing.T) {
	m, _ := open(t)
	m = press(t, m, "n")
	out := m.View().Content
	if !strings.Contains(out, "Alpha") || !strings.Contains(out, "no next page") {
		t.Errorf("view:\n%s", out)
	}
	m = press(t, m, "p")
	if !strings.Contains(m.View().Content, "no previous page") {
		t.Errorf("view:\n%s", m.View().Content)
	}
}

func TestPreviousNextDoNotInterruptATypedNumber(t *testing.T) {
	m, _ := open(t)
	m = press(t, m, "1", "n") // while typing a number, n is not a command
	if strings.Contains(m.View().Content, "no next page") {
		t.Errorf("n acted while a number was being typed:\n%s", m.View().Content)
	}
}
