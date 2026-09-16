<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
pf-cli-managed: yes
-->

[Español](docs/es/README.md) · [Українська](docs/uk/README.md)

# Projectfile CLI

pf-cli is the CLI for reading, writing and validating projectfiles

[![Stand with Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://damian-buho.github.io/support-ukraine/) [![Projectfile inside](https://badges.kiota.ch/static/v1?label=projectfile&message=inside&labelColor=0d0d0d&color=8c6723&style=flat-square)](https://projectfile.org) [![License](https://badges.kiota.ch/static/v1?label=license&message=MIT&color=1e5913&style=flat-square)](LICENSE) ![Commit style](https://badges.kiota.ch/static/v1?label=commits&message=conventional&color=1877aa&style=flat-square) ![Workflow](https://badges.kiota.ch/static/v1?label=workflow&message=git-flow&color=1877aa&style=flat-square) ![Versioning](https://badges.kiota.ch/static/v1?label=versioning&message=semantic&color=1877aa&style=flat-square) [![PRs welcome](https://badges.kiota.ch/static/v1?label=PRs&message=welcome&color=1e5913&style=flat-square)](CONTRIBUTING.md) [![Citation](https://badges.kiota.ch/static/v1?label=citation&message=cff&color=1877aa&style=flat-square)](CITATION.cff) [![REUSE compliance](https://api.reuse.software/badge/codeberg.org/projectfile/cli)](https://api.reuse.software/info/codeberg.org/projectfile/cli)

![Project status](https://badges.kiota.ch/static/v1?label=status&message=maintained&color=1d63ed&style=flat-square) [![Last commit on kiota.ch](https://badges.kiota.ch/gitea/last-commit/projectfile/cli?gitea_url=https://kiota.ch&style=flat-square)](https://kiota.ch/projectfile/cli) [![Latest release](https://badges.kiota.ch/gitea/v/release/projectfile/cli?gitea_url=https://codeberg.org&style=flat-square)](https://codeberg.org/projectfile/cli/releases) [![Last commit on GitHub](https://badges.kiota.ch/github/last-commit/projectfile-org/cli?style=flat-square)](https://github.com/projectfile-org/cli)

[![Publish pipeline on GitHub](https://github.com/projectfile-org/cli/actions/workflows/published.yaml/badge.svg?style=flat-square)](https://github.com/projectfile-org/cli/actions) [![Vulnerability audit on GitHub](https://github.com/projectfile-org/cli/actions/workflows/audited.yaml/badge.svg?style=flat-square)](https://github.com/projectfile-org/cli/actions) [![Dependency freshness on GitHub](https://github.com/projectfile-org/cli/actions/workflows/check-outdated.yaml/badge.svg?style=flat-square)](https://github.com/projectfile-org/cli/actions) [![Analysis sweep on GitHub](https://github.com/projectfile-org/cli/actions/workflows/analyze.yaml/badge.svg?style=flat-square)](https://github.com/projectfile-org/cli/actions)

[![Publish pipeline on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/published.yaml/badge.svg?style=flat-square)](https://kiota.ch/projectfile/cli/actions) [![Vulnerability audit on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/audited.yaml/badge.svg?style=flat-square)](https://kiota.ch/projectfile/cli/actions) [![Dependency freshness on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/check-outdated.yaml/badge.svg?style=flat-square)](https://kiota.ch/projectfile/cli/actions) [![Analysis sweep on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/analyze.yaml/badge.svg?style=flat-square)](https://kiota.ch/projectfile/cli/actions)

## Features

- Document read, write and query

See [FEATURES.md](FEATURES.md) for the full list.

## What this provides

- **Executable** `pf-cli`
- **Container image** `ghcr.io/projectfile-org/cli:latest`

## Supported platforms

- `linux/amd64`
- `linux/arm64`
- `linux/riscv64`

## Installation

Pull the published container image:

### Pull from GHCR

```sh
docker pull ghcr.io/projectfile-org/cli:latest
```

Stable releases also publish `X.Y.Z`, `X.Y` and `X` tags — pull the precision you want to pin.

If the registries above are unreachable, pull from the origin instead:

### Pull from Kiota

```sh
docker pull kiota.ch/projectfile/cli:latest
```

## Usage

Read and write projectfile fields from the shell:

```sh
pf-cli get identity.name
pf-cli get 'links[type=source-code].url'
pf-cli set org.projectfile.status maintained
pf-cli validate
```

## Building

Run `make` with no arguments for the default target; run `make help` to list every target.

For the local dev loop, `make dev-container` brings up the dev-container.

Pipeline entry points:

- `make analyze` — Run the heavy analysis sweep (mutation testing, benchmarks)
- `make audited` — Re-scan the pinned dependencies and published artifacts for new vulnerabilities
- `make check-outdated` — Report every pinned dependency that lags upstream
- `make ready-to-publish` — Run the pseudo-CI pipeline locally — build, test and scan, without publishing

## Policies

- [How to contribute](CONTRIBUTING.md)
- [Security policy](SECURITY.md)
- [Getting support](SUPPORT.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [AI and LLM Policy](AI_POLICY.md)

## Links

- [Projectfile Specification](https://projectfile.org)

## License

This project is licensed under MIT — see the [LICENSE](LICENSE) file for details.
