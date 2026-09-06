package gui

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

// LooksLikeOGID reports whether text carries an OpenGammon Position ID (#260).
//
// The decoding itself is NOT bound: parser.ParsePosition reads an OGID like it
// reads an XGID, so the frontend's existing ParsePositionText already returns
// the position. What the interface cannot do on its own is RECOGNISE one — an
// OGID has no mandatory prefix, so the test is a small parse and belongs with
// the reader rather than reimplemented in JavaScript.
func (a *App) LooksLikeOGID(text string) bool {
	return domain.LooksLikeOGID(text)
}
