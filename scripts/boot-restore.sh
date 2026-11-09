#!/bin/bash
# udev hook: make alt-shift-ime survive a ChromeOS update.
#
# udev runs this from /usr/local/lib/udev/rules.d/99-alt-shift-ime.rules when a
# block or input device is added.  It is idempotent and does at most one useful
# thing per boot.
#
# Normal boot: /etc/init/alt-shift-ime.conf exists, so Upstart starts the job.
# After a ChromeOS update: the job is gone, /usr/local survived, so start the
# daemon directly and let boot-restore-late.sh install the job for next boot.
set -u

PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
export PATH

DESTDIR=/usr/local/lib/alt-shift-ime
MARKER=/run/alt-shift-ime-boot-hook
LOG=/var/log/alt-shift-ime.log
HELPER_LOG=/var/log/alt-shift-ime-restore.log

# Do the work at most once per boot.  /run is a tmpfs, so this is reset by a
# reboot.
if [ -e "$MARKER" ]; then
  exit 0
fi

# Normal case: Upstart owns the job.  Mark the boot as handled and let the
# job start on its usual "started system-services" event.
if [ -f /etc/init/alt-shift-ime.conf ]; then
  : > "$MARKER"
  exit 0
fi

# Uninstalled (or an uninstall is in progress).
if [ -e "$DESTDIR/disabled" ]; then
  exit 0
fi

# The binary must be present.  If /usr/local is not mounted yet, the rule
# itself would not have been found, so this is only a safety check.
if [ ! -x /usr/local/bin/alt-shift-ime ]; then
  exit 0
fi

# /var/log is on the encrypted stateful partition.  It is not writable until
# chromeos_startup has mounted /var; the block-device trigger after
# boot-services runs us again after that.
if [ ! -w /var/log ]; then
  exit 0
fi

# uinput.conf loads the module at system-services.  Try now as well so the
# daemon does not have to wait; the daemon retries for up to 300 s anyway.
modprobe uinput 2>/dev/null || true

echo "$(date -Is) job missing: starting alt-shift-ime directly" >>"$HELPER_LOG"

/usr/local/bin/alt-shift-ime run \
  --config /usr/local/etc/alt-shift-ime.conf \
  --uinput-timeout 300s \
  >>"$LOG" 2>&1 </dev/null &

: > "$MARKER"

# Install the Upstart job for the next boot in the background.  This also
# moves the running daemon back under Upstart once boot has settled.
/usr/local/lib/alt-shift-ime/boot-restore-late.sh >>"$HELPER_LOG" 2>&1 </dev/null &

exit 0
