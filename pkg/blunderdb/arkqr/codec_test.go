package arkqr

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// corpusMatch is one match of the repository's testdata, as a document.
type corpusMatch struct {
	name string
	doc  transcript.Document
}

var corpusCache []corpusMatch

func loadCorpus(t *testing.T) []corpusMatch {
	t.Helper()
	if corpusCache == nil {
		corpusCache = readCorpus(t)
	}
	out := make([]corpusMatch, len(corpusCache))
	copy(out, corpusCache)
	return out
}

func readCorpus(t *testing.T) []corpusMatch {
	t.Helper()
	root := "../../../testdata"
	var out []corpusMatch
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		path := filepath.Join(root, e.Name())
		if strings.ToLower(filepath.Ext(e.Name())) != ".mat" {
			continue
		}
		text, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := transcript.FromMAT(string(text))
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		out = append(out, corpusMatch{e.Name(), doc})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	if len(out) < 4 {
		t.Fatalf("corpus: %d matches", len(out))
	}
	return out
}

// normalized is the document as the archive restores it: the steps of a
// legal play are the ones the codec writes for the board it leaves.
func normalized(doc transcript.Document) transcript.Document {
	ann := transcript.Replay(doc, 0)
	out := doc
	out.Actions = make([]transcript.Action, len(doc.Actions))
	for i, a := range doc.Actions {
		if std, _, ps, idx := standard(&a, &ann.Actions[i]); std && a.Kind == transcript.KindChecker {
			a.Steps = domainSteps(ps[idx].steps, a.Side)
		}
		if len(a.Steps) == 0 {
			a.Steps = nil
		}
		out.Actions[i] = a
	}
	if len(out.Actions) == 0 {
		out.Actions = nil
	}
	out.Cursor = len(out.Actions)
	if !out.Header.Date.IsZero() {
		out.Header.Date = out.Header.Date.UTC()
	}
	out.Header.TournamentID, out.Header.MatchID = nil, nil
	out.Entry = nil
	return out
}

func roundTrip(t *testing.T, name string, m Match) []byte {
	t.Helper()
	data, err := Encode(m)
	if err != nil {
		t.Fatalf("%s: encode: %v", name, err)
	}
	back, err := Decode(data)
	if err != nil {
		t.Fatalf("%s: decode: %v", name, err)
	}
	want := normalized(m.Doc)
	if !reflect.DeepEqual(back.Doc, want) {
		for i := range min(len(back.Doc.Actions), len(want.Actions)) {
			if !reflect.DeepEqual(back.Doc.Actions[i], want.Actions[i]) {
				t.Fatalf("%s: action %d differs:\n got %+v\nwant %+v", name, i, back.Doc.Actions[i], want.Actions[i])
			}
		}
		t.Fatalf("%s: documents differ (%d vs %d actions)\n got %+v\nwant %+v", name, len(back.Doc.Actions), len(want.Actions), back.Doc.Header, want.Header)
	}
	if back.Comment != m.Comment || !reflect.DeepEqual(back.Notes, m.Notes) || !reflect.DeepEqual(back.Elo, m.Elo) ||
		!reflect.DeepEqual(back.Experience, m.Experience) || back.EngineVersion != m.EngineVersion {
		t.Fatalf("%s: notes or comment differ", name)
	}
	// The boards every Action is played from are the original ones.
	a1, a2 := transcript.Replay(m.Doc, 0), transcript.Replay(back.Doc, 0)
	for i := range a1.Actions {
		if !domain.SameBoard(a1.Actions[i].Before.Board, a2.Actions[i].Before.Board) || !domain.SameBoard(a1.Actions[i].After, a2.Actions[i].After) {
			t.Fatalf("%s: action %d: boards differ", name, i)
		}
	}
	again, err := Encode(back)
	if err != nil || !bytes.Equal(again, data) {
		t.Fatalf("%s: re-encoding the decoded match changes it (%v)", name, err)
	}
	return data
}

func TestRoundTripCorpus(t *testing.T) {
	for _, cm := range loadCorpus(t) {
		data := roundTrip(t, cm.name, Match{Doc: cm.doc})
		t.Logf("%s: %d actions, %d bytes", cm.name, len(cm.doc.Actions), len(data))
	}
}

// TestRoundTripDegenerate covers what the models do not predict: rolls
// whose play is unknown, impossible dice, a board entered by hand, a
// declared score, durations and timecodes, notes, and a forfeit.
func TestRoundTripDegenerate(t *testing.T) {
	doc := loadCorpus(t)[0].doc
	acts := append([]transcript.Action(nil), doc.Actions...)
	ms := func(v int64) *int64 { return &v }
	acts[3].Kind, acts[3].Steps = transcript.KindUnrecorded, nil
	acts[7].Dice = [2]int{0, 7}
	board := transcript.InitialBoard()
	acts[9].BoardAfter = &board
	acts[9].BoardBefore = &board
	acts[11].Score = &[2]int{2, 1}
	acts[12].DecisionMS, acts[12].CubeDecisionMS = ms(4200), ms(1300)
	acts[13].RollTickMS, acts[13].TickMS = ms(61_000), ms(-5)
	acts[14].Steps = acts[14].Steps[:1]
	acts[15].Side = 1 - acts[15].Side
	acts = append(acts, transcript.Action{Side: 1, Kind: transcript.KindForfeit})
	doc.Actions = acts
	doc.NextScore = &[2]int{3, 4}
	doc.Header.Jacoby, doc.Header.Beaver, doc.Header.MaxCube = true, true, 6
	doc.Header.Event, doc.Header.Round, doc.Header.Location = "Open d'automne", "Quart de finale", "Lyon"
	doc.Header.VideoSource = "https://example.org/v?t=1"

	notes := make([]Note, len(acts))
	notes[0] = Note{Comments: []string{"première note", "#ouverture"}}
	notes[20] = Note{Flagged: true}
	roundTrip(t, "degenerate", Match{Doc: doc, Comment: "Commentaire du match", Notes: notes})

	empty := transcript.New(5)
	roundTrip(t, "empty", Match{Doc: empty})
}

// TestDecodeDamaged feeds the decoder truncated and altered streams: it
// refuses or decodes something, and never panics or loops.
func TestDecodeDamaged(t *testing.T) {
	data, err := Encode(Match{Doc: loadCorpus(t)[0].doc})
	if err != nil {
		t.Fatal(err)
	}
	for cut := 0; cut < len(data); cut += 7 {
		_, _ = Decode(data[:cut])
	}
	for i := 1; i < len(data); i += 5 {
		bad := append([]byte(nil), data...)
		bad[i] ^= 0x5a
		_, _ = Decode(bad)
	}
	if _, err := Decode([]byte{FormatVersion + 1, 1, 2}); err == nil {
		t.Fatal("a later format version decoded")
	}
}

// TestArchiveSize fixes the size of the corpus once coded: a change to the
// models that makes it grow is a regression of the compression, and one
// that changes it at all is a new format version.
func TestArchiveSize(t *testing.T) {
	want := map[string]int{
		"test.mat": 364,
		"charlot1-charlot2_7p_2025-11-08-2305.mat": 202,
	}
	total := 0
	for _, cm := range loadCorpus(t) {
		data, err := Encode(Match{Doc: cm.doc})
		if err != nil {
			t.Fatal(err)
		}
		total += len(data)
		if w, ok := want[cm.name]; ok && w != len(data) {
			t.Errorf("%s: %d bytes, want %d", cm.name, len(data), w)
		}
	}
	if total != wantCorpusBytes {
		t.Errorf("corpus: %d bytes, want %d", total, wantCorpusBytes)
	}
}

// TestLegalPlaysMatchDomain checks the codec's frozen generator against the
// domain's on every position of the corpus: the same set of boards.
func TestLegalPlaysMatchDomain(t *testing.T) {
	n := 0
	for _, cm := range loadCorpus(t) {
		ann := transcript.Replay(cm.doc, 0)
		for i, a := range cm.doc.Actions {
			if a.Kind != transcript.KindChecker || !validDice(a.Dice) || !ann.Actions[i].HasPosition {
				continue
			}
			pos := ann.Actions[i].Before
			pos.Dice = [2]int{a.Dice[0], a.Dice[1]}
			pos.PlayerOnRoll = a.Side
			ours := relOf(pos.Board, a.Side).plays(a.Dice[0], a.Dice[1])
			theirs := domain.LegalMoves(&pos)
			set := map[rel]bool{}
			for _, p := range theirs {
				set[relOf(p.Result.Board, a.Side)] = true
			}
			if len(set) != len(ours) {
				t.Fatalf("%s action %d: %d plays, domain %d", cm.name, i, len(ours), len(set))
			}
			for _, p := range ours {
				if !set[p.after] {
					t.Fatalf("%s action %d: play %v unknown to the domain", cm.name, i, p.steps)
				}
			}
			n++
		}
	}
	if n < 500 {
		t.Fatalf("only %d positions compared", n)
	}
}

const wantCorpusBytes = 799

func TestEncodeRefusesWhatDecodeRefuses(t *testing.T) {
	doc := loadCorpus(t)[0].doc
	long := strings.Repeat("x", maxLen+1)
	for name, m := range map[string]Match{
		"comment":  {Doc: doc, Comment: long},
		"player":   {Doc: func() transcript.Document { d := doc; d.Header.Player1 = long; return d }()},
		"note":     {Doc: doc, Notes: append([]Note{{Comments: []string{long}}}, make([]Note, len(doc.Actions)-1)...)},
		"comments": {Doc: doc, Notes: append([]Note{{Comments: make([]string, maxLen+1)}}, make([]Note, len(doc.Actions)-1)...)},
	} {
		if _, err := Encode(m); err == nil {
			t.Errorf("%s: encoded what Decode refuses", name)
		}
	}
	at := strings.Repeat("y", maxLen)
	roundTrip(t, "longest", Match{Doc: doc, Comment: at})
}

func TestSourceMetadataRoundTrip(t *testing.T) {
	elo, exp := 1834.25, 412
	roundTrip(t, "source", Match{Doc: loadCorpus(t)[0].doc, Elo: [2]*float64{&elo, nil}, Experience: [2]*int{nil, &exp}, EngineVersion: "eXtreme Gammon 2.19"})
}
