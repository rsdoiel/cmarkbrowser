#
# Makefile for cmarkbrowser, a concept experiment. There is no release flow
# yet (see TODO.md); add one from knowledge/Makefile when there is something
# to release.
#
PROJECT = cmarkbrowser

PROGRAMS = cmarkbrowser

VERSION = $(shell grep '"version":' codemeta.json | cut -d\"  -f 4)

PREFIX = $(HOME)
ifneq ($(prefix),)
	PREFIX = $(prefix)
endif

.PHONY: all build test generate clean install docs website publish

all: build

build:
	mkdir -p bin
	for P in $(PROGRAMS); do go build -o bin/$$P cmd/$$P/main.go; done

test:
	go vet ./...
	go test ./...

# generate rebuilds version.go from codemeta.json. It is not part of build.
generate:
	cmt codemeta.json version.go

install: build
	for P in $(PROGRAMS); do install -m 755 bin/$$P $(PREFIX)/bin/$$P; done

# docs writes the command reference from the program's own help, so the two
# cannot differ.
docs: build
	( printf '# cmarkbrowser command reference\n\n'; \
	  printf 'This page is written by `cmarkbrowser -help`.\n\n'; \
	  ./bin/cmarkbrowser -help | sed '1,3d; s/^# /## /' ) > cmarkbrowser.1.md

# website builds the HTML pages, index.md, .nojekyll and .well-known/toc.opml.
website: docs
	$(MAKE) -f website.mak

# publish merges into the gh-pages branch and pushes it (see publish.bash).
publish: website
	./publish.bash

clean:
	rm -rf bin dist
