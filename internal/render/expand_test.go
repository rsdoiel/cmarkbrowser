package render

import (
	"strings"
	"testing"
)

var expandMeta = map[string]any{
	"title":    "Hello",
	"author":   "Dave",
	"pub_date": "Fri, 23 Oct 2026 15:19:45 GMT",
	"source":   map[string]any{"title": "Scripting News", "url": "http://scripting.com/"},
	"tags":     []any{"a", "b"},
	"count":    3,
}

func TestExpandPageMacros(t *testing.T) {
	src := "# {{page.title}}\n\nBy {{page.author}} on {{page.pub_date}}. {{page.count}} items.\n"
	got, missing := Expand(src, expandMeta)
	want := "# Hello\n\nBy Dave on Fri, 23 Oct 2026 15:19:45 GMT. 3 items.\n"
	if got != want || len(missing) != 0 {
		t.Errorf("got %q missing %v, want %q", got, missing, want)
	}
}

func TestExpandNestedAndListKeys(t *testing.T) {
	got, missing := Expand("From {{page.source.title}} ({{page.tags}})\n", expandMeta)
	if got != "From Scripting News (a, b)\n" || len(missing) != 0 {
		t.Errorf("got %q missing %v", got, missing)
	}
}

func TestExpandLeavesCodeAlone(t *testing.T) {
	src := "Use `{{page.title}}` here.\n\n```\n{{page.title}}\n```\n\n    {{page.title}}\n\n{{page.title}}\n"
	got, _ := Expand(src, expandMeta)
	if strings.Count(got, "{{page.title}}") != 3 || strings.Count(got, "Hello") != 1 {
		t.Errorf("code was expanded or text was not: %q", got)
	}
}

func TestExpandIsOnePass(t *testing.T) {
	meta := map[string]any{"title": "{{page.author}}", "author": "Dave"}
	got, _ := Expand("{{page.title}}\n", meta)
	if strings.Contains(got, "Dave") || !strings.Contains(got, "page.author") {
		t.Errorf("a value was expanded again: %q", got)
	}
}

func TestExpandMissingKeyStaysVisibleAndIsReported(t *testing.T) {
	got, missing := Expand("{{page.nope}} and {{page.title}} and {{page.source}}\n", expandMeta)
	if !strings.Contains(got, "{{page.nope}}") || !strings.Contains(got, "Hello") {
		t.Errorf("got %q", got)
	}
	// a key that holds a table, not a value, is as good as missing
	if !strings.Contains(got, "{{page.source}}") {
		t.Errorf("table key was expanded: %q", got)
	}
	if len(missing) != 2 || missing[0] != "nope" || missing[1] != "source" {
		t.Errorf("missing = %v", missing)
	}
}

func TestExpandEscapesMarkupInValues(t *testing.T) {
	meta := map[string]any{"title": "*x* <b>y</b> [z](http://evil)\nnext"}
	got, _ := Expand("{{page.title}}\n", meta)
	clean, links := Prepare(got)
	if len(links) != 0 {
		t.Errorf("a value made a link: %#v in %q", links, got)
	}
	if strings.Contains(clean, "*x*") && !strings.Contains(clean, `\*x\*`) {
		t.Errorf("emphasis not escaped: %q", clean)
	}
	if strings.Contains(got, "\n"+"next") {
		t.Errorf("newline in a value broke the line: %q", got)
	}
}

func TestExpandInLinkTextAndEmphasis(t *testing.T) {
	got, _ := Expand("[{{page.title}}](a.md) and **{{page.author}}**\n", expandMeta)
	if got != "[Hello](a.md) and **Dave**\n" {
		t.Errorf("got %q", got)
	}
}

func TestExpandNoMacrosReturnsSourceUnchanged(t *testing.T) {
	src := "plain {text} and {{ other }} stuff\n"
	got, missing := Expand(src, expandMeta)
	if got != src || len(missing) != 0 {
		t.Errorf("got %q missing %v", got, missing)
	}
}
