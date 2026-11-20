package render

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestPrepareStripsHTMLAndScriptContent(t *testing.T) {
	src := "A <b>bold</b> word and <script>alert(1)</script> here.\n\n<div class=\"x\">block html</div>\n\n<!-- note -->\n\nend\n"
	got, _ := Prepare(src)
	for _, bad := range []string{"<", "alert(1)", "<!--", "note"} {
		if strings.Contains(got, bad) {
			t.Errorf("%q still in %q", bad, got)
		}
	}
	for _, keep := range []string{"bold", "word", "here", "end"} {
		if !strings.Contains(got, keep) {
			t.Errorf("%q lost from %q", keep, got)
		}
	}
}

func TestPrepareKeepsHTMLInCodeSpansAndBlocks(t *testing.T) {
	src := "Use `<b>` here.\n\n```html\n<script>x</script>\n```\n"
	got, _ := Prepare(src)
	if !strings.Contains(got, "`<b>`") || !strings.Contains(got, "<script>x</script>") {
		t.Errorf("code was changed: %q", got)
	}
}

func TestPrepareNumbersLinksInOrder(t *testing.T) {
	src := "See [one](a.md) and [two](https://example.org/b.md \"t\").\n\n- [three](c.md)\n"
	got, links := Prepare(src)
	want := []string{"a.md", "https://example.org/b.md", "c.md"}
	if len(links) != 3 {
		t.Fatalf("links %#v", links)
	}
	for i, w := range want {
		if links[i].Dest != w || links[i].N != i+1 {
			t.Errorf("link %d: %#v", i, links[i])
		}
	}
	for _, marker := range []string{"one " + Sentinel + "[1]", "two " + Sentinel + "[2]", "three " + Sentinel + "[3]"} {
		if !strings.Contains(strings.ReplaceAll(got, "\\", ""), marker) {
			t.Errorf("no %q in %q", marker, got)
		}
	}
}

func TestPrepareLeavesImagesAlone(t *testing.T) {
	_, links := Prepare("![pic](p.png)\n")
	if len(links) != 0 {
		t.Errorf("image counted as link: %#v", links)
	}
}

func TestPrepareNoLinksNoChange(t *testing.T) {
	src := "# Title\n\nplain text\n"
	got, links := Prepare(src)
	if got != src || len(links) != 0 {
		t.Errorf("%q %#v", got, links)
	}
}

// The antenna front page is a table of links. Prepare must parse it the way
// glamour does (GFM tables), or link numbers and markers disagree with what is
// rendered.
func TestPrepareNumbersLinksInTables(t *testing.T) {
	src := "| Collection | Feeds |\n| --- | --- |\n| [SoCal](socal.html) | [RSS](socal.xml), [OPML](socal.opml) |\n| [Pacific](pacific.html) | |\n"
	got, links := Prepare(src)
	if len(links) != 4 || links[0].Dest != "socal.html" || links[3].Dest != "pacific.html" {
		t.Fatalf("links %#v", links)
	}
	plain := strings.ReplaceAll(strings.ReplaceAll(got, "\\", ""), Sentinel, "")
	for _, marker := range []string{"SoCal [1]", "RSS [2]", "OPML [3]", "Pacific [4]"} {
		if !strings.Contains(plain, marker) {
			t.Errorf("no %q in %q", marker, plain)
		}
	}
}

// applyEditsReference is the original algorithm, kept as the specification:
// edits are applied from the end, one at a time, and an edit whose end passes
// the start of the one before it is dropped.
func applyEditsReference(b []byte, edits []edit) string {
	es := append([]edit(nil), edits...)
	sort.SliceStable(es, func(i, j int) bool { return es[i].start > es[j].start })
	out := b
	lastStart := len(b) + 1
	for _, e := range es {
		if e.stop > lastStart {
			continue
		}
		out = append(append(append([]byte{}, out[:e.start]...), e.repl...), out[e.stop:]...)
		lastStart = e.start
	}
	return string(out)
}

func TestApplyEditsMatchesTheReferenceOnRandomEdits(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for round := 0; round < 3000; round++ {
		n := rng.Intn(60) + 1
		b := make([]byte, n)
		for i := range b {
			b[i] = byte('a' + rng.Intn(26))
		}
		var edits []edit
		for k := rng.Intn(8); k > 0; k-- {
			start := rng.Intn(n + 1)
			stop := start + rng.Intn(min(n-start, 6)+1) // empty edits (inserts) and removals
			edits = append(edits, edit{start, stop, strings.Repeat("X", rng.Intn(4))})
		}
		want := applyEditsReference(b, edits)
		if got := applyEdits(b, append([]edit(nil), edits...)); got != want {
			t.Fatalf("round %d: src %q edits %v\n got  %q\n want %q", round, b, edits, got, want)
		}
	}
}

// Numbering the links of a big page must not copy the whole page for every
// link: a 4.7 MB page with 8,000 links took 17 seconds.
func TestPrepareBigPageWithManyLinksIsFast(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 8000; i++ {
		fmt.Fprintf(&sb, "## [Item %d](https://example.org/%d)\n\n%s\n\n", i, i, strings.Repeat("filler text ", 45))
	}
	src := sb.String()
	if len(src) < 4_000_000 {
		t.Fatalf("test page is only %d bytes", len(src))
	}
	start := time.Now()
	out, links := Prepare(src)
	if took := time.Since(start); took > 3*time.Second {
		t.Errorf("Prepare took %s for %d bytes and %d links", took, len(src), len(links))
	}
	if len(links) != 8000 || !strings.Contains(out, `\[8000\]`) {
		t.Errorf("links %d, last marker missing", len(links))
	}
}
