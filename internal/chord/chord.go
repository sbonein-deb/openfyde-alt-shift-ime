// Package chord implements the left Alt + left Shift detector.
//
// The detector is deliberately small and free of I/O so it can be unit tested.
// It triggers only after both modifiers have been pressed together and then
// released, and only if no other key was pressed during the chord. This
// matches the way ChromeOS tracks modifier state globally: injecting
// Ctrl+Space while Alt or Shift is still down would be seen as
// Ctrl+Alt+Shift+Space and would not match the accelerator.
package chord

import "github.com/sbonein-deb/openfyde-alt-shift-ime/internal/evdev"

// Detector tracks the physical left Alt and left Shift keys.
type Detector struct {
	alt   bool
	shift bool
	chord bool
	other bool
}

// Reset clears all state.
func (d *Detector) Reset() {
	d.alt = false
	d.shift = false
	d.chord = false
	d.other = false
}

// Active reports whether either trigger modifier is currently down.
func (d *Detector) Active() bool {
	return d.alt || d.shift
}

// Key consumes one EV_KEY event. It returns true exactly once, when the chord
// has been completed and both modifiers have been released without any other
// key being pressed in between.
func (d *Detector) Key(code uint16, value int32) bool {
	if value == 2 { // autorepeat
		return false
	}
	pressed := value == 1
	released := value == 0

	switch code {
	case evdev.KEY_LEFTALT:
		if pressed {
			if !d.alt && !d.shift {
				d.other = false
			}
			d.alt = true
			if d.shift {
				d.chord = true
			}
		} else if released {
			d.alt = false
			return d.finishIfReleased()
		}
	case evdev.KEY_LEFTSHIFT:
		if pressed {
			if !d.alt && !d.shift {
				d.other = false
			}
			d.shift = true
			if d.alt {
				d.chord = true
			}
		} else if released {
			d.shift = false
			return d.finishIfReleased()
		}
	default:
		if pressed && (d.alt || d.shift) {
			d.other = true
		}
	}
	return false
}

func (d *Detector) finishIfReleased() bool {
	if d.alt || d.shift {
		return false
	}
	trigger := d.chord && !d.other
	d.chord = false
	d.other = false
	return trigger
}
