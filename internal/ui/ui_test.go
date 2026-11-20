package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/rsdoiel/cmarkbrowser/internal/debuglog"
	"github.com/rsdoiel/cmarkbrowser/internal/render"
	"github.com/rsdoiel/cmarkbrowser/internal/search"
	"github.com/rsdoiel/cmarkbrowser/internal/toc"
)

func site(t *testing.T) *httptest.Server {
	t.Helper()
	pages := map[string]string{
		"/a.md":      "---\ntitle: Page A\n---\n# Alpha\n\nGo to [b](b.md) or [c](c.md).\n",
		"/b.md":      "---\ntitle: Page B\n---\n# Beta\n\n[home](a.md)\n",
		"/c.md":      "# Gamma\n",
		"/search.md": searchPageText(),
		"/s/1.md":    "---\ntitle: One\nurlNext: 2.md\n---\n# Story One\n",
		"/s/2.md":    "---\ntitle: Two\nurlPrev: 1.md\nurlNext: /c.md\n---\n# Story Two\n",
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
	m = press(t, m, "]") // relative urlNext
	if !strings.Contains(m.View().Content, "Story Two") {
		t.Fatalf("after n:\n%s", m.View().Content)
	}
	m = press(t, m, "[") // relative urlPrev
	if !strings.Contains(m.View().Content, "Story One") {
		t.Fatalf("after p:\n%s", m.View().Content)
	}
	m = press(t, m, "]", "]") // absolute-path urlNext on page two
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
	m = press(t, m, "]")
	out := m.View().Content
	if !strings.Contains(out, "Alpha") || !strings.Contains(out, "no next page") {
		t.Errorf("view:\n%s", out)
	}
	m = press(t, m, "[")
	if !strings.Contains(m.View().Content, "no previous page") {
		t.Errorf("view:\n%s", m.View().Content)
	}
}

func TestPreviousNextDoNotInterruptATypedNumber(t *testing.T) {
	m, _ := open(t)
	m = press(t, m, "1", "]") // while typing a number, ] is not a command
	if strings.Contains(m.View().Content, "no next page") {
		t.Errorf("n acted while a number was being typed:\n%s", m.View().Content)
	}
}

// debugSession drives a short session with a debug logger and returns the events.
func debugSession(t *testing.T, keys ...string) []map[string]any {
	t.Helper()
	srv := site(t)
	var buf bytes.Buffer
	m := New(srv.URL+"/a.md", render.Glamour{Style: "notty"}).WithDebug(debuglog.New(&buf))
	m = drive(t, m, m.Init())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	press(t, m, keys...)
	var evs []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var e map[string]any
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("not JSON: %q", line)
		}
		evs = append(evs, e)
	}
	return evs
}

func eventsNamed(evs []map[string]any, name string) []map[string]any {
	var out []map[string]any
	for _, e := range evs {
		if e["event"] == name {
			out = append(out, e)
		}
	}
	return out
}

func TestDebugLogFollowsASession(t *testing.T) {
	evs := debugSession(t, "tab", "1", "enter", "]", "b")
	var keys []string
	for _, e := range eventsNamed(evs, "key") {
		keys = append(keys, e["key"].(string))
	}
	if strings.Join(keys, ",") != "tab,1,enter,],b" {
		t.Errorf("keys %v", keys)
	}
	starts := eventsNamed(evs, "load.start")
	var why []string
	for _, e := range starts {
		why = append(why, e["why"].(string))
	}
	if strings.Join(why, ",") != "start,link,history" {
		t.Errorf("load reasons %v (events %v)", why, starts)
	}
	for _, e := range eventsNamed(evs, "load.done") {
		if e["ok"] != true || e["final"] == "" {
			t.Errorf("load.done %v", e)
		}
	}
	if st := eventsNamed(evs, "status"); len(st) == 0 || !strings.Contains(fmt.Sprint(st), "no next page") {
		t.Errorf("status events %v", st)
	}
	if f := eventsNamed(evs, "focus"); len(f) != 1 || f[0]["n"] != float64(1) || !strings.HasSuffix(f[0]["url"].(string), "/b.md") {
		t.Errorf("focus events %v", f)
	}
	if r := eventsNamed(evs, "resize"); len(r) != 1 || r[0]["w"] != float64(80) {
		t.Errorf("resize events %v", r)
	}
}

func TestDebugLogRecordsAFailedLoad(t *testing.T) {
	srv := site(t)
	var buf bytes.Buffer
	m := New(srv.URL+"/a.md", render.Glamour{Style: "notty"}).WithDebug(debuglog.New(&buf))
	m = drive(t, m, m.Init())
	m = press(t, m, "9", "9", "enter") // no such link
	if !strings.Contains(buf.String(), `"event":"status"`) || !strings.Contains(buf.String(), "no link 99") {
		t.Errorf("log:\n%s", buf.String())
	}
	// a fetch error
	m2 := New(srv.URL+"/missing.md", render.Glamour{Style: "notty"}).WithDebug(debuglog.New(&buf))
	drive(t, m2, m2.Init())
	done := false
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.Contains(line, `"event":"load.done"`) && strings.Contains(line, `"ok":false`) && strings.Contains(line, `"err":`) {
			done = true
		}
	}
	if !done {
		t.Errorf("no failed load.done:\n%s", buf.String())
	}
}

func TestDebugLogRecordsScrollAndQuit(t *testing.T) {
	evs := debugSession(t, "j", "q")
	if len(eventsNamed(evs, "scroll")) == 0 {
		t.Errorf("no scroll event: %v", evs)
	}
	if len(eventsNamed(evs, "quit")) != 1 {
		t.Errorf("no quit event: %v", evs)
	}
}

func TestDebugLogHoldsNoPageContentAndTypedTextIsOnlyKeys(t *testing.T) {
	srv := site(t)
	var buf bytes.Buffer
	m := New(srv.URL+"/a.md", render.Glamour{Style: "notty"}).WithDebug(debuglog.New(&buf))
	m = drive(t, m, m.Init())
	press(t, m, "1", "enter")
	if strings.Contains(buf.String(), "Alpha") || strings.Contains(buf.String(), "Beta") {
		t.Errorf("log holds page content:\n%s", buf.String())
	}
}

func TestNoDebugLoggerIsFine(t *testing.T) {
	m, _ := open(t) // no WithDebug
	press(t, m, "tab", "1", "enter", "b", "q")
}

// longPage is a page taller than the screen with a "needle" near the end, a
// mixed-case Yap State twice, and a phrase for smart-case.
func searchPageText() string {
	var sb strings.Builder
	sb.WriteString("---\ntitle: Long\n---\n# Long page\n\n")
	for i := 1; i <= 60; i++ {
		switch i {
		case 5:
			sb.WriteString("The Yap State news item.\n\n")
		case 30:
			sb.WriteString("Another yap state mention here.\n\n")
		case 55:
			sb.WriteString("A needle in the haystack, and a second needle.\n\n")
		default:
			fmt.Fprintf(&sb, "Filler paragraph number %d.\n\n", i)
		}
	}
	return sb.String()
}

func typeQuery(t *testing.T, m Model, q string) Model {
	t.Helper()
	keys := []string{"/"}
	for _, r := range q {
		keys = append(keys, string(r))
	}
	return press(t, m, keys...)
}

func openSearch(t *testing.T) Model {
	t.Helper()
	srv := site(t)
	m := New(srv.URL+"/search.md", render.Glamour{Style: "notty"})
	m = drive(t, m, m.Init())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return next.(Model)
}

func TestSlashOpensThePromptAndKeysGoToIt(t *testing.T) {
	m := openSearch(t)
	m = typeQuery(t, m, "q1")
	out := m.View().Content
	if !strings.Contains(out, "/q1") {
		t.Errorf("prompt not shown:\n%s", out)
	}
	// q did not quit and digits did not become a link number
	if m.num != "" {
		t.Errorf("keys leaked out of the prompt: num %q", m.num)
	}
	_, cmd := m.Update(key("q"))
	if cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Error("q quit the browser while a search was being typed")
		}
	}
}

func TestSearchFindsAndScrollsToTheFirstMatch(t *testing.T) {
	m := openSearch(t)
	m = typeQuery(t, m, "needle")
	m = press(t, m, "enter")
	out := m.View().Content
	if !strings.Contains(out, "needle") {
		t.Fatalf("match not scrolled into view:\n%s", out)
	}
	if !strings.Contains(out, search.CurrentOn) {
		t.Errorf("current match not highlighted:\n%q", out)
	}
	if !strings.Contains(out, "1 of 2") || !strings.Contains(out, "needle") {
		t.Errorf("footer should say where we are:\n%s", out)
	}
}

func TestNAndShiftNMoveThroughMatchesAndWrap(t *testing.T) {
	m := openSearch(t)
	m = typeQuery(t, m, "yap state")
	m = press(t, m, "enter")
	if !strings.Contains(m.View().Content, "1 of 2") {
		t.Fatalf("after search:\n%s", m.View().Content)
	}
	m = press(t, m, "n")
	if out := m.View().Content; !strings.Contains(out, "2 of 2") || !strings.Contains(search.StripANSI(out), "yap state mention") {
		t.Errorf("after n:\n%s", out)
	}
	m = press(t, m, "n") // past the last: wraps to the first
	if out := m.View().Content; !strings.Contains(out, "1 of 2") || !strings.Contains(out, "wrapped") {
		t.Errorf("after wrapping n:\n%s", out)
	}
	m = press(t, m, "N") // back past the first: wraps to the last
	if out := m.View().Content; !strings.Contains(out, "2 of 2") || !strings.Contains(out, "wrapped") {
		t.Errorf("after wrapping N:\n%s", out)
	}
}

func TestSearchIsSmartCaseAndAcceptsGoRegexp(t *testing.T) {
	m := openSearch(t)
	m = typeQuery(t, m, `yap (state|sate)`)
	m = press(t, m, "enter")
	if !strings.Contains(m.View().Content, "of 2") {
		t.Errorf("regexp with alternation: %s", m.View().Content)
	}
	m = press(t, m, "esc")
	m = typeQuery(t, m, `YAP`)
	m = press(t, m, "enter")
	if !strings.Contains(m.View().Content, "no match") {
		t.Errorf("an upper-case pattern is case sensitive: %s", m.View().Content)
	}
}

func TestNoMatchAndBadRegexpAreReported(t *testing.T) {
	m := openSearch(t)
	m = typeQuery(t, m, "zzzzqq")
	m = press(t, m, "enter")
	if !strings.Contains(m.View().Content, "no match for /zzzzqq/") {
		t.Errorf("no match: %s", m.View().Content)
	}
	m = press(t, m, "esc")
	m = typeQuery(t, m, `(unclosed`)
	m = press(t, m, "enter")
	if !strings.Contains(m.View().Content, "invalid regexp") {
		t.Errorf("bad regexp: %s", m.View().Content)
	}
	// a bad pattern does not throw away the search before it
	m = press(t, m, "esc")
	m = typeQuery(t, m, "needle")
	m = press(t, m, "enter")
	m = typeQuery(t, m, `(unclosed`)
	m = press(t, m, "enter")
	m = press(t, m, "n")
	if !strings.Contains(m.View().Content, "of 2") {
		t.Errorf("the earlier search was lost: %s", m.View().Content)
	}
}

func TestNWithoutASearchSaysSo(t *testing.T) {
	m := openSearch(t)
	m = press(t, m, "n")
	if !strings.Contains(m.View().Content, "no search") {
		t.Errorf("view:\n%s", m.View().Content)
	}
}

func TestEscapeCancelsThePromptAndThenClearsTheSearch(t *testing.T) {
	m := openSearch(t)
	m = typeQuery(t, m, "need")
	m = press(t, m, "esc") // cancels the prompt: nothing is searched
	if len(m.matches) != 0 || m.prompt {
		t.Errorf("prompt not cancelled: prompt %v matches %d", m.prompt, len(m.matches))
	}
	m = typeQuery(t, m, "needle")
	m = press(t, m, "enter")
	if len(m.matches) == 0 {
		t.Fatal("no matches")
	}
	m = press(t, m, "esc") // clears the highlight
	if len(m.matches) != 0 || strings.Contains(m.View().Content, search.CurrentOn) {
		t.Errorf("search not cleared")
	}
}

func TestEnterOnAnEmptyPromptRepeatsTheLastSearch(t *testing.T) {
	m := openSearch(t)
	m = typeQuery(t, m, "needle")
	m = press(t, m, "enter")
	m = press(t, m, "esc")
	m = press(t, m, "/", "enter")
	if len(m.matches) != 2 {
		t.Errorf("last search not repeated: %d matches", len(m.matches))
	}
}

func TestPromptEditing(t *testing.T) {
	m := openSearch(t)
	m = press(t, m, "/", "a", "b", "c", "backspace")
	if m.query != "ab" {
		t.Errorf("query %q", m.query)
	}
	m = press(t, m, "backspace", "backspace")
	if !m.prompt {
		// one more backspace on an empty prompt leaves it
	}
	m = press(t, m, "backspace")
	if m.prompt {
		t.Error("backspace on an empty prompt should leave it")
	}
}

func TestANewPageClearsTheSearch(t *testing.T) {
	srv := site(t)
	m := New(srv.URL+"/a.md", render.Glamour{Style: "notty"})
	m = drive(t, m, m.Init())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	m = typeQuery(t, m, "alpha")
	m = press(t, m, "enter")
	if len(m.matches) == 0 {
		t.Fatal("no match for alpha on page a")
	}
	m = press(t, m, "1", "enter")
	if len(m.matches) != 0 || m.prompt {
		t.Error("the search of the old page is still active")
	}
}

func TestResizeKeepsTheSearchAndCurrentMatch(t *testing.T) {
	m := openSearch(t)
	m = typeQuery(t, m, "needle")
	m = press(t, m, "enter", "n")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 50, Height: 24})
	m = next.(Model)
	if len(m.matches) != 2 || m.cur != 1 {
		t.Errorf("after resize: %d matches, current %d", len(m.matches), m.cur)
	}
}

func TestSearchIsLogged(t *testing.T) {
	srv := site(t)
	var buf bytes.Buffer
	m := New(srv.URL+"/search.md", render.Glamour{Style: "notty"}).WithDebug(debuglog.New(&buf))
	m = drive(t, m, m.Init())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	press(t, m, "/", "n", "e", "e", "d", "l", "e", "enter", "n")
	var search, match int
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var e map[string]any
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("not JSON: %q", line)
		}
		switch e["event"] {
		case "search":
			search++
			if e["pattern"] != "needle" || e["matches"] != float64(2) {
				t.Errorf("search event %v", e)
			}
		case "match":
			match++
		}
	}
	if search != 1 || match != 2 {
		t.Errorf("search events %d, match events %d", search, match)
	}
}

func TestTheTitleLineMarksAReaderView(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a.md":
			w.Header().Set("Content-Type", "text/markdown")
			fmt.Fprint(w, "# Start\n\n[leave](/outside)\n")
		case "/outside":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><head><title>Outside</title></head><body><article><h1>Outside</h1><p>`+strings.Repeat("HTML words. ", 30)+`</p></article></body></html>`)
		}
	}))
	defer srv.Close()
	m := New(srv.URL+"/a.md", render.Glamour{Style: "notty"})
	m = drive(t, m, m.Init())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m = next.(Model)
	first := strings.SplitN(m.View().Content, "\n", 2)[0]
	if strings.Contains(first, "reader view") {
		t.Errorf("a Markdown page was marked as a reader view: %q", first)
	}
	m = press(t, m, "1", "enter")
	first = strings.SplitN(m.View().Content, "\n", 2)[0]
	if !strings.Contains(first, "Outside") || !strings.Contains(first, "reader view") {
		t.Errorf("title line %q: want the title and a reader-view marker", first)
	}
	if !strings.Contains(m.View().Content, "HTML words") {
		t.Errorf("page not shown:\n%s", m.View().Content)
	}
}

// bookmarkSite serves a Markdown page and an HTML page with no Markdown copy.
func bookmarkSite(t *testing.T, secretPath string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/page.md":
			w.Header().Set("Content-Type", "text/markdown")
			fmt.Fprint(w, "---\ntitle: A Good Page\n---\n# Good\n\nbody\n")
		case "/other.md":
			w.Header().Set("Content-Type", "text/markdown")
			fmt.Fprint(w, "# Other page\n")
		case "/html":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><head><title>An HTML Page</title></head><body><article><h1>An HTML Page</h1><p>`+strings.Repeat("Words here. ", 30)+`</p></article></body></html>`)
		case "/evil.md":
			w.Header().Set("Content-Type", "text/markdown")
			fmt.Fprintf(w, "# Evil\n\n[read me](file://%s)\n[and me](%s)\n", secretPath, secretPath)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func openBM(t *testing.T, start string, bm string) Model {
	t.Helper()
	m := New(start, render.Glamour{Style: "notty"}).WithBookmarks(bm)
	m = drive(t, m, m.Init())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	return next.(Model)
}

func TestMBookmarksThePageAndNotTwice(t *testing.T) {
	srv := bookmarkSite(t, "")
	bm := filepath.Join(t.TempDir(), "bookmarks.md")
	m := openBM(t, srv.URL+"/page.md", bm)
	m = press(t, m, "m")
	if !strings.Contains(m.View().Content, "bookmarked") {
		t.Errorf("status:\n%s", m.View().Content)
	}
	b, err := os.ReadFile(bm)
	if err != nil || !strings.Contains(string(b), "[A Good Page]("+srv.URL+"/page.md)") {
		t.Errorf("file %q err %v", b, err)
	}
	m = press(t, m, "m")
	if !strings.Contains(m.View().Content, "already bookmarked") {
		t.Errorf("status:\n%s", m.View().Content)
	}
	if n := strings.Count(mustRead(t, bm), "page.md"); n != 1 {
		t.Errorf("%d entries", n)
	}
}

func mustRead(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestBOpensTheBookmarksAndALinkLeadsOutAndBackReturns(t *testing.T) {
	srv := bookmarkSite(t, "")
	bm := filepath.Join(t.TempDir(), "bookmarks.md")
	m := openBM(t, srv.URL+"/page.md", bm)
	m = press(t, m, "m", "B")
	out := m.View().Content
	if !strings.Contains(out, "Bookmarks") || !strings.Contains(out, "A Good Page") {
		t.Fatalf("bookmarks page:\n%s", out)
	}
	if first := strings.SplitN(out, "\n", 2)[0]; !strings.Contains(first, "bookmarks.md") {
		t.Errorf("title line %q should name the file", first)
	}
	m = press(t, m, "1", "enter") // follow the bookmark
	if !strings.Contains(m.View().Content, "Good") {
		t.Fatalf("did not follow the bookmark:\n%s", m.View().Content)
	}
	m = press(t, m, "b") // back to the local bookmarks page
	if !strings.Contains(m.View().Content, "A Good Page") || strings.Contains(m.View().Content, "not an http") {
		t.Errorf("back to the bookmarks failed:\n%s", m.View().Content)
	}
}

func TestBWithNoBookmarksSaysSo(t *testing.T) {
	srv := bookmarkSite(t, "")
	m := openBM(t, srv.URL+"/page.md", filepath.Join(t.TempDir(), "none.md"))
	m = press(t, m, "B")
	if !strings.Contains(m.View().Content, "no bookmarks") || !strings.Contains(m.View().Content, "Good") {
		t.Errorf("view:\n%s", m.View().Content)
	}
}

func TestMBookmarksAnHTMLPageByItsAddress(t *testing.T) {
	srv := bookmarkSite(t, "")
	bm := filepath.Join(t.TempDir(), "b.md")
	m := openBM(t, srv.URL+"/html", bm)
	m = press(t, m, "m")
	b := mustRead(t, bm)
	if !strings.Contains(b, "[An HTML Page]("+srv.URL+"/html)") {
		t.Errorf("file:\n%s", b)
	}
}

func TestAWriteFailureIsShownNotFatal(t *testing.T) {
	srv := bookmarkSite(t, "")
	m := openBM(t, srv.URL+"/page.md", filepath.Join(t.TempDir(), "no", "such", "dir", "b.md"))
	m = press(t, m, "m")
	if out := m.View().Content; !strings.Contains(out, "bookmark failed") || !strings.Contains(out, "Good") {
		t.Errorf("view:\n%s", out)
	}
}

// A page from the network cannot make the browser read a local file.
func TestARemotePageCannotOpenALocalFile(t *testing.T) {
	secret := filepath.Join(t.TempDir(), "secret.md")
	os.WriteFile(secret, []byte("# TOP SECRET CONTENT\n"), 0o644)
	srv := bookmarkSite(t, secret)
	m := openBM(t, srv.URL+"/evil.md", filepath.Join(t.TempDir(), "b.md"))
	for _, n := range []string{"1", "2"} {
		m = press(t, m, n, "enter")
		out := m.View().Content
		if strings.Contains(out, "TOP SECRET") {
			t.Fatalf("link %s read a local file:\n%s", n, out)
		}
		// link 1 is a file: address and is refused, so the page stays; link 2 is a
		// bare path, which is an ordinary address on the same server
		if n == "1" && (!strings.Contains(out, "Evil") || !strings.Contains(out, "local files")) {
			t.Errorf("a file: link should be refused with a message and the page kept:\n%s", out)
		}
	}
}

func TestLocalFilesAreNotBookmarked(t *testing.T) {
	bm := filepath.Join(t.TempDir(), "bookmarks.md")
	srv := bookmarkSite(t, "")
	m := openBM(t, srv.URL+"/page.md", bm)
	m = press(t, m, "m", "B") // now showing the local bookmarks file
	m = press(t, m, "m")
	if !strings.Contains(m.View().Content, "local file") {
		t.Errorf("view:\n%s", m.View().Content)
	}
	if strings.Count(mustRead(t, bm), "](") != 1 {
		t.Errorf("a local file was bookmarked:\n%s", mustRead(t, bm))
	}
}

func TestAStartFileIsLocalAndHistoryCanReturnToIt(t *testing.T) {
	srv := bookmarkSite(t, "")
	notes := filepath.Join(t.TempDir(), "notes.md")
	os.WriteFile(notes, []byte("---\ntitle: Notes\n---\n# My notes\n\n[the page]("+srv.URL+"/other.md)\n"), 0o644)
	m := New(notes, render.Glamour{Style: "notty"}).WithLocalStart().WithBookmarks(filepath.Join(t.TempDir(), "b.md"))
	m = drive(t, m, m.Init())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m = next.(Model)
	if !strings.Contains(m.View().Content, "My notes") {
		t.Fatalf("start file not shown:\n%s", m.View().Content)
	}
	m = press(t, m, "1", "enter")
	if !strings.Contains(m.View().Content, "Other page") {
		t.Fatalf("link not followed:\n%s", m.View().Content)
	}
	m = press(t, m, "b")
	if !strings.Contains(m.View().Content, "My notes") {
		t.Errorf("did not return to the start file:\n%s", m.View().Content)
	}
}

func TestBookmarkIsLogged(t *testing.T) {
	srv := bookmarkSite(t, "")
	var buf bytes.Buffer
	m := New(srv.URL+"/page.md", render.Glamour{Style: "notty"}).WithBookmarks(filepath.Join(t.TempDir(), "b.md")).WithDebug(debuglog.New(&buf))
	m = drive(t, m, m.Init())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	press(t, next.(Model), "m")
	if !strings.Contains(buf.String(), `"event":"bookmark"`) || !strings.Contains(buf.String(), `"added":true`) {
		t.Errorf("log:\n%s", buf.String())
	}
}

// A folder or HTML address answered with Markdown from its own file: the title
// line, the bookmark and history all name the Markdown.
func TestContentLocationIsWhatIsShownBookmarkedAndRevisited(t *testing.T) {
	hits := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[r.URL.Path]++
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		switch r.URL.Path {
		case "/start.md":
			fmt.Fprint(w, "# Start\n\n[go](/dir/)\n")
		case "/dir/":
			w.Header().Set("Content-Location", "/dir/index.md")
			fmt.Fprint(w, "# Folder page\n\n[sibling](other.md)\n")
		case "/dir/index.md":
			fmt.Fprint(w, "# Folder page\n\n[sibling](other.md)\n")
		case "/dir/other.md":
			fmt.Fprint(w, "# Sibling\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	bm := filepath.Join(t.TempDir(), "bookmarks.md")
	m := New(srv.URL+"/start.md", render.Glamour{Style: "notty"}).WithBookmarks(bm)
	m = drive(t, m, m.Init())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	m = next.(Model)
	m = press(t, m, "1", "enter") // the folder address
	if first := strings.SplitN(m.View().Content, "\n", 2)[0]; !strings.Contains(first, srv.URL+"/dir/index.md") {
		t.Errorf("title line %q should name the Markdown file", first)
	}
	m = press(t, m, "m")
	if b := mustRead(t, bm); !strings.Contains(b, "("+srv.URL+"/dir/index.md)") {
		t.Errorf("the bookmark should name the Markdown file:\n%s", b)
	}
	m = press(t, m, "1", "enter") // a relative link in it: /dir/other.md, not /other.md
	if !strings.Contains(m.View().Content, "Sibling") {
		t.Fatalf("relative link did not resolve against the folder:\n%s", m.View().Content)
	}
	m = press(t, m, "b") // back: the Markdown address is fetched, not the folder's
	if !strings.Contains(m.View().Content, "Folder page") {
		t.Errorf("back:\n%s", m.View().Content)
	}
	if hits["/dir/index.md"] == 0 {
		t.Errorf("history should return to /dir/index.md; hits %v", hits)
	}
}

// tocSite serves pages with no urlNext or urlPrev of their own, and the table
// of contents that lists them: Pages a, b, c, and Posts newest first.
func tocUISite(t *testing.T) (*httptest.Server, map[string]int) {
	t.Helper()
	hits := map[string]int{}
	pages := map[string]string{
		"/p/a.md":      "# Page A\n",
		"/p/b.md":      "# Page B\n",
		"/p/c.md":      "# Page C\n",
		"/p/keyed.md":  "---\nurlNext: c.md\n---\n# Keyed\n",
		"/blog/new.md": "# New post\n",
		"/blog/mid.md": "# Mid post\n",
		"/blog/old.md": "# Old post\n",
		"/lone.md":     "# Lone page\n",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[r.URL.Path]++
		if r.URL.Path == "/.well-known/toc.opml" {
			w.Header().Set("Content-Type", "text/x-opml; charset=utf-8")
			fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><opml version="2.0"><head/><body>
			<outline text="Pages">
				<outline text="A" url="http://%[1]s/p/a.md"/><outline text="B" url="http://%[1]s/p/b.md"/><outline text="C" url="http://%[1]s/p/c.md"/><outline text="Keyed" url="http://%[1]s/p/keyed.md"/></outline>
			<outline text="Posts"><outline text="2026" type="calendarYear">
				<outline text="New" url="http://%[1]s/blog/new.md"/><outline text="Mid" url="http://%[1]s/blog/mid.md"/><outline text="Old" url="http://%[1]s/blog/old.md"/></outline></outline>
			</body></opml>`, r.Host)
			return
		}
		body, ok := pages[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/markdown")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, hits
}

func openTOC(t *testing.T, srv *httptest.Server, path string) Model {
	t.Helper()
	m := New(srv.URL+path, render.Glamour{Style: "notty"})
	m = drive(t, m, m.Init())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	return next.(Model)
}

func TestNextAndPreviousUseTheTableOfContentsWhenThePageNamesNone(t *testing.T) {
	srv, _ := tocUISite(t)
	m := openTOC(t, srv, "/p/b.md")
	m = press(t, m, "]")
	if !strings.Contains(m.View().Content, "Page C") {
		t.Fatalf("after ]:\n%s", m.View().Content)
	}
	m = press(t, m, "[", "[")
	if !strings.Contains(m.View().Content, "Page A") {
		t.Errorf("after [ [:\n%s", m.View().Content)
	}
	// they are visits: back returns to the page before
	m = press(t, m, "b")
	if !strings.Contains(m.View().Content, "Page B") {
		t.Errorf("back:\n%s", m.View().Content)
	}
}

// In a list with calendar outlines the lines run newest first, so the
// previous page is the older one, the line below (antennaApp DR-0013 5).
func TestInPostsPreviousIsOlderAndNextIsNewer(t *testing.T) {
	srv, _ := tocUISite(t)
	m := openTOC(t, srv, "/blog/mid.md")
	m = press(t, m, "[")
	if !strings.Contains(m.View().Content, "Old post") {
		t.Errorf("previous of the middle post should be the old one:\n%s", m.View().Content)
	}
	m = press(t, m, "]", "]")
	if !strings.Contains(m.View().Content, "New post") {
		t.Errorf("two steps newer:\n%s", m.View().Content)
	}
}

func TestAPagesOwnKeysWinAndNoTableOfContentsIsFetchedForThem(t *testing.T) {
	srv, hits := tocUISite(t)
	m := openTOC(t, srv, "/p/keyed.md")
	m = press(t, m, "]")
	if !strings.Contains(m.View().Content, "Page C") {
		t.Fatalf("urlNext not followed:\n%s", m.View().Content)
	}
	if hits["/.well-known/toc.opml"] != 0 {
		t.Errorf("the table of contents was fetched although the page named its next page: %v", hits)
	}
}

// The table of contents is read when a key needs it, not when a page loads.
func TestNothingIsFetchedUntilAKeyNeedsIt(t *testing.T) {
	srv, hits := tocUISite(t)
	m := openTOC(t, srv, "/p/b.md")
	press(t, m, "j", "tab")
	if hits["/.well-known/toc.opml"] != 0 {
		t.Errorf("the table of contents was fetched without being needed: %v", hits)
	}
}

func TestTheEndsOfAListAndUnlistedPagesSaySoAndStay(t *testing.T) {
	srv, _ := tocUISite(t)
	m := openTOC(t, srv, "/p/a.md")
	m = press(t, m, "[")
	if out := m.View().Content; !strings.Contains(out, "no previous page") || !strings.Contains(out, "Page A") {
		t.Errorf("first page, [:\n%s", out)
	}
	m = openTOC(t, srv, "/lone.md") // a page the outline does not list
	m = press(t, m, "]")
	if out := m.View().Content; !strings.Contains(out, "no next page") || !strings.Contains(out, "table of contents") || !strings.Contains(out, "Lone page") {
		t.Errorf("unlisted page, ]:\n%s", out)
	}
}

func TestASiteWithNoTableOfContentsSaysSo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/x.md" {
			w.Header().Set("Content-Type", "text/markdown")
			fmt.Fprint(w, "# Only page\n")
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	m := openTOC(t, srv, "/x.md")
	m = press(t, m, "]")
	if out := m.View().Content; !strings.Contains(out, "no next page") || !strings.Contains(out, "Only page") {
		t.Errorf("view:\n%s", out)
	}
}

// A reader view or a local file is not a page of the Markdown web: no lookup.
func TestNoLookupForReaderViewsOrLocalFiles(t *testing.T) {
	hits := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[r.URL.Path]++
		switch r.URL.Path {
		case "/h":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><head><title>H</title></head><body><article><h1>H</h1><p>`+strings.Repeat("Words here. ", 30)+`</p></article></body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	m := openTOC(t, srv, "/h")
	press(t, m, "]")
	if hits["/.well-known/toc.opml"] != 0 {
		t.Errorf("a reader view looked for a table of contents: %v", hits)
	}
	notes := filepath.Join(t.TempDir(), "n.md")
	os.WriteFile(notes, []byte("# Notes\n"), 0o644)
	lm := New(notes, render.Glamour{Style: "notty"}).WithLocalStart()
	lm = drive(t, lm, lm.Init())
	press(t, lm, "]") // must not panic or fetch
}

// An answer that arrives after the person has moved on is dropped.
func TestAStaleAnswerIsDropped(t *testing.T) {
	srv, _ := tocUISite(t)
	m := openTOC(t, srv, "/p/b.md")
	stale := tocMsgFor("http://elsewhere.example/other.md", 1, srv.URL+"/p/c.md")
	next, cmd := m.Update(stale)
	if cmd != nil {
		t.Error("a stale answer started a load")
	}
	if !strings.Contains(next.(Model).View().Content, "Page B") {
		t.Error("the page changed")
	}
}

func TestTheLookupIsLogged(t *testing.T) {
	srv, _ := tocUISite(t)
	var buf bytes.Buffer
	m := New(srv.URL+"/p/b.md", render.Glamour{Style: "notty"}).WithDebug(debuglog.New(&buf))
	m = drive(t, m, m.Init())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	press(t, next.(Model), "]")
	for _, want := range []string{`"event":"toc.fetch"`, `"event":"toc.lookup"`, `"found":true`, `"why":"next"`} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("log lacks %s", want)
		}
	}
}

// tocMsgFor builds the message a finished lookup delivers, for the page given.
func tocMsgFor(page string, dir int, next string) tea.Msg {
	return tocMsg{page: page, dir: dir, res: toc.Result{Found: true, Next: next}}
}
