// Package keycombo parses and describes key combinations such as
// "ctrl+space" or "ctrl+shift+space".
package keycombo

import (
	"fmt"
	"strings"

	"github.com/sbonein-deb/openfyde-alt-shift-ime/internal/evdev"
)

// Combo is a modifier list plus a non-modifier key.
type Combo struct {
	Mods []uint16
	Key  uint16
}

var modifierNames = map[string]uint16{
	"ctrl":    evdev.KEY_LEFTCTRL,
	"control": evdev.KEY_LEFTCTRL,
	"shift":   evdev.KEY_LEFTSHIFT,
	"alt":     evdev.KEY_LEFTALT,
	"search":  evdev.KEY_LEFTMETA,
	"meta":    evdev.KEY_LEFTMETA,
	"super":   evdev.KEY_LEFTMETA,
	"win":     evdev.KEY_LEFTMETA,
}

var keyNames = map[string]uint16{
	"space":  evdev.KEY_SPACE,
	"tab":    evdev.KEY_TAB,
	"enter":  evdev.KEY_ENTER,
	"esc":    evdev.KEY_ESC,
	"escape": evdev.KEY_ESC,
	"a":      30,
	"b":      48,
	"c":      46,
	"d":      32,
	"e":      18,
	"f":      33,
	"g":      34,
	"h":      35,
	"i":      23,
	"j":      36,
	"k":      37,
	"l":      38,
	"m":      50,
	"n":      49,
	"o":      24,
	"p":      25,
	"q":      16,
	"r":      19,
	"s":      31,
	"t":      20,
	"u":      22,
	"v":      47,
	"w":      17,
	"x":      45,
	"y":      21,
	"z":      44,
}

// Parse parses a combination. Modifiers may appear in any order; the last
// non-modifier token is the key.
func Parse(s string) (Combo, error) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(s)), "+")
	var c Combo
	seenKey := false
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return Combo{}, fmt.Errorf("empty component in %q", s)
		}
		if mod, ok := modifierNames[part]; ok {
			if seenKey {
				return Combo{}, fmt.Errorf("modifier %q after key in %q", part, s)
			}
			c.Mods = append(c.Mods, mod)
			continue
		}
		key, ok := keyNames[part]
		if !ok {
			return Combo{}, fmt.Errorf("unknown key %q in %q", part, s)
		}
		if seenKey {
			return Combo{}, fmt.Errorf("more than one key in %q", s)
		}
		c.Key = key
		seenKey = true
	}
	if !seenKey {
		return Combo{}, fmt.Errorf("no key in %q", s)
	}
	return c, nil
}

// String returns a canonical representation.
func (c Combo) String() string {
	parts := make([]string, 0, len(c.Mods)+1)
	for _, m := range c.Mods {
		parts = append(parts, modifierLabel(m))
	}
	parts = append(parts, keyLabel(c.Key))
	return strings.Join(parts, "+")
}

func modifierLabel(code uint16) string {
	switch code {
	case evdev.KEY_LEFTCTRL:
		return "ctrl"
	case evdev.KEY_LEFTSHIFT:
		return "shift"
	case evdev.KEY_LEFTALT:
		return "alt"
	case evdev.KEY_LEFTMETA:
		return "search"
	default:
		return evdev.KeyName(code)
	}
}

func keyLabel(code uint16) string {
	switch code {
	case evdev.KEY_SPACE:
		return "space"
	case evdev.KEY_TAB:
		return "tab"
	case evdev.KEY_ENTER:
		return "enter"
	default:
		return strings.ToLower(evdev.KeyName(code))
	}
}
