package cli

import "github.com/kevung/blunderdb/internal/server"

// runHealthcheck handles the healthcheck command: one GET on a running
// daemon's /readyz, exit 0 when it is ready. The probe itself lives in
// internal/server so the headless cmd/serve binary — the one the container
// image runs, without curl — offers the same subcommand.
func (cli *CLI) runHealthcheck(args []string) error {
	return server.RunHealthcheck(args)
}
