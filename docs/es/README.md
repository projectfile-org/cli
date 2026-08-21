<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
pf-cli-managed: yes
-->

<!-- textlint-disable terminology,common-misspellings -->

[English](../../README.md) · [Українська](../uk/README.md)

# Projectfile CLI

pf-cli es la CLI para leer, escribir y validar projectfiles

[![Stand with Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://damian-buho.github.io/support-ukraine/) [![Projectfile inside](https://badges.kiota.ch/badge/projectfile-inside-c99b46?style=flat-square)](https://projectfile.org) [![License](https://badges.kiota.ch/static/v1?label=license&message=MIT&color=4c1&style=flat-square)](LICENSE) ![Commit style](https://badges.kiota.ch/static/v1?label=commits&message=conventional&color=blue&style=flat-square) ![Workflow](https://badges.kiota.ch/static/v1?label=workflow&message=git-flow&color=blue&style=flat-square) ![Versioning](https://badges.kiota.ch/static/v1?label=versioning&message=semantic&color=blue&style=flat-square) [![PRs welcome](https://badges.kiota.ch/static/v1?label=PRs&message=welcome&color=4c1&style=flat-square)](CONTRIBUTING.md) [![Citation](https://badges.kiota.ch/static/v1?label=citation&message=cff&color=blue&style=flat-square)](CITATION.cff)

![Project status](https://badges.kiota.ch/static/v1?label=status&message=maintained&color=1d63ed&style=flat-square) [![Last commit on kiota.ch](https://badges.kiota.ch/gitea/last-commit/projectfile/cli?gitea_url=https://kiota.ch&style=flat-square)](https://kiota.ch/projectfile/cli)

[![Publish pipeline on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/published.yaml/badge.svg?style=flat-square)](https://kiota.ch/projectfile/cli/actions) [![Vulnerability audit on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/audited.yaml/badge.svg?style=flat-square)](https://kiota.ch/projectfile/cli/actions) [![Dependency freshness on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/check-outdated.yaml/badge.svg?style=flat-square)](https://kiota.ch/projectfile/cli/actions) [![Analysis sweep on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/analyze.yaml/badge.svg?style=flat-square)](https://kiota.ch/projectfile/cli/actions)

## Características

- Lectura, escritura y consulta de documentos

Consulta [FEATURES.md](FEATURES.md) para ver la lista completa.

## Qué entrega este proyecto

- **Ejecutable** `pf-cli`
- **Imagen de contenedor** `ghcr.io/damian-buho/projectfile/cli:latest`
- **Imagen de contenedor** `docker.io/damianbuho/projectfile-cli:latest`

## Instalación

Descarga la imagen de contenedor publicada:

### Descargar de GHCR

```sh
docker pull ghcr.io/damian-buho/projectfile/cli:latest
```

### Descargar de DockerHub

```sh
docker pull docker.io/damianbuho/projectfile-cli:latest
```

Las versiones estables también publican las etiquetas `X.Y.Z`, `X.Y` y `X`: descarga el nivel de precisión que quieras fijar.

Si los registros anteriores no están disponibles, descarga desde el origen:

### Descargar de Kiota

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

Ejecuta `make` sin argumentos para el destino predeterminado; ejecuta `make help` para listar todos los destinos.

Para el bucle de desarrollo local, `make dev-container` levanta el dev-container.

Puntos de entrada de la canalización:

- `make analyze` — Run the heavy analysis sweep (mutation testing, benchmarks)
- `make audited` — Re-scan the pinned dependencies and published artifacts for new vulnerabilities
- `make check-outdated` — Report every pinned dependency that lags upstream
- `make ready-to-publish` — Run the pseudo-CI pipeline locally — build, test and scan, without publishing

## Políticas

- [Cómo contribuir](CONTRIBUTING.md)
- [Política de seguridad](SECURITY.md)
- [Cómo obtener ayuda](SUPPORT.md)
- [Código de conducta](CODE_OF_CONDUCT.md)
- [Política sobre IA y LLM](AI_POLICY.md)

## Enlaces

- [Especificación de Projectfile](https://projectfile.org)

## Licencia

Este proyecto se publica bajo la licencia MIT — consulta el archivo [LICENSE](LICENSE) para más detalles.

<!-- textlint-enable -->
