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
