package database

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

// TranscriptionMark is one line of the manifest of Repères: a move of a match
// whose video is a file on this machine, with the instants it was timed at and
// the position before and after it. It carries no image and no pixel; whoever
// holds the file reads the frames.
type TranscriptionMark struct {
	// Source is the video's path, as the match records it.
	Source  string `json:"source"`
	MatchID int64  `json:"match_id"`
	// Game and Move place the move as the match shows it; Action is its index
	// in the replayed transcription.
	Game   int   `json:"game"`
	Move   int32 `json:"move"`
	Action int   `json:"action"`
	// Player is 1 or 2, the side that acted.
	Player int `json:"player"`
	// Kind is the action's kind (checker, double, take, pass...).
	Kind string `json:"kind"`
	// Dice is {0, 0} for an action that rolls none.
	Dice [2]int `json:"dice"`
	// Play is the move as a transcript writes it, empty for a cube action.
	Play string `json:"play,omitempty"`
	// RollTickMS and TickMS are the Repères, nil when unknown (never zero); a
	// line carries at least one.
	RollTickMS *int64 `json:"roll_tick_ms,omitempty"`
	TickMS     *int64 `json:"tick_ms,omitempty"`
	// BoardBefore is the position the action was played from, BoardAfter the
	// board it left.
	BoardBefore domain.Board `json:"board_before"`
	BoardAfter  domain.Board `json:"board_after"`
	// Inconsistencies are the replay's findings on this action.
	Inconsistencies []string `json:"inconsistencies,omitempty"`
}

// TranscriptionMarksSummary accounts for the matches the manifest looked at.
type TranscriptionMarksSummary struct {
	// Matches is how many matches contributed lines.
	Matches int `json:"matches"`
	Marks   int `json:"marks"`
	// URLSources counts the matches left out because their video is an http(s)
	// URL: such a source never enters the manifest.
	URLSources int `json:"url_sources"`
	// Missing lists the local sources that are not on this disk, left out too.
	Missing []string `json:"missing,omitempty"`
}

// TranscriptionMarks reads the manifest of Repères of the library, or of
// match matchID when it is not zero: one line per move that carries a Repère,
// for the matches whose video is a local file. A URL source is counted and
// skipped, a local source absent from the disk is named in Missing and
// skipped, a match without a video contributes nothing. It writes nothing.
func (d *Database) TranscriptionMarks(matchID int64) ([]TranscriptionMark, TranscriptionMarksSummary, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var sum TranscriptionMarksSummary
	ctx := context.Background()

	var matches []*domain.Match
	if matchID != 0 {
		m, err := d.store.Matches().Get(ctx, "", matchID)
		if err != nil {
			return nil, sum, fmt.Errorf("match %d: %w", matchID, err)
		}
		matches = append(matches, m)
	} else {
		for m, err := range d.store.Matches().List(ctx, "", storage.MatchListOpts{}) {
			if err != nil {
				return nil, sum, err
			}
			matches = append(matches, m)
		}
	}

	var marks []TranscriptionMark
	for _, m := range matches {
		if m.VideoSource == nil || strings.TrimSpace(*m.VideoSource) == "" {
			continue
		}
		source := strings.TrimSpace(*m.VideoSource)
		if issuance.IsVideoURL(source) {
			sum.URLSources++
			continue
		}
		if _, err := os.Stat(source); err != nil {
			sum.Missing = append(sum.Missing, source)
			continue
		}
		got, err := d.matchMarks(ctx, m.ID, source)
		if err != nil {
			return nil, sum, err
		}
		if len(got) > 0 {
			sum.Matches++
			marks = append(marks, got...)
		}
	}
	sum.Marks = len(marks)
	return marks, sum, nil
}

// matchMarks replays match id through its .mat rendering, puts each Move's
// Repères back on its Action, and keeps the Actions that carry one.
func (d *Database) matchMarks(ctx context.Context, id int64, source string) ([]TranscriptionMark, error) {
	m, games, moves, err := ingest.ReadMatchForMAT(ctx, d.store, "", id)
	if err != nil {
		return nil, fmt.Errorf("match %d: %w", id, err)
	}
	doc, err := transcript.FromMAT(ingest.RenderMAT(m, games, moves))
	if err != nil {
		return nil, fmt.Errorf("match %d: %w", id, err)
	}
	transcription.RestoreTimecodes(&doc, games, moves)
	ann := transcript.Replay(doc, 0)

	var out []TranscriptionMark
	for i, info := range ann.Actions {
		a := doc.Actions[i]
		if a.RollTickMS == nil && a.TickMS == nil {
			continue
		}
		mark := TranscriptionMark{
			Source: source, MatchID: id,
			Game: info.GameNumber, Move: info.MoveNumber, Action: info.Index,
			Player: info.Side + 1, Kind: string(info.Kind),
			Play:       info.Notation,
			RollTickMS: a.RollTickMS, TickMS: a.TickMS,
			BoardBefore: info.Before.Board, BoardAfter: info.After,
		}
		switch info.Kind {
		case transcript.KindChecker, transcript.KindDance, transcript.KindUnrecorded:
			mark.Dice = [2]int{info.Before.Dice[0], info.Before.Dice[1]}
		default:
			mark.Play = ""
		}
		for _, bad := range info.Inconsistencies {
			mark.Inconsistencies = append(mark.Inconsistencies, string(bad.Kind))
		}
		out = append(out, mark)
	}
	return out, nil
}
