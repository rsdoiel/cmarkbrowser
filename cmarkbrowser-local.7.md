%cmarkbrowser-local(7) user manual | version 0.0.4 d68dce9
% R. S. Doiel
% 2026-11-26

# NAME

cmarkbrowser-local — browse a site on your own file system

# DESCRIPTION

Given a path instead of an address, cmarkbrowser browses a site on your own file
system, with no web server and no protocol: cmarkbrowser ./ browses the current
folder, starting at its index.md; a Markdown file starts at that file.

The site root is the folder you gave (for a file, the nearest folder above it
with a .well-known/toc.opml, else its own folder). A link in a page is a path:
b.md is next to the page, ../c.md is above it, and /d.md is relative to the site
root, as on the web; a link to x.html opens x.md when there is one. Links to
http:// and https:// addresses go to the web as usual.

A link that would leave the site root, by .. or through a symlink, is refused,
and so is any file: address, in any page: nothing a page says can read a file
outside the folder you named. ] and [ use the site's .well-known/toc.opml read
from the folder (its addresses match by path, so a site not yet published works).

The point is to preview a site and edit it: local pages are never recorded in
the history file and cannot be bookmarked. A directory with no index.md is exit
66; a file: address on the command line is a usage error. Local files are
otherwise opened only through B and H, never from a link.

# SEE ALSO

cmarkbrowser(1), cmarkbrowser-editing(7), cmarkbrowser-neighbours(7)
