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
VERSION=0.2.0
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

mkdir -p /usr/local/lib/alt-shift-ime
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

restorecon -v /usr/local/bin/alt-shift-ime /etc/init/alt-shift-ime.conf || true
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
is lost. The binary, config and restore copies under `/usr/local` survive.

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

No reboot is required.

## Uninstall

Keep config and restore copies:

```bash
bash /usr/local/lib/alt-shift-ime/uninstall-alt-shift-ime.sh
```

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
| `/usr/local/lib/alt-shift-ime/alt-shift-ime.upstart.conf` | Restore copy of the job. |
| `/usr/local/lib/alt-shift-ime/alt-shift-ime.conf` | Restore copy of the config. |
| `/usr/local/lib/alt-shift-ime/setup-alt-shift-ime.sh` | Installer copy. |
| `/usr/local/lib/alt-shift-ime/restore-alt-shift-ime.sh` | Post-update restore. |
| `/usr/local/lib/alt-shift-ime/uninstall-alt-shift-ime.sh` | Uninstaller copy. |
| `/var/log/alt-shift-ime.log` | Daemon log. |
| `/run/alt-shift-ime.lock` | Single-instance lock. |

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
./scripts/package.sh v0.2.0
```
