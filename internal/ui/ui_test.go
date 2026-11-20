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
		"/a.md": "---\ntitle: Page A\n---\n# Alpha\n\nGo to [b](b.md) or [c](c.md).\n",
		"/b.md": "---\ntitle: Page B\n---\n# Beta\n\n[home](a.md)\n",
		"/c.md": "# Gamma\n",
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
