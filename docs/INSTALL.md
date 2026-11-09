# Installation

## Before you start

- This program runs on the **OpenFyde/ChromeOS host**, not in Crostini.
- You need a root shell on the host.
- Check that `/dev/uinput` exists and that the kernel has the `uinput` module:

  ```bash
  ls -l /dev/uinput
  grep -w uinput /proc/modules || modprobe uinput
  ```

- Check that at least one keyboard advertises left Alt and left Shift:

  ```bash
  alt-shift-ime probe
  ```

  If the binary is not installed yet, use the `probe` from the release tarball
  after extracting it.

## Install from a release

Set `VERSION` and `ARCH` to the values you need:

```bash
VERSION=0.3.0
ARCH=amd64   # or arm64

cd /tmp
BASE="https://github.com/sbonein-deb/openfyde-alt-shift-ime/releases/download/v${VERSION}"
curl -fsSLO "${BASE}/alt-shift-ime_${VERSION}_linux_${ARCH}.tar.gz"
curl -fsSLO "${BASE}/checksums.txt"
sha256sum -c checksums.txt --ignore-missing

rm -rf "/tmp/alt-shift-ime-${VERSION}"
mkdir -p "/tmp/alt-shift-ime-${VERSION}"
tar xzf "alt-shift-ime_${VERSION}_linux_${ARCH}.tar.gz" \
  -C "/tmp/alt-shift-ime-${VERSION}" --strip-components=1

bash "/tmp/alt-shift-ime-${VERSION}/setup-alt-shift-ime.sh"
```

The installer is idempotent. It keeps an existing
`/usr/local/etc/alt-shift-ime.conf` and backs up replaced files under
`/usr/local/lib/alt-shift-ime/backup-<timestamp>/`.

### Install without starting

```bash
bash setup-alt-shift-ime.sh --no-start
```

Then start manually:

```bash
initctl start alt-shift-ime
```

## Manual installation

If you prefer not to run the installer, the equivalent steps are:

```bash
mount -o remount,rw /

install -m 0755 alt-shift-ime /usr/local/bin/alt-shift-ime
install -m 0644 alt-shift-ime.conf /usr/local/etc/alt-shift-ime.conf
install -m 0644 alt-shift-ime.upstart.conf /etc/init/alt-shift-ime.conf

mkdir -p /usr/local/lib/alt-shift-ime /usr/local/lib/udev/rules.d
install -m 0644 alt-shift-ime.upstart.conf \
  /usr/local/lib/alt-shift-ime/alt-shift-ime.upstart.conf
install -m 0644 alt-shift-ime.conf \
  /usr/local/lib/alt-shift-ime/alt-shift-ime.conf
install -m 0755 setup-alt-shift-ime.sh \
  /usr/local/lib/alt-shift-ime/setup-alt-shift-ime.sh
install -m 0755 restore-alt-shift-ime.sh \
  /usr/local/lib/alt-shift-ime/restore-alt-shift-ime.sh
install -m 0755 uninstall-alt-shift-ime.sh \
  /usr/local/lib/alt-shift-ime/uninstall-alt-shift-ime.sh
install -m 0755 boot-restore.sh \
  /usr/local/lib/alt-shift-ime/boot-restore.sh
install -m 0755 boot-restore-late.sh \
  /usr/local/lib/alt-shift-ime/boot-restore-late.sh
install -m 0644 99-alt-shift-ime.rules \
  /usr/local/lib/udev/rules.d/99-alt-shift-ime.rules

restorecon -v /usr/local/bin/alt-shift-ime /etc/init/alt-shift-ime.conf \
  /usr/local/lib/udev/rules.d/99-alt-shift-ime.rules || true
udevadm control --reload-rules
initctl reload-configuration
initctl start alt-shift-ime

mount -o remount,ro /
```

## Verify

```bash
initctl status alt-shift-ime
tail -n 20 /var/log/alt-shift-ime.log
```

Expected status:

```
alt-shift-ime start/running, process <pid>
```

Expected log:

```
... starting: combo=ctrl+space delay=80ms rescan=2s dry_run=false verbose=false
... lock acquired: /run/alt-shift-ime.lock
... created "alt-shift-ime virtual keyboard", warming up 700ms
... watching /dev/input/event2 ...
... ready: combo=ctrl+space delay=80ms rescan=2s dry_run=false
```

Press and release left Alt+Shift and check the log for `injected ctrl+space`.

## Upgrading

Download the new tarball and run the installer again. The existing config is
kept. The installer backs up the previous binary and job first.

```bash
bash setup-alt-shift-ime.sh
initctl status alt-shift-ime
```

## Restore after a ChromeOS update

ChromeOS updates replace the root filesystem, so `/etc/init/alt-shift-ime.conf`
is lost. The binary, config, udev rule and restore copies under `/usr/local`
survive.

Recovery is automatic. The udev rule
`/usr/local/lib/udev/rules.d/99-alt-shift-ime.rules` runs
`/usr/local/lib/alt-shift-ime/boot-restore.sh` during the normal boot device
trigger:

1. If `/etc/init/alt-shift-ime.conf` exists, the hook marks the boot as handled
   and Upstart starts the job normally.
2. If the job is missing and the install is not disabled, the hook starts the
   daemon directly with `/usr/local/bin/alt-shift-ime run`.
3. `boot-restore-late.sh` waits for boot to settle, installs the Upstart job
   with `restore-alt-shift-ime.sh --install-only`, reloads Upstart, stops the
   directly started daemon and starts the job under Upstart.

The hook runs at most once per boot (`/run/alt-shift-ime-boot-hook`) and logs to
`/var/log/alt-shift-ime-restore.log`. No manual step and no extra reboot are
required.

Manual fallback:

```bash
bash /usr/local/lib/alt-shift-ime/restore-alt-shift-ime.sh
```

The script:

1. remounts `/` read-write;
2. restores `/etc/init/alt-shift-ime.conf` from
   `/usr/local/lib/alt-shift-ime/alt-shift-ime.upstart.conf`;
3. restores the default config if it is missing;
4. reloads Upstart;
5. starts or restarts the daemon;
6. remounts `/` read-only.

`--install-only` performs steps 1–3 and 6 only; the automatic recovery helper
uses it.

No reboot is required.

## Test the automatic recovery

To simulate an update without reinstalling the system:

```bash
initctl stop alt-shift-ime
rm -f /etc/init/alt-shift-ime.conf
rm -f /run/alt-shift-ime-boot-hook
udevadm trigger --action=add --subsystem-match=block --sysname-match=sda
```

Adjust `sda` if your root disk has a different name. Then check:

```bash
cat /var/log/alt-shift-ime-restore.log
initctl status alt-shift-ime
```

The log should show the direct start followed by the takeover under Upstart.

## Uninstall

Keep config and restore copies:

```bash
bash /usr/local/lib/alt-shift-ime/uninstall-alt-shift-ime.sh
```

This stops the daemon, removes the binary, Upstart job and udev rule, and
creates `/usr/local/lib/alt-shift-ime/disabled` so the boot hook cannot restore
the job.

Remove everything:

```bash
bash /usr/local/lib/alt-shift-ime/uninstall-alt-shift-ime.sh --purge
```

After `--purge`, `/usr/local/lib/alt-shift-ime/` is gone, so keep a copy of the
release tarball if you want to reinstall later.

## Offline installation

On a machine with network access, download the tarball and `checksums.txt`.
Transfer both to the host, verify with `sha256sum -c`, extract and run the
installer. No network access is needed on the host.

## Architecture notes

- `amd64` covers the tested Celeron N3350 OpenFyde device.
- `arm64` covers ARM OpenFyde boards (for example Raspberry Pi class devices).
- The binary is static (`CGO_ENABLED=0`); no glibc or Go runtime is required on
  the host.

## Files installed

| Path | Purpose |
| --- | --- |
| `/usr/local/bin/alt-shift-ime` | Static binary. |
| `/usr/local/etc/alt-shift-ime.conf` | Configuration. |
| `/etc/init/alt-shift-ime.conf` | Upstart job. |
| `/usr/local/lib/udev/rules.d/99-alt-shift-ime.rules` | Boot-time recovery hook. |
| `/usr/local/lib/alt-shift-ime/alt-shift-ime.upstart.conf` | Restore copy of the job. |
| `/usr/local/lib/alt-shift-ime/alt-shift-ime.conf` | Restore copy of the config. |
| `/usr/local/lib/alt-shift-ime/setup-alt-shift-ime.sh` | Installer copy. |
| `/usr/local/lib/alt-shift-ime/restore-alt-shift-ime.sh` | Post-update restore. |
| `/usr/local/lib/alt-shift-ime/uninstall-alt-shift-ime.sh` | Uninstaller copy. |
| `/usr/local/lib/alt-shift-ime/boot-restore.sh` | udev boot hook. |
| `/usr/local/lib/alt-shift-ime/boot-restore-late.sh` | Job installation and Upstart takeover. |
| `/var/log/alt-shift-ime.log` | Daemon log. |
| `/var/log/alt-shift-ime-restore.log` | Automatic recovery log. |
| `/run/alt-shift-ime.lock` | Single-instance lock. |
| `/run/alt-shift-ime-boot-hook` | Once-per-boot marker. |

## Build and install from source

Inside Crostini or another Linux machine:

```bash
git clone https://github.com/sbonein-deb/openfyde-alt-shift-ime.git
cd openfyde-alt-shift-ime
./scripts/build.sh
```

Copy `dist/alt-shift-ime` to the host and use `BINARY=` with the installer:

```bash
BINARY=/tmp/alt-shift-ime bash setup-alt-shift-ime.sh
```

Or build the release tarballs locally:

```bash
./scripts/package.sh v0.3.0
```
