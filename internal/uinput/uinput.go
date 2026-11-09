// Package uinput creates a virtual keyboard and injects key combinations.
//
// The virtual device is persistent for the lifetime of the Injector, which
// avoids the 100-200 ms device-creation latency on every trigger and gives
// ChromeOS time to notice the device once at startup.
package uinput

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"
	"syscall"
	"time"

	"github.com/sbonein-deb/openfyde-alt-shift-ime/internal/evdev"
	"github.com/sbonein-deb/openfyde-alt-shift-ime/internal/keycombo"
)

const (
	evSyn      = 0x00
	evKey      = 0x01
	synReport  = 0
	busUSB     = 0x03
	vendorID   = 0x1209
	productID  = 0x0001
	deviceVer  = 0x0001
	deviceName = "alt-shift-ime virtual keyboard"
)

const (
	iocRead  = 2
	iocWrite = 1
)

func ioc(dir, typ, nr, size uintptr) uintptr {
	return dir<<30 | size<<16 | typ<<8 | nr
}

var (
	uiSetEvBit   = ioc(iocWrite, 'U', 100, 4)
	uiSetKeyBit  = ioc(iocWrite, 'U', 101, 4)
	uiDevCreate  = ioc(0, 'U', 1, 0)
	uiDevDestroy = ioc(0, 'U', 2, 0)
)

// ioctl passes arg by value. This is correct for uinput's UI_SET_* ioctls:
// despite being declared with _IOW, the kernel reads the integer argument
// directly from the ioctl argument rather than dereferencing a pointer.
func ioctl(fd int, req uintptr, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, arg)
	if errno != 0 {
		return errno
	}
	return nil
}

type inputID struct {
	BusType uint16
	Vendor  uint16
	Product uint16
	Version uint16
}

// uinputUserDev mirrors struct uinput_user_dev. binary.Write emits it without
// padding, which is required by the kernel's legacy uinput ABI.
type uinputUserDev struct {
	Name         [80]byte
	ID           inputID
	FFEffectsMax uint32
	Absmax       [64]int32
	Absmin       [64]int32
	Absfuzz      [64]int32
	Absflat      [64]int32
}

type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

// Injector owns one virtual keyboard.
type Injector struct {
	fd     int
	mu     sync.Mutex
	closed bool
}

// New creates the virtual keyboard. The caller should wait a short time before
// injecting so udev and ChromeOS can notice the device.
func New() (*Injector, error) {
	fd, err := syscall.Open("/dev/uinput", syscall.O_WRONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open /dev/uinput: %w", err)
	}
	fail := func(err error) (*Injector, error) {
		syscall.Close(fd)
		return nil, err
	}

	if err := ioctl(fd, uiSetEvBit, uintptr(evKey)); err != nil {
		return fail(fmt.Errorf("UI_SET_EVBIT(EV_KEY): %w", err))
	}
	// ChromeOS classifies a device as a full keyboard only if it advertises
	// every key from KEY_ESC (1) through KEY_D (32). Advertise the standard
	// 1..127 keyboard range so the virtual device is unambiguously a
	// keyboard. We only ever emit Ctrl, Space, and (for future combos) the
	// other modifiers.
	for key := uint16(1); key <= 127; key++ {
		if err := ioctl(fd, uiSetKeyBit, uintptr(key)); err != nil {
			return fail(fmt.Errorf("UI_SET_KEYBIT(%s): %w", evdev.KeyName(key), err))
		}
	}

	dev := uinputUserDev{
		ID: inputID{
			BusType: busUSB,
			Vendor:  vendorID,
			Product: productID,
			Version: deviceVer,
		},
	}
	copy(dev.Name[:], deviceName)
	var devBuf bytes.Buffer
	if err := binary.Write(&devBuf, binary.LittleEndian, &dev); err != nil {
		return fail(fmt.Errorf("encode uinput_user_dev: %w", err))
	}
	if err := writeAll(fd, devBuf.Bytes()); err != nil {
		return fail(fmt.Errorf("write uinput_user_dev: %w", err))
	}
	if err := ioctl(fd, uiDevCreate, 0); err != nil {
		return fail(fmt.Errorf("UI_DEV_CREATE: %w", err))
	}
	return &Injector{fd: fd}, nil
}

// Name returns the virtual device name used to exclude it from monitoring.
func (i *Injector) Name() string { return deviceName }

// Inject presses the combination's modifiers, taps the key, and releases the
// modifiers in reverse order.
func (i *Injector) Inject(c keycombo.Combo) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return fmt.Errorf("injector is closed")
	}

	for _, mod := range c.Mods {
		if err := i.emit(evKey, mod, 1); err != nil {
			return err
		}
	}
	if err := i.emit(evKey, c.Key, 1); err != nil {
		return err
	}
	if err := i.emit(evKey, c.Key, 0); err != nil {
		return err
	}
	for n := len(c.Mods) - 1; n >= 0; n-- {
		if err := i.emit(evKey, c.Mods[n], 0); err != nil {
			return err
		}
	}
	return nil
}

func (i *Injector) emit(typ, code uint16, value int32) error {
	if err := i.write(inputEvent{Type: typ, Code: code, Value: value}); err != nil {
		return err
	}
	if err := i.write(inputEvent{Type: evSyn, Code: synReport}); err != nil {
		return err
	}
	// A tiny gap keeps the injected sequence looking like real typing and
	// avoids coalescing in some consumers.
	time.Sleep(5 * time.Millisecond)
	return nil
}

func (i *Injector) write(e inputEvent) error {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, &e); err != nil {
		return err
	}
	return writeAll(i.fd, buf.Bytes())
}

func writeAll(fd int, b []byte) error {
	for len(b) > 0 {
		n, err := syscall.Write(fd, b)
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}

// Close destroys the virtual device.
func (i *Injector) Close() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return nil
	}
	i.closed = true
	_ = ioctl(i.fd, uiDevDestroy, 0)
	return syscall.Close(i.fd)
}
