package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// indexNames reads the topic names from `-help index`.
func indexNames(t *testing.T) []string {
	t.Helper()
	code, out, errs := run("-help", "index")
	if code != 0 {
		t.Fatalf("-help index: exit %d, stderr %q", code, errs)
	}
	var names []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if f := strings.Fields(line); len(f) > 0 {
			names = append(names, f[0])
		}
	}
	return names
}

func TestHelpIndexListsTheTopics(t *testing.T) {
	got := " " + strings.Join(indexNames(t), " ") + " "
	for _, want := range []string{"reading", "neighbours", "search", "corpus", "bookmarks", "local", "editing", "files", "debug", "config", "exit-status"} {
		if !strings.Contains(got, " "+want+" ") {
			t.Errorf("-help index lacks %q:%s", want, got)
		}
	}
}

func TestHelpIndexAcceptsEveryHelpSpelling(t *testing.T) {
	want := strings.Join(indexNames(t), " ")
	for _, opt := range []string{"-h", "--help", "-help"} {
		code, out, _ := run(opt, "index")
		if code != 0 || len(strings.Fields(out)) == 0 {
			t.Errorf("%s index: exit %d, stdout %q", opt, code, out)
		}
		var names []string
		for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
			names = append(names, strings.Fields(l)[0])
		}
		if strings.Join(names, " ") != want {
			t.Errorf("%s index lists %v", opt, names)
		}
	}
}

// Every topic is a manual page of section 7 that pandoc can render.
func TestEveryTopicIsASectionSevenPage(t *testing.T) {
	for _, name := range indexNames(t) {
		code, out, errs := run("-help", name)
		if code != 0 {
			t.Errorf("-help %s: exit %d, stderr %q", name, code, errs)
			continue
		}
		if !strings.HasPrefix(out, "%cmarkbrowser-"+name+"(7) user manual | version ") {
			t.Errorf("-help %s does not start with a section 7 title block:\n%.100s", name, out)
		}
		for _, h := range []string{"# NAME", "# DESCRIPTION"} {
			if !strings.Contains(out, "\n"+h+"\n") {
				t.Errorf("-help %s has no %q heading", name, h)
			}
		}
		if strings.Contains(out, "{app_name}") || strings.Contains(out, "{version}") {
			t.Errorf("-help %s leaves a placeholder unfilled", name)
		}
	}
}

func TestUnknownOrExtraTopicIsUsage(t *testing.T) {
	code, _, errs := run("-help", "nope")
	if code != 2 || !strings.Contains(errs, "-help index") {
		t.Errorf("unknown topic: exit %d, stderr %q: want 2 and a pointer to `-help index`", code, errs)
	}
	if code, _, errs := run("-help", "search", "corpus"); code != 2 {
		t.Errorf("two topics: exit %d, stderr %q, want 2", code, errs)
	}
}

func TestTheMainPageNamesTheIndexAndEveryTopic(t *testing.T) {
	_, page, _ := run("-help")
	if !strings.Contains(page, "-help index") {
		t.Error("the cmarkbrowser(1) page does not mention `-help index`")
	}
	for _, name := range indexNames(t) {
		if !strings.Contains(page, "cmarkbrowser-"+name+"(7)") {
			t.Errorf("SEE ALSO of the cmarkbrowser(1) page lacks cmarkbrowser-%s(7)", name)
		}
	}
}

// body drops the title block, which carries the version and the commit.
func body(s string) string {
	lines := strings.SplitAfter(s, "\n")
	i := 0
	for i < len(lines) && strings.HasPrefix(lines[i], "%") {
		i++
	}
	return strings.Join(lines[i:], "")
}

// The committed pages are what `-help TOPIC` says now: `make build` writes them.
func TestTopicPagesAreCurrent(t *testing.T) {
	names := indexNames(t)
	for _, name := range names {
		file := filepath.Join("..", "..", "cmarkbrowser-"+name+".7.md")
		have, err := os.ReadFile(file)
		if err != nil {
			t.Errorf("%s: %v (run make build)", file, err)
			continue
		}
		_, out, _ := run("-help", name)
		if body(string(have)) != body(out) {
			t.Errorf("%s is out of date: run make build", file)
		}
	}
	pages, _ := filepath.Glob(filepath.Join("..", "..", "cmarkbrowser-*.7.md"))
	var onDisk []string
	for _, p := range pages {
		onDisk = append(onDisk, strings.TrimSuffix(strings.TrimPrefix(filepath.Base(p), "cmarkbrowser-"), ".7.md"))
	}
	listed := append([]string(nil), names...)
	sort.Strings(onDisk)
	sort.Strings(listed)
	if strings.Join(onDisk, " ") != strings.Join(listed, " ") {
		t.Errorf("*.7.md pages %v, -help index %v", onDisk, listed)
	}
}

// The Makefile's TOPICS line is what `make build` loops over.
func TestMakefileTopicsAreTheIndex(t *testing.T) {
	mk, err := os.ReadFile(filepath.Join("..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^TOPICS *= *(.*)$`).FindSubmatch(mk)
	if m == nil {
		t.Fatal("Makefile has no TOPICS line")
	}
	got := strings.Fields(string(m[1]))
	want := indexNames(t)
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("Makefile TOPICS = %v, -help index = %v", got, want)
	}
}
