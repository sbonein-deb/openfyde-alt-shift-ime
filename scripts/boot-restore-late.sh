#!/bin/bash
# Finish the post-update recovery started by boot-restore.sh.
#
# Wait until the boot has settled, install /etc/init/alt-shift-ime.conf for the
# next boot, then stop the directly started daemon and let Upstart own it.
set -u

DESTDIR=/usr/local/lib/alt-shift-ime
RULE=/usr/local/lib/udev/rules.d/99-alt-shift-ime.rules
HELPER_LOCK=/run/alt-shift-ime-boot-helper.lock

# Only one helper per boot.
mkdir "$HELPER_LOCK" 2>/dev/null || exit 0
echo $$ >"$HELPER_LOCK/pid"

cleanup() {
  rm -f "$HELPER_LOCK/pid" 2>/dev/null || true
  rmdir "$HELPER_LOCK" 2>/dev/null || true
}
trap cleanup EXIT

echo "== $(date -Is) boot restore helper started"

# Wait for boot to settle.  system-services may be a long-running job or a
# task that has already completed, so only require that Upstart knows it.
i=0
while [ "$i" -lt 60 ]; do
  if [ -e /dev/uinput ] && initctl status system-services >/dev/null 2>&1; then
    break
  fi
  sleep 1
  i=$((i + 1))
done
sleep 10

if [ -e "$DESTDIR/disabled" ]; then
  echo "disabled marker present; exiting"
  exit 0
fi
if [ ! -f "$RULE" ]; then
  echo "udev rule missing; exiting"
  exit 0
fi
if [ ! -x /usr/local/bin/alt-shift-ime ]; then
  echo "binary missing; exiting"
  exit 0
fi

echo "== installing the Upstart job for the next boot"
if ! /usr/local/lib/alt-shift-ime/restore-alt-shift-ime.sh --install-only; then
  echo "restore --install-only failed"
  exit 1
fi

# An uninstall may have started while restore was running.
if [ -e "$DESTDIR/disabled" ]; then
  echo "disabled marker appeared; exiting"
  exit 0
fi

echo "== reloading Upstart"
initctl reload-configuration || true

# Stop a daemon that boot-restore.sh started directly.  Verify the PID from
# the lock file before signalling it.
pid=$(head -n 1 /run/alt-shift-ime.lock 2>/dev/null || true)
if [ -n "${pid:-}" ] && kill -0 "$pid" 2>/dev/null; then
  if grep -qa '/usr/local/bin/alt-shift-ime' "/proc/$pid/cmdline" 2>/dev/null; then
    echo "== stopping directly started daemon pid $pid"
    kill -TERM "$pid" 2>/dev/null || true
    for _ in 1 2 3 4 5; do
      kill -0 "$pid" 2>/dev/null || break
      sleep 1
    done
  fi
fi

if [ -e "$DESTDIR/disabled" ]; then
  echo "disabled marker appeared; not starting the job"
  exit 0
fi

echo "== starting alt-shift-ime under Upstart"
if initctl status alt-shift-ime 2>/dev/null | grep -q 'start/running'; then
  initctl restart alt-shift-ime || true
else
  initctl start alt-shift-ime || true
fi
sleep 1
initctl status alt-shift-ime || true
echo "== $(date -Is) boot restore helper done"
