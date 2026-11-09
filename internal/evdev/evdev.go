// Package evdev provides the small subset of Linux evdev functionality needed
// by alt-shift-ime: enumerating input devices, reading key events, and
// identifying keyboards. It deliberately does not grab devices.
package evdev

import (
	"encoding/binary"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const (
	EV_SYN = 0x00
	EV_KEY = 0x01

	KEY_ESC       = 1
	KEY_BACKSPACE = 14
	KEY_TAB       = 15
	KEY_ENTER     = 28
	KEY_LEFTCTRL  = 29
	KEY_LEFTSHIFT = 42
	KEY_LEFTALT   = 56
	KEY_SPACE     = 57
	KEY_LEFTMETA  = 125

	KEY_A = 30
	KEY_Z = 44

	KEY_MAX = 0x2ff

	// EventSize is sizeof(struct input_event) on 64-bit Linux.
	EventSize = 24
)

const (
	iocRead  = 2
	iocWrite = 1
)

func ioc(dir, typ, nr, size uintptr) uintptr {
	return dir<<30 | size<<16 | typ<<8 | nr
}

func eviocgname(size uintptr) uintptr { return ioc(iocRead, 'E', 0x06, size) }
func eviocgbit(ev, size uintptr) uintptr {
	return ioc(iocRead, 'E', 0x20+ev, size)
}
func eviocgid() uintptr { return ioc(iocRead, 'E', 0x02, 8) }

func ioctl(fd int, req uintptr, arg unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

// ID mirrors struct input_id.
type ID struct {
	BusType uint16
	Vendor  uint16
	Product uint16
	Version uint16
}

func (id ID) String() string {
	return fmt.Sprintf("bus=%04x vendor=%04x product=%04x version=%04x",
		id.BusType, id.Vendor, id.Product, id.Version)
}

// Event is a decoded struct input_event. Timestamps are not exposed because
// Phase 0 only needs ordering, not absolute timing.
type Event struct {
	Type  uint16
	Code  uint16
	Value int32
}

// DecodeEvent decodes one 24-byte input_event.
func DecodeEvent(b []byte) Event {
	return Event{
		Type:  binary.LittleEndian.Uint16(b[16:18]),
		Code:  binary.LittleEndian.Uint16(b[18:20]),
		Value: int32(binary.LittleEndian.Uint32(b[20:24])),
	}
}

// Device is an open evdev character device.
type Device struct {
	Path string
	FD   int
	Name string
	ID   ID

	eventBits [4]byte
	keyBits   [96]byte
}

// Open opens an evdev device read-only and non-blocking. It never grabs the
// device, so the normal desktop continues to receive all events.
func Open(path string) (*Device, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	d := &Device{Path: path, FD: fd}
	d.load()
	return d, nil
}

func (d *Device) load() {
	var name [256]byte
	if err := ioctl(d.FD, eviocgname(uintptr(len(name))), unsafe.Pointer(&name[0])); err == nil {
		d.Name = strings.TrimRight(string(name[:]), "\x00")
	}
	var id [8]byte
	if err := ioctl(d.FD, eviocgid(), unsafe.Pointer(&id[0])); err == nil {
		d.ID = ID{
			BusType: binary.LittleEndian.Uint16(id[0:2]),
			Vendor:  binary.LittleEndian.Uint16(id[2:4]),
			Product: binary.LittleEndian.Uint16(id[4:6]),
			Version: binary.LittleEndian.Uint16(id[6:8]),
		}
	}
	_ = ioctl(d.FD, eviocgbit(0, uintptr(len(d.eventBits))), unsafe.Pointer(&d.eventBits[0]))
	_ = ioctl(d.FD, eviocgbit(EV_KEY, uintptr(len(d.keyBits))), unsafe.Pointer(&d.keyBits[0]))
}

// Close closes the device.
func (d *Device) Close() error {
	if d == nil || d.FD < 0 {
		return nil
	}
	err := syscall.Close(d.FD)
	d.FD = -1
	return err
}

// HasEvent reports whether the device supports an event type.
func (d *Device) HasEvent(ev uint16) bool {
	if int(ev) >= len(d.eventBits)*8 {
		return false
	}
	return d.eventBits[ev/8]&(1<<(ev%8)) != 0
}

// HasKey reports whether the device supports a key code.
func (d *Device) HasKey(code uint16) bool {
	if int(code) >= len(d.keyBits)*8 {
		return false
	}
	return d.keyBits[code/8]&(1<<(code%8)) != 0
}

// KeyCount returns the number of key codes the device advertises.
func (d *Device) KeyCount() int {
	n := 0
	for _, b := range d.keyBits {
		for b != 0 {
			n += int(b & 1)
			b >>= 1
		}
	}
	return n
}

// IsKeyboard reports whether the device looks like a full keyboard. This is a
// heuristic: it requires EV_KEY plus the letter keys A and Z and Space.
func (d *Device) IsKeyboard() bool {
	return d.HasEvent(EV_KEY) &&
		d.HasKey(KEY_A) && d.HasKey(KEY_Z) && d.HasKey(KEY_SPACE)
}

// HasChordKeys reports whether the device advertises the physical keys this
// project listens for.
func (d *Device) HasChordKeys() bool {
	return d.HasEvent(EV_KEY) && d.HasKey(KEY_LEFTALT) && d.HasKey(KEY_LEFTSHIFT)
}

// Read reads raw events into buf. The caller must handle EAGAIN when the
// device is non-blocking.
func (d *Device) Read(buf []byte) (int, error) {
	n, err := syscall.Read(d.FD, buf)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// IsVirtualName reports whether name belongs to this project's uinput device.
func IsVirtualName(name string) bool {
	return strings.Contains(strings.ToLower(name), "alt-shift-ime")
}

// ListPaths returns /dev/input/event* paths sorted numerically.
func ListPaths() []string {
	paths, _ := filepath.Glob("/dev/input/event*")
	sort.Slice(paths, func(i, j int) bool {
		return eventNumber(paths[i]) < eventNumber(paths[j])
	})
	return paths
}

func eventNumber(path string) int {
	base := filepath.Base(path)
	n, err := strconv.Atoi(strings.TrimPrefix(base, "event"))
	if err != nil {
		return 1 << 30
	}
	return n
}

// ErrAgain reports whether err means "no events available right now".
func ErrAgain(err error) bool {
	return errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK)
}
