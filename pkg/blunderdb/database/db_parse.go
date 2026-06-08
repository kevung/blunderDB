package database

import "github.com/kevung/blunderdb/pkg/blunderdb/parser"

// ParsePositionText parses pasted clipboard / file text (a bare XGID, an XG
// human-readable export, or blunderDB's internal export format) into a Position
// plus optional analysis and comment. It is the single backend home for what
// used to be the frontend `parsePosition`; the GUI now calls this over Wails so
// the parsing logic exists in exactly one place (see pkg/blunderdb/parser).
//
// Pure: it performs no storage and takes no lock, mirroring the other "FromText"
// helpers that only parse.
func (d *Database) ParsePositionText(text string) (parser.Result, error) {
	return parser.ParsePosition(text)
}
