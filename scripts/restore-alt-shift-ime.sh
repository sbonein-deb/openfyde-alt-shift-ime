#!/bin/bash
# Recreate the alt-shift-ime upstart job after a ChromeOS update.
#
# ChromeOS updates replace the root filesystem, so /etc/init/alt-shift-ime.conf
# disappears while /usr/local/bin, /usr/local/etc and /usr/local/lib survive.
# Run this after an update, then reboot or let the job start.
set -u

DESTDIR=/usr/local/lib/alt-shift-ime
BINDIR=/usr/local/bin
CONFDIR=/usr/local/etc
INITDIR=/etc/init
JOB=alt-shift-ime.conf

if [ "$(id -u)" -ne 0 ]; then
  echo "run as root" >&2
  exit 1
fi
if [ ! -x "$BINDIR/alt-shift-ime" ]; then
  echo "missing $BINDIR/alt-shift-ime; reinstall from a release first" >&2
  exit 1
fi
if [ ! -f "$DESTDIR/alt-shift-ime.upstart.conf" ]; then
  echo "missing $DESTDIR/alt-shift-ime.upstart.conf; reinstall from a release first" >&2
  exit 1
fi

REMOUNTED=0
cleanup() {
  if [ "$REMOUNTED" = 1 ]; then
    mount -o remount,ro / 2>/dev/null || true
  fi
}
trap cleanup EXIT

echo "== remounting / read-write"
if mount -o remount,rw / 2>/dev/null; then
  REMOUNTED=1
else
  echo "error: could not remount / read-write" >&2
  exit 1
fi

echo "== restoring $INITDIR/$JOB"
install -m 0644 "$DESTDIR/alt-shift-ime.upstart.conf" "$INITDIR/$JOB"
restorecon -v "$INITDIR/$JOB" 2>/dev/null || true

if [ ! -e "$CONFDIR/alt-shift-ime.conf" ] && [ -e "$DESTDIR/alt-shift-ime.conf" ]; then
  echo "== restoring default config"
  install -m 0644 "$DESTDIR/alt-shift-ime.conf" "$CONFDIR/alt-shift-ime.conf"
fi

echo "== reloading upstart configuration"
initctl reload-configuration
if initctl status alt-shift-ime 2>/dev/null | grep -q 'start/running'; then
  initctl restart alt-shift-ime
else
  initctl start alt-shift-ime
fi
sleep 1
initctl status alt-shift-ime || true
echo "== last log lines"
tail -n 10 /var/log/alt-shift-ime.log 2>/dev/null || true
