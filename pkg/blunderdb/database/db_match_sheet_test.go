package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/arkqr"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/printsheet"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// tableCounts is the size of the tables printing must leave alone.
func tableCounts(t *testing.T, db *Database) string {
	t.Helper()
	var out []string
	for _, tb := range []string{"position", "analysis", "comment", "metadata", "match", "move"} {
		var n int
		if err := db.db.QueryRow(`SELECT COUNT(*) FROM ` + tb).Scan(&n); err != nil {
			t.Fatal(err)
		}
		out = append(out, fmt.Sprintf("%s=%d", tb, n))
	}
	return strings.Join(out, " ")
}

// TestMatchSheetOracle prints the sheet of an analysed match and checks it
// against the library: the grades are the library's, every listed position
// is numbered once in the transcription at the line its entry names, the
// opponent's blunders are listed only when asked, the ArkIDs read back as the
// stored positions, and the embedded code is the match's archive.
func TestMatchSheetOracle(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	matchID, err := db.ImportXGMatch(filepath.Join("testdata", "match_with_comment.xg"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	// A position player 1 played, marked and commented: a remarkable entry.
	var remarkPos int64
	seen := 0
	for mv, err := range db.store.Matches().MovesByMatch(ctx, "", matchID) {
		if err != nil {
			t.Fatal(err)
		}
		// Below the error threshold, so that it is listed as remarkable alone.
		if mv.Player == domain.WinnerPlayer1 && mv.MoveType == "checker" && mv.ErrorMP != nil && *mv.ErrorMP == 0 {
			if seen++; seen == 4 {
				remarkPos = mv.PositionID
			}
		}
	}
	if err := db.AddComment(remarkPos, "à revoir sur papier"); err != nil {
		t.Fatal(err)
	}
	before := tableCounts(t, db)

	grades, err := db.store.Stats().MatchMoveGrades(ctx, "", matchID)
	if err != nil {
		t.Fatal(err)
	}
	var want [2][2]int // [player][error, blunder]
	byMove := map[int64]string{}
	for _, g := range grades {
		byMove[g.MoveID] = g.Grade
	}
	players := map[int64]int{}
	positionOf := map[int64]int64{}
	for mv, err := range db.store.Matches().MovesByMatch(ctx, "", matchID) {
		if err != nil {
			t.Fatal(err)
		}
		sd := 0
		if mv.Player == domain.WinnerPlayer2 {
			sd = 1
		}
		players[mv.ID], positionOf[mv.ID] = sd, mv.PositionID
		switch byMove[mv.ID] {
		case storage.MoveGradeBlunder:
			want[sd][1]++
			want[sd][0]++
		case storage.MoveGradeError:
			want[sd][0]++
		}
	}
	if want[0][1]+want[1][1] == 0 {
		t.Fatal("the test match has no blunder: pick another one")
	}

	for _, opp := range []bool{false, true} {
		t.Run(fmt.Sprintf("opponent=%v", opp), func(t *testing.T) {
			o := printsheet.DefaultMatchOptions()
			o.OpponentBlunders = opp
			ms, err := printsheet.ReadMatch(ctx, db.store, "", matchID, o)
			if err != nil {
				t.Fatal(err)
			}
			for sd := range 2 {
				if got := [2]int{ms.Stats[sd].Errors, ms.Stats[sd].Blunders}; got != want[sd] {
					t.Errorf("player %d: errors/blunders %v, the library grades %v", sd+1, got, want[sd])
				}
			}
			// Every ref of the transcription, where it stands.
			type at struct{ game, row int }
			refs := map[int]at{}
			blunderParts := [2]int{}
			for _, g := range ms.Games {
				for ri, row := range g.Rows {
					for sd, cell := range row {
						for _, p := range cell.Parts {
							if p.Grade == printsheet.GradeBlunder {
								blunderParts[sd]++
								if (sd == ms.Self || opp) && p.Ref == 0 {
									t.Errorf("game %d line %d: a blunder of player %d is not numbered", g.Number, ri+1, sd+1)
								}
								if sd != ms.Self && !opp && p.Ref != 0 {
									t.Errorf("game %d line %d: the opponent's blunder is numbered %d", g.Number, ri+1, p.Ref)
								}
							}
							if p.Ref == 0 {
								continue
							}
							if _, dup := refs[p.Ref]; dup {
								t.Errorf("number %d appears twice in the transcription", p.Ref)
							}
							refs[p.Ref] = at{g.Number, ri + 1}
						}
					}
				}
			}
			if blunderParts != [2]int{want[0][1], want[1][1]} {
				t.Errorf("?? marks %v, the library grades %v blunders", blunderParts, [2]int{want[0][1], want[1][1]})
			}
			if len(refs) != len(ms.Entries) {
				t.Fatalf("%d numbers in the transcription, %d entries", len(refs), len(ms.Entries))
			}
			sawRemark := false
			for i, e := range ms.Entries {
				if e.Ref != i+1 {
					t.Errorf("entry %d is numbered %d", i+1, e.Ref)
				}
				if got := refs[e.Ref]; got != (at{e.Game, e.Row}) {
					t.Errorf("entry %d says game %d line %d, the transcription has it at %v", e.Ref, e.Game, e.Row, got)
				}
				if players[e.MoveID] != e.Player || positionOf[e.MoveID] != e.PositionID {
					t.Errorf("entry %d is not its move's", e.Ref)
				}
				if e.Kind == printsheet.KindBlunder && byMove[e.MoveID] != storage.MoveGradeBlunder {
					t.Errorf("entry %d is listed as a blunder the library does not grade one", e.Ref)
				}
				if e.PositionID == remarkPos {
					sawRemark = true
					if e.Kind != printsheet.KindRemark {
						t.Errorf("the commented position is listed as %s", e.Kind)
					}
				}
				if e.Kind == printsheet.KindRemark {
					if len(e.Comments) == 0 {
						t.Errorf("remark %d carries no comment", e.Ref)
					}
				}
				// The ArkID reads back as the stored position.
				stored, err := db.store.Positions().LoadByIDs(ctx, "", []int64{e.PositionID})
				if err != nil || len(stored) != 1 {
					t.Fatalf("position %d: %v", e.PositionID, err)
				}
				want, _ := domain.EncodeArkID(&stored[0])
				dec, err := domain.DecodeArkID(e.ArkID)
				if err != nil {
					t.Fatalf("entry %d: ArkID %q does not read back: %v", e.Ref, e.ArkID, err)
				}
				if got, _ := domain.EncodeArkID(&dec); got != want || e.ArkID != want {
					t.Errorf("entry %d: ArkID %q reads back as %q, the position is %q", e.Ref, e.ArkID, got, want)
				}
			}
			if !sawRemark {
				t.Error("the commented position is not listed")
			}

			page, err := db.MatchSheetHTML([]int64{matchID}, o)
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range ms.Entries {
				if !strings.Contains(page, `<span class="nt">`+e.ArkID+`</span>`) {
					t.Errorf("entry %d: ArkID missing from the page", e.Ref)
				}
			}
			frames, err := arkqr.ReadPage([]byte(page))
			if err != nil {
				t.Fatal(err)
			}
			if got := arkqr.Assemble(frames).Matches; len(got) != 1 {
				t.Fatalf("the page embeds %d matches", len(got))
			}
			if strings.Count(page, `<section class="doc"`) != 1 || !strings.Contains(page, "<svg") {
				t.Error("one sheet with its code expected")
			}
		})
	}

	// A batch: one sheet a match, each code embedded.
	page, err := db.MatchSheetHTML([]int64{matchID, matchID}, printsheet.DefaultMatchOptions())
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(page, `<section class="doc"`); n != 2 {
		t.Errorf("a batch of two gives %d sheets", n)
	}
	// The saved sheet imports as an archive page does; the two sheets of one
	// match carry one archive.
	saved := filepath.Join(t.TempDir(), "feuille.html")
	if err := os.WriteFile(saved, []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := newTestDB(t)
	res, err := dst.ImportPaperArchive([]string{saved})
	if err != nil {
		t.Fatal(err)
	}
	if res.Codes != 2 || len(res.Imported) != 1 || len(res.Failed) > 0 || len(res.Missing) > 0 {
		t.Errorf("reimporting the sheet: %+v", res)
	}
	if after := tableCounts(t, db); after != before {
		t.Errorf("printing wrote to the library: %s, then %s", before, after)
	}
	if _, err := db.MatchSheetHTML([]int64{matchID + 999}, printsheet.DefaultMatchOptions()); err == nil {
		t.Error("an unknown match prints")
	}
}

// TestMatchSheetExpanded writes the developed ArkID, which reads back as the
// same position.
func TestMatchSheetExpanded(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	matchID, err := db.ImportXGMatch(filepath.Join("testdata", "match_with_comment.xg"))
	if err != nil {
		t.Fatal(err)
	}
	o := printsheet.DefaultMatchOptions()
	o.Expanded, o.OpponentBlunders = true, true
	ms, err := printsheet.ReadMatch(context.Background(), db.store, "", matchID, o)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms.Entries) == 0 {
		t.Fatal("no entry")
	}
	for _, e := range ms.Entries {
		p, err := domain.DecodeArkID(e.ArkID)
		if err != nil {
			t.Fatalf("%q: %v", e.ArkID, err)
		}
		if again, _ := domain.EncodeArkIDExpanded(&p); again != e.ArkID {
			t.Errorf("%q reads back as %q", e.ArkID, again)
		}
	}
}
