// Package render turns a Markdown body into text for the terminal. The
// Renderer interface exists so glamour can be replaced by our own goldmark
// AST renderer without touching the UI.
package render

import (
	"fmt"

	glamour "charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
)

/** Renderer renders Markdown to terminal text.
 *
 * Render parameters:
 *   markdown (string) — the page body, front matter already removed
 *   width (int) — the columns to wrap to
 *
 * Render returns:
 *   string — text with ANSI styling
 *   error — a rendering failure
 */
type Renderer interface {
	Render(markdown string, width int) (string, error)
}

// Glamour renders with glamour using a fixed style.
type Glamour struct {
	// Style is a glamour style name such as "dark", "light" or "notty".
	Style string
}

/** Render implements Renderer with glamour.
 *
 * Parameters:
 *   markdown (string) — the body
 *   width (int) — wrap width in columns
 *
 * Returns:
 *   string — styled text
 *   error — the glamour error
 *
 * Example:
 *   out, err := render.Glamour{Style: "notty"}.Render("# Hi", 80)
 */
func (g Glamour) Render(markdown string, width int) (string, error) {
	style := g.Style
	if style == "" {
		style = "dark"
	}
	cfg, ok := styles.DefaultStyles[style]
	if !ok {
		return "", fmt.Errorf("render: unknown style %q", style)
	}
	quiet := *cfg
	// Glamour prints each link's URL after its text. The status line and the
	// Links list give the address, so print nothing for it; the text stays a
	// terminal hyperlink.
	quiet.Link.Format = `{{ "" }}`
	// A picture left over (no alt text, or one Prepare could not place) shows
	// nothing: no "Image:" label and no address.
	quiet.ImageText.Format = ""
	quiet.Image.Format = `{{ "" }}`
	r, err := glamour.NewTermRenderer(glamour.WithStyles(quiet), glamour.WithWordWrap(width), glamour.WithInlineTableLinks(true))
	if err != nil {
		return "", err
	}
	return r.Render(markdown)
}
