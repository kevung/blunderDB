package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testMatchVideoRoundTrip: a Match's video source and its Moves' Repères
// (ADR-0079) are read back as written, through every read of a match and its
// moves, and a Match without a video reads them back nil — a NULL read as ""
// or 0 would claim a video, or a Repère at the start of the media. Zero is a
// Repère like any other. ReplaceHeader replaces the source when given one,
// keeps it when given nil and clears it when given "".
func testMatchVideoRoundTrip(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()
	source := "https://example.org/finale.mp4"
	zero, roll, done := int64(0), int64(61_250), int64(64_800)

	write := func(video *string, ticks [][2]*int64) int64 {
		t.Helper()
		matchID, err := ms.Save(ctx, "", &domain.Match{Player1Name: "Alice", Player2Name: "Bob",
			MatchLength: 7, VideoSource: video})
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		gameID, err := ms.CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		p := checkerPos()
		posID, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position: %v", err)
		}
		for i, tk := range ticks {
			mv := domain.Move{GameID: gameID, MoveNumber: int32(i + 1), MoveType: "checker",
				PositionID: posID, Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5",
				RollTickMS: tk[0], TickMS: tk[1]}
			if _, err := ms.CreateMove(ctx, "", &mv); err != nil {
				t.Fatalf("CreateMove: %v", err)
			}
		}
		return matchID
	}
	same := func(a, b *int64) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
	check := func(name string, matchID int64, video *string, ticks [][2]*int64) {
		t.Helper()
		m, err := ms.Get(ctx, "", matchID)
		if err != nil {
			t.Fatalf("%s: Get: %v", name, err)
		}
		if (m.VideoSource == nil) != (video == nil) || (video != nil && *m.VideoSource != *video) {
			t.Errorf("%s: VideoSource = %v, want %v", name, m.VideoSource, video)
		}
		var got [][2]*int64
		for mv, err := range ms.MovesByMatch(ctx, "", matchID) {
			if err != nil {
				t.Fatalf("%s: MovesByMatch: %v", name, err)
			}
			got = append(got, [2]*int64{mv.RollTickMS, mv.TickMS})
		}
		var viaPositions [][2]*int64
		for mp, err := range ms.MovePositions(ctx, "", matchID) {
			if err != nil {
				t.Fatalf("%s: MovePositions: %v", name, err)
			}
			viaPositions = append(viaPositions, [2]*int64{mp.RollTickMS, mp.TickMS})
		}
		for read, rows := range map[string][][2]*int64{"MovesByMatch": got, "MovePositions": viaPositions} {
			if len(rows) != len(ticks) {
				t.Fatalf("%s, %s: %d moves, want %d", name, read, len(rows), len(ticks))
			}
			for i := range ticks {
				if !same(rows[i][0], ticks[i][0]) || !same(rows[i][1], ticks[i][1]) {
					t.Errorf("%s, %s, move %d: Repères %v/%v, want %v/%v",
						name, read, i+1, rows[i][0], rows[i][1], ticks[i][0], ticks[i][1])
				}
			}
		}
	}

	withTicks := [][2]*int64{{&roll, &done}, {nil, &zero}, {nil, nil}}
	withID := write(&source, withTicks)
	check("with a video", withID, &source, withTicks)

	bare := [][2]*int64{{nil, nil}}
	bareID := write(nil, bare)
	check("without a video", bareID, nil, bare)

	m, err := ms.Get(ctx, "", bareID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	local := "/home/alice/videos/finale.mkv"
	m.VideoSource = &local
	if err := ms.ReplaceHeader(ctx, "", bareID, m); err != nil {
		t.Fatalf("ReplaceHeader: %v", err)
	}
	check("after ReplaceHeader", bareID, &local, bare)

	m.VideoSource = nil
	if err := ms.ReplaceHeader(ctx, "", bareID, m); err != nil {
		t.Fatalf("ReplaceHeader(nil): %v", err)
	}
	check("after ReplaceHeader with nil", bareID, &local, bare)

	empty := ""
	m.VideoSource = &empty
	if err := ms.ReplaceHeader(ctx, "", bareID, m); err != nil {
		t.Fatalf(`ReplaceHeader(""): %v`, err)
	}
	check(`after ReplaceHeader with ""`, bareID, nil, bare)
}
