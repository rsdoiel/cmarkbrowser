// Package cli is the cmarkbrowser command line: option parsing, help and the
// exit codes of workspace DR-0003.
package cli

import (
	"errors"

	"github.com/rsdoiel/cmarkbrowser/internal/page"
)

// Exit codes, per workspace DR-0003. Only the classes this program can
// produce so far are listed; add the others when something returns them.
const (
	ExitOK          = 0
	ExitNegative    = 1
	ExitUsage       = 2
	ExitData        = 65
	ExitNoInput     = 66
	ExitUnavailable = 69
	ExitInternal    = 70
	ExitCantCreate  = 73
)

// ErrUsage marks a bad command line; wrap it with %w to exit 2.
var ErrUsage = errors.New("usage")

// ErrCantCreate marks output that cannot be created (a debug log file); wrap
// it with %w to exit 73.
var ErrCantCreate = errors.New("cannot create")

/** ExitCode is the one place an error becomes an exit code. It classifies by
 * sentinel with errors.Is, never by message text.
 *
 * Parameters:
 *   err (error) — nil, or an error wrapping one of this package's sentinels
 *
 * Returns:
 *   int — 0 for nil, the class code (2, 65, 66, 69, 73), or ExitInternal for an unclassified error
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
	case errors.Is(err, ErrCantCreate):
		return ExitCantCreate
	case errors.Is(err, page.ErrData):
		return ExitData
	case errors.Is(err, page.ErrNoInput):
		return ExitNoInput
	case errors.Is(err, page.ErrUnavailable):
		return ExitUnavailable
	}
	return ExitInternal
}
