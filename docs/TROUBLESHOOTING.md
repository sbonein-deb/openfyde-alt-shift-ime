# Troubleshooting

## Quick triage

Run these first:

```bash
alt-shift-ime version
alt-shift-ime probe
initctl status alt-shift-ime
tail -n 50 /var/log/alt-shift-ime.log
dmesg | grep -i avc | tail -20
```

Useful foreground checks:

```bash
# Detect the chord without injecting
alt-shift-ime selftest --no-inject

# Prove that the virtual keyboard can reach ChromeOS
alt-shift-ime inject --combo ctrl+space

# Verbose daemon in the foreground
initctl stop alt-shift-ime
alt-shift-ime run --verbose
```

Remember to start the Upstart job again after a foreground test:

```bash
initctl start alt-shift-ime
```

## `initctl: Unknown job: alt-shift-ime`

The Upstart job is missing. This is expected after a ChromeOS update, or after
an uninstall.

```bash
ls -l /etc/init/alt-shift-ime.conf
bash /usr/local/lib/alt-shift-ime/restore-alt-shift-ime.sh
```

If the restore script is also missing, reinstall from the release tarball.

## The job is not running

```bash
initctl status alt-shift-ime
tail -n 50 /var/log/alt-shift-ime.log
```

Common causes:

- **Lock held by another process.** Look for:

  ```
  another alt-shift-ime instance holds /run/alt-shift-ime.lock
  ```

  Check for a leftover foreground `run` or `selftest`:

  ```bash
  ps aux | grep alt-shift-ime
  fuser -v /run/alt-shift-ime.lock 2>/dev/null || true
  ```

  Stop the other process, then:

  ```bash
  initctl start alt-shift-ime
  ```

- **`/dev/uinput` missing.** See the next section.

- **Upstart gave up after repeated respawns.** Check the log for the real error,
  fix it, then:

  ```bash
  initctl start alt-shift-ime
  ```

## `/dev/uinput` is missing or not writable

```bash
ls -l /dev/uinput
grep -w uinput /proc/modules || modprobe uinput
```

Expected permissions: `root:uinput 0660` or `root:root 0600`. Run the daemon as
root.

If the device node is missing even after `modprobe`, the kernel may not have
`CONFIG_INPUT_UINPUT` enabled. This is unusual on OpenFyde; check the kernel
config or use a build that includes it.

## `probe` finds no keyboard

`probe` only shows devices that look like keyboards or advertise left Alt and
left Shift. Use:

```bash
alt-shift-ime probe --all
```

Look for a device with:

```
chord keys: true (left_alt=true left_shift=true)
```

If no device has both keys, the daemon cannot see the chord. Possible reasons:

- the keyboard is not connected or not enumerated yet;
- the keyboard exposes modifiers on a different event node (rare, but some
  composite devices split keys across nodes);
- the device is not readable by root (unusual);
- you are running inside Crostini, where `/dev/input` is not the host's input
  tree.

## The chord is detected but the layout does not switch

If the log shows:

```
chord: left Alt + left Shift released cleanly ...
injected ctrl+space
```

then the daemon did its job. Check the ChromeOS side:

1. `Ctrl+Space` must be assigned to the input-method switch. Open ChromeOS
   keyboard settings and verify the shortcut.
2. Try a direct injection:

   ```bash
   alt-shift-ime inject --combo ctrl+space
   ```

   If this also does nothing, the virtual keyboard is not reaching ChromeOS.

3. Try another combination that you know works, for example:

   ```bash
   alt-shift-ime inject --combo search+space
   ```

   and configure it in `/usr/local/etc/alt-shift-ime.conf` if needed.

4. Check that only one input method / layout is enabled. With a single layout,
   `Ctrl+Space` has nothing to switch to.

## `inject` works but the chord does not

- Run `selftest --no-inject` and press the chord. If no `chord:` line appears,
  the daemon is not seeing the physical events. Re-check `probe`.
- Make sure you press **left** Alt and **left** Shift. Right Alt/Shift are
  intentionally ignored.
- Do not press any other key while holding the modifiers. `Alt+Shift+Tab`,
  `Alt+Shift+arrow`, etc. cancel the chord by design.
- Release both modifiers. The trigger happens on release, not on press.
- Wait at least `delay_ms` (80 ms by default) after release.

## False triggers

The detector cancels the chord if any other key is pressed while either
modifier is down. If you still see unexpected switches:

```bash
initctl stop alt-shift-ime
alt-shift-ime run --verbose
```

and inspect the key sequence. Then adjust:

- increase `delay_ms` if the injection races with the physical release;
- set `verbose=false` in normal operation;
- check that no other tool is injecting `Ctrl+Space`.

## `Alt+Shift+Tab` switches the layout

It should not. The detector cancels the chord when Tab is pressed. If it still
happens:

- verify the installed version:

  ```bash
  alt-shift-ime version
  ```

- run `selftest --verbose` and check that Tab is seen as a key event;
- reinstall the latest release.

## External USB keyboard is not picked up

The daemon rescans `/dev/input` every `rescan_seconds` (2 seconds by default).
After plugging the keyboard:

```bash
alt-shift-ime probe
tail -n 20 /var/log/alt-shift-ime.log
```

The log should show a new `watching ...` line. If not:

- increase `rescan_seconds` temporarily? No — decrease it, for example to `1`;
- check `dmesg` for USB enumeration errors;
- run `probe --all` to see whether the device appears at all;
- remember that a keyboard must advertise both `KEY_LEFTALT` and
  `KEY_LEFTSHIFT` on the same event node.

The first release implemented this path but was produced without physical
external keyboard hardware, so report issues with `probe --all` output and the
daemon log.

## After a ChromeOS update

The job in `/etc/init` is gone, but `/usr/local` survived:

```bash
bash /usr/local/lib/alt-shift-ime/restore-alt-shift-ime.sh
```

If `/usr/local/lib/alt-shift-ime/` is also gone, reinstall from the release
tarball.

## SELinux AVC denials

```bash
dmesg | grep -i avc | tail -20
```

On the tested OpenFyde build there were no denials because `cros_init` and
`cros_init_scripts` are permissive. If you see denials for the binary or
`/dev/uinput`, check that `restorecon` was run:

```bash
restorecon -Rv /usr/local/lib/alt-shift-ime
restorecon -v /usr/local/bin/alt-shift-ime /etc/init/alt-shift-ime.conf
```

## `noexec` on `/tmp`

ChromeOS mounts `/tmp` with `noexec`. You may see:

```
warning: script from noexec mount
```

That is harmless when you run scripts with `bash /path/script.sh`. Do not rely
on `./script.sh` or on `test -x` for files in `/tmp`; the mode bits may be
correct but execution is still blocked.

## Logs and diagnostics

- Daemon log: `/var/log/alt-shift-ime.log`
- Upstart status: `initctl status alt-shift-ime`
- Lock file: `/run/alt-shift-ime.lock`
- Config: `/usr/local/etc/alt-shift-ime.conf`
- Restore copies: `/usr/local/lib/alt-shift-ime/`

For a bug report, include:

```bash
alt-shift-ime version
alt-shift-ime probe --all
initctl status alt-shift-ime
tail -n 100 /var/log/alt-shift-ime.log
dmesg | grep -i avc | tail -20
```

Remove any sensitive information before posting publicly.
