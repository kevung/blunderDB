package parser

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestParsePositionWritesTheCrawfordSentinel pins what a pasted XGID says about
// the Crawford rule, on the one entry point the clipboard, `import <XGID>`, and
// /v1/positions.parseText share.
//
// Field 7 of an XGID is the Crawford flag at a match score (eXtreme Gammon 2
// Help, « XGID », part 8: 1 means the current game is Crawford, 0 that it is
// not played with the Crawford rule). The away score carries that rule inside
// the number (CONTEXT.md, « Away score »): a player one point away is `1` in
// the Crawford game and `0` after it. The parser has always written that
// distinction; since #360 it is domain.DecodeXGID that reads it, once, and this
// test is what says the parser did not lose it on the way.
func TestParsePositionWritesTheCrawfordSentinel(t *testing.T) {
	const board = "-B-CBBB---a---A---ABcbbbd-"
	for _, c := range []struct {
		name string
		xgid string
		want [2]int
	}{
		{"after the Crawford game", "XGID=" + board + ":1:-1:1:21:3:6:0:7:10", [2]int{4, domain.PostCrawford}},
		{"in the Crawford game", "XGID=" + board + ":1:-1:1:21:3:6:1:7:10", [2]int{4, domain.Crawford}},
		{"both one point away after Crawford", "XGID=" + board + ":1:-1:1:21:6:6:0:7:10", [2]int{domain.PostCrawford, domain.PostCrawford}},
		// An empty field 7 states nothing about the rule: the ambiguous 1 is
		// what it leaves, as before.
		{"empty field 7", "XGID=" + board + ":1:-1:1:21:3:6::7:10", [2]int{4, domain.Crawford}},
		{"money play is not touched", "XGID=" + board + ":1:-1:1:21:0:0:0:0:10", [2]int{domain.Unlimited, domain.Unlimited}},
	} {
		t.Run(c.name, func(t *testing.T) {
			res, err := ParsePosition(c.xgid)
			if err != nil {
				t.Fatalf("ParsePosition(%q): %v", c.xgid, err)
			}
			if res.Position.Score != c.want {
				t.Errorf("away score = %v, want %v", res.Position.Score, c.want)
			}
		})
	}
}
