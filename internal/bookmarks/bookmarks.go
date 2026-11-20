// Package bookmarks keeps a reading list as a Markdown file: one list item per
// page, a link and the date it was added. The file is plain text you can read
// in the browser itself, edit by hand and keep under version control; nothing
// else is stored anywhere. Existing text in the file is never rewritten, only
// added to.
package bookmarks

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"
)

// ErrNotWeb marks an address that is not http or https: file, script and data
// addresses, and relative ones, are never bookmarked.
var ErrNotWeb = errors.New("only http and https pages can be bookmarked")

const heading = "# Bookmarks\n\n"

/** Add appends a bookmark to the Markdown file at path, creating the file with
 * a "# Bookmarks" heading when it is missing. An address already linked in the
 * file is not added again. The title is escaped so it cannot become markup or
 * HTML; an empty title is replaced by the address.
 *
 * Parameters:
 *   path (string) — the bookmarks file
 *   title (string) — the page title
 *   address (string) — an http or https address
 *   now (time.Time) — the date shown beside the entry
 *
 * Returns:
 *   bool — true when an entry was written, false when the address was already there
 *   error — ErrNotWeb for another kind of address, or the error from the file
 *     system (an *os.PathError), so the caller can say what failed
 *
 * Example:
 *   added, err := bookmarks.Add("bookmarks.md", "A Page", "https://x.org/a", time.Now())
 */
func Add(path, title, address string, now time.Time) (bool, error) {
	u, err := url.Parse(strings.TrimSpace(address))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return false, fmt.Errorf("%w: %q", ErrNotWeb, address)
	}
	address = u.String()
	label := strings.Join(strings.Fields(title), " ")
	if label == "" {
		label = address
	}
	entry := Line(label, address, now.Format("2006-01-02")) + "\n"

	existing, err := os.ReadFile(path)
	switch {
	case err == nil:
		if strings.Contains(string(existing), "]("+dest(address)+")") {
			return false, nil
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return false, err
		}
		defer f.Close()
		prefix := ""
		if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
			prefix = "\n"
		}
		if _, err := f.WriteString(prefix + entry); err != nil {
			return false, err
		}
		return true, f.Close()
	case errors.Is(err, os.ErrNotExist):
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return false, err
		}
		defer f.Close()
		if _, err := f.WriteString(heading + entry); err != nil {
			return false, err
		}
		return true, f.Close()
	}
	return false, err
}

/** Line returns one Markdown list item linking address, with title and stamp
 * escaped so neither can become markup or HTML. It is how bookmarks and the
 * history file write their entries.
 *
 * Parameters:
 *   title (string) — the link text
 *   address (string) — the link destination
 *   stamp (string) — the text shown in parentheses after the link
 *
 * Returns:
 *   string — "- [title](address) (stamp)", with no newline
 *
 * Example:
 *   bookmarks.Line("A Page", "https://x.org/a", "2026-11-20") // "- [A Page](https://x.org/a) (2026-11-20)"
 */
func Line(title, address, stamp string) string {
	return fmt.Sprintf("- [%s](%s) (%s)", escape(title), dest(address), stamp)
}

/** Has reports whether the file at path already links address.
 *
 * Example:
 *   if bookmarks.Has("bookmarks.md", page.URL) { ... }
 */
func Has(path, address string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	u, err := url.Parse(strings.TrimSpace(address))
	if err != nil {
		return false
	}
	return strings.Contains(string(b), "]("+dest(u.String())+")")
}

// escape backslash-escapes what Markdown would read as markup or HTML.
func escape(s string) string {
	var sb strings.Builder
	for i, r := range s {
		switch r {
		case '\\', '`', '*', '_', '[', ']', '<', '>', '~', '|':
			sb.WriteByte('\\')
		case '&':
			if end := strings.IndexByte(s[i:], ';'); end > 1 && end < 12 && isEntity(s[i+1:i+end]) {
				sb.WriteByte('\\')
			}
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

func isEntity(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '#' {
			return false
		}
	}
	return s != ""
}

// dest writes an address as a link destination: characters that would end or
// confuse it are percent-encoded, and an address with parentheses is wrapped
// in angle brackets.
func dest(a string) string {
	a = strings.NewReplacer(" ", "%20", "<", "%3C", ">", "%3E", "\n", "%0A", "\\", "%5C").Replace(a)
	if strings.ContainsAny(a, "()") {
		return "<" + a + ">"
	}
	return a
}
