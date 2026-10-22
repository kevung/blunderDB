package transcription

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/kevung/gnubgparser"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// recorded is one move of a .mat as the parser reads it: who, what, which dice,
// and the play with its steps in a fixed order.
type recorded struct {
	Player int
	Type   gnubgparser.MoveType
	Dice   [2]int
	Move   string
}

func recordedMoves(t *testing.T, text string) []recorded {
	t.Helper()
	m, err := gnubgparser.ParseMAT(strings.NewReader(text))
	if err != nil {
		t.Fatalf("ParseMAT: %v", err)
	}
	var out []recorded
	for _, g := range m.Games {
		for _, mv := range g.Moves {
			switch mv.Type {
			case gnubgparser.MoveTypeNormal, gnubgparser.MoveTypeDouble, gnubgparser.MoveTypeTake, gnubgparser.MoveTypeDrop:
			default:
				continue
			}
			r := recorded{Player: mv.Player, Type: mv.Type}
			if mv.Type == gnubgparser.MoveTypeNormal {
				d := mv.Dice
				if d[0] > d[1] {
					d[0], d[1] = d[1], d[0]
				}
				r.Dice = d
				var steps []string
				for i := 0; i+1 < len(mv.Move); i += 2 {
					if mv.Move[i] >= 0 {
						steps = append(steps, strconv.Itoa(mv.Move[i])+"/"+strconv.Itoa(mv.Move[i+1]))
					}
				}
				sort.Strings(steps)
				r.Move = strings.Join(steps, " ")
			}
			out = append(out, r)
		}
	}
	return out
}

func TestImportMATOpensADraftThatPlaysTheFile(t *testing.T) {
	ctx := context.Background()
	for _, name := range []string{"charlot1-charlot2_7p_2025-11-08-2305.mat", "gnubg_selfplay_drops_7p.mat"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", name))
			if err != nil {
				t.Fatal(err)
			}
			svc := New(newStore(t), Options{})
			st, err := svc.Import(ctx, "", string(raw))
			if err != nil {
				t.Fatalf("Import: %v", err)
			}
			if st.SessionID == "" || st.Revision == 0 {
				t.Fatalf("the draft is not open: session %q, revision %d", st.SessionID, st.Revision)
			}
			if st.Annotated.Document.Header.MatchID != nil {
				t.Fatal("an imported draft owns no Match")
			}

			// The draft is on disk: a fresh read plays the file's moves and cube.
			got, err := svc.Get(ctx, "", st.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := recordedMoves(t, string(raw))
			var cube, plays int
			for _, a := range got.Annotated.Document.Actions {
				switch a.Kind {
				case transcript.KindDouble, transcript.KindTake, transcript.KindPass:
					cube++
				case transcript.KindChecker, transcript.KindDance:
					plays++
				}
			}
			var wantCube, wantPlays int
			for _, r := range want {
				if r.Type == gnubgparser.MoveTypeNormal {
					wantPlays++
				} else {
					wantCube++
				}
			}
			if cube != wantCube || plays != wantPlays {
				t.Fatalf("draft has %d plays and %d cube actions, the file %d and %d", plays, cube, wantPlays, wantCube)
			}
			if wantCube == 0 {
				t.Fatal("the fixture must carry cube actions")
			}

			text, err := svc.MAT(ctx, "", st.ID)
			if err != nil {
				t.Fatal(err)
			}
			again := recordedMoves(t, text)
			if len(again) != len(want) {
				t.Fatalf("the draft renders %d moves, the file has %d", len(again), len(want))
			}
			for i := range want {
				if again[i] != want[i] {
					t.Fatalf("move %d: draft %+v, file %+v", i, again[i], want[i])
				}
			}
		})
	}
}

func TestImportDocumentKeepsAnIllegalPlayMarked(t *testing.T) {
	ctx := context.Background()
	svc := New(newStore(t), Options{})
	id := int64(42)
	doc := transcript.Document{
		Header: transcript.Header{MatchLength: 5, Player1: "A", Player2: "B", MatchID: &id},
		Actions: []transcript.Action{
			{Side: domain.Black, Kind: transcript.KindChecker, Dice: [2]int{3, 1},
				Steps: []domain.CheckerStep{{From: 8, To: 5}, {From: 6, To: 5}}},
			{Side: domain.Black, Kind: transcript.KindChecker, Dice: [2]int{6, 5}},
		},
	}
	st, err := svc.ImportDocument(ctx, "", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	if got := st.Annotated.Document; len(got.Actions) != 2 || got.Header.MatchID != nil || got.FormatVersion != transcript.FormatVersion {
		t.Fatalf("draft: %d actions, match %v, format %d", len(got.Actions), got.Header.MatchID, got.FormatVersion)
	}
	if !st.Annotated.Inconsistent() {
		t.Fatal("two turns in a row are kept and marked, not refused")
	}
}

func TestImportRefusesWhatNoReplayCanMean(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	svc := New(store, Options{})
	cases := map[string]string{
		"not json":       `{"header":`,
		"later format":   `{"format_version":99,"header":{"match_length":5},"actions":[]}`,
		"negative":       `{"header":{"match_length":-3},"actions":[]}`,
		"unknown kind":   `{"header":{"match_length":5},"actions":[{"side":0,"kind":"teleport"}]}`,
		"bad side":       `{"header":{"match_length":5},"actions":[{"side":4,"kind":"double"}]}`,
		"bad dice":       `{"header":{"match_length":5},"actions":[{"side":0,"kind":"checker","dice":[7,1]}]}`,
		"off the board":  `{"header":{"match_length":5},"actions":[{"side":0,"kind":"checker","dice":[3,1],"steps":[{"from":80,"to":77}]}]}`,
		"bad resign":     `{"header":{"match_length":5},"actions":[{"side":0,"kind":"resign","level":9}]}`,
		"empty mat":      "nothing to see here\n",
		"trailing bytes": `{"header":{"match_length":5},"actions":[]} {}`,
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.Import(ctx, "", text); !errors.Is(err, storage.ErrInvalid) {
				t.Fatalf("Import = %v, want ErrInvalid", err)
			}
		})
	}
	drafts, err := svc.List(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(drafts) != 0 {
		t.Fatalf("%d draft(s) written by refused imports", len(drafts))
	}
}

// A stored draft's own document imports back to the same draft: the .mat it
// renders is the .mat the first draft renders.
func TestImportStoredDocumentRoundTrip(t *testing.T) {
	ctx := context.Background()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "charlot1-charlot2_7p_2025-11-08-2305.mat"))
	if err != nil {
		t.Fatal(err)
	}
	svc := New(newStore(t), Options{})
	first, err := svc.ImportMAT(ctx, "", string(raw))
	if err != nil {
		t.Fatal(err)
	}
	stored, err := svc.Get(ctx, "", first.ID)
	if err != nil {
		t.Fatal(err)
	}
	blob, err := DurableJSON(stored.Annotated.Document)
	if err != nil {
		t.Fatal(err)
	}
	// With a byte-order mark, as an editor may save it: still a document.
	second, err := svc.Import(ctx, "", "\uFEFF"+string(blob))
	if err != nil {
		t.Fatalf("re-import of the stored document: %v", err)
	}
	want, err := svc.MAT(ctx, "", first.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.MAT(ctx, "", second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("the re-imported draft renders another .mat:\n%s\n---\n%s", got, want)
	}
}

func TestImportMATWithByteOrderMark(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "gnubg_selfplay_drops_7p.mat"))
	if err != nil {
		t.Fatal(err)
	}
	svc := New(newStore(t), Options{})
	if _, err := svc.Import(context.Background(), "", "\uFEFF"+string(raw)); err != nil {
		t.Fatalf("a .mat with a byte-order mark: %v", err)
	}
}
