%cmarkbrowser-bookmarks(7) user manual | version 0.0.4 d68dce9
% R. S. Doiel
% 2026-11-26

# NAME

cmarkbrowser-bookmarks — bookmarks and history

# DESCRIPTION

m adds the page being shown to a bookmarks file and B opens it. The file is
ordinary Markdown (bookmarks.md in the working directory, or the file named by
-bookmarks FILE): a list with a link and the date for each page, created with a
"# Bookmarks" heading when it is missing and never rewritten, only added to. A
page already in it is not added twice. Open it with B, or give it as the
argument, and follow its links like any page; b comes back to it. Edit it by
hand if you like.

Every page shown is recorded in a history file, history.md in the working
directory (or the file named by -history FILE), newest first: a list with a link
and the date and time, and an address visited again moves to the top. The list
keeps the 500 latest. H opens it as a page, and with no URL on the command line
cmarkbrowser starts from it, so you can click to where you left off. The file is
rewritten by cmarkbrowser; keep notes elsewhere.

-no-history records nothing; -dump never records. -history and -no-history
cannot be combined. The session history (b and f) is kept in memory either way.
Local pages are never recorded and cannot be bookmarked.

# SEE ALSO

cmarkbrowser(1), cmarkbrowser-reading(7), cmarkbrowser-local(7)
