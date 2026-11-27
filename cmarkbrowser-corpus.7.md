%cmarkbrowser-corpus(7) user manual | version 0.0.4 e22c5d6
% R. S. Doiel
% 2026-11-26

# NAME

cmarkbrowser-corpus — search and fill a cmarkfind corpus

# SYNOPSIS

cmarkbrowser -find QUERY [-corpus FILE]

# DESCRIPTION

S searches a corpus of pages that cmarkfind keeps in one SQLite file: type a
query at the bottom of the screen and press Enter (Esc cancels). The results are
a page like any other, with numbered links, the matching text and a count
(including how many pages the sites list that are not fetched yet); b comes back
to it and runs the query again. The query is words, "phrases", AND, OR, NOT and
word*; anything else is searched as plain words.

F fills the corpus from the site being read: one batch of 25 pages of the table
of contents that lists the page (a local site is filled whole). Press F again
for the next batch. Nothing is added to the corpus unless F is pressed. To keep a
big site up to date, use the cmarkfind program.

-find QUERY does the search with no browser: it prints the results page's
Markdown on standard output. Exit 0 with matches, 1 with none, 66 when there is
no corpus, 65 when the file is not a corpus or has a newer schema.

The file is named by -corpus FILE, else the corpus key of the configuration
file, else the CMARKFIND_DB variable, else cmarkfind's own default. -no-corpus
turns S and F off. -corpus and -no-corpus cannot be combined.

Text stored in the corpus comes from sites you do not control. A result shows it
as text only: it can make no link, image or terminal hyperlink of its own.

# SEE ALSO

cmarkbrowser(1), cmarkbrowser-config(7), cmarkfind(1), cmarkfind-query(7)
