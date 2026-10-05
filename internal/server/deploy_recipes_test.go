package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDeployRecipesDropReadTenants: X-Read-Tenants widens a read to other tenants, so a recipe
// that knows no such relation must erase what the client sent.
func TestDeployRecipesDropReadTenants(t *testing.T) {
	for _, f := range []string{"Caddyfile", "Caddyfile.oidc", "nginx-tenant-proxy.conf"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "deploy", f))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "X-Read-Tenants") {
			t.Errorf("deploy/%s does not drop X-Read-Tenants", f)
		}
	}
}
