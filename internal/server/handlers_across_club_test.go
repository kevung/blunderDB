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

func (p recordingPositions) Exists(ctx context.Context, scope string, zobrist uint64) (int64, bool, error) {
	p.log.read(scope)
	return p.PositionStore.Exists(ctx, scope, zobrist)
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
	rec := serveAcross(t, srv, "1", "2", "/v1/across.collectionPositions", `{"tenant":"2","collectionId":1}`)
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Fatalf("across.collectionPositions: %d %s", rec.Code, rec.Body)
	}
	if reads, _ := log.snapshot(); !slices.Equal(reads, []string{"2"}) {
		t.Errorf("store read %v, want [2]", reads)
	}
	srv, log = newRecordingServer(t, false)
	if rec := serveAcross(t, srv, "1", "2", "/v1/across.collectionPositions", `{"tenant":"9","collectionId":1}`); rec.Code != http.StatusBadRequest {
		t.Errorf("unlisted tenant: status %d, want 400", rec.Code)
	}
	if reads, _ := log.snapshot(); len(reads) != 0 {
		t.Errorf("an unlisted tenant reached the store: %v", reads)
	}
	if rec := serveAcross(t, srv, "1", "2", "/v1/across.collectionPositions", `{"tenant":"2","collectionId":1,"limit":`+strconv.Itoa(maxPageSize+1)+`}`); rec.Code != http.StatusBadRequest {
		t.Errorf("limit over maxPageSize: status %d, want 400", rec.Code)
	}
}

func TestAcrossClub_RankingRefusesNegativeMinDecisions(t *testing.T) {
	srv, _ := newRecordingServer(t, false)
	if rec := serveAcross(t, srv, "1", "", "/v1/across.clubRanking", `{"minDecisions":-1}`); rec.Code != http.StatusBadRequest {
		t.Errorf("minDecisions -1: status %d, want 400", rec.Code)
	}
	rec := serveAcross(t, srv, "1", "2", "/v1/across.clubRanking", `{}`)
	var got acrossClubRankingResp
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Rows == nil {
		t.Errorf("across.clubRanking = %d %s, want a rows array", rec.Code, rec.Body)
	}
}
