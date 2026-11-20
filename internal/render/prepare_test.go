package render

import (
	"strings"
	"testing"
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
