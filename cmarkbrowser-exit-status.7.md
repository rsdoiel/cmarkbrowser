%cmarkbrowser-exit-status(7) user manual | version 0.0.4 d68dce9
% R. S. Doiel
% 2026-11-26

# NAME

cmarkbrowser-exit-status — what the exit codes mean

# DESCRIPTION

cmarkbrowser uses the exit codes of the workspace convention.

# EXIT STATUS

0
: Success.

1
: -find matched nothing.

2
: Bad command line.

65
: The page is not Markdown (and not HTML), its front matter is not valid, or the
  file named is not text; or the corpus is not a corpus or is newer than this
  program reads.

66
: The page is not there (404 or 410), the path named does not exist, the
  directory named has no index.md, or -find has no corpus to read.

69
: The server cannot be reached.

70
: A bug.

73
: The -debug file cannot be created.

75
: The corpus is locked by another program; try again.

77
: The operating system refused access to the path named.

78
: The configuration file is there and wrong.

# SEE ALSO

cmarkbrowser(1), cmarkbrowser-config(7)
