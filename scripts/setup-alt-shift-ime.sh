#!/bin/bash
# Install alt-shift-ime on an OpenFyde/ChromeOS host and start it at boot.
#
# Usage:
#   sudo bash setup-alt-shift-ime.sh
#   sudo BINARY=/path/to/alt-shift-ime bash setup-alt-shift-ime.sh
#   sudo bash setup-alt-shift-ime.sh --no-start
#
# The script is safe to re-run. It backs up existing files, keeps an existing
# /usr/local/etc/alt-shift-ime.conf, and stores restore copies under
# /usr/local/lib/alt-shift-ime/ so the upstart job can be recreated after a
# ChromeOS update.
set -u

DESTDIR=/usr/local/lib/alt-shift-ime
BINDIR=/usr/local/bin
CONFDIR=/usr/local/etc
INITDIR=/etc/init
JOB=alt-shift-ime.conf
BIN=alt-shift-ime
CONF=alt-shift-ime.conf
LOG=/var/log/alt-shift-ime.log
START=1

for arg in "$@"; do
  case "$arg" in
    --no-start) START=0 ;;
    -h|--help)
      sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'
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

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

pick_file() {
  for candidate in "$@"; do
    if [ -f "$candidate" ]; then
      echo "$candidate"
      return 0
    fi
  done
  return 1
}

if [ -z "${BINARY:-}" ]; then
  BINARY=$(pick_file \
    "$SCRIPT_DIR/../dist/alt-shift-ime" \
    "$SCRIPT_DIR/alt-shift-ime" \
    "$SCRIPT_DIR/../alt-shift-ime" \
    "$DESTDIR/alt-shift-ime" \
    "$BINDIR/alt-shift-ime") || {
    echo "cannot find the alt-shift-ime binary; set BINARY=/path/to/alt-shift-ime" >&2
    exit 1
  }
fi
if [ ! -f "$BINARY" ]; then
  echo "binary not found: $BINARY" >&2
  exit 1
fi

CONFIG_SRC=$(pick_file \
  "$SCRIPT_DIR/../config/alt-shift-ime.conf" \
  "$SCRIPT_DIR/alt-shift-ime.conf" \
  "$DESTDIR/alt-shift-ime.conf") || CONFIG_SRC=

JOB_SRC=$(pick_file \
  "$SCRIPT_DIR/../init/alt-shift-ime.conf" \
  "$SCRIPT_DIR/alt-shift-ime.upstart.conf" \
  "$DESTDIR/alt-shift-ime.upstart.conf") || {
  echo "cannot find the upstart job file" >&2
  exit 1
}

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
  echo "warning: could not remount / read-write; /etc/init may not be writable" >&2
fi

echo "== creating $DESTDIR"
mkdir -p "$DESTDIR" "$CONFDIR" "$BINDIR"

BACKUP="$DESTDIR/backup-$TIMESTAMP"
if [ -e "$INITDIR/$JOB" ] || [ -e "$BINDIR/$BIN" ] || [ -e "$CONFDIR/$CONF" ]; then
  echo "== backing up existing files to $BACKUP"
  mkdir -p "$BACKUP"
  for f in "$INITDIR/$JOB" "$BINDIR/$BIN" "$CONFDIR/$CONF"; do
    if [ -e "$f" ]; then
      cp -a "$f" "$BACKUP/" 2>/dev/null || true
    fi
  done
fi

echo "== installing $BINDIR/$BIN"
install -m 0755 "$BINARY" "$BINDIR/$BIN"

if [ -n "$CONFIG_SRC" ]; then
  if [ -e "$CONFDIR/$CONF" ]; then
    echo "== keeping existing $CONFDIR/$CONF"
  else
    echo "== installing default $CONFDIR/$CONF"
    install -m 0644 "$CONFIG_SRC" "$CONFDIR/$CONF"
  fi
  install -m 0644 "$CONFIG_SRC" "$DESTDIR/alt-shift-ime.conf"
else
  echo "warning: no default config found; the daemon will use built-in defaults" >&2
fi

echo "== installing $INITDIR/$JOB"
install -m 0644 "$JOB_SRC" "$INITDIR/$JOB"
install -m 0644 "$JOB_SRC" "$DESTDIR/alt-shift-ime.upstart.conf"

echo "== storing restore copies"
install -m 0755 "$0" "$DESTDIR/setup-alt-shift-ime.sh"
if [ -f "$SCRIPT_DIR/restore-alt-shift-ime.sh" ]; then
  install -m 0755 "$SCRIPT_DIR/restore-alt-shift-ime.sh" "$DESTDIR/restore-alt-shift-ime.sh"
fi
if [ -f "$SCRIPT_DIR/uninstall-alt-shift-ime.sh" ]; then
  install -m 0755 "$SCRIPT_DIR/uninstall-alt-shift-ime.sh" "$DESTDIR/uninstall-alt-shift-ime.sh"
fi

restorecon -v "$BINDIR/$BIN" "$INITDIR/$JOB" 2>/dev/null || true
restorecon -Rv "$DESTDIR" 2>/dev/null || true

echo "== reloading upstart configuration"
initctl reload-configuration

if [ "$START" = 1 ]; then
  if initctl status alt-shift-ime 2>/dev/null | grep -q 'start/running'; then
    echo "== restarting alt-shift-ime"
    initctl restart alt-shift-ime
  else
    echo "== starting alt-shift-ime"
    initctl start alt-shift-ime
  fi
  sleep 1
  initctl status alt-shift-ime || true
  echo "== last log lines"
  tail -n 10 "$LOG" 2>/dev/null || true
else
  echo "== not starting alt-shift-ime (--no-start)"
fi

echo "== done"
echo "binary:  $BINDIR/$BIN"
echo "config:  $CONFDIR/$CONF"
echo "job:     $INITDIR/$JOB"
echo "restore: $DESTDIR/restore-alt-shift-ime.sh"
echo "log:     $LOG"
