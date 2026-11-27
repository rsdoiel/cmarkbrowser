%cmarkbrowser-editing(7) user manual | version 0.0.4 e22c5d6
% R. S. Doiel
% 2026-11-26

# NAME

cmarkbrowser-editing — edit the page you are reading

# DESCRIPTION

e edits the page being shown in your text editor ($VISUAL, else $EDITOR; a value
such as "code -w" works; neither set is a message, not a guess). The browser
steps aside while the editor runs and shows the page again, at the same place,
when it exits.

A local file (a page of a local site, the bookmarks file, the history file) is
edited in place. A remote page cannot be saved back, so its source, front matter
included, is saved as a new file in your Downloads folder (never replacing one,
a.md then a-1.md) and that copy is edited; keep it or delete it. A reader view
has no Markdown source and is refused.

# SEE ALSO

cmarkbrowser(1), cmarkbrowser-local(7), cmarkbrowser-files(7)
