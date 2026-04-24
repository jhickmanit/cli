// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/ory/cli/cmd"
	"github.com/ory/x/profilex"
)

func main() {
	defer profilex.Profile().Stop()
	os.Exit(cmd.Execute())
}
