# Design

This document explains why `openfyde-alt-shift-ime` is built the way it is and
which ChromeOS-specific details matter.

## The problem

ChromeOS has no built-in shortcut that switches the input method on Alt+Shift.
`Ctrl+Space` and `Search+Space` are the standard accelerators. Users who come
from Linux/Windows expect Alt+Shift, and remapping it inside Crostini only
affects Linux applications, not the host UI.

The goal is therefore a host-side component that turns a physical left
Alt+Shift chord into the existing ChromeOS input-method shortcut, without
modifying signed ChromeOS binaries.

## Options considered

### 1. Patch Chromium/Ash

The shortcut handling lives in Chromium/Ash. Patching it would require building
and replacing signed system components. On OpenFyde this is possible in
principle, but it is fragile across updates and far too invasive for a small
keyboard convenience.

### 2. Remap keys inside Crostini

Crostini has its own X11/Wayland input stack. A remap there only affects Linux
apps. It cannot change the layout of the host ChromeOS UI.

### 3. Use xkb or a ChromeOS keyboard-layout setting

ChromeOS does not expose an xkb configuration hook for the host UI, and the
standard keyboard settings do not include an Alt+Shift accelerator.

### 4. Host evdev watcher + uinput injector

This is the chosen approach:

- read physical keyboard events from `/dev/input/event*`;
- detect left Alt + left Shift;
- inject `Ctrl+Space` through a virtual keyboard created with `/dev/uinput`.

It is small, self-contained, survives ChromeOS updates (with a restore step for
`/etc/init`), and does not touch the normal input path.

## Architecture

```
cmd/alt-shift-ime        CLI: probe, inject, selftest, run
internal/evdev           evdev ioctls, device enumeration, event decoding
internal/chord           pure left-Alt/left-Shift state machine
internal/uinput          virtual keyboard and key injection
internal/daemon          epoll watcher, hotplug, delayed injection
internal/config          key=value configuration parser
internal/keycombo        combo parser (ctrl+space, search+space, ...)
internal/lock            flock single-instance lock
```

The daemon uses one `epoll` instance and one non-blocking read loop. A timer
performs a hotplug rescan of `/dev/input/event*` every `rescan_seconds`.

## Why the keyboard is not grabbed

`EVIOCGRAB` would give the daemon exclusive access to the keyboard. That would
break normal typing unless every event were re-injected, and it would add a
large failure surface. Instead, devices are opened `O_RDONLY | O_NONBLOCK` and
never grabbed. ChromeOS keeps receiving all events; the daemon only observes
them.

The trade-off is that the daemon cannot suppress the original Alt+Shift keys.
That is fine: ChromeOS does not assign a conflicting action to left Alt+Shift
by default.

## Why injection happens on release

ChromeOS tracks modifier state globally. If the daemon injects `Ctrl+Space`
while the physical Alt or Shift is still down, the effective combination seen
by the UI is `Ctrl+Alt+Shift+Space`, which does not match the `Ctrl+Space`
accelerator.

The detector therefore:

1. waits until both left Alt and left Shift have been pressed together;
2. cancels the chord if any other key is pressed while either modifier is down;
3. triggers only when the last of the two modifiers is released;
4. waits `delay_ms` (80 ms by default) so ChromeOS processes the physical
   releases before the injected events arrive;
5. injects the configured combination.

This also prevents `Alt+Shift+Tab` from switching layouts: Tab is another key,
so the chord is canceled.

The detector is a pure state machine in `internal/chord`, which makes the edge
cases unit-testable without any devices.

## Virtual keyboard details

The virtual device is created once at daemon startup and kept open. Creating a
uinput device takes 100–200 ms and ChromeOS needs a short warm-up before it
starts routing events from it, so the daemon waits 700 ms after creation.

Device properties:

- name: `alt-shift-ime virtual keyboard`
- bus: USB
- vendor/product: `0x1209:0x0001`
- key bitmap: key codes 1 through 127

The wide key bitmap is deliberate. ChromeOS classifies a device as a full
keyboard only if it advertises the contiguous range from `KEY_ESC` (1) through
`KEY_D` (32). Advertising 1..127 makes the virtual device unambiguously a
keyboard. The daemon only emits the keys needed by the configured combination.

### uinput ioctl ABI

`UI_SET_EVBIT` and `UI_SET_KEYBIT` are declared with `_IOW`, but the kernel
reads the integer argument directly from the ioctl argument rather than
dereferencing a pointer. Passing the value by value is correct here; passing a
pointer would set bit 0 or fail.

The legacy `uinput_user_dev` structure is written without padding, matching the
kernel ABI expected by `UI_DEV_CREATE`.

## Device detection

A device is watched if it advertises `EV_KEY`, `KEY_LEFTALT` and
`KEY_LEFTSHIFT`. It does not have to pass the stricter full-keyboard heuristic,
because some compact or programmable keyboards may expose a reduced key set.

`probe` additionally reports a stricter `keyboard` flag (A, Z and Space
present) for diagnostics.

The daemon excludes its own virtual device by name (`alt-shift-ime` substring)
and by the injector's exact name, so injected events can never feed back into
the detector.

## Hotplug and removal

Every `rescan_seconds`:

- devices whose `/dev/input/event*` node disappeared are removed;
- new event nodes are opened and added to `epoll`.

On `EPOLLHUP`/`EPOLLERR`, read error, or a missing node:

- the device is closed and removed;
- the chord detector is reset;
- any pending injection is canceled.

This avoids a stuck "Alt is still down" state after unplugging a keyboard
mid-chord.

## Single instance

Both `run` and `selftest` take an exclusive non-blocking `flock` on
`/run/alt-shift-ime.lock`. The PID is written into the file for diagnostics.
A second instance exits with:

```
another alt-shift-ime instance holds /run/alt-shift-ime.lock: resource temporarily unavailable
```

The lock is advisory and process-scoped, so it is released automatically when
the process exits.

## Boot and persistence

The Upstart job starts on `started system-services`, respawns up to 10 times in
60 seconds, and logs to `/var/log/alt-shift-ime.log`.

ChromeOS mounts `/` read-only. The installer remounts it read-write, installs
`/etc/init/alt-shift-ime.conf`, and remounts it read-only.

`/usr/local` is on the stateful partition and survives ChromeOS updates, while
`/etc/init` does not. The installer therefore stores:

- the binary and config;
- the Upstart job as `alt-shift-ime.upstart.conf`;
- the setup, restore, boot-recovery and uninstall scripts

under `/usr/local/lib/alt-shift-ime/`. `restore-alt-shift-ime.sh` copies the
job back into `/etc/init`, reloads Upstart and starts the daemon.

## Automatic recovery after an update

Running the restore script by hand is easy to forget. The installer also writes
`/usr/local/lib/udev/rules.d/99-alt-shift-ime.rules`. udev 249 reads that
directory (`CONF_PATHS` includes `/usr/local/lib/udev/rules.d`), and the
`cros_udevd` domain is permissive on the tested build, so the rule works without
touching `/etc/udev` or the signed root filesystem.

The rule runs `boot-restore.sh` for `add` events on block devices and input
event nodes. Input events also cover a keyboard plugged in after boot. The hook
is idempotent:

1. `/run/alt-shift-ime-boot-hook` makes it act at most once per boot. `/run` is
   a tmpfs, so the marker disappears on reboot.
2. If `/etc/init/alt-shift-ime.conf` exists, Upstart owns the job; the hook only
   writes the marker.
3. If the install is disabled (`/usr/local/lib/alt-shift-ime/disabled`), it
   exits.
4. Otherwise it starts the daemon directly with the same arguments as the
   Upstart job and launches `boot-restore-late.sh` in the background.

The block-device trigger may run before `/var` is writable. The hook therefore
checks `-w /var/log` and returns; the later input/block events after
`boot-services` run it again. The daemon also retries `/dev/uinput` for up to
300 seconds, so starting before `uinput.conf` has loaded the module is safe.

`boot-restore-late.sh` waits for boot to settle, installs the job with
`restore-alt-shift-ime.sh --install-only`, reloads Upstart, stops the directly
started daemon (PID from `/run/alt-shift-ime.lock`, verified against
`/proc/<pid>/cmdline`) and starts `alt-shift-ime` under Upstart. From then on the
next boots use the normal job.

The uninstaller removes the udev rule and creates the `disabled` sentinel before
stopping anything, so a recovery helper that is already running cannot
reinstall the job during uninstall.

## SELinux and noexec

On the tested OpenFyde build, SELinux is enforcing, but the `cros_init` and
`cros_init_scripts` domains are permissive and no AVC denials were observed.
The installer still calls `restorecon` on installed files.

`/tmp` is mounted `noexec`. The installer must not test `-x` on a binary in
`/tmp`, because that check fails even when the mode bits are `0755`. It only
checks that the file exists and installs it with `install -m 0755` into
`/usr/local/bin`. Shell scripts are run with `bash /path/script.sh`.

## Failure handling

- `/dev/uinput` missing at boot: `run` retries for up to 30 seconds.
- uinput creation errors other than `ENOENT`: fail fast and let Upstart respawn.
- inject errors: logged; the daemon keeps running.
- no keyboard found at startup: the daemon keeps running and waits for hotplug
  (`run`); `selftest` exits with an error because it is interactive.
- SIGTERM/SIGINT: context cancellation, cleanup, virtual device destroyed.

## Testing strategy

- Pure unit tests for the chord state machine and combo/config parsers.
- evdev ABI tests for ioctl sizes and event decoding.
- `go test -race` for the daemon's timer and device maps.
- Manual host tests for injection, `Search+Space`, `Alt+Shift+Tab`, AVC, boot
  persistence, restore and uninstall.

The external USB keyboard hotplug path is implemented and covered at the
device-detection level, but the first release was produced without physical
external keyboard hardware.
