package server

import (
	"bufio"
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"slices"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func (p recordingPositions) ExistsMany(ctx context.Context, scope string, zobrists []uint64) (map[uint64]int64, error) {
	p.log.read(scope)
	return p.PositionStore.ExistsMany(ctx, scope, zobrists)
}

func (r recordingStorage) Collections() storage.CollectionStore {
	return recordingCollections{r.Storage.Collections(), r.log}
}

type recordingCollections struct {
	storage.CollectionStore
	log *scopeLog
}

func (c recordingCollections) List(ctx context.Context, scope string) iter.Seq2[*storage.Collection, error] {
	c.log.read(scope)
	return c.CollectionStore.List(ctx, scope)
}

func (c recordingCollections) Positions(ctx context.Context, scope string, id int64, o storage.ListOpts) iterPositions {
	c.log.read(scope)
	return c.CollectionStore.Positions(ctx, scope, id, o)
}

// clubCalls is one call per club route that spans the whole read set.
var clubCalls = []struct{ path, body string }{
	{"/v1/across.commentsByZobrist", `{"zobrists":[1]}`},
	{"/v1/across.collectionsList", `{}`},
	{"/v1/across.clubRanking", `{}`},
}

func TestAcrossClub_ReadsTheReadSetInOrder(t *testing.T) {
	for _, c := range clubCalls {
		t.Run(c.path, func(t *testing.T) {
			srv, log := newRecordingServer(t, false)
			rec := serveAcross(t, srv, "1", "3,2", c.path, c.body)
			if rec.Code != http.StatusOK {
				t.Fatalf("status %d: %s", rec.Code, rec.Body)
			}
			reads, writes := log.snapshot()
			if !slices.Equal(reads, []string{"1", "3", "2"}) {
				t.Errorf("store read %v, want [1 3 2]", reads)
			}
			if len(writes) != 0 {
				t.Errorf("a club read wrote under %v", writes)
			}
		})
	}
}

func TestAcrossClub_RefusedWithoutReadTenants(t *testing.T) {
	for _, c := range clubCalls {
		srv, log := newRecordingServerWith(t, Options{})
		if rec := serveAcross(t, srv, "1", "2", c.path, c.body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s with the header untrusted: status %d, want 400", c.path, rec.Code)
		}
		if reads, _ := log.snapshot(); len(reads) != 0 {
			t.Errorf("%s: the store was read %v", c.path, reads)
		}
	}
}

// TestAcrossClub_CoachCommentJoinsByZobrist: the coach saves the student's
// board in the coach's own tenant and comments it there; the comment comes
// back on the hash the student's position carries.
func TestAcrossClub_CoachCommentJoinsByZobrist(t *testing.T) {
	srv, _ := newRecordingServer(t, false)
	save := serveAcross(t, srv, "1", "", "/v1/positions.save", `{"position":`+initialPositionJSON(t)+`}`)
	var id idResp
	if err := json.Unmarshal(save.Body.Bytes(), &id); err != nil || id.ID == 0 {
		t.Fatalf("positions.save: %d %s", save.Code, save.Body)
	}
	if rec := serveAcross(t, srv, "1", "", "/v1/comments.add", `{"positionId":`+jsonInt(id.ID)+`,"text":"coach: hit"}`); rec.Code != http.StatusOK {
		t.Fatalf("comments.add: %d %s", rec.Code, rec.Body)
	}
	found := serveAcross(t, srv, "1", "", "/v1/across.searchFind", `{}`)
	var pos acrossPosition
	if err := json.Unmarshal(found.Body.Bytes(), &pos); err != nil || pos.Zobrist == 0 {
		t.Fatalf("across.searchFind = %s (%v)", found.Body, err)
	}

	rec := serveAcross(t, srv, "1", "", "/v1/across.commentsByZobrist", `{"zobrists":[`+strconv.FormatUint(pos.Zobrist, 10)+`]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("across.commentsByZobrist: %d %s", rec.Code, rec.Body)
	}
	var got []acrossComment
	sc := bufio.NewScanner(rec.Body)
	for sc.Scan() {
		var c acrossComment
		if err := json.Unmarshal(sc.Bytes(), &c); err != nil {
			t.Fatalf("line %q: %v", sc.Text(), err)
		}
		got = append(got, c)
	}
	if len(got) != 1 || got[0].Tenant != "1" || got[0].Zobrist != pos.Zobrist || got[0].PositionID != id.ID ||
		got[0].Comment == nil || got[0].Comment.Text != "coach: hit" || got[0].Comment.Origin != "user" {
		t.Errorf("across.commentsByZobrist = %+v, want the coach's comment on tenant 1", got)
	}

	tooMany := make([]uint64, storage.MaxZobristLookups+1)
	body, _ := json.Marshal(acrossZobristReq{Zobrists: tooMany})
	if rec := serveAcross(t, srv, "1", "", "/v1/across.commentsByZobrist", string(body)); rec.Code != http.StatusBadRequest {
		t.Errorf("more than MaxZobristLookups hashes: status %d, want 400", rec.Code)
	}
}

func TestAcrossClub_CollectionPositionsNameTheirTenant(t *testing.T) {
	srv, log := newRecordingServer(t, false)
	save := serveAcross(t, srv, "1", "", "/v1/positions.save", `{"position":`+initialPositionJSON(t)+`}`)
	var pid, cid idResp
	if err := json.Unmarshal(save.Body.Bytes(), &pid); err != nil || pid.ID == 0 {
		t.Fatalf("positions.save: %d %s", save.Code, save.Body)
	}
	create := serveAcross(t, srv, "1", "", "/v1/collections.create", `{"name":"library"}`)
	if err := json.Unmarshal(create.Body.Bytes(), &cid); err != nil || cid.ID == 0 {
		t.Fatalf("collections.create: %d %s", create.Code, create.Body)
	}
	if rec := serveAcross(t, srv, "1", "", "/v1/collections.addPosition", `{"collectionId":`+jsonInt(cid.ID)+`,"positionId":`+jsonInt(pid.ID)+`}`); rec.Code != http.StatusOK {
		t.Fatalf("collections.addPosition: %d %s", rec.Code, rec.Body)
	}
	before, _ := log.snapshot()

	// SQLite ignores the scope, so tenant 2 answers tenant 1's collection:
	// what counts is that the line names tenant 2 and carries the hash.
	rec := serveAcross(t, srv, "1", "2", "/v1/across.collectionPositions", `{"tenant":"2","collectionId":`+jsonInt(cid.ID)+`}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("across.collectionPositions: %d %s", rec.Code, rec.Body)
	}
	var got acrossPosition
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Tenant != "2" || got.Zobrist == 0 || got.Position == nil || got.Position.ID != pid.ID {
		t.Errorf("across.collectionPositions = %s (%v), want position %d tagged 2 with its hash", rec.Body, err, pid.ID)
	}
	if reads, _ := log.snapshot(); !slices.Equal(reads[len(before):], []string{"2"}) {
		t.Errorf("store read %v, want [2]", reads[len(before):])
	}
	if rec := serveAcross(t, srv, "1", "2", "/v1/across.collectionPositions", `{"tenant":"2","collectionId":1,"limit":`+strconv.Itoa(maxPageSize+1)+`}`); rec.Code != http.StatusBadRequest {
		t.Errorf("limit over maxPageSize: status %d, want 400", rec.Code)
	}
}

// TestAcrossClub_UnlistedTenantNeverReachesTheStore: on SQLite, with no
// PostgreSQL, a route naming a tenant outside the read set answers 400 before
// any store call.
func TestAcrossClub_UnlistedTenantNeverReachesTheStore(t *testing.T) {
	for _, c := range []struct{ path, body string }{
		{"/v1/across.collectionPositions", `{"tenant":"9","collectionId":1}`},
		{"/v1/across.matchMovePositions", `{"tenant":"9","matchId":1,"limit":5}`},
	} {
		srv, log := newRecordingServer(t, false)
		if rec := serveAcross(t, srv, "1", "2", c.path, c.body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s, unlisted tenant: status %d, want 400", c.path, rec.Code)
		}
		if reads, writes := log.snapshot(); len(reads)+len(writes) != 0 {
			t.Errorf("%s: an unlisted tenant reached the store: reads %v, writes %v", c.path, reads, writes)
		}
	}
}

func TestAcrossClub_RankingRequestIsBounded(t *testing.T) {
	srv, log := newRecordingServer(t, false)
	for _, body := range []string{
		`{"minDecisions":-1}`,
		`{"offset":-1}`,
		`{"limit":` + strconv.Itoa(maxPageSize+1) + `}`,
		`{"filter":{"TournamentIDs":[1]}}`,
	} {
		if rec := serveAcross(t, srv, "1", "2", "/v1/across.clubRanking", body); rec.Code != http.StatusBadRequest {
			t.Errorf("across.clubRanking %s: status %d, want 400", body, rec.Code)
		}
	}
	if reads, _ := log.snapshot(); len(reads) != 0 {
		t.Errorf("a refused ranking read the store: %v", reads)
	}
	rec := serveAcross(t, srv, "1", "2", "/v1/across.clubRanking", `{"limit":1,"offset":5}`)
	var got acrossClubRankingResp
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Rows == nil {
		t.Errorf("across.clubRanking = %d %s, want a rows array", rec.Code, rec.Body)
	}
}

func TestPageOf_StopsTheStreamOnceThePageIsFull(t *testing.T) {
	pulled := 0
	seq := func(yield func(int, error) bool) {
		for i := range 100 {
			pulled++
			if !yield(i, nil) {
				return
			}
		}
	}
	var got []int
	for v, err := range pageOf(seq, 3, 2) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, v)
	}
	if !slices.Equal(got, []int{3, 4}) || pulled != 5 {
		t.Errorf("pageOf(3, 2) = %v after %d items pulled, want [3 4] after 5", got, pulled)
	}
	for _, err := range pageOf(seq, -1, 2) {
		if err == nil {
			t.Error("a negative offset is accepted")
		}
	}
}
