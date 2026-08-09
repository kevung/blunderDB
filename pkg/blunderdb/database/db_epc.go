package database

import "github.com/kevung/blunderdb/pkg/blunderdb/engine/race"

// ComputeEPCFromPosition computes the EPC for both players from a full board
// position. It delegates to engine/race, the single implementation shared
// with the serve daemon and the CLI (ADR-0009); this method only exists as a
// Wails binding surface.
func (d *Database) ComputeEPCFromPosition(position Position) (race.EPC, error) {
	return race.ComputeEPC(&position.Board), nil
}
