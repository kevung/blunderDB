// Command blunderdb-load is the offline bulk load of the public library
// (package bulkload). It ships apart from the server image on purpose: the
// web server has no way to run it.
package main

import (
	"fmt"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/bulkload"
	_ "github.com/kevung/blunderdb/pkg/blunderdb/database"
)

func main() {
	if err := bulkload.RunCLI(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
