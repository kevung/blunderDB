package database

import "github.com/kevung/blunderdb/pkg/blunderdb/engine/race"

// ComputeEPCFromPosition computes the EPC blocks and the race zone (win
// probability, and money cube verdict in the exact regime) for a position.
// It delegates to engine/race, the single implementation shared with the
// serve daemon and the CLI (ADR-0009); this method only exists as a Wails
// binding surface.
func (d *Database) ComputeEPCFromPosition(position Position) (race.Result, error) {
	return race.Evaluate(&position), nil
}
