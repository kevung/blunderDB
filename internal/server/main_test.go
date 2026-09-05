package server

import (
	"os"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen/bearofftest"
)

// The daemon serves the EPC, and the one-sided table it reads left the binary
// with ADR-0027. In production `serve` generates or is pointed at one; here it
// is taken from the repository fixture, and cached across packages, by
// bearofftest. Failures panic: a TestMain has no *testing.T, and a zero-value
// one turns any failure into an unreadable "main called runtime.Goexit".
func TestMain(m *testing.M) {
	path, err := bearofftest.EnsureOneSided()
	if err != nil {
		panic("server tests: " + err.Error())
	}
	if err := engine.LoadOneSided(path); err != nil {
		panic("server tests: loading the one-sided table: " + err.Error())
	}
	os.Exit(m.Run())
}
