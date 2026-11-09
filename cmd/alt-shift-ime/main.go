package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sbonein-deb/openfyde-alt-shift-ime/internal/config"
	"github.com/sbonein-deb/openfyde-alt-shift-ime/internal/daemon"
	"github.com/sbonein-deb/openfyde-alt-shift-ime/internal/evdev"
	"github.com/sbonein-deb/openfyde-alt-shift-ime/internal/keycombo"
	"github.com/sbonein-deb/openfyde-alt-shift-ime/internal/lock"
	"github.com/sbonein-deb/openfyde-alt-shift-ime/internal/uinput"
)

var version = "0.3.0-dev"

const defaultConfigPath = "/usr/local/etc/alt-shift-ime.conf"
const defaultLockPath = "/run/alt-shift-ime.lock"

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "probe":
		err = cmdProbe(os.Args[2:])
	case "inject":
		err = cmdInject(os.Args[2:])
	case "selftest":
		err = cmdSelftest(os.Args[2:])
	case "run":
		err = cmdRun(os.Args[2:])
	case "version", "--version", "-version":
		fmt.Println(version)
	case "help", "--help", "-h":
		usage()
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `alt-shift-ime %s

Host-side left Alt + left Shift language switcher for OpenFyde/ChromeOS.

Usage:
  alt-shift-ime probe [--all]
  alt-shift-ime inject [--combo ctrl+space] [--warmup 500ms]
  alt-shift-ime selftest [--combo ctrl+space] [--delay 80ms] [--no-inject] [--verbose]
  alt-shift-ime run [--config /usr/local/etc/alt-shift-ime.conf] [--dry-run]

probe     list input devices and show which ones look like keyboards
inject    create a virtual keyboard and inject one combination
selftest  watch physical keyboards and inject on left Alt + left Shift release
run       long-running daemon, normally started by upstart

Run as root on the OpenFyde host. The physical keyboard is never grabbed.
`, version)
}

func cmdProbe(args []string) error {
	fs := flag.NewFlagSet("probe", flag.ExitOnError)
	all := fs.Bool("all", false, "show non-keyboard devices too")
	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Printf("alt-shift-ime %s\n", version)
	fmt.Printf("uid=%d\n", os.Getuid())
	fmt.Printf("/dev/uinput: %s\n", uinputStatus())
	fmt.Printf("/dev/input: %s\n\n", inputStatus())

	paths := evdev.ListPaths()
	if len(paths) == 0 {
		fmt.Println("no /dev/input/event* devices found")
		return nil
	}

	found := 0
	for _, path := range paths {
		d, err := evdev.Open(path)
		if err != nil {
			fmt.Printf("%s\n  ERROR: %v\n\n", path, err)
			continue
		}
		isKeyboard := d.IsKeyboard()
		hasChordKeys := d.HasChordKeys()
		if !*all && !isKeyboard && !hasChordKeys {
			d.Close()
			continue
		}
		found++
		fmt.Printf("%s\n", path)
		fmt.Printf("  name:       %q\n", d.Name)
		fmt.Printf("  id:         %s\n", d.ID)
		fmt.Printf("  keyboard:   %t\n", isKeyboard)
		fmt.Printf("  chord keys: %t (left_alt=%t left_shift=%t)\n",
			hasChordKeys, d.HasKey(evdev.KEY_LEFTALT), d.HasKey(evdev.KEY_LEFTSHIFT))
		fmt.Printf("  key count:  %d\n", d.KeyCount())
		if evdev.IsVirtualName(d.Name) {
			fmt.Printf("  note:       this is alt-shift-ime's own virtual device\n")
		}
		fmt.Println()
		d.Close()
	}
	if found == 0 {
		fmt.Println("no keyboard devices matched; use --all to show everything")
	}
	return nil
}

func cmdInject(args []string) error {
	fs := flag.NewFlagSet("inject", flag.ExitOnError)
	comboStr := fs.String("combo", "ctrl+space", "combination to inject")
	warmup := fs.Duration("warmup", 500*time.Millisecond, "wait after creating the virtual device")
	hold := fs.Duration("hold", 150*time.Millisecond, "wait before destroying the virtual device")
	if err := fs.Parse(args); err != nil {
		return err
	}
	c, err := keycombo.Parse(*comboStr)
	if err != nil {
		return err
	}

	inj, err := uinput.New()
	if err != nil {
		return err
	}
	defer inj.Close()
	log.Printf("created %q, waiting %s for ChromeOS to notice it", inj.Name(), *warmup)
	time.Sleep(*warmup)

	log.Printf("injecting %s", c)
	if err := inj.Inject(c); err != nil {
		return err
	}
	time.Sleep(*hold)
	log.Printf("done")
	return nil
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", defaultConfigPath, "configuration file")
	comboStr := fs.String("combo", "", "override the injected combination")
	delay := fs.Duration("delay", -1, "override the delay after release (0 is allowed)")
	rescan := fs.Duration("rescan", -1, "override the hotplug rescan interval")
	dryRun := fs.Bool("dry-run", false, "detect the chord but do not inject")
	verbose := fs.Bool("verbose", false, "log every key event")
	lockFile := fs.String("lock-file", defaultLockPath, "single-instance lock file")
	noLock := fs.Bool("no-lock", false, "do not take the single-instance lock")
	uinputTimeout := fs.Duration("uinput-timeout", 30*time.Second, "how long to wait for /dev/uinput")
	warmup := fs.Duration("warmup", 700*time.Millisecond, "wait after creating the virtual keyboard")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	if *comboStr != "" {
		cfg.Combo, err = keycombo.Parse(*comboStr)
		if err != nil {
			return err
		}
	}
	if *delay >= 0 {
		cfg.Delay = *delay
	}
	if *rescan > 0 {
		cfg.Rescan = *rescan
	}
	if *dryRun {
		cfg.DryRun = true
	}
	if *verbose {
		cfg.Verbose = true
	}

	logger := log.Default()
	logger.Printf("alt-shift-ime %s starting: combo=%s delay=%s rescan=%s dry_run=%t verbose=%t",
		version, cfg.Combo, cfg.Delay, cfg.Rescan, cfg.DryRun, cfg.Verbose)
	logger.Printf("config: %s", *configPath)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var lk *lock.Lock
	if !*noLock {
		lk, err = lock.Acquire(*lockFile)
		if err != nil {
			return err
		}
		defer lk.Close()
		logger.Printf("lock acquired: %s", *lockFile)
	}

	var injector daemon.Injector
	if !cfg.DryRun {
		inj, err := waitForInjector(ctx, *uinputTimeout, *warmup, logger)
		if err != nil {
			return err
		}
		defer inj.Close()
		injector = inj
	} else {
		logger.Printf("dry-run: no virtual keyboard will be created")
	}

	return daemon.Run(ctx, daemon.Options{
		Combo:    cfg.Combo,
		Delay:    cfg.Delay,
		Rescan:   cfg.Rescan,
		DryRun:   cfg.DryRun,
		Verbose:  cfg.Verbose,
		Injector: injector,
		Logger:   logger,
	})
}

func cmdSelftest(args []string) error {
	fs := flag.NewFlagSet("selftest", flag.ExitOnError)
	comboStr := fs.String("combo", "ctrl+space", "combination to inject")
	delay := fs.Duration("delay", 80*time.Millisecond, "delay after the final modifier release")
	rescan := fs.Duration("rescan", 2*time.Second, "how often to look for new keyboards")
	warmup := fs.Duration("warmup", 700*time.Millisecond, "wait after creating the virtual device")
	noInject := fs.Bool("no-inject", false, "detect the chord but do not inject")
	verbose := fs.Bool("verbose", true, "log every key event")
	lockFile := fs.String("lock-file", defaultLockPath, "single-instance lock file")
	noLock := fs.Bool("no-lock", false, "do not take the single-instance lock")
	uinputTimeout := fs.Duration("uinput-timeout", 10*time.Second, "how long to wait for /dev/uinput")
	if err := fs.Parse(args); err != nil {
		return err
	}
	c, err := keycombo.Parse(*comboStr)
	if err != nil {
		return err
	}

	logger := log.Default()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var lk *lock.Lock
	if !*noLock {
		lk, err = lock.Acquire(*lockFile)
		if err != nil {
			return err
		}
		defer lk.Close()
	}

	var injector daemon.Injector
	if !*noInject {
		inj, err := waitForInjector(ctx, *uinputTimeout, *warmup, logger)
		if err != nil {
			return err
		}
		defer inj.Close()
		injector = inj
	} else {
		logger.Printf("dry-run mode: the chord will be detected but not injected")
	}

	return daemon.Run(ctx, daemon.Options{
		Combo:         c,
		Delay:         *delay,
		Rescan:        *rescan,
		DryRun:        *noInject,
		Verbose:       *verbose,
		RequireDevice: true,
		Injector:      injector,
		Logger:        logger,
	})
}

func waitForInjector(ctx context.Context, timeout, warmup time.Duration, logger *log.Logger) (*uinput.Injector, error) {
	deadline := time.Now().Add(timeout)
	for {
		inj, err := uinput.New()
		if err == nil {
			logger.Printf("created %q, warming up %s", inj.Name(), warmup)
			if err := sleepContext(ctx, warmup); err != nil {
				inj.Close()
				return nil, err
			}
			return inj, nil
		}
		if !errors.Is(err, syscall.ENOENT) {
			return nil, err
		}
		logger.Printf("waiting for /dev/uinput: %v", err)
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("uinput not available after %s: %w", timeout, err)
		}
		if err := sleepContext(ctx, 500*time.Millisecond); err != nil {
			return nil, err
		}
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func uinputStatus() string {
	fd, err := syscall.Open("/dev/uinput", syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return "missing"
		}
		if err == syscall.EACCES || err == syscall.EPERM {
			return "present, not writable (run as root)"
		}
		return fmt.Sprintf("error: %v", err)
	}
	syscall.Close(fd)
	return "present, writable"
}

func inputStatus() string {
	if _, err := os.Stat("/dev/input"); err != nil {
		if os.IsNotExist(err) {
			return "missing"
		}
		return fmt.Sprintf("error: %v", err)
	}
	paths := evdev.ListPaths()
	return fmt.Sprintf("%d event device(s)", len(paths))
}
