package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/rsdoiel/cmarkbrowser/internal/debuglog"
	"github.com/rsdoiel/cmarkbrowser/internal/render"
	"github.com/rsdoiel/cmarkbrowser/internal/search"
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
