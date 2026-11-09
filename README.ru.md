# openfyde-alt-shift-ime

[![CI](https://github.com/sbonein-deb/openfyde-alt-shift-ime/actions/workflows/ci.yml/badge.svg)](https://github.com/sbonein-deb/openfyde-alt-shift-ime/actions/workflows/ci.yml)

Небольшой хостовый демон для OpenFyde/ChromeOS, который переключает
раскладку/метод ввода при отпускании **левых Alt + Shift**. Он читает события
физических клавиатур через `evdev` и инжектит `Ctrl+Space` через виртуальную
клавиатуру `uinput`. Chromium/Ash не патчится, физическая клавиатура не
захватывается.

**English documentation:** [README.md](README.md)

## Зачем

В ChromeOS нет штатного переключения раскладки по Alt+Shift. Обычные обходные
пути работают только внутри Crostini или требуют патчить Chromium, что
непрактично на подписанной сборке ChromeOS/OpenFyde. Этот проект работает на
хосте, рядом с input-стеком ChromeOS, и использует существующий акселератор
`Ctrl+Space`.

`Search+Space` продолжает работать как раньше.

## Как это работает

```
физическая клавиатура
      │  evdev (только чтение, без EVIOCGRAB)
      ▼
наблюдатель alt-shift-ime ──► левые Alt + Shift отпущены чисто?
      │                              │ да
      │                              ▼
      │                     пауза 80 мс (настраивается)
      │                              │
      ▼                              ▼
 виртуальная клавиатура /dev/uinput ──► Ctrl+Space ──► смена IME в ChromeOS
```

Демон срабатывает по **отпусканию**, а не по нажатию. ChromeOS хранит
состояние модификаторов глобально: инжект, пока Alt или Shift ещё удерживаются,
дал бы `Ctrl+Alt+Shift+Space`, что не совпадает с акселератором. Поэтому демон
ждёт отпускания обоих модификаторов и небольшую паузу.

Если во время аккорда нажата любая другая клавиша, аккорд отменяется. Именно
поэтому `Alt+Shift+Tab` не переключает раскладку.

## Возможности

- Только левые `Alt` и `Shift`; правые модификаторы игнорируются.
- По умолчанию инжектится `Ctrl+Space`; комбинация настраивается.
- Срабатывание после отпускания модификаторов, задержка 80 мс по умолчанию.
- Штатный `Search+Space` не ломается.
- Наблюдение за всеми клавиатурами, которые объявляют левые Alt и Shift.
- Внешние USB-клавиатуры подхватываются на лету, без перезапуска.
- Single-instance `flock`: второй демон или `selftest` не запустится.
- При отключении клавиатуры состояние детектора сбрасывается, отложенный
  инжект отменяется.
- Статический Go-бинарник для `amd64` и `arm64`, без runtime-зависимостей.
- Диагностические команды: `probe`, `inject`, `selftest`.
- Upstart-job, installer, restore после обновления и uninstall.

## Требования

- OpenFyde/ChromeOS с Upstart (проверено на OpenFyde `16093.91.20.8`, ChromeOS
  M132, ядро 5.4).
- Root на хосте. Это **не** программа для Crostini/контейнера.
- Доступный и записываемый для root `/dev/uinput`. Модуль `uinput` обычно уже
  загружен.
- `amd64` или `arm64`.

## Установка из релиза

Скачайте tarball под вашу архитектуру со
[страницы релизов](https://github.com/sbonein-deb/openfyde-alt-shift-ime/releases),
проверьте его и запустите installer от root:

```bash
cd /tmp
curl -fsSLO https://github.com/sbonein-deb/openfyde-alt-shift-ime/releases/download/v0.2.0/alt-shift-ime_0.2.0_linux_amd64.tar.gz
curl -fsSLO https://github.com/sbonein-deb/openfyde-alt-shift-ime/releases/download/v0.2.0/checksums.txt
sha256sum -c checksums.txt --ignore-missing

rm -rf /tmp/alt-shift-ime-install
mkdir -p /tmp/alt-shift-ime-install
tar xzf alt-shift-ime_0.2.0_linux_amd64.tar.gz -C /tmp/alt-shift-ime-install --strip-components=1
bash /tmp/alt-shift-ime-install/setup-alt-shift-ime.sh
```

Для `arm64` замените `amd64` на `arm64` в имени файла.

Installer:

- перемонтирует `/` в read-write на время установки;
- делает бэкап существующих файлов в `/usr/local/lib/alt-shift-ime/`;
- ставит `/usr/local/bin/alt-shift-ime`;
- ставит `/usr/local/etc/alt-shift-ime.conf`, если конфига ещё нет;
- ставит Upstart-job `/etc/init/alt-shift-ime.conf`;
- сохраняет restore-копии в `/usr/local/lib/alt-shift-ime/`;
- перезагружает Upstart и запускает демон.

Повторный запуск безопасен.

> `/tmp` на ChromeOS смонтирован с `noexec`. Запускайте shell-скрипты через
> `bash /path/script.sh`, не полагайтесь на прямой запуск.

## Проверка

```bash
alt-shift-ime probe
initctl status alt-shift-ime
tail -n 20 /var/log/alt-shift-ime.log
```

Здоровый старт выглядит так:

```
alt-shift-ime 0.2.0 starting: combo=ctrl+space delay=80ms rescan=2s dry_run=false verbose=false
config: /usr/local/etc/alt-shift-ime.conf
lock acquired: /run/alt-shift-ime.lock
created "alt-shift-ime virtual keyboard", warming up 700ms
watching /dev/input/event2 "AT Translated Set 2 keyboard" (bus=0011 vendor=0001 product=0001 version=ab83)
ready: combo=ctrl+space delay=80ms rescan=2s dry_run=false
```

Нажмите и отпустите левые Alt+Shift. В логе должно появиться:

```
chord: left Alt + left Shift released cleanly on "AT Translated Set 2 keyboard"
TRIGGER from "AT Translated Set 2 keyboard": injecting ctrl+space
injected ctrl+space
```

## Конфигурация

Конфиг по умолчанию — `/usr/local/etc/alt-shift-ime.conf`:

```ini
combo=ctrl+space
delay_ms=80
rescan_seconds=2
# dry_run=false
# verbose=false
```

| Ключ | Значение | По умолчанию |
| --- | --- | --- |
| `combo` | Комбинация для инжекта. | `ctrl+space` |
| `delay_ms` | Задержка после отпускания последнего модификатора. | `80` |
| `rescan_seconds` | Интервал поиска новых клавиатур. | `2` |
| `dry_run` | Ловить аккорд, но не инжектить. | `false` |
| `verbose` | Логировать каждое событие клавиши. | `false` |

Поддерживаемые модификаторы: `ctrl`, `shift`, `alt`, `search` (алиасы:
`meta`, `super`, `win`). Поддерживаемые клавиши: `space`, `tab`, `enter`,
`esc`, `a`–`z`.

После изменения конфига:

```bash
initctl restart alt-shift-ime
```

Флаги командной строки перекрывают конфиг, например:

```bash
alt-shift-ime run --combo ctrl+shift+space --delay 120ms --verbose
```

## Диагностические команды

```bash
alt-shift-ime probe              # список input-устройств и определение клавиатур
alt-shift-ime probe --all        # показать все event-устройства
alt-shift-ime inject             # один раз инжектнуть Ctrl+Space
alt-shift-ime inject --combo search+space
alt-shift-ime selftest           # foreground-режим с подробным логом
alt-shift-ime selftest --no-inject
alt-shift-ime run --dry-run      # ловить аккорд без создания виртуальной клавиатуры
```

`inject` полезен, чтобы убедиться, что виртуальная клавиатура доходит до
ChromeOS, ещё до отладки детектора. `selftest` — foreground-версия демона,
берёт тот же single-instance lock.

## Восстановление после обновления ChromeOS

Обновление ChromeOS заменяет root-раздел, поэтому
`/etc/init/alt-shift-ime.conf` исчезает. `/usr/local` лежит на stateful-разделе
и переживает обновление. Восстановите job:

```bash
bash /usr/local/lib/alt-shift-ime/restore-alt-shift-ime.sh
```

Скрипт восстановит Upstart-job и конфиг, перезагрузит Upstart и запустит демон.
Перезагрузка не нужна.

## Удаление

```bash
bash /usr/local/lib/alt-shift-ime/uninstall-alt-shift-ime.sh
```

Останавливает демон, удаляет бинарник и Upstart-job, сохраняя конфиг и
restore-копии. С `--purge` удаляется всё:

```bash
bash /usr/local/lib/alt-shift-ime/uninstall-alt-shift-ime.sh --purge
```

## Сборка из исходников

Достаточно Go 1.19 или новее. Собирайте в Crostini или на любой Linux-машине;
на хосте OpenFyde Go не нужен.

```bash
git clone https://github.com/sbonein-deb/openfyde-alt-shift-ime.git
cd openfyde-alt-shift-ime

go vet ./...
go test -race ./...
./scripts/build.sh
ARCH=arm64 OUT=dist/alt-shift-ime-arm64 ./scripts/build.sh
```

Создать release-tarball и checksums:

```bash
./scripts/package.sh v0.2.0
ls dist/release/
```

## Документация

- [docs/INSTALL.ru.md](docs/INSTALL.ru.md) — установка, обновление, restore,
  удаление.
- [docs/DESIGN.ru.md](docs/DESIGN.ru.md) — архитектура и детали, специфичные
  для ChromeOS.
- [docs/TROUBLESHOOTING.ru.md](docs/TROUBLESHOOTING.ru.md) — диагностика и
  типовые сбои.
- [CHANGELOG.md](CHANGELOG.md) — история релизов.

English versions: [docs/INSTALL.md](docs/INSTALL.md),
[docs/DESIGN.md](docs/DESIGN.md),
[docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md).

## Безопасность

- Демон работает от root, потому что `evdev` и `/dev/uinput` требуют этого.
- Физические клавиатуры открываются только на чтение. `EVIOCGRAB` не
  используется, поэтому штатный input-путь ChromeOS получает все клавиши.
- Виртуальное устройство используется только для отправки настроенной
  комбинации.
- Демон не использует сеть.
- Единственность экземпляра обеспечивается `flock` на
  `/run/alt-shift-ime.lock`.

## Известные ограничения

- Поддерживается только загрузка OpenFyde/ChromeOS на Upstart. Systemd-unit
  нет.
- Правые Alt и Shift намеренно игнорируются.
- Hotplug внешней клавиатуры реализован и покрыт тестами на уровне
  детектора/устройств, но на момент первого релиза не проверялся на реальном
  внешнем железе.
- Имена клавиш в `combo` — это позиционные evdev-коды, а не символы,
  зависящие от раскладки.
- После обновления ChromeOS нужно запустить restore-скрипт (или переустановить).

## Лицензия

GPL-2.0. См. [LICENSE](LICENSE).
