%cmarkbrowser-search(7) user manual | version 0.0.4 e22c5d6
% R. S. Doiel
% 2026-11-26

# NAME

cmarkbrowser-search — search the page you are reading

# DESCRIPTION

/ searches the page. Type a Go regular expression (the RE2 syntax of the regexp
package) and press Enter: the page scrolls to the first match at or below the
top of the window, every match is marked and the current one is highlighted. n
goes to the next match and N to the previous one, wrapping at either end.

A pattern with no upper-case letter ignores case; an upper-case letter makes it
case sensitive; write (?i) or (?-i) to decide yourself.

Text is searched as you read it: the lines of one paragraph count as one run of
words, so a phrase is found even where the screen wrapped it, and a match never
runs from one paragraph into the next. / then Enter on an empty prompt repeats
the last search. A search belongs to the page it was run on.

To search the sites you have read, not the page, see cmarkbrowser-corpus(7).

# SEE ALSO

cmarkbrowser(1), cmarkbrowser-corpus(7), cmarkbrowser-reading(7)
