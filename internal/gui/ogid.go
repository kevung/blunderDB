package gui

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

// LooksLikeOGID reports whether text carries an OpenGammon Position ID.
//
// Decoding goes through ParsePositionText; recognising one needs a small parse
// (an OGID has no prefix), kept with the reader rather than in JavaScript.
func (a *App) LooksLikeOGID(text string) bool {
	return domain.LooksLikeOGID(text)
}
