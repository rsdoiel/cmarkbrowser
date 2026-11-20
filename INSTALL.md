# Installing cmarkbrowser

cmarkbrowser is a single program written in Go. There is no release yet, so you
build it from the source. Releases, when there are some, are zip files on the
[releases page](https://github.com/rsdoiel/cmarkbrowser/releases) for Linux
(x86_64 and aarch64, which includes the Raspberry Pi), macOS (x86_64 and arm64)
and Windows (x86_64 and arm64): unzip one and put `bin/cmarkbrowser` on your PATH.

## Requirements

- Go 1.27.3 or later
- GNU Make 3 or later (optional; the commands below are what it runs)
- Pandoc 3.1.11 or later, only to build this website

## Build and install

~~~shell
git clone https://github.com/rsdoiel/cmarkbrowser
cd cmarkbrowser
make
make install
~~~

`make` builds `bin/cmarkbrowser`. `make install` copies it to `$HOME/bin`, or to
`prefix/bin` when you run `make install prefix=/usr/local`. Without make:

~~~shell
go build -o bin/cmarkbrowser ./cmd/cmarkbrowser
~~~

Run the tests with `make test`.

## Try it

~~~shell
cmarkbrowser -version
cmarkbrowser https://rsdoiel.github.io/blog/index.md
cmarkbrowser -dump https://dave.writersweb.org/
~~~

The browser needs a terminal. `-dump` prints a page and its links and works in a
pipe. See the [User Manual](user_manual.md) for the keys.

## Editing pages

Press `e` to edit the page you are reading. cmarkbrowser starts the editor named
by `$VISUAL`, else `$EDITOR`, and guesses none, so set one:

~~~shell
export EDITOR=nano
~~~

## Files it writes

cmarkbrowser has no settings file. In the folder you start it from it keeps
`history.md` (every page you visit, newest first) and, if you press `m`,
`bookmarks.md`. `-no-history` turns the history off. A copy of a remote page that
you edit goes in your `Downloads` folder.

## Building this website

~~~shell
make website
~~~

writes an HTML page beside each Markdown page, `index.md`, `.nojekyll` and
`.well-known/toc.opml`. `make publish` merges into the `gh-pages` branch and
pushes it. See [BonBon](bonbon.md) for what the Markdown side is for.
