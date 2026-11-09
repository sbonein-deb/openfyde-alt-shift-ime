package chord

import (
	"testing"

	"github.com/sbonein-deb/openfyde-alt-shift-ime/internal/evdev"
)

type testEvent struct {
	code  uint16
	value int32
}

func feed(d *Detector, events ...testEvent) bool {
	trigger := false
	for _, e := range events {
		if d.Key(e.code, e.value) {
			trigger = true
		}
	}
	return trigger
}

func ev(code uint16, value int32) testEvent {
	return testEvent{code: code, value: value}
}

func TestAltThenShiftTriggersOnFinalRelease(t *testing.T) {
	d := &Detector{}
	if feed(d, ev(evdev.KEY_LEFTALT, 1), ev(evdev.KEY_LEFTSHIFT, 1)) {
		t.Fatal("must not trigger while modifiers are held")
	}
	if feed(d, ev(evdev.KEY_LEFTALT, 0)) {
		t.Fatal("must not trigger while shift is still held")
	}
	if !feed(d, ev(evdev.KEY_LEFTSHIFT, 0)) {
		t.Fatal("expected trigger on final release")
	}
}

func TestShiftThenAltTriggersOnFinalRelease(t *testing.T) {
	d := &Detector{}
	feed(d, ev(evdev.KEY_LEFTSHIFT, 1), ev(evdev.KEY_LEFTALT, 1))
	if !feed(d, ev(evdev.KEY_LEFTSHIFT, 0), ev(evdev.KEY_LEFTALT, 0)) {
		t.Fatal("expected trigger")
	}
}

func TestOtherKeyCancelsChord(t *testing.T) {
	d := &Detector{}
	if feed(d,
		ev(evdev.KEY_LEFTALT, 1),
		ev(evdev.KEY_LEFTSHIFT, 1),
		ev(evdev.KEY_TAB, 1),
		ev(evdev.KEY_TAB, 0),
		ev(evdev.KEY_LEFTALT, 0),
		ev(evdev.KEY_LEFTSHIFT, 0),
	) {
		t.Fatal("must not trigger after another key was pressed")
	}
}

func TestOtherKeyBeforeChordCancels(t *testing.T) {
	d := &Detector{}
	feed(d,
		ev(evdev.KEY_LEFTALT, 1),
		ev(evdev.KEY_A, 1),
		ev(evdev.KEY_A, 0),
		ev(evdev.KEY_LEFTSHIFT, 1),
	)
	if feed(d, ev(evdev.KEY_LEFTALT, 0), ev(evdev.KEY_LEFTSHIFT, 0)) {
		t.Fatal("must not trigger if a key was typed while alt was held")
	}
}

func TestSingleModifiersDoNotTrigger(t *testing.T) {
	d := &Detector{}
	if feed(d, ev(evdev.KEY_LEFTALT, 1), ev(evdev.KEY_LEFTALT, 0)) {
		t.Fatal("alt alone must not trigger")
	}
	if feed(d, ev(evdev.KEY_LEFTSHIFT, 1), ev(evdev.KEY_LEFTSHIFT, 0)) {
		t.Fatal("shift alone must not trigger")
	}
}

func TestAutorepeatIgnored(t *testing.T) {
	d := &Detector{}
	feed(d, ev(evdev.KEY_LEFTALT, 1), ev(evdev.KEY_LEFTSHIFT, 1))
	if feed(d, ev(evdev.KEY_LEFTALT, 2), ev(evdev.KEY_LEFTSHIFT, 2)) {
		t.Fatal("autorepeat must not trigger")
	}
	if !feed(d, ev(evdev.KEY_LEFTALT, 0), ev(evdev.KEY_LEFTSHIFT, 0)) {
		t.Fatal("expected trigger after repeats")
	}
}

func TestDetectorResetsAfterTrigger(t *testing.T) {
	d := &Detector{}
	feed(d, ev(evdev.KEY_LEFTALT, 1), ev(evdev.KEY_LEFTSHIFT, 1), ev(evdev.KEY_LEFTALT, 0), ev(evdev.KEY_LEFTSHIFT, 0))
	if !feed(d, ev(evdev.KEY_LEFTALT, 1), ev(evdev.KEY_LEFTSHIFT, 1), ev(evdev.KEY_LEFTALT, 0), ev(evdev.KEY_LEFTSHIFT, 0)) {
		t.Fatal("second chord must trigger")
	}
}
