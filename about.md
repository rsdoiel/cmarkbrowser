---
title: cmarkbrowser
abstract: |-
  A terminal browser for the Markdown Web inspired by Dave Winer's BonBon: it fetches pages as Markdown, strips HTML, follows links and a site's table of contents, and lets the reader choose how the text looks.
authors:
  - family_name: Doiel
    given_name: R. S.
    id: https://orcid.org/0000-0003-0900-6903



repository_code: https://github.com/rsdoiel/cmarkbrowser
version: 0.0.4
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

## cmarkbrowser 0.0.4

- S searches a cmarkfind corpus (a SQLite3 FTS5 file) and shows the results as a page; F fills the corpus from the site being read, 25 pages a press; -corpus, -no-corpus, the corpus config key and CMARKFIND_DB name the file; -find QUERY prints the results for scripts (exit 0, 1 none, 66 no corpus, 65 not a corpus). Stored text is escaped so a result can make no link of its own (cmarkbrowser DR-0003)
- ] and [ pass over a neighbour with nothing at its address, as BonBon does: each address asked once, at most 12 candidates a key press
- Needs markdownweb v0.0.5 and cmarkfind v0.0.1 (a pure-Go SQLite driver comes with it; still no cgo)
- -help index lists eleven help topics; -help TOPIC shows one, and each is a section 7 man page (cmarkbrowser-TOPIC.7)

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
- github.com/rsdoiel/markdownweb >= 0.0.5
- github.com/rsdoiel/cmarkfind >= 0.0.1


## Software Suggestions

- CMTools >= 0.0.46
- Pandoc >= 3.1.11
- GNU Make >= 3
- jq, zip and the GitHub CLI (gh), to cut a release
- Bash or PowerShell


