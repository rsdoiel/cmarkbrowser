// Package config reads cmarkbrowser's configuration file: cmarkbrowser.yaml in
// the user's configuration directory (~/.config/cmarkbrowser on Linux, with
// $XDG_CONFIG_HOME honoured). A missing file is fine; a present but wrong one
// is ErrConfig (exit 78, workspace DR-0003).
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"unicode"

	"gopkg.in/yaml.v3"
)

// ErrConfig marks a configuration file that is present and wrong; wrap it with
// %w to exit 78.
var ErrConfig = errors.New("configuration")

// FileName is the configuration file's name inside the configuration directory.
const FileName = "cmarkbrowser.yaml"

/** Config is what the configuration file holds. Keys the program does not know
 * yet are ignored, so a file written for a later version still loads.
 *
 * Fields:
 *   UserAgent (string) — the User-Agent sent with every request (key user-agent);
 *     empty means cmarkbrowser/VERSION
 *
 * Example:
 *   c, err := config.Load()
 *   if err == nil && c.UserAgent != "" { ... }
 */
type Config struct {
	UserAgent string `yaml:"user-agent"`
}

/** Path returns where the configuration file is, whether or not it exists.
 *
 * Returns:
 *   string — the file's path, under os.UserConfigDir ($XDG_CONFIG_HOME, else ~/.config, on Linux)
 *   error  — the configuration directory cannot be found
 *
 * Example:
 *   p, _ := config.Path() // "/home/me/.config/cmarkbrowser/cmarkbrowser.yaml"
 */
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cmarkbrowser", FileName), nil
}

/** Load reads the configuration file.
 *
 * Returns:
 *   Config — the settings; the zero value when there is no file
 *   error  — wraps ErrConfig when the file is unreadable, not a YAML mapping,
 *     has a value of the wrong type, or a User-Agent with a control character
 *
 * Example:
 *   c, err := config.Load()
 */
func Load() (Config, error) {
	var c Config
	p, err := Path()
	if err != nil {
		return c, nil // no home directory, so no file to read
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, fmt.Errorf("%w: %s: %s", ErrConfig, p, err)
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("%w: %s: %s", ErrConfig, p, err)
	}
	for _, r := range c.UserAgent {
		if unicode.IsControl(r) {
			return Config{}, fmt.Errorf("%w: %s: user-agent has a control character", ErrConfig, p)
		}
	}
	return c, nil
}
