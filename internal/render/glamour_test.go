package render

import (
	"regexp"
	"strings"
	"testing"
)

// Links in a table cell were listed a second time under the table by
// glamour ("[1]: text [1] URL"), and the cell showed "[1][1]". The page's own
// numbered marker and the Links list already say it all, so the table must hold
// each link once.
// osc8 matches a terminal hyperlink escape, which carries the URL invisibly.
var osc8 = regexp.MustCompile("\x1b\\]8;[^\x07\x1b]*(\x07|\x1b\\\\)")

func TestGlamourTableLinksAreNotRepeatedAfterTheTable(t *testing.T) {
	src := "| Name | Link |\n| --- | --- |\n| One | [first](https://example.org/one.md) |\n| Two | [second](https://example.org/two.md) |\n\nAfter the table.\n"
	prepared, _ := Prepare(src)
	out, err := Glamour{Style: "notty"}.Render(prepared, 70)
	if err != nil {
		t.Fatal(err)
	}
	plain := osc8.ReplaceAllString(strings.ReplaceAll(out, Sentinel, ""), "")
	if strings.Contains(plain, "]: ") {
		t.Errorf("link reference lines after the table:\n%s", plain)
	}
	if strings.Contains(plain, "[1][1]") || strings.Count(plain, "[1]") != 1 {
		t.Errorf("link 1 is not marked exactly once:\n%s", plain)
	}
}

// Glamour prints each link's URL after its text. The page already shows the
// address in the status line and the Links list, so the body must not repeat
// it, in a paragraph or a table cell, and the spacing around the marker stays
// single, with punctuation right after it.
func TestGlamourDoesNotPrintLinkURLsInTheBody(t *testing.T) {
	src := "See [first](https://example.org/one.md) now, or [again](https://example.org/again.md).\n\nA [ref][r] link.\n\n[r]: https://example.org/r.md\n\n| A | B |\n| --- | --- |\n| [two](https://example.org/two.md) | x |\n"
	prepared, _ := Prepare(src)
	for _, style := range []string{"notty", "dark", "light"} {
		out, err := Glamour{Style: style}.Render(prepared, 70)
		if err != nil {
			t.Fatal(err)
		}
		plain := osc8.ReplaceAllString(strings.ReplaceAll(out, Sentinel, ""), "")
		plain = ansiSGR.ReplaceAllString(plain, "")
		if strings.Contains(plain, "example.org") {
			t.Errorf("%s: a URL is printed in the body:\n%s", style, plain)
		}
		if !strings.Contains(plain, "See first [1] now, or again [2].") {
			t.Errorf("%s: want %q in:\n%s", style, "See first [1] now, or again [2].", plain)
		}
	}
}

// ansiSGR matches a colour or style escape.
var ansiSGR = regexp.MustCompile("\x1b\\[[0-9;]*m")

// What the person sees for pictures: alt text as "[image: alt] [N]", nothing at
// all for a picture with no alt text, and no address anywhere in the body.
func TestGlamourShowsImagesAsAltTextAndNeverAnAddress(t *testing.T) {
	src := "Text ![A cat](cat.png) inline.\n\n![](decor.png)\n\n[![badge](b.svg)](https://x.org/)\n\n| a |\n| --- |\n| ![in table](t.png) |\n"
	prepared, _ := Prepare(src)
	for _, style := range []string{"notty", "dark", "light"} {
		out, err := Glamour{Style: style}.Render(prepared, 70)
		if err != nil {
			t.Fatal(err)
		}
		plain := ansiSGR.ReplaceAllString(osc8.ReplaceAllString(strings.ReplaceAll(out, Sentinel, ""), ""), "")
		for _, bad := range []string{"cat.png", "decor", "b.svg", "x.org", "t.png", "Image:", "→"} {
			if strings.Contains(plain, bad) {
				t.Errorf("%s: %q in the body:\n%s", style, bad, plain)
			}
		}
		for _, want := range []string{"Text [image: A cat] [1] inline.", "[image: badge] [2]", "[image: in table] [3]"} {
			if !strings.Contains(plain, want) {
				t.Errorf("%s: want %q in:\n%s", style, want, plain)
			}
		}
	}
}
