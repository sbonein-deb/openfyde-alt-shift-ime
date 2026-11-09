# openfyde-alt-shift-ime

[![CI](https://github.com/sbonein-deb/openfyde-alt-shift-ime/actions/workflows/ci.yml/badge.svg)](https://github.com/sbonein-deb/openfyde-alt-shift-ime/actions/workflows/ci.yml)

A small host-side daemon for OpenFyde/ChromeOS that switches the input
method/layout when you release **left Alt + left Shift**. It watches physical
keyboards through `evdev` and injects `Ctrl+Space` through a virtual `uinput`
keyboard. Chromium/Ash is not patched, and the physical keyboard is never
grabbed.

**Русская документация:** [README.ru.md](README.ru.md)

## Why

ChromeOS does not provide an Alt+Shift layout shortcut out of the box. The
usual workarounds only work inside Crostini or require patching Chromium, which
is not practical on a signed ChromeOS/OpenFyde release. This project runs on
the host, next to the ChromeOS input stack, and reuses the existing
`Ctrl+Space` input-method accelerator.

`Search+Space` keeps working exactly as before.

## How it works

```
physical keyboard
      │  evdev (read-only, no EVIOCGRAB)
      ▼
alt-shift-ime watcher ──► left Alt + left Shift released cleanly?
      │                              │ yes
      │                              ▼
      │                     wait 80 ms (configurable)
      │                              │
      ▼                              ▼
 /dev/uinput virtual keyboard ──► Ctrl+Space ──► ChromeOS IME switch
```

The daemon triggers on **release**, not on press. ChromeOS tracks modifier
state globally: injecting while Alt or Shift is still held would produce
`Ctrl+Alt+Shift+Space`, which does not match the accelerator. Waiting for both
modifiers to be released and for a short delay avoids that.

If any other key is pressed during the chord, the chord is canceled. This is
why `Alt+Shift+Tab` does not switch layouts.

## Features

- Left `Alt` + left `Shift` only; right-side modifiers are ignored.
- Injects `Ctrl+Space` by default; the combination is configurable.
- Triggers after the modifiers are released, with an 80 ms delay by default.
- Preserves the standard `Search+Space` shortcut.
- Watches every keyboard that advertises left Alt and left Shift.
- Picks up external USB keyboards on hotplug without a restart.
- Single-instance `flock`; a second daemon or `selftest` refuses to run.
- Resets detector state and cancels a pending injection if a keyboard is
  unplugged mid-chord.
- Static Go binary for `amd64` and `arm64`; no runtime dependencies.
- Diagnostic commands: `probe`, `inject`, `selftest`.
- Upstart job, installer, restore-after-update and uninstall scripts.

## Requirements

- OpenFyde/ChromeOS with Upstart (tested on OpenFyde `16093.91.20.8`, ChromeOS
  M132, kernel 5.4).
- Root access on the host. This is **not** a Crostini/container program.
- `/dev/uinput` available and writable by root. The `uinput` module is usually
  already loaded.
- `amd64` or `arm64`.

## Install from a release

Download the tarball for your architecture from the
[releases page](https://github.com/sbonein-deb/openfyde-alt-shift-ime/releases),
verify it, and run the installer as root:

```bash
cd /tmp
curl -fsSLO https://github.com/sbonein-deb/openfyde-alt-shift-ime/releases/download/v0.2.0/alt-shift-ime_0.2.0_linux_amd64.tar.gz
curl -fsSLO https://github.com/sbonein-deb/openfyde-alt-shift-ime/releases/download/v0.2.0/checksums.txt
sha256sum -c checksums.txt --ignore-missing

rm -rf /tmp/alt-shift-ime-install
mkdir -p /tmp/alt-shift-ime-install
tar xzf alt-shift-ime_0.2.0_linux_amd64.tar.gz -C /tmp/alt-shift-ime-install --strip-components=1
bash /tmp/alt-shift-ime-install/setup-alt-shift-ime.sh
```

For `arm64`, replace `amd64` with `arm64` in the file name.

The installer:

- remounts `/` read-write for the duration of the install;
- backs up existing files under `/usr/local/lib/alt-shift-ime/`;
- installs `/usr/local/bin/alt-shift-ime`;
- installs `/usr/local/etc/alt-shift-ime.conf` if it does not exist;
- installs the Upstart job `/etc/init/alt-shift-ime.conf`;
- stores restore copies under `/usr/local/lib/alt-shift-ime/`;
- reloads Upstart and starts the daemon.

It is safe to run again.

> `/tmp` is mounted `noexec` on ChromeOS. Run shell scripts with `bash
> /path/script.sh`; do not rely on executing them directly.

## Verify

```bash
alt-shift-ime probe
initctl status alt-shift-ime
tail -n 20 /var/log/alt-shift-ime.log
```

A healthy start looks like this:

```
alt-shift-ime 0.2.0 starting: combo=ctrl+space delay=80ms rescan=2s dry_run=false verbose=false
config: /usr/local/etc/alt-shift-ime.conf
lock acquired: /run/alt-shift-ime.lock
created "alt-shift-ime virtual keyboard", warming up 700ms
watching /dev/input/event2 "AT Translated Set 2 keyboard" (bus=0011 vendor=0001 product=0001 version=ab83)
ready: combo=ctrl+space delay=80ms rescan=2s dry_run=false
```

Press and release left Alt+Shift. The log should show:

```
chord: left Alt + left Shift released cleanly on "AT Translated Set 2 keyboard"
TRIGGER from "AT Translated Set 2 keyboard": injecting ctrl+space
injected ctrl+space
```

## Configuration

The default configuration is `/usr/local/etc/alt-shift-ime.conf`:

```ini
combo=ctrl+space
delay_ms=80
rescan_seconds=2
# dry_run=false
# verbose=false
```

| Key | Meaning | Default |
| --- | --- | --- |
| `combo` | Combination to inject. | `ctrl+space` |
| `delay_ms` | Delay after the final modifier release. | `80` |
| `rescan_seconds` | Hotplug rescan interval. | `2` |
| `dry_run` | Detect the chord but do not inject. | `false` |
| `verbose` | Log every key event. | `false` |

Supported modifiers: `ctrl`, `shift`, `alt`, `search` (aliases: `meta`,
`super`, `win`). Supported keys: `space`, `tab`, `enter`, `esc`, `a`–`z`.

After editing the configuration:

```bash
initctl restart alt-shift-ime
```

Command-line flags override the configuration, for example:

```bash
alt-shift-ime run --combo ctrl+shift+space --delay 120ms --verbose
```

## Diagnostic commands

```bash
alt-shift-ime probe              # list input devices and keyboard detection
alt-shift-ime probe --all        # show every event device
alt-shift-ime inject             # inject Ctrl+Space once
alt-shift-ime inject --combo search+space
alt-shift-ime selftest           # watch the chord and inject, verbose
alt-shift-ime selftest --no-inject
alt-shift-ime run --dry-run      # detect without creating a virtual keyboard
```

`inject` is useful to prove that the virtual keyboard can reach ChromeOS
before debugging the chord detector. `selftest` is a foreground version of the
daemon and takes the same single-instance lock.

## Restore after a ChromeOS update

ChromeOS updates replace the root filesystem, so `/etc/init/alt-shift-ime.conf`
disappears. `/usr/local` lives on the stateful partition and survives. Restore
the job with:

```bash
bash /usr/local/lib/alt-shift-ime/restore-alt-shift-ime.sh
```

The script restores the Upstart job and config, reloads Upstart and starts the
daemon. A reboot is not required.

## Uninstall

```bash
bash /usr/local/lib/alt-shift-ime/uninstall-alt-shift-ime.sh
```

This stops the daemon and removes the binary and Upstart job, keeping the
configuration and restore copies. Add `--purge` to remove everything:

```bash
bash /usr/local/lib/alt-shift-ime/uninstall-alt-shift-ime.sh --purge
```

## Build from source

Go 1.19 or newer is enough. Build inside Crostini or any Linux machine; the
OpenFyde host does not need a Go toolchain.

```bash
git clone https://github.com/sbonein-deb/openfyde-alt-shift-ime.git
cd openfyde-alt-shift-ime

go vet ./...
go test -race ./...
./scripts/build.sh
ARCH=arm64 OUT=dist/alt-shift-ime-arm64 ./scripts/build.sh
```

Create release tarballs and checksums:

```bash
./scripts/package.sh v0.2.0
ls dist/release/
```

## Documentation

- [docs/INSTALL.md](docs/INSTALL.md) — install, upgrade, restore, uninstall.
- [docs/DESIGN.md](docs/DESIGN.md) — architecture and the ChromeOS-specific
  details.
- [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) — diagnostics and common
  failures.
- [CHANGELOG.md](CHANGELOG.md) — release history.

Russian versions: [docs/INSTALL.ru.md](docs/INSTALL.ru.md),
[docs/DESIGN.ru.md](docs/DESIGN.ru.md),
[docs/TROUBLESHOOTING.ru.md](docs/TROUBLESHOOTING.ru.md).

## Security notes

- The daemon runs as root because both `evdev` and `/dev/uinput` require it.
- Physical keyboards are opened read-only. `EVIOCGRAB` is never used, so the
  normal ChromeOS input path still receives every key.
- The virtual device is only used to emit the configured combination.
- The daemon does not use the network.
- A single instance is enforced with `flock` on `/run/alt-shift-ime.lock`.

## Known limitations

- Only the Upstart-based OpenFyde/ChromeOS boot is supported. There is no
  systemd unit.
- Right Alt and right Shift are intentionally ignored.
- The external-keyboard hotplug path is implemented and unit-tested at the
  detector/device level, but at the time of the first release it had not been
  verified with physical external hardware.
- Key names in `combo` are positional evdev key codes, not layout-dependent
  characters.
- ChromeOS updates require running the restore script (or reinstalling).

## License

GPL-2.0. See [LICENSE](LICENSE).
