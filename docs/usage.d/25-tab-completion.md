<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# Tab completion

`pf-cli completion` prints the shell-completion script for the shell named as
its argument — `bash`, `zsh`, `fish` or `powershell`. Load it once per shell
session, or from your shell’s startup file:

```sh
source <(pf-cli completion bash)
source <(pf-cli completion zsh)
source <(pf-cli completion fish)
```

Every command, subcommand, flag and flag value is completed, including the
`--format` list on `get`. Print the script to inspect it, or install it under
your shell’s completion directory:

```console
$ pf-cli completion zsh --help
Generate the autocompletion script for the zsh shell.
...
```
