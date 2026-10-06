package transcription

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// typedGame types a short game gesture by gesture into a draft and returns
// the draft and its Actions: legal dice at random, a double and its take in
// the middle, a dance when a roll allows no play.
func typedGame(t *testing.T, svc *Service, scope string, header transcript.Header, plays int) (*State, []transcript.Action) {
	t.Helper()
	ctx := context.Background()
	st, err := svc.Create(ctx, scope, header)
	if err != nil {
		t.Fatal(err)
	}
	apply := func(g transcript.Gesture) error {
		next, err := svc.Apply(ctx, scope, st.ID, Expect{Session: st.SessionID, Revision: st.Revision}, g)
		if err == nil {
			st = next
		}
		return err
	}
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < plays; i++ {
		if i == plays/2 {
			for _, k := range []transcript.GestureKind{transcript.GestureDouble, transcript.GestureTake} {
				if err := apply(transcript.Gesture{Kind: k}); err != nil {
					t.Fatalf("cube %s: %v", k, err)
				}
			}
		}
		for _, d := range []int{rng.Intn(6) + 1, rng.Intn(6) + 1} {
			if err := apply(transcript.Gesture{Kind: transcript.GestureEnterDie, Die: d}); err != nil {
				t.Fatal(err)
			}
		}
		if err := apply(transcript.Gesture{Kind: transcript.GestureSelectCandidate, Candidate: 0}); err != nil {
			if err := apply(transcript.Gesture{Kind: transcript.GestureDance}); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := apply(transcript.Gesture{Kind: transcript.GestureValidate}); err != nil {
			t.Fatal(err)
		}
	}
	if st.Annotated.Inconsistent() {
		t.Fatal("the typed game is inconsistent")
	}
	return st, st.Annotated.Document.Actions
}

func matchHeader(t *testing.T, svc *Service, scope string, id int64) *domain.Match {
	t.Helper()
	m, err := svc.store.Matches().Get(context.Background(), scope, id)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMaterializeEqualsCreateApplyFinish(t *testing.T) {
	ctx := context.Background()
	svc := New(newStore(t), Options{})
	header := transcript.Header{
		MatchLength: 7, Player1: "Alice", Player2: "Bob", Event: "Open", Round: "1",
		Date: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Transcriber: "T",
	}
	st, actions := typedGame(t, svc, "1", header, 40)
	if len(actions) < 40 {
		t.Fatalf("only %d actions typed", len(actions))
	}
	fin, err := svc.Finish(ctx, "1", st.ID, Expect{Session: st.SessionID})
	if err != nil {
		t.Fatal(err)
	}
	// Another library: the canonical hash is unique in one.
	other := New(newStore(t), Options{})
	mat, err := other.Materialize(ctx, "1", header, actions)
	if err != nil {
		t.Fatal(err)
	}
	if mat.Games != fin.Games || mat.Moves != fin.Moves || mat.Positions != fin.Positions || mat.ToAnalyze != fin.ToAnalyze || mat.Inconsistent {
		t.Fatalf("outline differs: materialize %+v, finish %+v", mat, fin)
	}
	a, b := matchHeader(t, svc, "1", fin.MatchID), matchHeader(t, other, "1", mat.MatchID)
	if a.MatchHash == "" || a.CanonicalHash == "" {
		t.Fatal("the finished match carries no hash")
	}
	if a.MatchHash != b.MatchHash || a.CanonicalHash != b.CanonicalHash {
		t.Fatalf("hashes differ: %s/%s vs %s/%s", a.MatchHash, a.CanonicalHash, b.MatchHash, b.CanonicalHash)
	}
	if a.Player1Name != b.Player1Name || a.MatchLength != b.MatchLength || a.GameCount != b.GameCount || a.Transcriber != b.Transcriber {
		t.Fatalf("headers differ: %+v vs %+v", a, b)
	}
	if rows, err := other.List(ctx, "1"); err != nil || len(rows) != 0 {
		t.Fatalf("materialising leaves no draft: %v %v", rows, err)
	}
}

func countMatches(t *testing.T, svc *Service, scope string) int {
	t.Helper()
	n, err := svc.store.Matches().Count(context.Background(), scope, storage.MatchListOpts{})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestMaterializeRefusalWritesNothing(t *testing.T) {
	ctx := context.Background()
	svc := New(newStore(t), Options{})
	header := transcript.Header{MatchLength: 5, Player1: "A", Player2: "B", Date: time.Now(), Transcriber: "T"}
	_, actions := typedGame(t, svc, "1", header, 12)
	before := countMatches(t, svc, "1")
	drafts, _ := svc.List(ctx, "1")

	// Rank k: the same side plays twice in a row.
	k := 7
	bad := append([]transcript.Action(nil), actions...)
	bad[k] = bad[k-1]
	_, err := svc.Materialize(ctx, "1", header, bad)
	var ref *RefusedAction
	if !errors.As(err, &ref) {
		t.Fatalf("want a RefusedAction, got %v", err)
	}
	if ref.Rank != k || ref.Refusal == nil || ref.Refusal.Detail == "" {
		t.Fatalf("refusal = %+v", ref)
	}
	if !errors.Is(err, storage.ErrInvalid) {
		t.Fatal("a refusal is an ErrInvalid")
	}
	if got := countMatches(t, svc, "1"); got != before {
		t.Fatalf("a refused document wrote %d match(es)", got-before)
	}
	if after, _ := svc.List(ctx, "1"); len(after) != len(drafts) {
		t.Fatal("a refused document left a draft")
	}

	// A header the machine refuses is rank -1.
	h := header
	h.Beaver = true
	if _, err := svc.Materialize(ctx, "1", h, actions); !errors.As(err, &ref) || ref.Rank != -1 {
		t.Fatalf("beaver header: %v", err)
	}
	if _, err := svc.Materialize(ctx, "1", header, nil); !errors.Is(err, storage.ErrInvalid) {
		t.Fatalf("empty document: %v", err)
	}
}

func TestMaterializeCarriesDecisionTimes(t *testing.T) {
	ctx := context.Background()
	svc := New(newStore(t), Options{})
	header := transcript.Header{MatchLength: 3, Player1: "A", Player2: "B", Date: time.Now(), Transcriber: "T"}
	_, actions := typedGame(t, svc, "1", header, 6)
	ms := int64(1500)
	actions[1].DecisionMS = &ms

	res, err := svc.Materialize(ctx, "1", header, actions)
	if err != nil {
		t.Fatal(err)
	}
	var got []*int64
	for mv, err := range svc.store.Matches().MovesByMatch(ctx, "1", res.MatchID) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, mv.DecisionMS)
	}
	if len(got) < 2 || got[1] == nil || *got[1] != ms {
		t.Fatalf("the second move should carry %d ms: %v", ms, got)
	}
	if got[0] != nil || got[2] != nil {
		t.Fatal("a move without a duration reads unknown, never zero")
	}
}
