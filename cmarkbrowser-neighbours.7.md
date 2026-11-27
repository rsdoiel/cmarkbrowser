%cmarkbrowser-neighbours(7) user manual | version 0.0.4 e22c5d6
% R. S. Doiel
% 2026-11-26

# NAME

cmarkbrowser-neighbours — the next and previous page

# DESCRIPTION

] opens the next page and [ the previous one. A page can name them in its own
front matter, as urlNext and urlPrev. When it names none, cmarkbrowser asks the
site's table of contents: the nearest .well-known/toc.opml above the page, with
the outlines it includes. It is read the first time one of these keys needs it,
and remembered.

As in BonBon the outline is one list. ] is the line below the page and [ the
line above, groups do not stop the arrows, the ends wrap, and a site the outline
includes is a list of its own.

A neighbour with nothing at its address (a non-2xx answer, or none) is passed
over, as BonBon does. Each address is asked about once and the answer is
remembered; at most 12 candidates are tried per key press. This is for web
pages; a local site already skips a file that is not there.

A site with no table of contents says so in the status line.

# SEE ALSO

cmarkbrowser(1), cmarkbrowser-local(7), cmarkbrowser-reading(7)
