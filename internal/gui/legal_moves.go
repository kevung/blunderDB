package gui

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

// LegalMoves is the GUI's face of domain.LegalMoves: every distinct legal
// play for the position's dice, each with its steps, its resulting position
// and its notation.
//
// It takes a position, not an id (scratch boards have none), and lives on App
// because no storage is behind it. The quiz board filters these plays, so
// move legality is written once, in Go.
//
// No dice returns nil; a dance returns an empty slice.
func (a *App) LegalMoves(pos domain.Position) []domain.LegalPlay {
	return domain.LegalMoves(&pos)
}
