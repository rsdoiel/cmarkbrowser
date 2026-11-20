package search

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

// styled builds text the way the renderer does: ANSI colours, an OSC 8
// hyperlink, and a zero-width sentinel that must never be counted.
const styled = "  \x1b[1;38;5;228mHealthy Eats\x1b[0m\n" +
	"\n" +
	"  The Yap⁠ State \x1b]8;id=1;https://x.example/\x1b\\news\x1b]8;;\x1b\\ item\n" +
	"  continues here and Yap State again.\n" +
	"\n" +
	"  end\n"

func mustFind(t *testing.T, ix *Index, pattern string) []Match {
	t.Helper()
	re, err := Compile(pattern)
	if err != nil {
		t.Fatalf("Compile(%q): %s", pattern, err)
	}
	return ix.Find(re)
}

func TestStripANSIRemovesColoursLinksAndTheSentinel(t *testing.T) {
	got := StripANSI(styled)
	if strings.ContainsAny(got, "\x1b⁠") || strings.Contains(got, "https://x.example") {
		t.Errorf("not stripped: %q", got)
	}
	if !strings.Contains(got, "The Yap State news item") {
		t.Errorf("text lost: %q", got)
	}
}

func TestFindSimpleMatchesAndTheirRanges(t *testing.T) {
	ix := NewIndex(styled)
	ms := mustFind(t, ix, "healthy")
	if len(ms) != 1 {
		t.Fatalf("matches %v", ms)
	}
	r := ms[0].Ranges
	if len(r) != 1 || r[0].Line != 0 || r[0].Start != 2 || r[0].Stop != 9 {
		t.Errorf("range %v, want line 0 columns 2 to 9", r)
	}
}

func TestFindAcrossAWrappedLine(t *testing.T) {
	ix := NewIndex(styled)
	// "item continues": the words are on two rendered lines
	ms := mustFind(t, ix, `item continues`)
	if len(ms) != 1 || len(ms[0].Ranges) != 2 {
		t.Fatalf("want one match over two lines, got %v", ms)
	}
	a, b := ms[0].Ranges[0], ms[0].Ranges[1]
	if a.Line != 2 || b.Line != 3 || b.Start != 2 {
		t.Errorf("ranges %v", ms[0].Ranges)
	}
}

func TestParagraphBreaksStopAMatch(t *testing.T) {
	ix := NewIndex(styled)
	if ms := mustFind(t, ix, `Healthy Eats The Yap`); len(ms) != 0 {
		t.Errorf("matched across a blank line: %v", ms)
	}
}

func TestMultipleMatchesInOrder(t *testing.T) {
	ix := NewIndex(styled)
	ms := mustFind(t, ix, `yap state`)
	if len(ms) != 2 {
		t.Fatalf("want 2, got %d: %v", len(ms), ms)
	}
	if ms[0].Ranges[0].Line != 2 || ms[1].Ranges[0].Line != 3 {
		t.Errorf("order %v", ms)
	}
}

func TestGoRegexpFeatures(t *testing.T) {
	ix := NewIndex(styled)
	for pattern, want := range map[string]int{
		`Yap (State|Sate)`:  2,
		`\bnews\b`:          1,
		`^\s*end$`:          0, // flow text is not line anchored; see the doc on Find
		`h.{4}hy`:           1,
		`(?-i)YAP`:          0, // explicit case-sensitive
		`(?i)YAP`:           2,
		`[A-Z]\w+ [A-Z]\w+`: 3,
	} {
		re, err := Compile(pattern)
		if err != nil {
			t.Fatalf("%q: %s", pattern, err)
		}
		if got := len(ix.Find(re)); got != want {
			t.Errorf("%q: %d matches, want %d", pattern, got, want)
		}
	}
}

func TestSmartCase(t *testing.T) {
	ix := NewIndex(styled)
	if n := len(mustFind(t, ix, "yap")); n != 2 {
		t.Errorf("all-lowercase pattern should ignore case: %d", n)
	}
	if n := len(mustFind(t, ix, "Yap")); n != 2 {
		t.Errorf("Yap: %d", n)
	}
	if n := len(mustFind(t, ix, "YAP")); n != 0 {
		t.Errorf("an uppercase letter makes the search case sensitive: %d", n)
	}
	if n := len(mustFind(t, ix, "yAp")); n != 0 {
		t.Errorf("yAp: %d", n)
	}
}

func TestCompileErrors(t *testing.T) {
	if _, err := Compile(`(unclosed`); err == nil {
		t.Error("bad pattern accepted")
	}
	if _, err := Compile(``); err == nil {
		t.Error("empty pattern accepted")
	}
}

func TestEmptyMatchesAreIgnored(t *testing.T) {
	ix := NewIndex(styled)
	if ms := mustFind(t, ix, `x*`); len(ms) != 0 {
		t.Errorf("empty matches reported: %d", len(ms))
	}
}

func TestMultibyteTextHasRuneColumns(t *testing.T) {
	ix := NewIndex("  日本語のページ Yap\n")
	ms := mustFind(t, ix, "yap")
	if len(ms) != 1 || ms[0].Ranges[0].Start != 10 || ms[0].Ranges[0].Stop != 13 {
		t.Errorf("ranges %v, want columns 10 to 13 counted in runes", ms)
	}
}

// Highlighting must leave the visible text exactly as it was.
func TestHighlightKeepsTheVisibleText(t *testing.T) {
	ix := NewIndex(styled)
	ms := mustFind(t, ix, `yap state`)
	out := Highlight(styled, ms, 0)
	if StripANSI(out) != StripANSI(styled) {
		t.Errorf("visible text changed:\n%q\n%q", StripANSI(out), StripANSI(styled))
	}
	if !strings.Contains(out, CurrentOn) || !strings.Contains(out, MatchOn) {
		t.Errorf("no highlight codes: %q", out)
	}
}

func TestHighlightWrapsExactlyTheMatchedColumns(t *testing.T) {
	src := "  one two three\n"
	ix := NewIndex(src)
	ms := mustFind(t, ix, "two")
	out := Highlight(src, ms, -1) // no current match: all use the ordinary style
	want := "  one " + MatchOn + "two" + MatchOff + " three\n"
	if out != want {
		t.Errorf("got %q want %q", out, want)
	}
	out = Highlight(src, ms, 0)
	want = "  one " + CurrentOn + "two" + CurrentOff + " three\n"
	if out != want {
		t.Errorf("current: got %q want %q", out, want)
	}
}

func TestHighlightSkipsEscapesWhenCountingColumns(t *testing.T) {
	src := "\x1b[1mab\x1b[0m cd \x1b]8;;https://x\x1b\\ef\x1b]8;;\x1b\\\n"
	ix := NewIndex(src)
	out := Highlight(src, mustFind(t, ix, "ef"), -1)
	want := "\x1b[1mab\x1b[0m cd \x1b]8;;https://x\x1b\\" + MatchOn + "ef" + MatchOff + "\x1b]8;;\x1b\\\n"
	if out != want {
		t.Errorf("got %q want %q", out, want)
	}
}

func TestHighlightAMatchOverTwoLines(t *testing.T) {
	ix := NewIndex(styled)
	ms := mustFind(t, ix, `item continues`)
	out := Highlight(styled, ms, 0)
	if strings.Count(out, CurrentOn) != 2 || strings.Count(out, CurrentOff) != 2 {
		t.Errorf("want the current style on both lines: %q", out)
	}
	if StripANSI(out) != StripANSI(styled) {
		t.Error("visible text changed")
	}
}

func TestHighlightWithNoMatchesIsTheInput(t *testing.T) {
	if Highlight(styled, nil, 0) != styled {
		t.Error("changed text with no matches")
	}
}

// Whatever the matches are, StripANSI(Highlight(x)) == StripANSI(x).
func TestHighlightNeverChangesVisibleTextOnRandomPatterns(t *testing.T) {
	ix := NewIndex(styled)
	for _, p := range []string{`e`, `\w+`, `[aeiou]{1,2}`, `.`, `n`, `Yap State.*here`, `s`} {
		re := regexp.MustCompile(p)
		ms := ix.Find(re)
		for cur := -1; cur < len(ms); cur++ {
			if StripANSI(Highlight(styled, ms, cur)) != StripANSI(styled) {
				t.Fatalf("pattern %q current %d changed the text", p, cur)
			}
		}
	}
}

func TestLineOfAMatch(t *testing.T) {
	ix := NewIndex(styled)
	ms := mustFind(t, ix, "again")
	if ms[0].Line() != 3 {
		t.Errorf("Line() = %d", ms[0].Line())
	}
}

// The Pacific collection page is about 97,000 lines with thousands of matches
// for a common word: indexing, finding and marking must stay quick.
func TestBigPageIsQuick(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 97000; i++ {
		if i%8 == 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("  \x1b[38;5;252mthe quick brown fox in the Pacific, item ")
		sb.WriteString(strings.Repeat("x", i%7))
		sb.WriteString("\x1b[0m\n")
	}
	display := sb.String()
	start := time.Now()
	ix := NewIndex(display)
	re, _ := Compile("pacific")
	ms := ix.Find(re)
	out := Highlight(display, ms, len(ms)/2)
	took := time.Since(start)
	if len(ms) != 97000 {
		t.Fatalf("%d matches", len(ms))
	}
	if took > 4*time.Second {
		t.Errorf("index, find and highlight of %d lines took %s", 97000, took)
	}
	if len(out) <= len(display) {
		t.Error("nothing marked")
	}
	t.Logf("took %s", took)
}

// Marked renders the same text as Highlight for every current match.
func TestMarkedMatchesHighlight(t *testing.T) {
	ix := NewIndex(styled)
	for _, pattern := range []string{`yap`, `item continues`, `e`, `again`} {
		ms := mustFind(t, ix, pattern)
		mk := NewMarked(styled, ms)
		for cur := -1; cur < len(ms); cur++ {
			if got, want := mk.Render(cur), Highlight(styled, ms, cur); got != want {
				t.Fatalf("%q current %d:\n got  %q\n want %q", pattern, cur, got, want)
			}
		}
		// and in any order, since n and N go both ways
		for _, cur := range []int{len(ms) - 1, 0, len(ms) / 2, -1, 0} {
			if got, want := mk.Render(cur), Highlight(styled, ms, cur); got != want {
				t.Fatalf("%q out of order, current %d differs", pattern, cur)
			}
		}
	}
}

func TestMarkedWithNoMatchesIsTheText(t *testing.T) {
	if NewMarked(styled, nil).Render(0) != styled {
		t.Error("changed the text")
	}
}

// Moving to the next match must not re-mark the whole page.
func TestMovingTheCurrentMatchIsQuickOnABigPage(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 97000; i++ {
		sb.WriteString("  \x1b[38;5;252mthe quick brown fox in the Pacific\x1b[0m\n")
	}
	display := sb.String()
	re, _ := Compile("pacific")
	ms := NewIndex(display).Find(re)
	mk := NewMarked(display, ms)
	start := time.Now()
	for i := 0; i < 20; i++ {
		mk.Render(i)
	}
	if per := time.Since(start) / 20; per > 150*time.Millisecond {
		t.Errorf("one Render took %s on %d lines", per, 97000)
	}
}

func TestStripANSIHandlesTruncatedEscapes(t *testing.T) {
	for _, s := range []string{"a\x1b", "a\x1b[", "a\x1b[1", "a\x1b]8;;http", "a\x1b]8;;x\x1b"} {
		if got := StripANSI(s); !strings.HasPrefix(got, "a") {
			t.Errorf("%q -> %q", s, got)
		}
	}
}
