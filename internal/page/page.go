// Package page fetches a Markdown page the way BonBon does and splits its
// front matter from its body.
package page

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Sentinels for the DR-0003 classes a fetch can fail in. Wrap with %w.
var (
	// ErrData marks content that was read and is wrong (exit 65).
	ErrData = errors.New("bad content")
	// ErrNoInput marks a page that is not there (exit 66).
	ErrNoInput = errors.New("no such page")
	// ErrUnavailable marks a server that cannot be reached (exit 69).
	ErrUnavailable = errors.New("server unavailable")
)

// errNotMarkdown marks a good response that is not a Markdown page. It is
// always wrapped with ErrData; Fetch uses it to decide to try a sibling.
var errNotMarkdown = errors.New("not Markdown")

// notMarkdown is the error for a good response that is not a page. It reads as
// one message, is an ErrData, and is also errNotMarkdown.
type notMarkdown struct{ msg string }

func (e *notMarkdown) Error() string        { return e.msg }
func (e *notMarkdown) Is(target error) bool { return target == errNotMarkdown }
func (e *notMarkdown) Unwrap() error        { return ErrData }

// Accept is the Accept header BonBon sends with every page request.
const Accept = "text/markdown, text/x-opml"

// maxBody caps what Fetch reads, so a hostile or broken server cannot fill memory.
const maxBody = 8 << 20

/** Page is a fetched Markdown page.
 *
 * Fields:
 *   URL (string) — the final address, after redirects
 *   Meta (map[string]any) — the front matter, empty when there is none
 *   Body (string) — the Markdown with the front matter removed
 */
type Page struct {
	URL  string
	Meta map[string]any
	Body string
}

// str returns the front matter value for key as a string, or "".
func (p *Page) str(key string) string {
	if v, ok := p.Meta[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

/** Title returns the front matter title, or "" when there is none.
 *
 * Returns:
 *   string — the title
 *
 * Example:
 *   p.Title() // "Hello"
 */
func (p *Page) Title() string { return p.str("title") }

/** URLNext returns the front matter urlNext, or "" when there is none.
 *
 * Returns:
 *   string — the address of the next page, possibly relative
 *
 * Example:
 *   p.URLNext() // "/two.md"
 */
func (p *Page) URLNext() string { return p.str("urlNext") }

/** URLPrev returns the front matter urlPrev, or "" when there is none.
 *
 * Returns:
 *   string — the address of the previous page, possibly relative
 *
 * Example:
 *   p.URLPrev() // "/zero.md"
 */
func (p *Page) URLPrev() string { return p.str("urlPrev") }

/** Parse splits optional YAML front matter (between two lines of three
 * dashes at the very start) from the Markdown body. A leading block that is
 * never closed is not front matter and stays in the body.
 *
 * Parameters:
 *   src ([]byte) — the page as served
 *
 * Returns:
 *   *Page — the page, with Meta never nil
 *   error — wraps ErrData when the front matter is not valid YAML
 *
 * Example:
 *   p, err := page.Parse([]byte("---\ntitle: Hi\n---\ntext\n"))
 */
func Parse(src []byte) (*Page, error) {
	p := &Page{Meta: map[string]any{}, Body: string(src)}
	s := strings.ReplaceAll(string(src), "\r\n", "\n")
	if !strings.HasPrefix(s, "---\n") {
		return p, nil
	}
	rest := s[4:]
	end := strings.Index("\n"+rest, "\n---\n")
	tail := ""
	if end < 0 {
		if strings.HasSuffix(rest, "\n---") {
			end, tail = len(rest)-4, ""
			end++
		} else {
			return p, nil
		}
	} else {
		tail = rest[end+4:]
		if end > 0 {
			end--
		}
	}
	block := rest[:max(end, 0)]
	if err := yaml.Unmarshal([]byte(block), &p.Meta); err != nil {
		return nil, fmt.Errorf("%w: front matter: %s", ErrData, err)
	}
	if p.Meta == nil {
		p.Meta = map[string]any{}
	}
	p.Body = tail
	return p, nil
}

/** IsMarkdown reports whether a response is a page by BonBon's rule: the type
 * is text/markdown, or it is text/plain or a generic binary type and the path
 * ends in .md or .markdown.
 *
 * Parameters:
 *   contentType (string) — the Content-Type header
 *   p (string) — the URL path
 *
 * Returns:
 *   bool — true for a page
 *
 * Example:
 *   page.IsMarkdown("text/plain", "/a.md") // true
 */
func IsMarkdown(contentType, p string) bool {
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mt = strings.ToLower(strings.TrimSpace(contentType))
	}
	if mt == "text/markdown" {
		return true
	}
	ext := strings.ToLower(path.Ext(p))
	if ext != ".md" && ext != ".markdown" {
		return false
	}
	return mt == "text/plain" || mt == "application/octet-stream" || mt == "binary/octet-stream" || mt == ""
}

/** Fetch gets a page with BonBon's Accept header and parses it. When the
 * server answers a link to an HTML page or a directory with something that is
 * not Markdown, it then tries the Markdown copy of the same page (a.html to
 * a.md, a directory to its index.md), the way BonBon's urlMirrorOf maps links.
 * The same address is always tried first, so a server that negotiates
 * Markdown on Accept (as some CDNs do for software agents) is respected.
 *
 * Parameters:
 *   ctx (context.Context) — cancellation and deadline
 *   rawurl (string) — an http or https address
 *
 * Returns:
 *   *Page — the page; its URL is the address that served it
 *   error — wraps ErrUnavailable (cannot connect), ErrNoInput (404 or 410),
 *     or ErrData (not Markdown, bad front matter, other bad status)
 *
 * Example:
 *   p, err := page.Fetch(ctx, "https://dave.writersweb.org/")
 */
func Fetch(ctx context.Context, rawurl string) (*Page, error) {
	p, err := fetchOne(ctx, rawurl)
	if err == nil || !errors.Is(err, errNotMarkdown) {
		return p, err
	}
	sib := markdownSibling(rawurl)
	if sib == "" {
		return nil, err
	}
	if p, err2 := fetchOne(ctx, sib); err2 == nil {
		return p, nil
	}
	return nil, err
}

// markdownSibling returns the address of the Markdown copy of an HTML page or
// directory address, or "" when the address is neither.
func markdownSibling(rawurl string) string {
	u, err := url.Parse(rawurl)
	if err != nil {
		return ""
	}
	switch ext := strings.ToLower(path.Ext(u.Path)); {
	case ext == ".html" || ext == ".htm":
		u.Path = strings.TrimSuffix(u.Path, path.Ext(u.Path)) + ".md"
	case strings.HasSuffix(u.Path, "/"):
		u.Path += "index.md"
	default:
		return ""
	}
	u.Fragment = ""
	return u.String()
}

// fetchOne gets one address and parses it as a page.
func fetchOne(ctx context.Context, rawurl string) (*Page, error) {
	u, err := url.Parse(rawurl)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("%w: %q is not an http or https address", ErrData, rawurl)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawurl, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrData, err)
	}
	req.Header.Set("Accept", Accept)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return nil, fmt.Errorf("%w: %s: %s", ErrNoInput, rawurl, resp.Status)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("%w: %s: %s", ErrData, rawurl, resp.Status)
	}
	if !IsMarkdown(resp.Header.Get("Content-Type"), resp.Request.URL.Path) {
		return nil, &notMarkdown{fmt.Sprintf("%s: %s is not Markdown (%s)", ErrData, rawurl, resp.Header.Get("Content-Type"))}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnavailable, err)
	}
	if len(body) > maxBody {
		return nil, fmt.Errorf("%w: %s is larger than %d bytes", ErrData, rawurl, maxBody)
	}
	p, err := Parse(bytes.TrimPrefix(body, []byte("\xef\xbb\xbf")))
	if err != nil {
		return nil, err
	}
	p.URL = resp.Request.URL.String()
	return p, nil
}
