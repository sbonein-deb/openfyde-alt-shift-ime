package evdev

import "fmt"

var keyNames = map[uint16]string{
	KEY_LEFTCTRL:  "KEY_LEFTCTRL",
	KEY_LEFTSHIFT: "KEY_LEFTSHIFT",
	KEY_LEFTALT:   "KEY_LEFTALT",
	KEY_SPACE:     "KEY_SPACE",
	KEY_LEFTMETA:  "KEY_LEFTMETA",
	KEY_TAB:       "KEY_TAB",
	16:            "KEY_Q",
	17:            "KEY_W",
	18:            "KEY_E",
	19:            "KEY_R",
	20:            "KEY_T",
	21:            "KEY_Y",
	22:            "KEY_U",
	23:            "KEY_I",
	24:            "KEY_O",
	25:            "KEY_P",
	30:            "KEY_A",
	31:            "KEY_S",
	32:            "KEY_D",
	33:            "KEY_F",
	34:            "KEY_G",
	35:            "KEY_H",
	36:            "KEY_J",
	37:            "KEY_K",
	38:            "KEY_L",
	44:            "KEY_Z",
	45:            "KEY_X",
	46:            "KEY_C",
	47:            "KEY_V",
	48:            "KEY_B",
	49:            "KEY_N",
	50:            "KEY_M",
	KEY_ENTER:     "KEY_ENTER",
	KEY_ESC:       "KEY_ESC",
	KEY_BACKSPACE: "KEY_BACKSPACE",
}

// KeyName returns a stable name for a key code.
func KeyName(code uint16) string {
	if name, ok := keyNames[code]; ok {
		return name
	}
	return fmt.Sprintf("KEY_%d", code)
}
