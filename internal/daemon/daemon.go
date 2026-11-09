// Package daemon watches physical keyboards and injects a configured key
// combination when left Alt and left Shift are released cleanly.
package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/sbonein-deb/openfyde-alt-shift-ime/internal/chord"
	"github.com/sbonein-deb/openfyde-alt-shift-ime/internal/evdev"
	"github.com/sbonein-deb/openfyde-alt-shift-ime/internal/keycombo"
)

// Injector is the minimal interface the daemon needs. Tests and dry-run mode
// can provide a no-op implementation.
type Injector interface {
	Inject(keycombo.Combo) error
	Name() string
}

// Options controls one daemon run.
type Options struct {
	Combo         keycombo.Combo
	Delay         time.Duration
	Rescan        time.Duration
	DryRun        bool
	Verbose       bool
	RequireDevice bool
	Injector      Injector
	Logger        *log.Logger
}

// Run watches keyboards until ctx is canceled. It does not create or close the
// Injector; the caller owns its lifetime.
func Run(ctx context.Context, opts Options) error {
	logger := opts.Logger
	if logger == nil {
		logger = log.Default()
	}
	if opts.Delay < 0 {
		return fmt.Errorf("delay must not be negative")
	}
	if opts.Rescan <= 0 {
		opts.Rescan = 2 * time.Second
	}

	epfd, err := syscall.EpollCreate1(0)
	if err != nil {
		return fmt.Errorf("epoll_create1: %w", err)
	}
	defer syscall.Close(epfd)

	devices := make(map[int]*evdev.Device)
	watchedPaths := make(map[string]bool)
	defer func() {
		for _, d := range devices {
			d.Close()
		}
	}()

	detector := &chord.Detector{}

	var pendingMu sync.Mutex
	var pending *time.Timer
	var pendingGen uint64

	cancelPending := func() {
		pendingMu.Lock()
		defer pendingMu.Unlock()
		pendingGen++
		if pending != nil {
			pending.Stop()
			pending = nil
		}
	}

	schedule := func(deviceName string) {
		pendingMu.Lock()
		defer pendingMu.Unlock()
		pendingGen++
		gen := pendingGen
		if pending != nil {
			pending.Stop()
		}
		pending = time.AfterFunc(opts.Delay, func() {
			pendingMu.Lock()
			if gen != pendingGen {
				pendingMu.Unlock()
				return
			}
			pending = nil
			pendingMu.Unlock()

			if opts.Injector == nil {
				logger.Printf("TRIGGER from %q: %s (dry-run, not injected)", deviceName, opts.Combo)
				return
			}
			logger.Printf("TRIGGER from %q: injecting %s", deviceName, opts.Combo)
			if err := opts.Injector.Inject(opts.Combo); err != nil {
				logger.Printf("inject failed: %v", err)
				return
			}
			logger.Printf("injected %s", opts.Combo)
		})
	}

	removeDevice := func(fd int) {
		d := devices[fd]
		if d == nil {
			return
		}
		delete(devices, fd)
		delete(watchedPaths, d.Path)
		d.Close()
		// If a modifier was held on the removed device, do not leave the
		// detector stuck in a half-pressed state.
		detector.Reset()
		cancelPending()
		logger.Printf("stopped watching %s %q; detector reset", d.Path, d.Name)
	}

	addDevice := func(path string) {
		if watchedPaths[path] {
			return
		}
		d, err := evdev.Open(path)
		if err != nil {
			if opts.Verbose {
				logger.Printf("skip %s: %v", path, err)
			}
			return
		}
		if evdev.IsVirtualName(d.Name) ||
			(opts.Injector != nil && d.Name == opts.Injector.Name()) {
			d.Close()
			return
		}
		if !d.HasChordKeys() {
			d.Close()
			if opts.Verbose {
				logger.Printf("skip %s %q: no left Alt/Shift keys", path, d.Name)
			}
			return
		}
		event := syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(d.FD)}
		if err := syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, d.FD, &event); err != nil {
			logger.Printf("epoll add %s: %v", path, err)
			d.Close()
			return
		}
		devices[d.FD] = d
		watchedPaths[path] = true
		logger.Printf("watching %s %q (%s)", d.Path, d.Name, d.ID)
	}

	scan := func() {
		// Remove devices whose event node disappeared before epoll reported
		// it. This also frees the path so a replugged keyboard can be picked
		// up again.
		for fd, d := range devices {
			if _, err := os.Stat(d.Path); err != nil {
				removeDevice(fd)
			}
		}
		for _, path := range evdev.ListPaths() {
			addDevice(path)
		}
	}

	scan()
	if len(devices) == 0 {
		if opts.RequireDevice {
			return fmt.Errorf("no keyboard with left Alt and left Shift found")
		}
		logger.Printf("no keyboard found yet; waiting for hotplug")
	}

	logger.Printf("ready: combo=%s delay=%s rescan=%s dry_run=%t",
		opts.Combo, opts.Delay, opts.Rescan, opts.DryRun)

	events := make([]syscall.EpollEvent, 16)
	buf := make([]byte, evdev.EventSize*64)
	ticker := time.NewTicker(opts.Rescan)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			logger.Printf("stopping")
			return nil
		}

		n, err := syscall.EpollWait(epfd, events, 500)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			if ctx.Err() != nil {
				logger.Printf("stopping")
				return nil
			}
			return fmt.Errorf("epoll_wait: %w", err)
		}

		for i := 0; i < n; i++ {
			event := events[i]
			fd := int(event.Fd)
			d := devices[fd]
			if d == nil {
				continue
			}
			if event.Events&(syscall.EPOLLHUP|syscall.EPOLLERR) != 0 {
				removeDevice(fd)
				continue
			}

			for {
				nr, err := d.Read(buf)
				if err != nil {
					if evdev.ErrAgain(err) {
						break
					}
					if opts.Verbose {
						logger.Printf("read %s: %v", d.Path, err)
					}
					removeDevice(fd)
					break
				}
				if nr == 0 {
					break
				}
				for off := 0; off+evdev.EventSize <= nr; off += evdev.EventSize {
					e := evdev.DecodeEvent(buf[off : off+evdev.EventSize])
					if e.Type != evdev.EV_KEY {
						continue
					}
					// Any new physical key event cancels a pending injection.
					// Injected events come from the excluded virtual device
					// and never reach this loop.
					cancelPending()
					trigger := detector.Key(e.Code, e.Value)
					if opts.Verbose {
						logger.Printf("%s: %s value=%d", d.Name, evdev.KeyName(e.Code), e.Value)
					}
					if trigger {
						logger.Printf("chord: left Alt + left Shift released cleanly on %q", d.Name)
						schedule(d.Name)
					}
				}
			}
		}

		select {
		case <-ticker.C:
			scan()
		case <-ctx.Done():
			logger.Printf("stopping")
			return nil
		default:
		}
	}
}
