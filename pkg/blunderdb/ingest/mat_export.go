package ingest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/matrender"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// mat_export.go renders a stored match as a Jellyfish/gnubg .mat text — the
// inverse of the .mat parser, and the format XG re-imports. The output
// round-trips through gnubgparser.ParseMAT; analysis is NOT part of .mat.

// Move.Player and Game.Winner share XG's encoding {1 = player 1, -1 = player 2},
// Game.Winner adding 0 for an unfinished game (domain.WinnerPlayer1).

// ReadMatchForMAT reconstructs a stored match into the shape RenderMAT wants:
// the match header, its games in order, and each game's moves keyed by game id.
// It is the read-side counterpart WriteMatch never had.
//
// A Match played from a Start other than the opening position is refused
// (ErrInvalid): a .mat begins every game at the opening position, and would
// write a first game that was never played (ADR-0072 rule 12).
func ReadMatchForMAT(ctx context.Context, s storage.Storage, scope string, matchID int64) (*domain.Match, []*domain.Game, map[int64][]*domain.Move, error) {
	m, err := s.Matches().Get(ctx, scope, matchID)
	if err != nil {
		return nil, nil, nil, err
	}
	origin, err := s.Duels().Origin(ctx, scope, matchID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, nil, nil, err
	}
	if origin != nil && origin.Start != "" {
		return nil, nil, nil, fmt.Errorf("match %d began at a Start (%s), which a .mat cannot write: %w",
			matchID, origin.Start, storage.ErrInvalid)
	}
	var games []*domain.Game
	for g, err := range s.Matches().Games(ctx, scope, matchID) {
		if err != nil {
			return nil, nil, nil, err
		}
		games = append(games, g)
	}
	movesByGame := map[int64][]*domain.Move{}
	for mv, err := range s.Matches().MovesByMatch(ctx, scope, matchID) {
		if err != nil {
			return nil, nil, nil, err
		}
		movesByGame[mv.GameID] = append(movesByGame[mv.GameID], mv)
	}
	if m.PostCrawford, err = firstGamePostCrawford(ctx, s, scope, m, games, movesByGame); err != nil {
		return nil, nil, nil, err
	}
	return m, games, movesByGame, nil
}

// firstGamePostCrawford reads [domain.Match.PostCrawford] back from the stored
// match: the first game opens with a player one point away, and its first
// Position carries the PostCrawford away score rather than the Crawford one.
func firstGamePostCrawford(ctx context.Context, s storage.Storage, scope string, m *domain.Match, games []*domain.Game, moves map[int64][]*domain.Move) (bool, error) {
	L := m.MatchLength
	if L <= 0 || len(games) == 0 {
		return false, nil
	}
	g := games[0]
	if g.InitialScore[0] != L-1 && g.InitialScore[1] != L-1 {
		return false, nil
	}
	first := moves[g.ID]
	if len(first) == 0 || first[0].PositionID == 0 {
		return false, nil
	}
	pos, err := s.Positions().Load(ctx, scope, first[0].PositionID)
	if err != nil {
		return false, err
	}
	return pos.Score[0] == domain.PostCrawford || pos.Score[1] == domain.PostCrawford, nil
}

// ExportMatchMAT reads a stored match and writes its .mat transcript to w.
func ExportMatchMAT(ctx context.Context, s storage.Storage, scope string, matchID int64, w io.Writer) error {
	m, games, moves, err := ReadMatchForMAT(ctx, s, scope, matchID)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, RenderMAT(m, games, moves))
	return err
}

// SuggestMATFilename builds a filesystem-friendly default name for a match's
// .mat export: "{Player1}_{Player2}_{YYYY-MM-DD}_{Np|unlimited}.mat". Each
// player's words are CamelCased and concatenated (no spaces), accents kept;
// the date is the match date (falling back to the import date, then omitted);
// MatchLength <= 0 renders as "unlimited" (money game). The same name feeds the
// GUI save dialog and the CLI batch export so both agree.
func SuggestMATFilename(m *domain.Match) string {
	p1 := camelConcat(m.Player1Name)
	if p1 == "" {
		p1 = "Player1"
	}
	p2 := camelConcat(m.Player2Name)
	if p2 == "" {
		p2 = "Player2"
	}
	parts := []string{p1, p2}
	if d := matFilenameDate(m); d != "" {
		parts = append(parts, d)
	}
	if m.MatchLength > 0 {
		parts = append(parts, fmt.Sprintf("%dp", m.MatchLength))
	} else {
		parts = append(parts, "unlimited")
	}
	return strings.Join(parts, "_") + ".mat"
}

// matFilenameDate returns YYYY-MM-DD from the match date, falling back to the
// import date, or "" when neither is set.
func matFilenameDate(m *domain.Match) string {
	d := m.MatchDate
	if d.IsZero() {
		d = m.ImportDate
	}
	if d.IsZero() {
		return ""
	}
	return d.Format("2006-01-02")
}

// camelConcat splits a player name on whitespace, capitalises each word's first
// rune, drops filesystem-forbidden characters, and concatenates the words.
func camelConcat(s string) string {
	var b strings.Builder
	for _, word := range strings.Fields(s) {
		word = stripForbidden(word)
		if word == "" {
			continue
		}
		r := []rune(word)
		r[0] = unicode.ToUpper(r[0])
		b.WriteString(string(r))
	}
	return b.String()
}

// stripForbidden removes path separators, characters illegal in filenames on
// common filesystems, and control runes. Accents and other letters are kept.
func stripForbidden(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return -1
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// RenderMAT writes match m with its games (in order) and each game's moves
// (keyed by game id, in order) as a .mat transcript.
func RenderMAT(m *domain.Match, games []*domain.Game, movesByGame map[int64][]*domain.Move) string {
	return matrender.Render(m, games, movesByGame)
}
