package gui

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

// LegalMoves is the GUI's face of domain.LegalMoves: every distinct legal
// play for the position's dice, each with its steps, its resulting position
// and its notation.
//
// It takes a POSITION and not an id, exactly like the daemon's
// /v1/positions.legalMoves, because the question is a pure function of the
// board — the quiz asks it of a library position, but the scratch board of
// EDIT and EPC has dice too, and neither has an id worth passing around.
//
// It lives on App rather than on Database for the same reason the route is
// listed among the pure ones in the parity table: there is no storage behind
// it. What it exists for is the quiz's board (#294, fiche J.4), which offers
// the legal destinations by filtering these plays on what the user has
// already played — so that "which point can this checker reach" is written
// once, in Go, and not a second time in JavaScript where the two copies would
// drift the day somebody fixes a bear-off overage in one of them.
//
// No dice returns nil; a dance returns an empty slice. The caller needs to
// tell "no question to ask" from "no answer to give", and JSON keeps both.
func (a *App) LegalMoves(pos domain.Position) []domain.LegalPlay {
	return domain.LegalMoves(&pos)
}
