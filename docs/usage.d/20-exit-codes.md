<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# Exit codes

Every non-zero exit is one documented failure class, so a script branches on
the kind of failure rather than treating every non-zero as “broken”. A
command that succeeded exits `0`; any other value means the work did not
finish.

| Code | Meaning | Examples |
| --- | --- | --- |
| `1` | Runtime failure — the document could not be read, an include could not be resolved, or a write was refused | `pf-cli get --path-file /nope.yaml identity.name`, an unreachable include, a failed write |
| `2` | Usage error — the invocation itself is wrong, and nothing ran | `pf-cli --nope`, `pf-cli bogus`, `pf-cli set` with no path |
| `3` | The requested field is absent | `pf-cli get identity.bogus`, `pf-cli get identity.name --exists` on a missing field |
| `4` | Validation failed — the document was read and violates the v1 schema | `pf-cli validate` on a document missing required fields |

```console
$ pf-cli get identity.bogus; echo $?
identity.bogus: no such field. To continue, list what is there with pf-cli get identity --format yaml, or pass --default <value> to fall back.
3
$ pf-cli bogus; echo $?
Error: unknown command "bogus" for "pf-cli". Run pf-cli --help to see every command.
2
```

Codes `2`, `3` and `4` were all `1` before this map existed. A caller that
branches on “non-zero versus zero” is unaffected; one that branched on `1`
alone now sees `3` for a missing field and `4` for a schema failure.
