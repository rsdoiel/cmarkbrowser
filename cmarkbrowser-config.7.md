%cmarkbrowser-config(7) user manual | version 0.0.4 d68dce9
% R. S. Doiel
% 2026-11-26

# NAME

cmarkbrowser-config — the configuration file

# DESCRIPTION

If ~/.config/cmarkbrowser/cmarkbrowser.yaml exists ($XDG_CONFIG_HOME is used when
set) it is read before anything is fetched. A missing file is fine; a file that
is there and wrong is an error (exit 78).

Settings so far:

user-agent
: The User-Agent sent with every request, so a site can tell who is asking. The
  default is cmarkbrowser/VERSION. Example:

      user-agent: cmarkbrowser/0.0.4 (+https://example.org/about)

corpus
: The cmarkfind corpus file for S, F and -find. -corpus FILE overrides it, and
  CMARKFIND_DB is used when it is not set. Example:

      corpus: ~/.local/share/cmarkfind/corpus.db

# SEE ALSO

cmarkbrowser(1), cmarkbrowser-corpus(7), cmarkbrowser-exit-status(7)
