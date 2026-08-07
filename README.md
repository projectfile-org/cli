<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

<!-- pf-cli-managed: yes -->
# projectfile/cli

pf-cli — the projectfile document-backend CLI (get/set/convert/validate/…)

[![License](https://img.shields.io/badge/license-MIT-4c1?style=flat-square)](LICENSE) [![PRs welcome](https://img.shields.io/badge/PRs-welcome-4c1?style=flat-square)](CONTRIBUTING.md) [![REUSE compliance](https://api.reuse.software/badge/codeberg.org/projectfile/cli)](https://api.reuse.software/info/codeberg.org/projectfile/cli)

![Project status](https://img.shields.io/badge/status-maintained-1d63ed?style=flat-square) [![Last commit](https://img.shields.io/gitea/last-commit/projectfile/cli?gitea_url=https://codeberg.org&style=flat-square)](https://codeberg.org/projectfile/cli)

[![Build status on kiota.ch](https://kiota.ch/projectfile/cli/badges/workflows/published.yaml/badge.svg)](https://kiota.ch/projectfile/cli/actions)

## Features

- Document read, write and query

See [Features](FEATURES.md) for the full list.

## What this provides

- **Executable** `pf-cli`
- **Container image** `kiota.ch/projectfile/cli:latest`

## Installation

Pull the published container image:

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

- [Makefile reference](docs/MAKEFILE.md)

Pipeline entry points:

- `make analyze` — Run the heavy analysis sweep (mutation testing, benchmarks)
- `make audited` — Re-scan the pinned dependencies and published artifacts for new vulnerabilities
- `make check-outdated` — Report every pinned dependency that lags upstream
- `make published` — Build, test, scan and publish the release artifacts

## Policies

- [How to contribute](CONTRIBUTING.md)
- [Security policy](SECURITY.md)
- [Getting support](SUPPORT.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)

## Links

### Project

- [Source Code on Codeberg](https://codeberg.org/projectfile/cli)
- [Source Code on GitHub](https://github.com/damian-buho/projectfile-cli)
- [Source Code on kiota.ch](https://kiota.ch/projectfile/cli)
- [Issues on Codeberg](https://codeberg.org/projectfile/cli/issues)
- [Issues on GitHub](https://github.com/damian-buho/projectfile-cli/issues)

## License

This project is licensed under MIT — see the [LICENSE](LICENSE) file for details.
