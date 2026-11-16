package gui

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

// LooksLikePositionID reports whether text carries an OpenGammon Position ID
// or a GNUbg identifier, the two that have no prefix to recognise them by.
//
// Decoding goes through ParsePositionText; recognising one needs a small parse,
// kept with the reader rather than in JavaScript.
func (a *App) LooksLikePositionID(text string) bool {
	if domain.LooksLikeOGID(text) {
		return true
	}
	_, ok := domain.FindGNUbgID(text)
	return ok
}
