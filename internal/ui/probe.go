package ui

import (
	"context"
	"sync"
	"time"

	"github.com/rsdoiel/markdownweb/debuglog"
	"github.com/rsdoiel/markdownweb/page"
)

// probeTimeout bounds one probe, so a neighbour that never answers costs a few
// seconds once and is then remembered as unavailable.
const probeTimeout = 8 * time.Second

// maxProbes is the most candidates one key press asks about; past it the key
// says there is no page rather than walking a whole list of dead addresses.
const maxProbes = 12

/** prober remembers which addresses answered, so the arrows pass over a
 * neighbour with nothing at its address (markdownweb DR-0004 Q1) the way
 * BonBon does, and ask about each address once. It is safe for concurrent use.
 */
type prober struct {
	mu   sync.Mutex
	seen map[string]bool
}

func newProber() *prober { return &prober{seen: map[string]bool{}} }

/** answers reports whether something is at addr: a 2xx answer, after
 * redirects, through the shared fetcher (its pause, User-Agent and robots.txt
 * rules). fetch.Fetcher has no HEAD, so the request is a GET whose body is
 * closed unread. The answer, either way, is remembered.
 *
 * Parameters:
 *   ctx (context.Context) — cancels the probe
 *   addr (string)         — an http or https address
 *
 * Returns:
 *   bool — true when the address answered 2xx
 *
 * Example:
 *   ok := p.answers(ctx, "https://example.org/b.md")
 */
func (p *prober) answers(ctx context.Context, addr string) bool {
	p.mu.Lock()
	ok, known := p.seen[addr]
	p.mu.Unlock()
	if known {
		return ok
	}
	pctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	resp, err := page.Fetcher().Open(pctx, addr, "")
	if err == nil {
		resp.Body.Close()
		ok = resp.StatusCode >= 200 && resp.StatusCode < 300
	}
	debuglog.FromContext(ctx).Event("toc.probe", "addr", addr, "ok", ok)
	if ctx.Err() == nil { // a cancelled lookup says nothing about the address
		p.mu.Lock()
		p.seen[addr] = ok
		p.mu.Unlock()
	}
	return ok
}

/** firstThatAnswers returns the first candidate that answers, nearest first,
 * skipping the page itself, or "" when none does within maxProbes.
 *
 * Parameters:
 *   ctx (context.Context) — cancels the probes
 *   cands ([]string)      — candidate addresses, nearest first
 *   self (string)         — the page being shown, never a neighbour of itself
 *
 * Returns:
 *   string — the address, or "" for none
 *
 * Example:
 *   next := p.firstThatAnswers(ctx, res.After, page)
 */
func (p *prober) firstThatAnswers(ctx context.Context, cands []string, self string) string {
	n := 0
	for _, c := range cands {
		if c == self {
			continue
		}
		if n++; n > maxProbes {
			break
		}
		if p.answers(ctx, c) {
			return c
		}
	}
	return ""
}
