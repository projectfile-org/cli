// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

// Package cli carries this repository’s own projectfile.yaml into the pf-cli binary.
package cli

import _ "embed"

//go:embed projectfile.yaml
var ProjectfileYAML []byte
