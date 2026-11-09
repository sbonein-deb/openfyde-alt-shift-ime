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
UDEVDIR=/usr/local/lib/udev/rules.d
JOB=alt-shift-ime.conf
BIN=alt-shift-ime
CONF=alt-shift-ime.conf
UDEV_RULE=99-alt-shift-ime.rules
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

# install(1) fails when source and destination are the same file. That happens
# when the installer is re-run from its installed copy under $DESTDIR.
install_file() {
  local src=$1
  local dst=$2
  local mode=$3
  if [ "$(readlink -f "$src")" = "$(readlink -f "$dst" 2>/dev/null)" ]; then
    chmod "$mode" "$dst"
    return 0
  fi
  install -m "$mode" "$src" "$dst"
}

if [ -z "${BINARY:-}" ]; then
  # Prefer the file next to this script: release tarballs ship the binary and
  # the installer in the same directory. Fall back to a repository checkout.
  BINARY=$(pick_file \
    "$SCRIPT_DIR/alt-shift-ime" \
    "$SCRIPT_DIR/../dist/alt-shift-ime" \
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
  "$SCRIPT_DIR/alt-shift-ime.conf" \
  "$SCRIPT_DIR/../config/alt-shift-ime.conf" \
  "$DESTDIR/alt-shift-ime.conf") || CONFIG_SRC=

JOB_SRC=$(pick_file \
  "$SCRIPT_DIR/alt-shift-ime.upstart.conf" \
  "$SCRIPT_DIR/../init/alt-shift-ime.conf" \
  "$DESTDIR/alt-shift-ime.upstart.conf") || {
  echo "cannot find the upstart job file" >&2
  exit 1
}

BOOT_RESTORE_SRC=$(pick_file \
  "$SCRIPT_DIR/boot-restore.sh" \
  "$SCRIPT_DIR/../scripts/boot-restore.sh" \
  "$DESTDIR/boot-restore.sh") || {
  echo "cannot find boot-restore.sh" >&2
  exit 1
}

BOOT_RESTORE_LATE_SRC=$(pick_file \
  "$SCRIPT_DIR/boot-restore-late.sh" \
  "$SCRIPT_DIR/../scripts/boot-restore-late.sh" \
  "$DESTDIR/boot-restore-late.sh") || {
  echo "cannot find boot-restore-late.sh" >&2
  exit 1
}

UDEV_RULE_SRC=$(pick_file \
  "$SCRIPT_DIR/$UDEV_RULE" \
  "$SCRIPT_DIR/../udev/$UDEV_RULE" \
  "$DESTDIR/$UDEV_RULE") || {
  echo "cannot find $UDEV_RULE" >&2
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
if [ -e "$INITDIR/$JOB" ] || [ -e "$BINDIR/$BIN" ] || [ -e "$CONFDIR/$CONF" ] || [ -e "$UDEVDIR/$UDEV_RULE" ]; then
  echo "== backing up existing files to $BACKUP"
  mkdir -p "$BACKUP"
  for f in "$INITDIR/$JOB" "$BINDIR/$BIN" "$CONFDIR/$CONF" "$UDEVDIR/$UDEV_RULE"; do
    if [ -e "$f" ]; then
      cp -a "$f" "$BACKUP/" 2>/dev/null || true
    fi
  done
fi

echo "== installing $BINDIR/$BIN"
install_file "$BINARY" "$BINDIR/$BIN" 0755

echo "== clearing disabled marker"
rm -f "$DESTDIR/disabled"

if [ -n "$CONFIG_SRC" ]; then
  if [ -e "$CONFDIR/$CONF" ]; then
    echo "== keeping existing $CONFDIR/$CONF"
  else
    echo "== installing default $CONFDIR/$CONF"
    install_file "$CONFIG_SRC" "$CONFDIR/$CONF" 0644
  fi
  install_file "$CONFIG_SRC" "$DESTDIR/alt-shift-ime.conf" 0644
else
  echo "warning: no default config found; the daemon will use built-in defaults" >&2
fi

echo "== installing $INITDIR/$JOB"
install_file "$JOB_SRC" "$INITDIR/$JOB" 0644
install_file "$JOB_SRC" "$DESTDIR/alt-shift-ime.upstart.conf" 0644

echo "== installing $UDEVDIR/$UDEV_RULE"
mkdir -p "$UDEVDIR"
install_file "$UDEV_RULE_SRC" "$UDEVDIR/$UDEV_RULE" 0644
install_file "$UDEV_RULE_SRC" "$DESTDIR/$UDEV_RULE" 0644

echo "== storing restore copies"
install_file "$0" "$DESTDIR/setup-alt-shift-ime.sh" 0755
install_file "$BOOT_RESTORE_SRC" "$DESTDIR/boot-restore.sh" 0755
install_file "$BOOT_RESTORE_LATE_SRC" "$DESTDIR/boot-restore-late.sh" 0755
if [ -f "$SCRIPT_DIR/restore-alt-shift-ime.sh" ]; then
  install_file "$SCRIPT_DIR/restore-alt-shift-ime.sh" "$DESTDIR/restore-alt-shift-ime.sh" 0755
elif [ -f "$SCRIPT_DIR/../scripts/restore-alt-shift-ime.sh" ]; then
  install_file "$SCRIPT_DIR/../scripts/restore-alt-shift-ime.sh" "$DESTDIR/restore-alt-shift-ime.sh" 0755
fi
if [ -f "$SCRIPT_DIR/uninstall-alt-shift-ime.sh" ]; then
  install_file "$SCRIPT_DIR/uninstall-alt-shift-ime.sh" "$DESTDIR/uninstall-alt-shift-ime.sh" 0755
elif [ -f "$SCRIPT_DIR/../scripts/uninstall-alt-shift-ime.sh" ]; then
  install_file "$SCRIPT_DIR/../scripts/uninstall-alt-shift-ime.sh" "$DESTDIR/uninstall-alt-shift-ime.sh" 0755
fi

restorecon -v "$BINDIR/$BIN" "$INITDIR/$JOB" "$UDEVDIR/$UDEV_RULE" 2>/dev/null || true
restorecon -Rv "$DESTDIR" 2>/dev/null || true

echo "== reloading udev rules"
if command -v udevadm >/dev/null 2>&1; then
  udevadm control --reload-rules 2>/dev/null || true
fi

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
echo "udev:    $UDEVDIR/$UDEV_RULE"
echo "restore: $DESTDIR/restore-alt-shift-ime.sh"
echo "boot:    $DESTDIR/boot-restore.sh"
echo "log:     $LOG"
