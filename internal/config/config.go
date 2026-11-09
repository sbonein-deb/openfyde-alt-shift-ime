// Package config loads the small key=value configuration used by
// alt-shift-ime.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sbonein-deb/openfyde-alt-shift-ime/internal/keycombo"
)

// Config holds the daemon settings.
type Config struct {
	Combo   keycombo.Combo
	Delay   time.Duration
	Rescan  time.Duration
	DryRun  bool
	Verbose bool
}

// Default returns the built-in defaults.
func Default() Config {
	combo, _ := keycombo.Parse("ctrl+space")
	return Config{
		Combo:  combo,
		Delay:  80 * time.Millisecond,
		Rescan: 2 * time.Second,
	}
}

// Load reads path. A missing file is not an error: it returns the defaults so
// the daemon can run before the installer has created a config.
func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read %s: %w", path, err)
	}
	return Parse(string(data), cfg)
}

// Parse applies key=value lines on top of base. Blank lines and lines starting
// with '#' are ignored.
func Parse(text string, base Config) (Config, error) {
	cfg := base
	for i, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return cfg, fmt.Errorf("line %d: expected key=value", i+1)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return cfg, fmt.Errorf("line %d: empty key", i+1)
		}
		if value == "" {
			return cfg, fmt.Errorf("line %d: empty value for %q", i+1, key)
		}

		switch key {
		case "combo":
			combo, err := keycombo.Parse(value)
			if err != nil {
				return cfg, fmt.Errorf("line %d: %w", i+1, err)
			}
			cfg.Combo = combo
		case "delay_ms":
			n, err := strconv.Atoi(value)
			if err != nil || n < 0 {
				return cfg, fmt.Errorf("line %d: invalid delay_ms %q", i+1, value)
			}
			cfg.Delay = time.Duration(n) * time.Millisecond
		case "rescan_seconds":
			n, err := strconv.Atoi(value)
			if err != nil || n <= 0 {
				return cfg, fmt.Errorf("line %d: invalid rescan_seconds %q", i+1, value)
			}
			cfg.Rescan = time.Duration(n) * time.Second
		case "dry_run":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return cfg, fmt.Errorf("line %d: invalid dry_run %q", i+1, value)
			}
			cfg.DryRun = b
		case "verbose":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return cfg, fmt.Errorf("line %d: invalid verbose %q", i+1, value)
			}
			cfg.Verbose = b
		default:
			return cfg, fmt.Errorf("line %d: unknown key %q", i+1, key)
		}
	}
	return cfg, nil
}
