// Package edit hands a Markdown file to the person's text editor. The browser
// suspends while the editor runs and shows the page again afterwards, so a site
// can be previewed and edited from one place (cmarkbrowser DR-0001 and the
// editor hand-off in TODO). Nothing here writes to a remote site.
package edit

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// ErrNoEditor means neither $VISUAL nor $EDITOR is set.
var ErrNoEditor = errors.New("set $VISUAL or $EDITOR to edit a page")

/** Command builds the command that opens file in the person's editor: $VISUAL,
 * else $EDITOR, split on white space so "code -w" works, with the file as the
 * last argument. No editor is guessed: with neither variable set it is an
 * error, so the person is never dropped into an editor they did not choose.
 *
 * Parameters:
 *   getenv (func(string) string) — os.Getenv, or a stand-in for tests
 *   file (string) — the file to edit
 *
 * Returns:
 *   *exec.Cmd — ready for tea.ExecProcess
 *   error — ErrNoEditor
 *
 * Example:
 *   cmd, err := edit.Command(os.Getenv, "/site/index.md")
 */
func Command(getenv func(string) string, file string) (*exec.Cmd, error) {
	for _, name := range []string{"VISUAL", "EDITOR"} {
		if f := strings.Fields(getenv(name)); len(f) > 0 {
			return exec.Command(f[0], append(f[1:], file)...), nil
		}
	}
	return nil, ErrNoEditor
}

/** SaveCopy writes the source of a remote page to a new file in dir, to be
 * edited and kept by the person; the page on the server is not touched. The
 * name is the last element of the address's path as written (case kept), with
 * anything but letters, digits, dot, dash and underscore replaced by "_", and
 * ".md" added when it has no Markdown or text extension; a name already taken
 * gets -1, -2, ... before the extension. An existing file is never replaced.
 *
 * Parameters:
 *   dir (string) — an existing folder, usually the Downloads folder
 *   address (string) — the page's address, for the name
 *   source (string) — the page as received, front matter included
 *
 * Returns:
 *   string — the path of the new file
 *   error — from the file system
 *
 * Example:
 *   p, err := edit.SaveCopy("/home/me/Downloads", "https://x.org/blog/a.md", src) // .../a.md
 */
func SaveCopy(dir, address, source string) (string, error) {
	name := copyName(address)
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 0; i < 1000; i++ {
		candidate := name
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d%s", stem, i, ext)
		}
		full := filepath.Join(dir, candidate)
		f, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if _, err := f.WriteString(source); err != nil {
			f.Close()
			return "", err
		}
		return full, f.Close()
	}
	return "", fmt.Errorf("%s: too many copies of %s", dir, name)
}

// copyName is the file name for a copy of the page at address.
func copyName(address string) string {
	p := address
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	if i := strings.Index(p, "://"); i >= 0 {
		p = p[i+3:]
		if j := strings.Index(p, "/"); j >= 0 {
			p = p[j:]
		} else {
			p = ""
		}
	}
	base := path.Base(strings.TrimRight(p, "/"))
	if base == "." || base == "/" || base == "" {
		base = "index.md"
	}
	var sb strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			sb.WriteRune(r)
		default:
			sb.WriteByte('_')
		}
	}
	name := strings.TrimLeft(sb.String(), ".") // no hidden files, no ".."
	switch strings.ToLower(path.Ext(name)) {
	case ".md", ".markdown", ".txt":
	default:
		name = strings.TrimSuffix(name, path.Ext(name)) + ".md"
	}
	if name == ".md" || name == "" {
		name = "page.md"
	}
	if len(name) > 100 {
		name = name[len(name)-100:]
	}
	return name
}

/** DownloadsDir is the person's Downloads folder, when it exists.
 *
 * Returns:
 *   string — the folder
 *   error — no home folder, or no Downloads folder in it
 *
 * Example:
 *   dir, err := edit.DownloadsDir() // "/home/me/Downloads"
 */
func DownloadsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "Downloads")
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return "", fmt.Errorf("no Downloads folder in %s", home)
	}
	return dir, nil
}
