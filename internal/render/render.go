// Package render turns a Markdown body into text for the terminal. The
// Renderer interface exists so glamour can be replaced by our own goldmark
// AST renderer without touching the UI.
package render

import (
	glamour "charm.land/glamour/v2"
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
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(width), glamour.WithInlineTableLinks(true))
	if err != nil {
		return "", err
	}
	return r.Render(markdown)
}
