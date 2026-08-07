<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>

SPDX-License-Identifier: MIT
-->

# Document read, write and query

- Reads and mutates dotted and selector field paths from the shell, so fields can be changed without hand-editing YAML.
- Converts between encodings (YAML/JSON), validates against the embedded schema, and strips include-redundant fields.
- Warms and purges the HTTP include-resolution cache.
- Runs an interactive first-run setup wizard for per-user configuration.
