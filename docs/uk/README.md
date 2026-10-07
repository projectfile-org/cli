<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
pf-cli-managed: yes
-->

<!-- textlint-disable terminology,common-misspellings -->

[English](../../README.md) · [Español](../es/README.md)

# Projectfile CLI

pf-cli — командний інтерфейс projectfile: читає, записує та валідує поля, розвʼязує віддалені includes і генерує похідні артефакти. Містить вбудовану JSON-схему v1 для офлайн-валідації та є канонічним інструментом редагування projectfile-файлів локально й у CI.

[![Stand with Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://damian-buho.github.io/support-ukraine/) [![Projectfile inside](https://badges.kiota.ch/static/v1?label=projectfile&message=inside&labelColor=0d0d0d&color=8c6723&style=flat-square)](https://projectfile.org) [![License](https://badges.kiota.ch/static/v1?label=license&message=MIT&color=1e5913&style=flat-square)](LICENSE) [![Cosign](https://badges.kiota.ch/static/v1?label=cosign&message=enabled&color=1e5913&style=flat-square)](https://docs.sigstore.dev/cosign/verifying/verify/) [![PRs welcome](https://badges.kiota.ch/static/v1?label=PRs&message=welcome&color=1e5913&style=flat-square)](CONTRIBUTING.md) [![REUSE compliance](https://api.reuse.software/badge/codeberg.org/projectfile/cli)](https://api.reuse.software/info/codeberg.org/projectfile/cli)

![Project status](https://badges.kiota.ch/static/v1?label=status&message=maintained&color=1d63ed&style=flat-square) [![Last commit on kiota.ch](https://badges.kiota.ch/gitea/last-commit/projectfile/cli?gitea_url=https://kiota.ch&label=last%20commit%20on%20kiota.ch&style=flat-square)](https://kiota.ch/projectfile/cli) [![Last commit on Codeberg](https://badges.kiota.ch/gitea/last-commit/projectfile/cli?gitea_url=https://codeberg.org&label=last%20commit%20on%20Codeberg&style=flat-square)](https://codeberg.org/projectfile/cli) [![Last commit on GitHub](https://badges.kiota.ch/github/last-commit/projectfile-org/cli?label=last%20commit%20on%20GitHub&style=flat-square)](https://github.com/projectfile-org/cli)

[![Publish pipeline on GitHub](https://github.com/projectfile-org/cli/actions/workflows/published.yaml/badge.svg?style=flat-square)](https://github.com/projectfile-org/cli/actions) [![Vulnerability audit on GitHub](https://github.com/projectfile-org/cli/actions/workflows/audited.yaml/badge.svg?style=flat-square)](https://github.com/projectfile-org/cli/actions) [![Dependency freshness on GitHub](https://github.com/projectfile-org/cli/actions/workflows/check-outdated.yaml/badge.svg?style=flat-square)](https://github.com/projectfile-org/cli/actions) [![Analysis sweep on GitHub](https://github.com/projectfile-org/cli/actions/workflows/analyzed.yaml/badge.svg?style=flat-square)](https://github.com/projectfile-org/cli/actions)

[![Publish pipeline on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/published.yaml/badge.svg?style=flat-square)](https://kiota.ch/projectfile/cli/actions) [![Vulnerability audit on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/audited.yaml/badge.svg?style=flat-square)](https://kiota.ch/projectfile/cli/actions) [![Dependency freshness on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/check-outdated.yaml/badge.svg?style=flat-square)](https://kiota.ch/projectfile/cli/actions) [![Analysis sweep on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/analyzed.yaml/badge.svg?style=flat-square)](https://kiota.ch/projectfile/cli/actions)

## Можливості

- Читання, запис і запити до документів

Див. [Можливості](FEATURES.md), щоб переглянути повний перелік.

## Що надає цей проєкт

- **Виконуваний файл** `pf-cli` — команда `pf-cli`
- **Образ контейнера** `ghcr.io/projectfile-org/cli:latest`
- **Образ контейнера** `damianbuho/projectfile-cli:latest`

## Встановлення

### Образ контейнера

Завантажте опублікований образ контейнера:

#### Завантажити з GHCR — linux/amd64, linux/arm64, linux/riscv64

```sh
docker pull ghcr.io/projectfile-org/cli:latest
alias pf-cli='docker run --rm --user "$(id -u):$(id -g)" --group-add 0 --volume "$PWD:/app/ws" --workdir /app/ws ghcr.io/projectfile-org/cli:latest pf-cli'
```

#### Завантажити з DockerHub — linux/amd64

```sh
docker pull damianbuho/projectfile-cli:latest
alias pf-cli='docker run --rm --user "$(id -u):$(id -g)" --group-add 0 --volume "$PWD:/app/ws" --workdir /app/ws damianbuho/projectfile-cli:latest pf-cli'
```

Стабільні випуски також публікують теґи `X.Y.Z`, `X.Y` і `X` — завантажте той рівень точності, який хочете зафіксувати.

Якщо наведені вище реєстри недоступні, завантажте з джерела:

#### Завантажити з Kiota — linux/amd64

```sh
docker pull kiota.ch/projectfile/cli:latest
alias pf-cli='docker run --rm --user "$(id -u):$(id -g)" --group-add 0 --volume "$PWD:/app/ws" --workdir /app/ws kiota.ch/projectfile/cli:latest pf-cli'
```

Потім запускайте його так, ніби його встановлено, — псевдонім виконує кожен приклад як написано в поточному каталозі:

```sh
pf-cli --help
```

### Готовий бінарний файл

Завантажте готовий бінарний файл для своєї платформи з випусків на GitHub:

```sh
mkdir -p ~/.local/bin
curl --fail --location --output ~/.local/bin/pf-cli https://github.com/projectfile-org/cli/releases/latest/download/pf-cli-$(uname -s | tr A-Z a-z)-$(uname -m)
chmod +x ~/.local/bin/pf-cli
~/.local/bin/pf-cli --help
```

Опубліковано для: `linux/amd64`, `linux/arm64`, `linux/riscv64`, `darwin/amd64`, `darwin/arm64`

## Використання

### pf-cli

```console
$ pf-cli --help
pf-cli is the command-line frontend for the projectfile: it reads, writes and
validates fields, resolves remote includes, and generates derived artifacts. It
embeds the v1 JSON Schema for offline validation and is the canonical tool for
editing projectfiles locally and in CI.

File and forge sync (bridge, forge, scan): use pf-bridge.

Usage:
  pf-cli [command]

Commands:
  add         Append items to a projectfile list
  cache       Manage pf-cli’s local cache for offline use
  completion  Generate the autocompletion script for the specified shell
  convert     Convert a projectfile between encodings (toml/yaml/json)
  del         Remove a field, list item, or map entry from projectfile
  get         Read one or more projectfile fields
  help        Help about any command
  init        Scaffold a new projectfile document
  optimize    Remove local fields that duplicate include values
  set         Write a value into a projectfile field
  setup       Edit your per-user configuration interactively
  validate    Validate a projectfile against the v1 JSON Schema

Flags:
      --colors string        colour output: auto|always|never; also PF_CLI_NO_COLOR=1 (default "auto")
      --fail-on string       abort includes at error|warning (default "error")
  -h, --help                 help for pf-cli
      --ignore-user-config   skip $XDG_CONFIG_HOME/projectfile/cli.* loading
      --offline              refuse network; use cache and embedded data
  -q, --quiet                mute info; warnings and errors still print
      --sorted               write YAML keys in sorted order
      --timeout duration     per-attempt network timeout (default 10s)
  -v, --verbose              show each step; also PF_CLI_VERBOSE=1
  -V, --version              print the version

Examples:
  pf-cli init --namespace org.example --name demo
  pf-cli get identity.name
  pf-cli set license.spdx MIT
  pf-cli add keywords rust wasm
  pf-cli del keywords[0]
  pf-cli validate
  pf-cli convert yaml toml
  pf-cli optimize
  pf-cli cache status

Use "pf-cli [command] --help" for more information about a command.
```

Приклади й довідка кожної команди — у [Використання](USAGE.md).

## Збирання

Клонуйте репозиторій разом із підмодулями:

```sh
git clone --recurse-submodules https://codeberg.org/projectfile/cli cli && cd cli
```

Зберіть образ контейнера локально:

```sh
make container-build
```

- [Довідник із Makefile](../how-to/MAKEFILE.md)

Виконайте `make` без аргументів для типової цілі; виконайте `make help`, щоб переглянути всі цілі.

Для локального циклу розробки `make dev-container` піднімає dev-container.

Точки входу конвеєра:

- `make analyzed` — Запускає важкий аналіз (мутаційне тестування, бенчмарки)
- `make audited` — Повторно сканує закріплені залежності й опубліковані артефакти на нові вразливості
- `make check-outdated` — Звітує про кожну закріплену залежність, що відстає від upstream
- `make ready-to-publish` — Запускає псевдо-CI локально — збирає, тестує й сканує без публікації

## Політики

- [Як зробити внесок](CONTRIBUTING.md)
- [Політика безпеки](SECURITY.md)
- [Як отримати підтримку](SUPPORT.md)
- [Кодекс поведінки](CODE_OF_CONDUCT.md)
- [Політика щодо ШІ та LLM](AI_POLICY.md)

## Посилання

- [Специфікація Projectfile](https://projectfile.org)

## Ліцензія

Цей проєкт ліцензовано на умовах MIT — див. файл [LICENSE](LICENSE) для подробиць.

<!-- textlint-enable -->
