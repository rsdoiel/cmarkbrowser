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

.PHONY: all build test generate clean install

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

clean:
	rm -rf bin dist
