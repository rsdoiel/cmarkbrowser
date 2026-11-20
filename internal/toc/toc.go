// Package toc reads a site's table of contents, the OPML outline BonBon looks
// for at /.well-known/toc.opml, and answers one question about it: what page
// comes before and what page after a given page. antennaApp writes the file
// (its DR-0013); Dave Winer's sites publish their own, so the parser reads
// what both write, including the ISO-8859-1 that Dave's declares.
//
// The rule for "before" and "after" is antenna's (DR-0013 5), which agrees with
// Dave Winer's explicit urlPrev and urlNext:
//
//   - Only outlines that are pages count: a url and no outlines under them. An
//     outline that has both (a month page with its days under it) is an index,
//     not a step in the sequence; an include outline is not a page.
//   - A group with no type and no url (antenna's Pages, Posts) is a list of its
//     own: arrows stop at its ends. Outlines of another kind (calendarYear and
//     the like, as in Dave's file) belong to the list around them.
//   - A list with calendar outlines in it is newest first, so the previous page
//     is the next line down (the older one) and the next page is the line
//     above. In any other list the previous page is the line above.
//
// What BonBon itself does with the lines of an outline, for its own arrows, is
// not documented and could not be tested.
package toc

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"strings"
)

// Entry is a page the outline lists.
type Entry struct {
	Text string
	URL  string // absolute
}

type node struct {
	text, typ, url string // url is absolute, "" when none
	include        bool   // type="include": url is another outline, not a page
	children       []*node
}

// Doc is a parsed table of contents.
type Doc struct {
	root *node
}

type xmlOutline struct {
	Text     string       `xml:"text,attr"`
	Type     string       `xml:"type,attr"`
	URL      string       `xml:"url,attr"`
	Children []xmlOutline `xml:"outline"`
}

type xmlOPML struct {
	XMLName xml.Name `xml:"opml"`
	Body    struct {
		Outlines []xmlOutline `xml:"outline"`
	} `xml:"body"`
}

/** Parse reads a table of contents. Addresses in it are resolved against base,
 * the address the file was fetched from. Encodings other than UTF-8 are read:
 * ISO-8859-1 (also latin1 and US-ASCII) and windows-1252.
 *
 * Parameters:
 *   src ([]byte) — the OPML document
 *   base (string) — the address it came from
 *
 * Returns:
 *   *Doc — the outline; an outline with nothing in it is a Doc with no entries
 *   error — the document is not well-formed XML, is not an opml document, or
 *     declares an encoding that is not supported
 *
 * Example:
 *   d, err := toc.Parse(body, "https://example.org/.well-known/toc.opml")
 */
func Parse(src []byte, base string) (*Doc, error) {
	b, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("toc: %q is not an address: %w", base, err)
	}
	var o xmlOPML
	dec := xml.NewDecoder(bytes.NewReader(src))
	dec.CharsetReader = charsetReader
	if err := dec.Decode(&o); err != nil {
		return nil, fmt.Errorf("toc: %w", err)
	}
	var build func(xs []xmlOutline) []*node
	build = func(xs []xmlOutline) []*node {
		var out []*node
		for _, x := range xs {
			n := &node{text: x.Text, typ: x.Type, include: strings.EqualFold(x.Type, "include")}
			if a := strings.TrimSpace(x.URL); a != "" {
				if ref, err := url.Parse(a); err == nil {
					n.url = b.ResolveReference(ref).String()
				}
			}
			n.children = build(x.Children)
			out = append(out, n)
		}
		return out
	}
	return &Doc{root: &node{children: build(o.Body.Outlines)}}, nil
}

// leaf is a page of the outline with where it sits.
type leaf struct {
	Entry
	scope   *node // the list it belongs to
	reverse bool  // newest first
	key     string
}

func isCalendar(typ string) bool {
	switch strings.ToLower(typ) {
	case "calendaryear", "calendarmonth", "calendarday":
		return true
	}
	return false
}

// leaves lists the pages of the outline in document order.
func (d *Doc) leaves() []leaf {
	var out []leaf
	var walk func(n, scope *node, reverse bool)
	walk = func(n, scope *node, reverse bool) {
		for _, c := range n.children {
			childScope, childReverse := scope, reverse
			if c.typ == "" && c.url == "" && !c.include && len(c.children) > 0 {
				childScope, childReverse = c, false // a plain group is a list of its own
			}
			if isCalendar(c.typ) {
				childReverse = true
			}
			if c.url != "" && len(c.children) == 0 && !c.include {
				out = append(out, leaf{Entry: Entry{Text: c.text, URL: c.url}, scope: childScope, reverse: childReverse, key: Key(c.url)})
			}
			walk(c, childScope, childReverse)
		}
	}
	walk(d.root, d.root, false)
	return out
}

// Entries returns the pages the outline lists, in document order.
func (d *Doc) Entries() []Entry {
	var out []Entry
	for _, l := range d.leaves() {
		out = append(out, l.Entry)
	}
	return out
}

// Includes returns the addresses of the include outlines, which are other
// outlines to read (not pages).
func (d *Doc) Includes() []string {
	var out []string
	var walk func(n *node)
	walk = func(n *node) {
		for _, c := range n.children {
			if c.include && c.url != "" {
				out = append(out, c.url)
			}
			walk(c)
		}
	}
	walk(d.root)
	return out
}

/** Splice makes the outlines of child part of d, in place of the include
 * outline that names it: the include becomes a plain group holding them, so
 * its pages are one list.
 *
 * Parameters:
 *   includeURL (string) — the address of the include outline (as in Includes)
 *   child (*Doc) — the outline that address held
 *
 * Example:
 *   for _, inc := range d.Includes() { d.Splice(inc, fetched[inc]) }
 */
func (d *Doc) Splice(includeURL string, child *Doc) {
	var walk func(n *node)
	walk = func(n *node) {
		for _, c := range n.children {
			if c.include && c.url == includeURL {
				c.children = append([]*node(nil), child.root.children...)
				c.include, c.url, c.typ = false, "", ""
				continue
			}
			walk(c)
		}
	}
	walk(d.root)
}

// Key is the form of an address two outlines and a browser agree on: scheme and
// host in lower case, no fragment, a folder as its index.md, and .html as .md,
// since a page is listed by its Markdown and may be shown by its HTML address.
func Key(address string) string {
	u, err := url.Parse(address)
	if err != nil {
		return address
	}
	p := u.Path
	switch {
	case p == "" || strings.HasSuffix(p, "/"):
		p += "index.md"
	case strings.HasSuffix(strings.ToLower(p), ".html"):
		p = p[:len(p)-5] + ".md"
	case strings.HasSuffix(strings.ToLower(p), ".htm"):
		p = p[:len(p)-4] + ".md"
	}
	k := strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host) + p
	if u.RawQuery != "" {
		k += "?" + u.RawQuery
	}
	return k
}

/** Neighbours returns the page before and the page after a page in the
 * outline, by the rule in the package comment. An empty string means there is
 * none: the page is first or last in its list.
 *
 * Parameters:
 *   page (string) — the address being shown; its .html, folder and fragment
 *     forms match the Markdown address the outline lists
 *
 * Returns:
 *   string — the previous page's address, or ""
 *   string — the next page's address, or ""
 *   bool — false when the outline does not list the page at all
 *
 * Example:
 *   prev, next, ok := d.Neighbours("https://example.org/blog/2026/11/19/a.html")
 */
func (d *Doc) Neighbours(page string) (prev, next string, ok bool) {
	want := Key(page)
	all := d.leaves()
	at := -1
	for i, l := range all {
		if l.key == want {
			at = i
			break
		}
	}
	if at < 0 {
		return "", "", false
	}
	// the list the page is in, in document order
	var list []leaf
	pos := 0
	for i, l := range all {
		if l.scope == all[at].scope {
			if i == at {
				pos = len(list)
			}
			list = append(list, l)
		}
	}
	before, after := "", ""
	if pos > 0 {
		before = list[pos-1].URL
	}
	if pos+1 < len(list) {
		after = list[pos+1].URL
	}
	if all[at].reverse {
		return after, before, true // newest first: previous is the older, the line below
	}
	return before, after, true
}

// charsetReader reads the encodings a table of contents is found in.
func charsetReader(label string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "utf-8", "utf8":
		return input, nil
	case "iso-8859-1", "iso8859-1", "latin1", "latin-1", "l1", "us-ascii", "ascii":
		return recode(input, nil)
	case "windows-1252", "cp1252":
		return recode(input, &cp1252)
	}
	return nil, fmt.Errorf("unsupported encoding %q", label)
}

// recode turns single-byte text into UTF-8. high maps the bytes 0x80 to 0x9f
// (windows-1252 differs from ISO-8859-1 only there); nil keeps them as they are.
func recode(r io.Reader, high *[32]rune) (io.Reader, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	for _, c := range b {
		switch {
		case c < 0x80:
			out.WriteByte(c)
		case high != nil && c < 0xa0:
			out.WriteRune(high[c-0x80])
		default:
			out.WriteRune(rune(c))
		}
	}
	return &out, nil
}

// cp1252 holds windows-1252's characters for the bytes 0x80 to 0x9f.
var cp1252 = [32]rune{
	0x20AC, 0x0081, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021, 0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0x008D, 0x017D, 0x008F,
	0x0090, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014, 0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0x009D, 0x017E, 0x0178,
}
