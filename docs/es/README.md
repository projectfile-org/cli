<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
pf-cli-managed: yes
-->

<!-- textlint-disable terminology,common-misspellings -->

[English](../../README.md) · [Українська](../uk/README.md)

# Projectfile CLI

pf-cli es la interfaz de línea de órdenes del projectfile: lee, escribe y valida campos, resuelve includes remotos y genera artefactos derivados. Embebe el esquema JSON v1 para validación sin conexión y es la herramienta canónica para editar projectfiles localmente y en CI.

[![Stand with Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://damian-buho.github.io/support-ukraine/) [![Projectfile inside](https://badges.kiota.ch/static/v1?label=projectfile&message=inside&labelColor=0d0d0d&color=8c6723&style=flat-square)](https://projectfile.org) [![License](https://badges.kiota.ch/static/v1?label=license&message=MIT&color=1e5913&style=flat-square)](LICENSE) [![Cosign](https://badges.kiota.ch/static/v1?label=cosign&message=enabled&color=1e5913&style=flat-square)](https://docs.sigstore.dev/cosign/verifying/verify/) [![PRs welcome](https://badges.kiota.ch/static/v1?label=PRs&message=welcome&color=1e5913&style=flat-square)](CONTRIBUTING.md) [![REUSE compliance](https://api.reuse.software/badge/codeberg.org/projectfile/cli)](https://api.reuse.software/info/codeberg.org/projectfile/cli)

![Project status](https://badges.kiota.ch/static/v1?label=status&message=maintained&color=1d63ed&style=flat-square) [![Last commit on kiota.ch](https://badges.kiota.ch/gitea/last-commit/projectfile/cli?gitea_url=https://kiota.ch&label=last%20commit%20on%20kiota.ch&style=flat-square)](https://kiota.ch/projectfile/cli) [![Last commit on Codeberg](https://badges.kiota.ch/gitea/last-commit/projectfile/cli?gitea_url=https://codeberg.org&label=last%20commit%20on%20Codeberg&style=flat-square)](https://codeberg.org/projectfile/cli) [![Last commit on GitHub](https://badges.kiota.ch/github/last-commit/projectfile-org/cli?label=last%20commit%20on%20GitHub&style=flat-square)](https://github.com/projectfile-org/cli)

[![Publish pipeline on GitHub](https://github.com/projectfile-org/cli/actions/workflows/published.yaml/badge.svg?style=flat-square)](https://github.com/projectfile-org/cli/actions) [![Vulnerability audit on GitHub](https://github.com/projectfile-org/cli/actions/workflows/audited.yaml/badge.svg?style=flat-square)](https://github.com/projectfile-org/cli/actions) [![Dependency freshness on GitHub](https://github.com/projectfile-org/cli/actions/workflows/check-outdated.yaml/badge.svg?style=flat-square)](https://github.com/projectfile-org/cli/actions) [![Analysis sweep on GitHub](https://github.com/projectfile-org/cli/actions/workflows/analyze.yaml/badge.svg?style=flat-square)](https://github.com/projectfile-org/cli/actions)

[![Publish pipeline on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/published.yaml/badge.svg?style=flat-square)](https://kiota.ch/projectfile/cli/actions) [![Vulnerability audit on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/audited.yaml/badge.svg?style=flat-square)](https://kiota.ch/projectfile/cli/actions) [![Dependency freshness on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/check-outdated.yaml/badge.svg?style=flat-square)](https://kiota.ch/projectfile/cli/actions) [![Analysis sweep on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/analyze.yaml/badge.svg?style=flat-square)](https://kiota.ch/projectfile/cli/actions)

## Características

- Lectura, escritura y consulta de documentos

Consulta [Características](FEATURES.md) para ver la lista completa.

## Qué entrega este proyecto

- **Ejecutable** `pf-cli` — comando `pf-cli`
- **Imagen de contenedor** `ghcr.io/projectfile-org/cli:latest`
- **Imagen de contenedor** `damianbuho/projectfile-cli:latest`

## Instalación

### Imagen de contenedor

Descarga la imagen de contenedor publicada:

#### Descargar de GHCR — linux/amd64, linux/arm64, linux/riscv64

```sh
docker pull ghcr.io/projectfile-org/cli:latest
alias pf-cli='docker run --rm --user "$(id -u):$(id -g)" --group-add 0 --volume "$PWD:/app/ws" --workdir /app/ws ghcr.io/projectfile-org/cli:latest pf-cli'
```

#### Descargar de DockerHub — linux/amd64

```sh
docker pull damianbuho/projectfile-cli:latest
alias pf-cli='docker run --rm --user "$(id -u):$(id -g)" --group-add 0 --volume "$PWD:/app/ws" --workdir /app/ws damianbuho/projectfile-cli:latest pf-cli'
```

Las versiones estables también publican las etiquetas `X.Y.Z`, `X.Y` y `X`: descarga el nivel de precisión que quieras fijar.

Si los registros anteriores no están disponibles, descarga desde el origen:

#### Descargar de Kiota — linux/amd64

```sh
docker pull kiota.ch/projectfile/cli:latest
alias pf-cli='docker run --rm --user "$(id -u):$(id -g)" --group-add 0 --volume "$PWD:/app/ws" --workdir /app/ws kiota.ch/projectfile/cli:latest pf-cli'
```

Después, ejecútalo como si estuviera instalado; el alias ejecuta cada ejemplo tal cual sobre el directorio actual:

```sh
pf-cli --help
```

### Binario precompilado

Descarga el binario precompilado para tu plataforma desde la última versión en GitHub:

```sh
curl --fail --location --output pf-cli https://github.com/projectfile-org/cli/releases/latest/download/pf-cli-$(uname -s | tr A-Z a-z)-$(uname -m | sed -e s/x86_64/amd64/ -e s/aarch64/arm64/) && chmod +x pf-cli
./pf-cli --help
```

Publicado para: `linux/amd64`, `linux/arm64`, `linux/riscv64`, `darwin/amd64`, `darwin/arm64`

## Uso

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

Los ejemplos y la ayuda de cada comando están en [Uso](USAGE.md).

## Compilación

Clona el repositorio con sus submódulos:

```sh
git clone --recurse-submodules https://codeberg.org/projectfile/cli cli && cd cli
```

Construye la imagen de contenedor en local:

```sh
make container-build
```

- [Referencia del Makefile](../how-to/MAKEFILE.md)

Ejecuta `make` sin argumentos para el destino predeterminado; ejecuta `make help` para listar todos los destinos.

Para el bucle de desarrollo local, `make dev-container` levanta el dev-container.

Puntos de entrada de la canalización:

- `make analyze` — Ejecuta el análisis pesado (pruebas de mutación, benchmarks)
- `make audited` — Vuelve a escanear las dependencias fijadas y los artefactos publicados en busca de vulnerabilidades nuevas
- `make check-outdated` — Informa de cada dependencia fijada que va por detrás de su versión upstream
- `make ready-to-publish` — Ejecuta localmente el pipeline pseudo-CI — compila, prueba y escanea, sin publicar

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
