// Command cmarkbrowser is a terminal browser for the Markdown web.
package main

import (
	"os"

	"github.com/rsdoiel/cmarkbrowser/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
