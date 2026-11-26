package cli

import (
	"os"
	"testing"
)

// TestMain keeps every test away from the real configuration directory.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "cmarkbrowser-config")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_CONFIG_HOME", dir)
	os.Setenv("HOME", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
