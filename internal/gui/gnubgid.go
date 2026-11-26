package gui

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

// GnubgID is the GUI's face of domain.EncodeGNUbgID: the identifier GNU
// Backgammon shows for the position, "<Position ID>:<Match ID>".
//
// It takes a position, not an id: scratch boards have none, and no storage is
// behind it.
func (a *App) GnubgID(pos domain.Position) (string, error) {
	return domain.EncodeGNUbgID(&pos)
}

// ArkID is the GUI's face of domain.EncodeArkID: the position as an ArkID,
// compact (the canonical form) or developed.
func (a *App) ArkID(pos domain.Position, expanded bool) (string, error) {
	if expanded {
		return domain.EncodeArkIDExpanded(&pos)
	}
	return domain.EncodeArkID(&pos)
}
