package ingest

import (
	"os"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/issuance/issuancetest"
)

// TestMain keeps exports from touching the real Issuer identity.
func TestMain(m *testing.M) { os.Exit(issuancetest.Run(m.Run)) }
