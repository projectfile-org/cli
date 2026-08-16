<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
pf-cli-managed: yes
-->

<!-- textlint-disable terminology,common-misspellings -->

[English](../../README.md) · [Español](../es/README.md)

# Projectfile CLI

pf-cli — CLI для читання, запису та валідації projectfile-файлів

[![Stand with Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://damian-buho.github.io/support-ukraine/) [![License](https://img.shields.io/static/v1?label=license&message=MIT&color=4c1&style=flat-square)](LICENSE) ![Commit style](https://img.shields.io/static/v1?label=commits&message=conventional&color=blue&style=flat-square) ![Workflow](https://img.shields.io/static/v1?label=workflow&message=git-flow&color=blue&style=flat-square) ![Versioning](https://img.shields.io/static/v1?label=versioning&message=semantic&color=blue&style=flat-square) [![PRs welcome](https://img.shields.io/static/v1?label=PRs&message=welcome&color=4c1&style=flat-square)](CONTRIBUTING.md) [![Citation](https://img.shields.io/static/v1?label=citation&message=cff&color=blue&style=flat-square)](CITATION.cff)

![Project status](https://img.shields.io/static/v1?label=status&message=maintained&color=1d63ed&style=flat-square)

[![Build status on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/published.yaml/badge.svg)](https://kiota.ch/projectfile/cli/actions)

## Можливості

- Читання, запис і запити до документів

Див. [FEATURES.md](FEATURES.md), щоб переглянути повний перелік.

## Що надає цей проєкт

- **Виконуваний файл** `pf-cli`
- **Образ контейнера** `ghcr.io/damian-buho/projectfile/cli:latest`
- **Образ контейнера** `docker.io/damianbuho/projectfile-cli:latest`

## Встановлення

Завантажте опублікований образ контейнера:

```sh
docker pull ghcr.io/damian-buho/projectfile/cli:latest
docker pull docker.io/damianbuho/projectfile-cli:latest
```

Якщо наведені вище реєстри недоступні, завантажте з джерела:

```sh
docker pull kiota.ch/projectfile/cli:latest
```

## Використання

Read and write projectfile fields from the shell:

```sh
pf-cli get identity.name
pf-cli get 'links[type=source-code].url'
pf-cli set org.projectfile.status maintained
pf-cli validate
```

## Збирання

- [Довідник із Makefile](../MAKEFILE.md)

Точки входу конвеєра:

- `make analyze` — Run the heavy analysis sweep (mutation testing, benchmarks)
- `make audited` — Re-scan the pinned dependencies and published artifacts for new vulnerabilities
- `make check-outdated` — Report every pinned dependency that lags upstream
- `make ready-to-publish` — Run the pseudo-CI pipeline locally — build, test and scan, without publishing

Виконайте `make` без аргументів для типової цілі; виконайте `make help`, щоб переглянути всі цілі.

Для локального циклу розробки `make dev-container` піднімає dev-container.

## Політики

- [Як зробити внесок](CONTRIBUTING.md)
- [Політика безпеки](SECURITY.md)
- [Як отримати підтримку](SUPPORT.md)
- [Кодекс поведінки](CODE_OF_CONDUCT.md)

## Посилання

### Проєкт

- [Специфікація Projectfile](https://projectfile.org)
- [Projectfile CLI на kiota.ch](https://kiota.ch/projectfile/cli)

### Інше

- [Від автора](https://dbuho.me)

## Ліцензія

Цей проєкт ліцензовано на умовах MIT — див. файл [LICENSE](LICENSE) для подробиць.

*Згенеровано з projectfile ([дізнатися як](https://projectfile.org/how-to/readme))*
<!-- textlint-enable -->
