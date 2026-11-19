// Package cli is the cmarkbrowser command line: option parsing, help and the
// exit codes of workspace DR-0003.
package cli

import "errors"

// Exit codes, per workspace DR-0003. Only the classes this program can
// produce so far are listed; add the others when something returns them.
const (
	ExitOK       = 0
	ExitNegative = 1
	ExitUsage    = 2
	ExitInternal = 70
)

// ErrUsage marks a bad command line; wrap it with %w to exit 2.
var ErrUsage = errors.New("usage")

// ErrNotImplemented marks a request for something the scaffold does not do
// yet. It is a bug-class condition (exit 70), not a user error, and goes away
// when browsing is built.
var ErrNotImplemented = errors.New("not implemented")

/** ExitCode is the one place an error becomes an exit code. It classifies by
 * sentinel with errors.Is, never by message text.
 *
 * Parameters:
 *   err (error) — nil, or an error wrapping one of this package's sentinels
 *
 * Returns:
 *   int — 0 for nil, the class code, or ExitInternal for an unclassified error
 *
 * Example:
 *   cli.ExitCode(fmt.Errorf("%w: no URL", cli.ErrUsage)) // 2
 */
func ExitCode(err error) int {
	switch {
	case err == nil:
		return ExitOK
	case errors.Is(err, ErrUsage):
		return ExitUsage
	}
	return ExitInternal
}
