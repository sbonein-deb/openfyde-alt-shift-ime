#!/bin/bash
# Remove alt-shift-ime from an OpenFyde/ChromeOS host.
#
# Usage:
#   sudo bash uninstall-alt-shift-ime.sh          # keep config and logs
#   sudo bash uninstall-alt-shift-ime.sh --purge  # remove everything
set -u

DESTDIR=/usr/local/lib/alt-shift-ime
UDEVDIR=/usr/local/lib/udev/rules.d
UDEV_RULE=99-alt-shift-ime.rules
LOCK=/run/alt-shift-ime.lock
HELPER_LOCK=/run/alt-shift-ime-boot-helper.lock
PURGE=0

for arg in "$@"; do
  case "$arg" in
    --purge) PURGE=1 ;;
    -h|--help)
      sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *)
      echo "unknown argument: $arg" >&2
      exit 2
      ;;
  esac
done

if [ "$(id -u)" -ne 0 ]; then
  echo "run as root" >&2
  exit 1
fi

stop_pid_from() {
  local pidfile=$1
  local pattern=$2
  local pid
  pid=$(head -n 1 "$pidfile" 2>/dev/null || true)
  if [ -z "${pid:-}" ] || ! kill -0 "$pid" 2>/dev/null; then
    return 0
  fi
  if ! grep -qa "$pattern" "/proc/$pid/cmdline" 2>/dev/null; then
    return 0
  fi
  echo "== stopping pid $pid"
  kill -TERM "$pid" 2>/dev/null || true
  for _ in 1 2 3 4 5; do
    kill -0 "$pid" 2>/dev/null || return 0
    sleep 1
  done
}

# Prevent boot-restore-late.sh from reinstalling the job while we uninstall.
mkdir -p "$DESTDIR"
: >"$DESTDIR/disabled"

if [ -e "$HELPER_LOCK/pid" ]; then
  stop_pid_from "$HELPER_LOCK/pid" 'boot-restore-late.sh'
fi

echo "== stopping alt-shift-ime"
initctl stop alt-shift-ime 2>/dev/null || true
stop_pid_from "$LOCK" '/usr/local/bin/alt-shift-ime'

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
  echo "warning: could not remount / read-write" >&2
fi

echo "== removing /etc/init/alt-shift-ime.conf"
rm -f /etc/init/alt-shift-ime.conf
initctl reload-configuration 2>/dev/null || true

echo "== removing $UDEVDIR/$UDEV_RULE"
rm -f "$UDEVDIR/$UDEV_RULE"
if command -v udevadm >/dev/null 2>&1; then
  udevadm control --reload-rules 2>/dev/null || true
fi

echo "== removing /usr/local/bin/alt-shift-ime"
rm -f /usr/local/bin/alt-shift-ime

rm -f /run/alt-shift-ime-boot-hook
rm -rf "$HELPER_LOCK"

if [ "$PURGE" = 1 ]; then
  echo "== purging config, restore copies and logs"
  rm -f /usr/local/etc/alt-shift-ime.conf
  rm -rf "$DESTDIR"
  rm -f /var/log/alt-shift-ime.log /var/log/alt-shift-ime-restore.log
else
  echo "== keeping /usr/local/etc/alt-shift-ime.conf and /usr/local/lib/alt-shift-ime"
fi

echo "== done"
