// Package browse joins fetching, preparing and rendering a page.
package browse

import (
	"context"
	"fmt"
	"github.com/rsdoiel/cmarkbrowser/internal/local"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rsdoiel/cmarkbrowser/internal/debuglog"
	"github.com/rsdoiel/cmarkbrowser/internal/page"
	"github.com/rsdoiel/cmarkbrowser/internal/render"
)

/** Link is a numbered link with its destination resolved against the page.
 *
 * Fields:
 *   N (int) — the number shown after the link text
 *   Text (string) — the link text
 *   URL (string) — the absolute destination
 */
type Link struct {
	N    int
	Text string
	URL  string
	// Path is set for a path link in a local page (DR-0001): the absolute file
	// it names, which may be outside the site root; the browser checks that
	// when the link is followed. URL is then its file: address, internal only.
	Path string
}

/** View is a page ready to show.
 *
 * Fields:
 *   Page (*page.Page) — the fetched page
 *   Clean (string) — the body after HTML was stripped and links numbered
 *   Text (string) — the rendered, styled text, with a Sentinel before each
 *     link marker; use Display to get text fit to show
 *   Links ([]Link) — the numbered links, in reading order
 *   Missing ([]string) — {{page.KEY}} macros left in the text because the
 *     front matter has no such value
 *   Site (*local.Site) — the site root when the page is a local page of a site
 *     the person opened (OpenLocal); nil for a web page and for files such as
 *     the bookmarks file, whose links are never paths
 */
type View struct {
	Page    *page.Page
	Clean   string
	Text    string
	Links   []Link
	Missing []string
	Site    *local.Site

	lines map[int]int // link number -> line of its marker in Text
	log   *debuglog.Logger
}

/** Resolve resolves a link destination against the address of the page it is on.
 *
 * Parameters:
 *   base (string) — the page address
 *   dest (string) — the destination as written, possibly relative
 *
 * Returns:
 *   string — the absolute address
 *   error — wraps page.ErrData when either is not a valid address
 *
 * Example:
 *   u, _ := browse.Resolve("https://x.org/a/b.md", "c.md") // "https://x.org/a/c.md"
 */
func Resolve(base, dest string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("%w: %s", page.ErrData, err)
	}
	d, err := url.Parse(dest)
	if err != nil {
		return "", fmt.Errorf("%w: link %q: %s", page.ErrData, dest, err)
	}
	return b.ResolveReference(d).String(), nil
}

/** Open fetches a page, strips HTML, numbers its links and renders it.
 *
 * Parameters:
 *   ctx (context.Context) — cancellation and deadline
 *   addr (string) — the page address
 *   r (render.Renderer) — the renderer
 *   width (int) — columns to wrap to
 *
 * Returns:
 *   *View — the page ready to show
 *   error — from page.Fetch, or the renderer
 *
 * Example:
 *   v, err := browse.Open(ctx, "https://x.org/a.md", render.Glamour{}, 80)
 */
func Open(ctx context.Context, addr string, r render.Renderer, width int) (*View, error) {
	t0 := time.Now()
	p, err := page.Fetch(ctx, addr)
	if err != nil && page.IsHTML(err) {
		// An HTML page with no Markdown copy: a link that left the Markdown
		// web. Show it in reader view instead of stopping.
		p, err = page.FetchReader(ctx, addr)
	}
	if err != nil {
		return nil, err
	}
	return build(ctx, addr, p, nil, r, width, t0)
}

/** OpenFile shows a local Markdown file the way Open shows a page. It is only
 * for a file the person named (the start argument, the bookmarks file); links
 * are never opened through it.
 *
 * Parameters:
 *   ctx (context.Context) — carries the debug logger
 *   path (string) — the file
 *   r (render.Renderer) — the renderer
 *   width (int) — columns to wrap to
 *
 * Returns:
 *   *View — the page ready to show
 *   error — from page.ReadFile, or the renderer
 *
 * Example:
 *   v, err := browse.OpenFile(ctx, "bookmarks.md", render.Glamour{}, 80)
 */
func OpenFile(ctx context.Context, path string, r render.Renderer, width int) (*View, error) {
	t0 := time.Now()
	p, err := page.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return build(ctx, path, p, nil, r, width, t0)
}

/** OpenLocal shows a page of a local site (DR-0001): a file inside the site
 * root, read the way OpenFile reads one, with its links resolved as paths
 * relative to the page, or to the site root when they start with "/". It is
 * for the start page and for links the person follows inside the root; the
 * caller checks the root first.
 *
 * Parameters:
 *   ctx (context.Context) — carries the debug logger
 *   path (string) — the file, absolute
 *   site (*local.Site) — the site root
 *   r (render.Renderer) — the renderer
 *   width (int) — columns to wrap to
 *
 * Returns:
 *   *View — the page ready to show, with Site set
 *   error — from page.ReadFile, or the renderer
 *
 * Example:
 *   v, err := browse.OpenLocal(ctx, "/site/index.md", site, render.Glamour{}, 80)
 */
func OpenLocal(ctx context.Context, path string, site *local.Site, r render.Renderer, width int) (*View, error) {
	t0 := time.Now()
	p, err := page.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return build(ctx, path, p, site, r, width, t0)
}

// build expands, prepares and renders a fetched page and numbers its links. t0
// is when fetching began, for the debug log.
func build(ctx context.Context, addr string, p *page.Page, site *local.Site, r render.Renderer, width int, t0 time.Time) (*View, error) {
	log := debuglog.FromContext(ctx)
	t1 := time.Now()
	body, missing := p.Body, []string(nil)
	if !p.Reader { // {{page.NAME}} macros are Markdown-web front matter, not HTML text
		body, missing = render.Expand(p.Body, p.Meta)
	}
	t2 := time.Now()
	clean, links := render.Prepare(body)
	t3 := time.Now()
	text, err := r.Render(clean, width)
	if err != nil {
		return nil, fmt.Errorf("rendering %s: %w", addr, err)
	}
	t4 := time.Now()
	v := &View{Page: p, Clean: clean, Text: text, Missing: missing, Site: site, log: log}
	v.index()
	for _, l := range links {
		if site != nil {
			// A link in a local page is a path, or a web address; anything else
			// (a file: address too) stays in the text but is not followable.
			pagePath, _ := local.PathOf(p.URL)
			ll, err := site.ResolveLink(pagePath, l.Dest)
			switch {
			case err != nil:
				// kept, so that following it can say it is refused
				if strings.HasPrefix(strings.ToLower(strings.TrimSpace(l.Dest)), "file:") {
					v.Links = append(v.Links, Link{N: l.N, Text: l.Text, URL: strings.TrimSpace(l.Dest)})
				}
			case ll.URL != "":
				v.Links = append(v.Links, Link{N: l.N, Text: l.Text, URL: ll.URL})
			default:
				v.Links = append(v.Links, Link{N: l.N, Text: l.Text, URL: local.Address(ll.Path), Path: ll.Path})
			}
			continue
		}
		// A link that cannot be resolved stays in the text but is not followable.
		u, err := Resolve(p.URL, l.Dest)
		if err != nil {
			continue
		}
		v.Links = append(v.Links, Link{N: l.N, Text: l.Text, URL: u})
	}
	log.Event("open", "url", addr, "final", p.URL, "width", width,
		"fetch_ms", t1.Sub(t0), "expand_ms", t2.Sub(t1), "prepare_ms", t3.Sub(t2), "render_ms", t4.Sub(t3),
		"total_ms", time.Since(t0), "body_bytes", len(p.Body), "text_bytes", len(text),
		"lines", strings.Count(text, "\n")+1, "links", len(v.Links), "missing_keys", missing, "reader", p.Reader)
	return v, nil
}

// SetLogger sets where Rerender writes its debug events. Open sets it from
// its context; this is for a View made another way.
func (v *View) SetLogger(l *debuglog.Logger) { v.log = l }

/** Rerender renders the page again at a new width, for a resized terminal,
 * without fetching it.
 *
 * Parameters:
 *   r (render.Renderer) — the renderer
 *   width (int) — columns to wrap to
 *
 * Returns:
 *   error — the renderer's error; the old text is kept on failure
 *
 * Example:
 *   err := v.Rerender(render.Glamour{}, 100)
 */
func (v *View) Rerender(r render.Renderer, width int) error {
	t0 := time.Now()
	text, err := r.Render(v.Clean, width)
	if err != nil {
		return fmt.Errorf("rendering %s: %w", v.Page.URL, err)
	}
	v.Text = text
	v.index()
	v.log.Event("rerender", "url", v.Page.URL, "width", width, "ms", time.Since(t0), "lines", strings.Count(text, "\n")+1)
	return nil
}

// markerRE matches a link marker in rendered text: the sentinel, optionally
// with styling codes between it and the bracket, then [N].
var markerRE = regexp.MustCompile(render.Sentinel + `((?:\x1b\[[0-9;]*m)*)\[(\d+)\]`)

// index finds the line of each link marker in Text.
func (v *View) index() {
	v.lines = map[int]int{}
	for i, line := range strings.Split(v.Text, "\n") {
		for _, m := range markerRE.FindAllStringSubmatch(line, -1) {
			if n, err := strconv.Atoi(m[2]); err == nil {
				if _, seen := v.lines[n]; !seen {
					v.lines[n] = i
				}
			}
		}
	}
}

/** LineOf returns the line of Display's output where a link's marker is.
 *
 * Parameters:
 *   n (int) — the link number
 *
 * Returns:
 *   int — the zero-based line
 *   bool — false when the marker was not found in the rendered text
 *
 * Example:
 *   line, ok := v.LineOf(2)
 */
func (v *View) LineOf(n int) (int, bool) {
	l, ok := v.lines[n]
	return l, ok
}

/** Display returns the rendered text ready to show: sentinels removed and the
 * marker of the focused link in reverse video.
 *
 * Parameters:
 *   focus (int) — the number of the focused link, or a value below 1 for none
 *
 * Returns:
 *   string — the text
 *
 * Example:
 *   viewport.SetContent(v.Display(3))
 */
func (v *View) Display(focus int) string {
	return markerRE.ReplaceAllStringFunc(v.Text, func(m string) string {
		sub := markerRE.FindStringSubmatch(m)
		if n, _ := strconv.Atoi(sub[2]); n == focus {
			return sub[1] + "\x1b[7m[" + sub[2] + "]\x1b[27m"
		}
		return sub[1] + "[" + sub[2] + "]"
	})
}
