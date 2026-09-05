package server

import (
	"os"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen/bearofftest"
)

// The daemon serves the EPC, and the one-sided table it reads left the binary
// with ADR-0027. In production `serve` generates or is pointed at one; here it
// is generated once, and cached across packages, by bearofftest.
func TestMain(m *testing.M) {
	t := &testing.T{}
	if err := engine.LoadOneSided(bearofftest.OneSidedPath(t)); err != nil {
		panic("server tests: loading the generated one-sided table: " + err.Error())
	}
	os.Exit(m.Run())
}
