<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# Configuration

pf-cli reads two environment variables, each mirroring a flag. The flag wins when both are set.

| Variable | Type | Default | Effect |
| --- | --- | --- | --- |
| `PF_CLI_VERBOSE` | boolean | unset | Same as `--verbose` — show each step |
| `PF_CLI_NO_COLOR` | boolean | unset | `1` forces `--colors never` |
| `PF_CACHE_TTL` | duration | 1h | Freshness window for cached HTTP includes |
| `PF_INCLUDE_CACHE_TTL` | duration | 1h | Alias of `PF_CACHE_TTL`, read second |
| `XDG_CACHE_HOME` | path | platform default | Where the shared `pf/` cache lives |

Both booleans are read with `strconv.ParseBool`, so `1`, `t`, `T`, `TRUE`, `true` and `True` enable them and `0`, `f` and `false` disable them. Any other value — `yes` included — is silently ignored, so set `1` or `0` and nothing else. A duration is parsed with `time.ParseDuration` (`30m`, `2h`); an unparsable value falls back to the default rather than failing.

## Precedence

Every setting resolves in one order, first match wins:

1. command-line flag
2. environment variable
3. your per-user config — `$XDG_CONFIG_HOME/projectfile/cli.*`, written by `pf-cli setup`
4. the built-in default

## Debugging your configuration

Pass `--ignore-user-config` to skip the user-config layer entirely. It answers “is this behaviour coming from my personal config?” in one run, and makes a run in CI reproducible against a teammate’s setup.
