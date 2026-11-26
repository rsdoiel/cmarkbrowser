---
title: cmarkbrowser
abstract: |-
  A terminal browser for the Markdown Web inspired by Dave Winer's BonBon: it fetches pages as Markdown, strips HTML, follows links and a site's table of contents, and lets the reader choose how the text looks.
authors:
  - family_name: Doiel
    given_name: R. S.
    id: https://orcid.org/0000-0003-0900-6903



repository_code: https://github.com/rsdoiel/cmarkbrowser
version: 0.0.3
license_url: https://www.gnu.org/licenses/agpl-3.0.txt

programming_language:
  - Go >= 1.27.4

keywords:
  - Markdown
  - CommonMark
  - browser
  - TUI
  - BonBon
  - Textcasting

date_released: 2026-11-26
---

About this software
===================

## cmarkbrowser 0.0.3

- ] and [ follow BonBon's rule for a page that names no urlPrev and urlNext (markdownweb v0.0.4, DR-0004): the outline is one flat list, the line below is next and the line above previous, the ends wrap, groups do not stop the arrows, and a list with calendar outlines is no longer reversed. A site the outline includes is a list of its own. A neighbour with nothing at its address is not skipped yet.
- markdownweb is v0.0.4: requests go through its polite fetcher, so the User-Agent is cmarkbrowser/VERSION, or the user-agent setting in ~/.config/cmarkbrowser/cmarkbrowser.yaml (a wrong file exits 78); new internal/config.
- Front matter and {{page.NAME}} macros come from github.com/rsdoiel/frontmatter; goldmark is v1.8.7; Needs Go 1.27.4.

## Authors

- [R. S. Doiel](https://orcid.org/0000-0003-0900-6903)






A terminal browser for the Markdown Web inspired by Dave Winer's BonBon: it fetches pages as Markdown, strips HTML, follows links and a site's table of contents, and lets the reader choose how the text looks.

- [License](https://www.gnu.org/licenses/agpl-3.0.txt)
- [Code Repository](https://github.com/rsdoiel/cmarkbrowser)
  - [Issue Tracker](https://github.com/rsdoiel/cmarkbrowser/issues)

## Programming languages

- Go >= 1.27.4




## Software Requirements

- Go >= 1.27.4
- github.com/rsdoiel/frontmatter >= 0.0.1
- github.com/rsdoiel/markdownweb >= 0.0.4


## Software Suggestions

- CMTools >= 0.0.46
- Pandoc >= 3.1.11
- GNU Make >= 3
- jq, zip and the GitHub CLI (gh), to cut a release
- Bash or PowerShell


