<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
pf-cli-managed: yes
-->

<!-- textlint-disable terminology -->

[English](../../README.md) · [Українська](../uk/README.md)

# Projectfile CLI

pf-cli is a tool for reading, writing, validating projectfiles

[![Stand with Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://damian-buho.github.io/support-ukraine/) [![License](https://img.shields.io/static/v1?label=license&message=MIT&color=4c1&style=flat-square)](LICENSE) ![Commit style](https://img.shields.io/static/v1?label=commits&message=conventional&color=blue&style=flat-square) ![Workflow](https://img.shields.io/static/v1?label=workflow&message=git-flow&color=blue&style=flat-square) ![Versioning](https://img.shields.io/static/v1?label=versioning&message=semantic&color=blue&style=flat-square) [![PRs welcome](https://img.shields.io/static/v1?label=PRs&message=welcome&color=4c1&style=flat-square)](CONTRIBUTING.md) [![Citation](https://img.shields.io/static/v1?label=citation&message=cff&color=blue&style=flat-square)](CITATION.cff)

![Project status](https://img.shields.io/static/v1?label=status&message=maintained&color=1d63ed&style=flat-square)

[![Build status on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/published.yaml/badge.svg)](https://kiota.ch/projectfile/cli/actions)

## Características

- Document read, write and query

Consulta [FEATURES.md](../../FEATURES.md) para ver la lista completa.

## Qué entrega este proyecto

- **Ejecutable** `pf-cli`
- **Imagen de contenedor** `ghcr.io/damian-buho/projectfile/cli:latest`
- **Imagen de contenedor** `docker.io/damianbuho/projectfile-cli:latest`

## Instalación

Descarga la imagen de contenedor publicada:

```sh
docker pull ghcr.io/damian-buho/projectfile/cli:latest
docker pull docker.io/damianbuho/projectfile-cli:latest
```

Si los registros anteriores no están disponibles, descarga desde el origen:

```sh
docker pull kiota.ch/projectfile/cli:latest
```

## Uso

Read and write projectfile fields from the shell:

```sh
pf-cli get identity.name
pf-cli get 'links[type=source-code].url'
pf-cli set org.projectfile.status maintained
pf-cli validate
```

## Compilación

- [Referencia del Makefile](../MAKEFILE.md)

Puntos de entrada de la canalización:

- `make analyze` — Run the heavy analysis sweep (mutation testing, benchmarks)
- `make audited` — Re-scan the pinned dependencies and published artifacts for new vulnerabilities
- `make check-outdated` — Report every pinned dependency that lags upstream
- `make ready-to-publish` — Run the pseudo-CI pipeline locally — build, test and scan, without publishing

Ejecuta `make` sin argumentos para el destino predeterminado; ejecuta `make help` para listar todos los destinos.

Para el bucle de desarrollo local, `make dev-container` levanta el dev-container.

## Políticas

- [Cómo contribuir](CONTRIBUTING.md)
- [Política de seguridad](SECURITY.md)
- [Cómo obtener ayuda](SUPPORT.md)
- [Código de conducta](CODE_OF_CONDUCT.md)

## Enlaces

- [Especificación de Projectfile](https://projectfile.org)
- [Projectfile CLI on kiota.ch](https://kiota.ch/projectfile/cli)

## Licencia

Este proyecto se publica bajo la licencia MIT — consulta el archivo [LICENSE](LICENSE) para más detalles.

*Generado desde projectfile ([saber cómo](https://projectfile.org/how-to/readme))*
<!-- textlint-enable -->
