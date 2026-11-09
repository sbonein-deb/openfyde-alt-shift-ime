package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sbonein-deb/openfyde-alt-shift-ime/internal/evdev"
)

func TestDefault(t *testing.T) {
	c := Default()
	if c.Combo.Key != evdev.KEY_SPACE || len(c.Combo.Mods) != 1 || c.Combo.Mods[0] != evdev.KEY_LEFTCTRL {
		t.Fatalf("unexpected default combo: %+v", c.Combo)
	}
	if c.Delay != 80*time.Millisecond || c.Rescan != 2*time.Second {
		t.Fatalf("unexpected defaults: %+v", c)
	}
}

func TestParse(t *testing.T) {
	text := `
# comment
combo = ctrl+shift+space
delay_ms=120
rescan_seconds=5
dry_run=true
verbose=true
`
	c, err := Parse(text, Default())
	if err != nil {
		t.Fatal(err)
	}
	if c.Combo.Key != evdev.KEY_SPACE || len(c.Combo.Mods) != 2 {
		t.Fatalf("combo = %+v", c.Combo)
	}
	if c.Delay != 120*time.Millisecond || c.Rescan != 5*time.Second {
		t.Fatalf("timings = %+v", c)
	}
	if !c.DryRun || !c.Verbose {
		t.Fatalf("booleans = %+v", c)
	}
}

func TestParseErrors(t *testing.T) {
	cases := []string{
		"combo",
		"=ctrl+space",
		"combo=",
		"combo=nope+space",
		"delay_ms=-1",
		"delay_ms=abc",
		"rescan_seconds=0",
		"dry_run=maybe",
		"verbose=2",
		"unknown=1",
	}
	for _, text := range cases {
		if _, err := Parse(text, Default()); err == nil {
			t.Fatalf("Parse(%q) unexpectedly succeeded", text)
		}
	}
}

func TestLoadMissingReturnsDefaults(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "missing.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Delay != Default().Delay {
		t.Fatalf("got %+v", c)
	}
}

func TestLoadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "alt-shift-ime.conf")
	if err := os.WriteFile(path, []byte("delay_ms=50\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Delay != 50*time.Millisecond {
		t.Fatalf("delay = %s", c.Delay)
	}
}
