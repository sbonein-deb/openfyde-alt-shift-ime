package uinput

import (
	"testing"
	"unsafe"
)

func TestUserDevABI(t *testing.T) {
	// struct uinput_user_dev is 1116 bytes on 64-bit Linux. A mismatch here
	// would corrupt the legacy uinput setup handshake.
	if got := unsafe.Sizeof(uinputUserDev{}); got != 1116 {
		t.Fatalf("uinput_user_dev size = %d, want 1116", got)
	}
}

func TestInputEventABI(t *testing.T) {
	if got := unsafe.Sizeof(inputEvent{}); got != 24 {
		t.Fatalf("input_event size = %d, want 24", got)
	}
}
