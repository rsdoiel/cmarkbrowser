// Package history keeps the pages visited as a Markdown file, newest first, so
// the browser can start from it and the person can click back to where they
// left off. Like the bookmarks file it is plain text anyone can read and keep
// under version control; unlike it, the browser rewrites it (an address visited
// again moves to the top, and the list is capped). Nothing is written unless the
// person named a file with -history.
package history

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rsdoiel/cmarkbrowser/internal/bookmarks"
)

// ErrNotWeb marks an address that is not http or https: local files and other
// schemes are never recorded.
var ErrNotWeb = errors.New("only http and https pages are recorded in the history")

// MaxEntries is how many entries the file keeps; the oldest are dropped.
const MaxEntries = 500

const heading = "# History\n\nPages visited, newest first. This file is rewritten by cmarkbrowser.\n\n"

/** Record puts a visit at the top of the history file at path, creating the
 * file when it is missing. An address already in the file moves to the top
 * with the new time and title. Only lines that are list items with a link are
 * kept from the old file; the heading text is written afresh. The file is
 * replaced in one rename, so an interrupted write leaves the old file.
 *
 * Parameters:
 *   path (string) — the history file
 *   title (string) — the page title; the address is used when it is empty
 *   address (string) — an http or https address
 *   now (time.Time) — shown beside the entry as "2006-01-02 15:04"
 *
 * Returns:
 *   error — ErrNotWeb for another kind of address, or the error from the file
 *     system (an *os.PathError)
 *
 * Example:
 *   err := history.Record("history.md", "A Page", "https://x.org/a", time.Now())
 */
func Record(path, title, address string, now time.Time) error {
	u, err := url.Parse(strings.TrimSpace(address))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("%w: %q", ErrNotWeb, address)
	}
	address = u.String()
	label := strings.Join(strings.Fields(title), " ")
	if label == "" {
		label = address
	}
	line := bookmarks.Line(label, address, now.Format("2006-01-02 15:04"))
	marker := strings.TrimSuffix(strings.TrimPrefix(bookmarks.Line("", address, ""), "- ["), " ()") // "](dest)"
	entries := []string{line}
	old, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, l := range strings.Split(string(old), "\n") {
		if !strings.HasPrefix(l, "- [") || !strings.Contains(l, "](") {
			continue
		}
		if strings.Contains(l, marker+" (") {
			continue // this address moved to the top
		}
		entries = append(entries, l)
	}
	if len(entries) > MaxEntries {
		entries = entries[:MaxEntries]
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".history-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(heading + strings.Join(entries, "\n") + "\n"); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
