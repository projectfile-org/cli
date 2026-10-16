// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package main

import (
	"projectfile.org/projectfile/cli/v3"
	"projectfile.org/projectfile/cli/v3/internal/cmd"
)

func main() {
	cmd.SetProjectfileYAML(cli.ProjectfileYAML)
	cmd.Execute()
}
