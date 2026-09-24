// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package main

import (
	_ "embed"

	"projectfile.org/projectfile/cli/internal/cmd"
)

//go:embed projectfile.yaml
var projectfileYAML []byte

func main() {
	cmd.SetProjectfileYAML(projectfileYAML)
	cmd.Execute()
}
