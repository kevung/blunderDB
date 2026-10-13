package transcription

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

func tournamentNames(t *testing.T, store storage.Storage, scope string) map[string]int64 {
	t.Helper()
	out := map[string]int64{}
	for tr, err := range store.Tournaments().List(context.Background(), scope, storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("List tournaments: %v", err)
		}
		out[tr.Name] = tr.ID
	}
	return out
}

func finishOne(t *testing.T, svc *Service, scope string, h transcript.Header) int64 {
	t.Helper()
	ctx := context.Background()
	st, err := svc.Create(ctx, scope, h)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	st = play(t, svc, scope, st, checkerAction(6, 3))
	res, err := svc.Finish(ctx, scope, st.ID, Expect{Session: st.SessionID, Revision: st.Revision})
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	return res.MatchID
}

func tournamentOf(t *testing.T, store storage.Storage, scope string, matchID int64) int64 {
	t.Helper()
	m, err := store.Matches().Get(context.Background(), scope, matchID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if m.TournamentID == nil {
		return 0
	}
	return *m.TournamentID
}

// A finished transcription with an event and no tournament chosen is filed
// under its event's tournament, as an import of the same match would be:
// never left outside the tournament the import step just created for it.
func TestFinishWithoutTournamentFilesUnderItsEvent(t *testing.T) {
	const scope = "1"
	store := newStore(t)
	svc := New(store, Options{})

	id := finishOne(t, svc, scope, transcript.Header{MatchLength: 9, Player1: "A", Player2: "B", Event: "Open"})

	names := tournamentNames(t, store, scope)
	tid, ok := names["Open"]
	if !ok {
		t.Fatalf("no tournament named after the event; have %v", names)
	}
	if got := tournamentOf(t, store, scope, id); got != tid {
		t.Fatalf("match in tournament %d, want %d (Open)", got, tid)
	}
}

// A tournament chosen in the header wins over the event, and the event does
// not leave an empty tournament of its own behind.
func TestFinishWithChosenTournamentCreatesNoEventTournament(t *testing.T) {
	const scope = "1"
	ctx := context.Background()
	store := newStore(t)
	svc := New(store, Options{})
	chosen, err := store.Tournaments().Create(ctx, scope, "Open", "", "")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}

	id := finishOne(t, svc, scope, transcript.Header{MatchLength: 9, Player1: "A", Player2: "B", Event: "Open 2026", TournamentID: &chosen})

	if got := tournamentOf(t, store, scope, id); got != chosen {
		t.Fatalf("match in tournament %d, want the chosen %d", got, chosen)
	}
	if names := tournamentNames(t, store, scope); len(names) != 1 {
		t.Fatalf("tournaments %v, want only the chosen one", names)
	}
}

// Materialize writes a new Match as Finish does.
func TestMaterializeFilesLikeFinish(t *testing.T) {
	const scope = "1"
	ctx := context.Background()
	store := newStore(t)
	svc := New(store, Options{})

	st, err := svc.Create(ctx, scope, transcript.Header{MatchLength: 9})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	st = play(t, svc, scope, st, checkerAction(6, 3))
	h := transcript.Header{MatchLength: 9, Player1: "A", Player2: "B", Event: "Open"}
	res, err := svc.Materialize(ctx, scope, h, st.Annotated.Document.Actions)
	if err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	names := tournamentNames(t, store, scope)
	if got := tournamentOf(t, store, scope, res.MatchID); got == 0 || got != names["Open"] {
		t.Fatalf("match in tournament %d, want Open (%v)", got, names)
	}
}
