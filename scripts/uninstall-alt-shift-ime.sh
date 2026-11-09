#!/bin/bash
# Remove alt-shift-ime from an OpenFyde/ChromeOS host.
#
# Usage:
#   sudo bash uninstall-alt-shift-ime.sh          # keep config and logs
#   sudo bash uninstall-alt-shift-ime.sh --purge  # remove everything
set -u

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

REMOUNTED=0
cleanup() {
  if [ "$REMOUNTED" = 1 ]; then
    mount -o remount,ro / 2>/dev/null || true
  fi
}
trap cleanup EXIT

echo "== stopping alt-shift-ime"
initctl stop alt-shift-ime 2>/dev/null || true

echo "== remounting / read-write"
if mount -o remount,rw / 2>/dev/null; then
  REMOUNTED=1
else
  echo "warning: could not remount / read-write" >&2
fi

echo "== removing /etc/init/alt-shift-ime.conf"
rm -f /etc/init/alt-shift-ime.conf
initctl reload-configuration 2>/dev/null || true

echo "== removing /usr/local/bin/alt-shift-ime"
rm -f /usr/local/bin/alt-shift-ime

if [ "$PURGE" = 1 ]; then
  echo "== purging config, restore copies and logs"
  rm -f /usr/local/etc/alt-shift-ime.conf
  rm -rf /usr/local/lib/alt-shift-ime
  rm -f /var/log/alt-shift-ime.log
else
  echo "== keeping /usr/local/etc/alt-shift-ime.conf and /usr/local/lib/alt-shift-ime"
fi

echo "== done"
