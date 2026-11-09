package evdev

import "testing"

func TestDecodeEvent(t *testing.T) {
	// One input_event: timestamp zero, EV_KEY, KEY_LEFTALT, value 1.
	b := make([]byte, EventSize)
	b[16] = EV_KEY
	b[18] = KEY_LEFTALT
	b[20] = 1
	e := DecodeEvent(b)
	if e.Type != EV_KEY || e.Code != KEY_LEFTALT || e.Value != 1 {
		t.Fatalf("decoded %+v", e)
	}
}

func TestKeyBits(t *testing.T) {
	d := &Device{}
	d.keyBits[KEY_LEFTALT/8] |= 1 << (KEY_LEFTALT % 8)
	if !d.HasKey(KEY_LEFTALT) {
		t.Fatal("expected KEY_LEFTALT")
	}
	if d.HasKey(KEY_LEFTSHIFT) {
		t.Fatal("did not expect KEY_LEFTSHIFT")
	}
}
