# Диагностика и типовые проблемы

## Быстрая проверка

Сначала выполните:

```bash
alt-shift-ime version
alt-shift-ime probe
initctl status alt-shift-ime
tail -n 50 /var/log/alt-shift-ime.log
dmesg | grep -i avc | tail -20
```

Полезные foreground-проверки:

```bash
# Ловить аккорд без инжекта
alt-shift-ime selftest --no-inject

# Убедиться, что виртуальная клавиатура доходит до ChromeOS
alt-shift-ime inject --combo ctrl+space

# Демон в foreground с подробным логом
initctl stop alt-shift-ime
alt-shift-ime run --verbose
```

После foreground-теста не забудьте снова запустить Upstart-job:

```bash
initctl start alt-shift-ime
```

## `initctl: Unknown job: alt-shift-ime`

Upstart-job отсутствует. Это ожидаемо после обновления ChromeOS или после
удаления.

Автоматическое восстановление должно вернуть его при загрузке. Проверьте:

```bash
ls -l /etc/init/alt-shift-ime.conf
cat /var/log/alt-shift-ime-restore.log
ls -l /usr/local/lib/udev/rules.d/99-alt-shift-ime.rules
```

Ручной fallback:

```bash
bash /usr/local/lib/alt-shift-ime/restore-alt-shift-ime.sh
```

Если restore-скрипта тоже нет, переустановите из release-tarball.

## Job не запущен

```bash
initctl status alt-shift-ime
tail -n 50 /var/log/alt-shift-ime.log
```

Частые причины:

- **Lock занят другим процессом.** В логе:

  ```
  another alt-shift-ime instance holds /run/alt-shift-ime.lock
  ```

  Проверьте, не остался ли foreground `run` или `selftest`:

  ```bash
  ps aux | grep alt-shift-ime
  fuser -v /run/alt-shift-ime.lock 2>/dev/null || true
  ```

  Остановите другой процесс, затем:

  ```bash
  initctl start alt-shift-ime
  ```

- **Нет `/dev/uinput`.** См. следующий раздел.

- **Upstart сдался после нескольких respawn.** Найдите в логе реальную ошибку,
  исправьте её, затем:

  ```bash
  initctl start alt-shift-ime
  ```

## `/dev/uinput` отсутствует или недоступен для записи

```bash
ls -l /dev/uinput
grep -w uinput /proc/modules || modprobe uinput
```

Ожидаемые права: `root:uinput 0660` или `root:root 0600`. Демон запускается от
root.

Если узел не появляется даже после `modprobe`, возможно, в ядре нет
`CONFIG_INPUT_UINPUT`. Для OpenFyde это необычно; проверьте конфиг ядра или
используйте сборку с этим модулем.

## `probe` не находит клавиатуру

`probe` показывает только устройства, похожие на клавиатуры или объявляющие
левые Alt и Shift. Используйте:

```bash
alt-shift-ime probe --all
```

Ищите устройство с:

```
chord keys: true (left_alt=true left_shift=true)
```

Если ни у одного устройства нет обоих ключей, демон не увидит аккорд.
Возможные причины:

- клавиатура не подключена или ещё не проинициализирована;
- клавиатура отдаёт модификаторы на другом event-узле (редко, но некоторые
  составные устройства разделяют клавиши по узлам);
- устройство недоступно для чтения от root (необычно);
- вы находитесь внутри Crostini, где `/dev/input` — не хостовое input-дерево.

## Аккорд ловится, но раскладка не переключается

Если в логе есть:

```
chord: left Alt + left Shift released cleanly ...
injected ctrl+space
```

значит, демон сделал свою работу. Проверьте сторону ChromeOS:

1. `Ctrl+Space` должен быть назначен на переключение метода ввода. Откройте
   настройки клавиатуры ChromeOS и проверьте шорткат.
2. Попробуйте прямой инжект:

   ```bash
   alt-shift-ime inject --combo ctrl+space
   ```

   Если это тоже ничего не делает, виртуальная клавиатура не доходит до
   ChromeOS.

3. Попробуйте другую заведомо рабочую комбинацию, например:

   ```bash
   alt-shift-ime inject --combo search+space
   ```

   и при необходимости настройте её в `/usr/local/etc/alt-shift-ime.conf`.

4. Убедитесь, что включён хотя бы один метод ввода / раскладка. Если раскладка
   одна, `Ctrl+Space` нечего переключать.

## `inject` работает, а аккорд — нет

- Запустите `selftest --no-inject` и нажмите аккорд. Если строки `chord:` нет,
  демон не видит физические события. Перепроверьте `probe`.
- Нажимайте именно **левые** Alt и Shift. Правые Alt/Shift намеренно
  игнорируются.
- Не нажимайте другие клавиши, пока удерживаете модификаторы.
  `Alt+Shift+Tab`, `Alt+Shift+стрелка` и т.п. отменяют аккорд по замыслу.
- Отпустите оба модификатора. Срабатывание происходит по отпусканию, а не по
  нажатию.
- Подождите минимум `delay_ms` (80 мс по умолчанию) после отпускания.

## Ложные срабатывания

Детектор отменяет аккорд, если при удерживаемом модификаторе нажата другая
клавиша. Если странные переключения всё равно есть:

```bash
initctl stop alt-shift-ime
alt-shift-ime run --verbose
```

и посмотрите последовательность клавиш. Затем:

- увеличьте `delay_ms`, если инжект гоняется с физическим отпусканием;
- в обычной работе держите `verbose=false`;
- проверьте, не инжектит ли `Ctrl+Space` другой инструмент.

## `Alt+Shift+Tab` переключает раскладку

Не должен. Детектор отменяет аккорд при нажатии Tab. Если это всё равно
происходит:

- проверьте установленную версию:

  ```bash
  alt-shift-ime version
  ```

- запустите `selftest --verbose` и убедитесь, что Tab виден как key-событие;
- переустановите последний релиз.

## Внешняя USB-клавиатура не подхватывается

Демон пересканирует `/dev/input` каждые `rescan_seconds` (2 секунды по
умолчанию). После подключения клавиатуры:

```bash
alt-shift-ime probe
tail -n 20 /var/log/alt-shift-ime.log
```

В логе должна появиться новая строка `watching ...`. Если её нет:

- уменьшите `rescan_seconds`, например до `1`;
- проверьте `dmesg` на ошибки USB-инициализации;
- запустите `probe --all`, чтобы увидеть, появляется ли устройство вообще;
- помните: клавиатура должна объявлять и `KEY_LEFTALT`, и `KEY_LEFTSHIFT` на
  одном event-узле.

Hotplug проверен на физическом внешнем USB-железе. Если конкретная клавиатура
всё равно не работает, приложите вывод `probe --all` и лог демона.

## После обновления ChromeOS

Job в `/etc/init` исчез, но `/usr/local` уцелел. udev-хук должен восстановить
его автоматически:

```bash
cat /var/log/alt-shift-ime-restore.log
initctl status alt-shift-ime
```

Ожидаемая последовательность в логе:

```
== ... boot restore helper started
== installing the Upstart job for the next boot
== reloading Upstart
== stopping directly started daemon pid ...
== starting alt-shift-ime under Upstart
alt-shift-ime start/running, process ...
```

Если лог отсутствует или пуст:

1. проверьте, что правило на месте:

   ```bash
   ls -l /usr/local/lib/udev/rules.d/99-alt-shift-ime.rules
   ```

2. проверьте, что скрипты хука на месте:

   ```bash
   ls -l /usr/local/lib/alt-shift-ime/boot-restore*.sh
   ```

3. проверьте маркер «один раз за загрузку». Это tmpfs-файл, он должен исчезать
   после перезагрузки:

   ```bash
   ls -l /run/alt-shift-ime-boot-hook
   ```

4. переустановите, если чего-то не хватает. Затем запустите ручной fallback:

   ```bash
   bash /usr/local/lib/alt-shift-ime/restore-alt-shift-ime.sh
   ```

Если `/usr/local/lib/alt-shift-ime/` тоже пропал, переустановите из
release-tarball.

### Демон запускается напрямую, но не переходит под Upstart

Проверьте лог helper и процессы:

```bash
cat /var/log/alt-shift-ime-restore.log
ps aux | grep -E 'alt-shift-ime|boot-restore'
ls -ld /run/alt-shift-ime-boot-helper.lock
```

Helper ждёт завершения загрузки до 60 секунд и ещё 10 секунд. Если его убили,
перезагрузитесь или выполните:

```bash
bash /usr/local/lib/alt-shift-ime/restore-alt-shift-ime.sh
```

### Хук восстанавливает демон после удаления

Uninstaller удаляет udev-правило и создаёт
`/usr/local/lib/alt-shift-ime/disabled`. Проверьте, что произошло и то и другое:

```bash
ls -l /usr/local/lib/udev/rules.d/99-alt-shift-ime.rules
ls -l /usr/local/lib/alt-shift-ime/disabled
```

Если правило всё ещё на месте, удалите его и перезагрузите udev:

```bash
rm -f /usr/local/lib/udev/rules.d/99-alt-shift-ime.rules
udevadm control --reload-rules
```

## AVC-отказы SELinux

```bash
dmesg | grep -i avc | tail -20
```

На проверенной сборке OpenFyde отказов не было, потому что `cros_init` и
`cros_init_scripts` permissive. Если видите отказы для бинарника или
`/dev/uinput`, проверьте, что вызывался `restorecon`:

```bash
restorecon -Rv /usr/local/lib/alt-shift-ime
restorecon -v /usr/local/bin/alt-shift-ime /etc/init/alt-shift-ime.conf
```

## `noexec` на `/tmp`

ChromeOS монтирует `/tmp` с `noexec`. Вы можете увидеть:

```
warning: script from noexec mount
```

Это безвредно, если запускать скрипты через `bash /path/script.sh`. Не
полагайтесь на `./script.sh` или `test -x` для файлов в `/tmp`: mode bits могут
быть корректными, но выполнение всё равно заблокировано.

## Логи и диагностика

- Лог демона: `/var/log/alt-shift-ime.log`
- Статус Upstart: `initctl status alt-shift-ime`
- Lock-файл: `/run/alt-shift-ime.lock`
- Конфиг: `/usr/local/etc/alt-shift-ime.conf`
- Restore-копии: `/usr/local/lib/alt-shift-ime/`

Для баг-репорта приложите:

```bash
alt-shift-ime version
alt-shift-ime probe --all
initctl status alt-shift-ime
tail -n 100 /var/log/alt-shift-ime.log
dmesg | grep -i avc | tail -20
```

Перед публикацией удалите чувствительную информацию.
