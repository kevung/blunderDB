package cli

import "github.com/kevung/blunderdb/internal/server"

// runMCP handles the mcp command. The server lives in internal/server beside
// `call`, whose in-process dispatch over the /v1 handlers it shares.
func (cli *CLI) runMCP(args []string) error {
	return server.RunMCP(args, appVersion)
}
