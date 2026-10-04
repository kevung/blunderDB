//go:build simulation

// t2-verify : la CLI de blunderDB (internal/cli) sans le GUI ni frontend/dist, pour lancer
// `tournament verify` sur la base de T2 après la fermeture brutale (CLI_USAGE.md § Tournament).
package main

import (
	"fmt"
	"os"

	"github.com/kevung/blunderdb/internal/cli"
)

func main() {
	if err := cli.NewCLI().Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
