#
# Makefile for the cmarkbrowser website: pandoc builds an HTML page beside each
# Markdown page, the Markdown pages are published as they are, for BonBon
# browsers and cmarkbrowser, and toc.bash writes /.well-known/toc.opml.
#
# Written by hand from knowledge's website.mak; do not regenerate it with cmt.
#
PROJECT = cmarkbrowser

# Where the site is published (GitHub Pages, gh-pages branch).
BASE_URL = https://rsdoiel.github.io/cmarkbrowser/

PANDOC = $(shell which pandoc)

# Internal notes, not website pages. index.md is a copy of README.md made below.
# history.md is the history cmarkbrowser writes in the folder it is run from (gitignored).
# cmarkbrowser.1.md and cmarkbrowser-TOPIC.7.md are the man page sources; command_reference.md is its website page.
NO_PUBLISH = TODO.md index.md cmarkbrowser.1.md history.md $(wildcard cmarkbrowser-*.7.md)

MD_PAGES = $(filter-out $(NO_PUBLISH),$(shell ls -1 *.md))

# The order of the table of contents. Every published page must be listed
# here (the build stops when one is missing), README.md first: it is the home page.
PAGE_ORDER = README.md user_manual.md command_reference.md bonbon.md INSTALL.md about.md CHANGES.md

HTML_PAGES = $(patsubst %.md,%.html,$(MD_PAGES))

TOC = .well-known/toc.opml

MISSING = $(filter-out $(PAGE_ORDER),$(MD_PAGES))

build: check-order $(HTML_PAGES) index.md $(TOC) .nojekyll

check-order:
	@if [ -n "$(MISSING)" ]; then echo "website.mak: add to PAGE_ORDER: $(MISSING)" >&2; exit 1; fi

$(HTML_PAGES): $(MD_PAGES) links-to-html.lua page.tmpl .FORCE
	@if [ -x "$(PANDOC)" ]; then $(PANDOC) --metadata title=$(basename $@) -V markdown-source=$(basename $@).md \
		-s --to html5 $(basename $@).md -o $(basename $@).html \
		--lua-filter=links-to-html.lua \
		--lua-filter=add-col-scope.lua \
		--template=page.tmpl; fi
	@if [ $@ = "README.html" ]; then sed -i.bak 's|href="README.md"|href="index.md"|' README.html && rm -f README.html.bak && mv README.html index.html; fi

# A BonBon browser asks for the site's address and gets index.md.
index.md: README.md
	cp README.md index.md

# GitHub Pages hides folders that start with a dot (and so /.well-known/)
# unless the site says it is not a Jekyll site.
.nojekyll:
	touch .nojekyll

$(TOC): $(MD_PAGES) toc.bash
	mkdir -p $(dir $(TOC))
	./toc.bash $(BASE_URL) $(PROJECT) $(PAGE_ORDER) > $(TOC).tmp
	mv $(TOC).tmp $(TOC)

toc: check-order
	@rm -f $(TOC)
	@$(MAKE) -f website.mak $(TOC)

clean:
	@rm -f $(HTML_PAGES) index.html index.md $(TOC)

.PHONY: build check-order toc clean .FORCE
.FORCE:
