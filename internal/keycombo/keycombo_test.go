package keycombo

import (
	"testing"

	"github.com/sbonein-deb/openfyde-alt-shift-ime/internal/evdev"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		want Combo
	}{
		{"ctrl+space", Combo{Mods: []uint16{evdev.KEY_LEFTCTRL}, Key: evdev.KEY_SPACE}},
		{"Ctrl+Shift+Space", Combo{Mods: []uint16{evdev.KEY_LEFTCTRL, evdev.KEY_LEFTSHIFT}, Key: evdev.KEY_SPACE}},
		{"search+space", Combo{Mods: []uint16{evdev.KEY_LEFTMETA}, Key: evdev.KEY_SPACE}},
		{"space", Combo{Key: evdev.KEY_SPACE}},
	}
	for _, tc := range cases {
		got, err := Parse(tc.in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.in, err)
		}
		if got.Key != tc.want.Key || len(got.Mods) != len(tc.want.Mods) {
			t.Fatalf("Parse(%q) = %+v, want %+v", tc.in, got, tc.want)
		}
		for i := range got.Mods {
			if got.Mods[i] != tc.want.Mods[i] {
				t.Fatalf("Parse(%q) mods = %v, want %v", tc.in, got.Mods, tc.want.Mods)
			}
		}
	}
}

func TestParseRejects(t *testing.T) {
	for _, in := range []string{"", "ctrl", "ctrl+", "ctrl+space+tab", "nope+space", "space+ctrl"} {
		if _, err := Parse(in); err == nil {
			t.Fatalf("Parse(%q) unexpectedly succeeded", in)
		}
	}
}

func TestString(t *testing.T) {
	c, err := Parse("ctrl+shift+space")
	if err != nil {
		t.Fatal(err)
	}
	if got := c.String(); got != "ctrl+shift+space" {
		t.Fatalf("String() = %q", got)
	}
}
