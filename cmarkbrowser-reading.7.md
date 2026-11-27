%cmarkbrowser-reading(7) user manual | version 0.0.4 d68dce9
% R. S. Doiel
% 2026-11-26

# NAME

cmarkbrowser-reading — moving around a page and following links

# DESCRIPTION

cmarkbrowser fetches a page as Markdown, shows it in the terminal and follows
its links. It is modelled on Dave Winer's BonBon: HTML in a page is stripped, a
page cannot carry its own styling, and a site's table of contents is read from
/.well-known/toc.opml.

Tab and Shift-Tab move between links (the focused link's address is shown at
the bottom) and Enter follows it. Or type a link's number and press Enter. b and
f go back and forward. Esc clears a typed number, then the focus, then a search.
The arrows, PgUp, PgDn, space, g and G scroll. q quits.

o asks for an address to open, at the bottom of the screen: type it and press
Enter (Esc cancels). An address starts with http:// or https://; a bare host
name such as example.org/a.md gets https://. A file name is refused: typed
addresses are for the web, and local files are opened only from the command line
and through B and H. The page joins the session history, so b comes back.

A link to an HTML page that has no Markdown copy is shown in reader view: its
text, links and pictures that have alt text, without scripts, styles or menus.

-dump prints the rendered page and its numbered links and exits, for a script;
-style NAME (dark, light, notty, ascii) and -width N shape it. See
cmarkbrowser-debug(7).

# SEE ALSO

cmarkbrowser(1), cmarkbrowser-neighbours(7), cmarkbrowser-search(7),
cmarkbrowser-files(7)
