%cmarkbrowser-debug(7) user manual | version 0.0.4 d68dce9
% R. S. Doiel
% 2026-11-26

# NAME

cmarkbrowser-debug — -debug, -dump and what they show

# DESCRIPTION

-debug FILE appends a log of the session to FILE, one JSON object per line:
every key pressed, every address tried (the same address, the page's declared
Markdown alternate, the .md sibling) with its status, size, time and outcome,
which of them produced the page, the time each step took, and the exit code. It
holds addresses, key names, counts and times, never page content. The file is
created with mode 0600 (exit 73 when it cannot be created). Use it to see why a
page opened as it did, and to send a session to someone who can read it.

-dump fetches the URL, prints the rendered page and its numbered links, and
exits. -style NAME picks the glamour style (dark, the default, light, notty or
ascii) and -width N the columns (default 80). -dump never records history.

# EXAMPLES

    cmarkbrowser -dump -style notty https://rsdoiel.github.io/blog/index.md | less
    cmarkbrowser -debug session.jsonl https://example.org/

# SEE ALSO

cmarkbrowser(1), cmarkbrowser-exit-status(7)
