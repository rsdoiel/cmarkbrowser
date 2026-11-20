---
title: cmarkbrowser
abstract: |-
  A terminal browser for the Markdown web, after Dave Winer's BonBon: it fetches pages as Markdown, strips HTML, follows links and a site's table of contents, and lets the reader choose how the text looks.
authors:
  - family_name: Doiel
    given_name: R. S.
    id: https://orcid.org/0000-0003-0900-6903



repository_code: https://github.com/rsdoiel/cmarkbrowser
version: 0.0.0
license_url: https://www.gnu.org/licenses/agpl-3.0.txt

programming_language:
  - Go >= 1.27.3

keywords:
  - Markdown
  - CommonMark
  - browser
  - TUI
  - BonBon
  - Textcasting

date_released: 2026-11-20
---

About this software
===================

## cmarkbrowser 0.0.0

v0.0.0 is the prototype, a proof of concept. A terminal browser for the Markdown web (after Dave Winer's BonBon). Opens pages as Markdown with BonBon's Accept header, strips front matter and HTML, numbers and focuses links, back and forward, search, reader view for HTML pages, source for XML and text, notes for images and files. Next and previous page from a page's urlNext and urlPrev or the site's /.well-known/toc.opml (root first), urlMirrorOf link translation from bonbon.json. Bookmarks (bookmarks.md) and history (history.md, on by default, -no-history turns it off), o to open an address, e to edit the page in $VISUAL or $EDITOR. Browses a Markdown folder as a site with no server: cmarkbrowser ./ (cmarkbrowser DR-0001). Documentation is a Markdown site for BonBon browsers as well as an HTML website. Not yet in this prototype: a configuration file, per-platform state folders, images, urlSearch.

## Authors

- [R. S. Doiel](https://orcid.org/0000-0003-0900-6903)






A terminal browser for the Markdown web, after Dave Winer's BonBon: it fetches pages as Markdown, strips HTML, follows links and a site's table of contents, and lets the reader choose how the text looks.

- [License](https://www.gnu.org/licenses/agpl-3.0.txt)
- [Code Repository](https://github.com/rsdoiel/cmarkbrowser)
  - [Issue Tracker](https://github.com/rsdoiel/cmarkbrowser/issues)

## Programming languages

- Go >= 1.27.3




## Software Requirements

- Go >= 1.27.3


## Software Suggestions

- CMTools >= 0.0.46
- Pandoc >= 3.1.11
- GNU Make >= 3
- jq, zip and the GitHub CLI (gh), to cut a release
- Bash or PowerShell


