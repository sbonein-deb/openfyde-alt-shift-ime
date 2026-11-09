# Changelog

All notable changes to this project are documented in this file. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this
project uses [Semantic Versioning](https://semver.org/).

## [0.3.0] - 2026-11-09

### Added

- Automatic recovery after a ChromeOS update. A udev rule in
  `/usr/local/lib/udev/rules.d/` runs `boot-restore.sh` during boot. If the
  Upstart job is missing, the hook starts the daemon directly, installs the job
  for the next boot in the background, and moves the running daemon back under
  Upstart.
- `restore-alt-shift-ime.sh --install-only` installs the job and config without
  reloading Upstart or starting the daemon.
- Installer installs the udev rule and the boot-recovery scripts, clears the
  uninstall sentinel and reloads udev rules.
- Uninstaller removes the udev rule, creates a `disabled` sentinel, stops a
  directly started daemon and terminates a running recovery helper.
- `/var/log/alt-shift-ime-restore.log` for automatic recovery diagnostics.

### Changed

- Version bumped to `0.3.0`.

### Verified

- External USB keyboard and hotplug were verified with physical hardware.

## [0.2.0] - 2026-11-09

First public release.

### Added

- Host-side daemon that injects a configurable combination when physical left
  Alt and left Shift are released.
- Left Alt + left Shift chord detector that ignores right-side modifiers and
  cancels the chord when any other key is pressed.
- Delayed injection (80 ms by default) so ChromeOS sees the modifiers as
  released before `Ctrl+Space` arrives.
- `uinput` virtual keyboard with a key bitmap covering key codes 1..127 so
  ChromeOS classifies it as a full keyboard.
- Passive `evdev` watcher using `epoll`; physical keyboards are never grabbed.
- Hotplug rescan of `/dev/input` every 2 seconds, with detector reset and
  pending-injection cancellation when a device disappears.
- Single-instance lock via `flock` on `/run/alt-shift-ime.lock`.
- Configuration file `/usr/local/etc/alt-shift-ime.conf` with `combo`,
  `delay_ms`, `rescan_seconds`, `dry_run` and `verbose`.
- Upstart job that starts on `system-services`, respawns on failure and logs to
  `/var/log/alt-shift-ime.log`.
- Installer, restore-after-ChromeOS-update script and uninstaller.
- Diagnostic commands: `probe`, `inject`, `selftest`, plus `run --dry-run`.
- Static `amd64` and `arm64` binaries.
- English and Russian documentation.
- GitHub Actions CI and tag-based release workflow.

### Known limitations

- External USB keyboard hotplug is implemented but was not verified with
  physical external hardware in this release.
- Only Upstart-based OpenFyde/ChromeOS is supported.
- ChromeOS updates remove `/etc/init/alt-shift-ime.conf`; run
  `restore-alt-shift-ime.sh` afterwards.

## [0.1.0] - 2026-11-09

Internal Phase 0 prototype, not publicly released.

### Added

- `probe`, `inject` and `selftest` commands.
- Initial `evdev`, `uinput`, `chord` and `keycombo` packages.
- Host verification: RU/US switching, `Search+Space`, `Alt+Shift+Tab`, AVC.

[0.3.0]: https://github.com/sbonein-deb/openfyde-alt-shift-ime/releases/tag/v0.3.0
[0.2.0]: https://github.com/sbonein-deb/openfyde-alt-shift-ime/releases/tag/v0.2.0
