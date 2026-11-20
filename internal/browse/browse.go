// Package browse joins fetching, preparing and rendering a page.
package browse

import (
	"context"
	"fmt"
	"net/url"

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
}

/** View is a page ready to show.
 *
 * Fields:
 *   Page (*page.Page) — the fetched page
 *   Clean (string) — the body after HTML was stripped and links numbered
 *   Text (string) — the rendered, styled text
 *   Links ([]Link) — the numbered links, in reading order
 */
type View struct {
	Page  *page.Page
	Clean string
	Text  string
	Links []Link
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
	p, err := page.Fetch(ctx, addr)
	if err != nil {
		return nil, err
	}
	clean, links := render.Prepare(p.Body)
	text, err := r.Render(clean, width)
	if err != nil {
		return nil, fmt.Errorf("rendering %s: %w", addr, err)
	}
	v := &View{Page: p, Clean: clean, Text: text}
	for _, l := range links {
		// A link that cannot be resolved stays in the text but is not followable.
		u, err := Resolve(p.URL, l.Dest)
		if err != nil {
			continue
		}
		v.Links = append(v.Links, Link{N: l.N, Text: l.Text, URL: u})
	}
	return v, nil
}

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
	text, err := r.Render(v.Clean, width)
	if err != nil {
		return fmt.Errorf("rendering %s: %w", v.Page.URL, err)
	}
	v.Text = text
	return nil
}
