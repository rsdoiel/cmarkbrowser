%cmarkbrowser-files(7) user manual | version 0.0.4 d68dce9
% R. S. Doiel
% 2026-11-26

# NAME

cmarkbrowser-files — links that are not pages

# DESCRIPTION

A link to something that is not a page is shown the way BonBon shows it.

XML (RSS, Atom, OPML, any XML) and text (plain text, JSON, YAML, CSV, code) are
shown as source, in a block, up to 1 MB, with control characters removed so
nothing in a file can steer your terminal.

An image or any other file (a PDF, an archive) is noted with its type, size and
address and is not downloaded; a terminal cannot show a picture.

These are ordinary pages: they go in the history, can be bookmarked and, if they
are source, e edits a copy of them. In a local site the same rules go by the
file's extension.

A link to an HTML page that has no Markdown copy is shown in reader view: its
text, links and pictures that have alt text, without scripts, styles or menus.

# SEE ALSO

cmarkbrowser(1), cmarkbrowser-reading(7), cmarkbrowser-editing(7)
