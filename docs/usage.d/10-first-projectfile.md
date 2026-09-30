<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# Create, edit and validate a projectfile

```console
$ pf-cli init --namespace org.example --name demo --license MIT --format yaml --non-interactive --no-scan
✓ created projectfile.yaml (3 fields from defaults)
$ pf-cli set identity.summary 'Demo project'
✓ set identity.summary = Demo project
$ pf-cli add keywords cli yaml
✓ add keywords: 2 added, 0 skipped
$ pf-cli get --batch identity.name identity.summary
identity.name=demo
identity.summary=Demo project
$ pf-cli get identity --format json
{
  "name": "demo",
  "namespace": "org.example",
  "summary": "Demo project"
}
$ pf-cli validate
/tmp/demo/projectfile.yaml: valid
```
