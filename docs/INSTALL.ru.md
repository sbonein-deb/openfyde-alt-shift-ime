# Установка

## Перед началом

- Программа работает на **хосте OpenFyde/ChromeOS**, а не в Crostini.
- Нужен root-shell на хосте.
- Проверьте, что `/dev/uinput` существует и модуль `uinput` загружен:

  ```bash
  ls -l /dev/uinput
  grep -w uinput /proc/modules || modprobe uinput
  ```

- Проверьте, что хотя бы одна клавиатура объявляет левые Alt и Shift:

  ```bash
  alt-shift-ime probe
  ```

  Если бинарник ещё не установлен, используйте `probe` из распакованного
  release-tarball.

## Установка из релиза

Задайте нужные `VERSION` и `ARCH`:

```bash
VERSION=0.2.0
ARCH=amd64   # или arm64

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

Installer идемпотентен. Он сохраняет существующий
`/usr/local/etc/alt-shift-ime.conf` и делает бэкап заменяемых файлов в
`/usr/local/lib/alt-shift-ime/backup-<timestamp>/`.

### Установка без запуска

```bash
bash setup-alt-shift-ime.sh --no-start
```

Затем запустите вручную:

```bash
initctl start alt-shift-ime
```

## Ручная установка

Если не хотите запускать installer, эквивалентные шаги:

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

## Проверка

```bash
initctl status alt-shift-ime
tail -n 20 /var/log/alt-shift-ime.log
```

Ожидаемый статус:

```
alt-shift-ime start/running, process <pid>
```

Ожидаемый лог:

```
... starting: combo=ctrl+space delay=80ms rescan=2s dry_run=false verbose=false
... lock acquired: /run/alt-shift-ime.lock
... created "alt-shift-ime virtual keyboard", warming up 700ms
... watching /dev/input/event2 ...
... ready: combo=ctrl+space delay=80ms rescan=2s dry_run=false
```

Нажмите и отпустите левые Alt+Shift и проверьте в логе `injected ctrl+space`.

## Обновление

Скачайте новый tarball и снова запустите installer. Существующий конфиг
сохраняется. Перед заменой installer делает бэкап предыдущего бинарника и job.

```bash
bash setup-alt-shift-ime.sh
initctl status alt-shift-ime
```

## Восстановление после обновления ChromeOS

Обновление ChromeOS заменяет root-раздел, поэтому
`/etc/init/alt-shift-ime.conf` теряется. Бинарник, конфиг и restore-копии в
`/usr/local` переживают обновление.

```bash
bash /usr/local/lib/alt-shift-ime/restore-alt-shift-ime.sh
```

Скрипт:

1. перемонтирует `/` в read-write;
2. восстанавливает `/etc/init/alt-shift-ime.conf` из
   `/usr/local/lib/alt-shift-ime/alt-shift-ime.upstart.conf`;
3. восстанавливает конфиг по умолчанию, если он пропал;
4. перезагружает Upstart;
5. запускает или перезапускает демон;
6. возвращает `/` в read-only.

Перезагрузка не нужна.

## Удаление

Сохранить конфиг и restore-копии:

```bash
bash /usr/local/lib/alt-shift-ime/uninstall-alt-shift-ime.sh
```

Удалить всё:

```bash
bash /usr/local/lib/alt-shift-ime/uninstall-alt-shift-ime.sh --purge
```

После `--purge` каталог `/usr/local/lib/alt-shift-ime/` исчезает, поэтому
сохраните копию release-tarball, если захотите переустановить позже.

## Офлайн-установка

На машине с сетью скачайте tarball и `checksums.txt`. Перенесите оба файла на
хост, проверьте через `sha256sum -c`, распакуйте и запустите installer. Сеть на
хосте не нужна.

## Архитектуры

- `amd64` покрывает проверенное устройство OpenFyde на Celeron N3350.
- `arm64` покрывает ARM-платы OpenFyde (например, класс Raspberry Pi).
- Бинарник статический (`CGO_ENABLED=0`); glibc и Go runtime на хосте не нужны.

## Устанавливаемые файлы

| Путь | Назначение |
| --- | --- |
| `/usr/local/bin/alt-shift-ime` | Статический бинарник. |
| `/usr/local/etc/alt-shift-ime.conf` | Конфигурация. |
| `/etc/init/alt-shift-ime.conf` | Upstart-job. |
| `/usr/local/lib/alt-shift-ime/alt-shift-ime.upstart.conf` | Restore-копия job. |
| `/usr/local/lib/alt-shift-ime/alt-shift-ime.conf` | Restore-копия конфига. |
| `/usr/local/lib/alt-shift-ime/setup-alt-shift-ime.sh` | Копия installer. |
| `/usr/local/lib/alt-shift-ime/restore-alt-shift-ime.sh` | Restore после обновления. |
| `/usr/local/lib/alt-shift-ime/uninstall-alt-shift-ime.sh` | Копия uninstaller. |
| `/var/log/alt-shift-ime.log` | Лог демона. |
| `/run/alt-shift-ime.lock` | Single-instance lock. |

## Сборка и установка из исходников

В Crostini или на другой Linux-машине:

```bash
git clone https://github.com/sbonein-deb/openfyde-alt-shift-ime.git
cd openfyde-alt-shift-ime
./scripts/build.sh
```

Скопируйте `dist/alt-shift-ime` на хост и используйте `BINARY=` с installer:

```bash
BINARY=/tmp/alt-shift-ime bash setup-alt-shift-ime.sh
```

Или соберите release-tarball локально:

```bash
./scripts/package.sh v0.2.0
```
