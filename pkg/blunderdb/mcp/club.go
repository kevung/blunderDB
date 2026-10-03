package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The tools over evaluation, spaced repetition, transcription and tournament
// direction: each one a call to a /v1 route, so a tool reads exactly what the
// HTTP client of the same tenant reads.

func registerClub(tb *Toolbox) {
	registerEvaluate(tb)
	registerAnkiReview(tb)
	registerTranscriptionReads(tb)
	registerDirectionReads(tb)
	registerAcrossReads(tb)
}

func registerEvaluate(tb *Toolbox) {
	type evalIn struct {
		Text       string `json:"text" jsonschema:"the position as text: XGID, OGID or GNU ID"`
		Ply        *int   `json:"ply,omitempty" jsonschema:"search depth, 0 to 2 (default 2)"`
		Candidates int    `json:"candidates,omitempty" jsonschema:"checker plays kept, 1 to 20 (default 5)"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "evaluate", Title: "Evaluate a position",
		Description: "Evaluate a position given as text with gammonNet, without storing it: with dice, the best plays and their equity; without dice, the cube decision. " +
			"Equities are money points in a money game, normalised equity (±1 = the current cube) at a match score."},
		func(ctx context.Context, req *sdk.CallToolRequest, a evalIn) (any, error) {
			if strings.TrimSpace(a.Text) == "" {
				return nil, errors.New("give text: an XGID, OGID or GNU ID")
			}
			p, err := tb.position(ctx, req, positionRef{Text: a.Text})
			if err != nil {
				return nil, err
			}
			// An absent ply is the route's default (2), not a zero-ply search.
			in := obj{"position": p, "candidates": a.Candidates}
			if a.Ply != nil {
				in["ply"] = *a.Ply
			}
			var out obj
			err = tb.Engine.Call(ctx, req, "gammonnet.evaluate", in, &out)
			return out, err
		})
}

func registerAnkiReview(tb *Toolbox) {
	type nextIn struct {
		DeckID int64 `json:"deckId" jsonschema:"a deck id from study_decks"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "anki_next", Title: "Next card to review",
		Description: "The next due card of a spaced-repetition deck: its card id and position. Show the position, let the user answer, then grade it with anki_review (on a server that writes). The card is null when nothing is due."},
		func(ctx context.Context, req *sdk.CallToolRequest, a nextIn) (any, error) {
			var card obj
			if err := tb.Engine.Call(ctx, req, "anki.nextCard", obj{"deckId": a.DeckID}, &card); err != nil {
				// The route answers 404 when no card is due: a null card, not a failure.
				var api *APIError
				if errors.As(err, &api) && api.Status == http.StatusNotFound {
					return obj{"card": nil}, nil
				}
				return nil, err
			}
			return obj{"card": card}, nil
		})
	if !tb.write {
		return
	}
	type reviewIn struct {
		CardID int64 `json:"cardId" jsonschema:"the card anki_next drew"`
		Rating int   `json:"rating" jsonschema:"1 again, 2 hard, 3 good, 4 easy"`
	}
	Add(tb, Writes, &sdk.Tool{Name: "anki_review", Title: "Grade a reviewed card",
		Description: "Record the user's grade of a card (1 again, 2 hard, 3 good, 4 easy) and reschedule it, as the review screen does. Answers the card with its next due date."},
		func(ctx context.Context, req *sdk.CallToolRequest, a reviewIn) (any, error) {
			if a.Rating < 1 || a.Rating > 4 {
				return nil, errors.New("rating is 1 (again), 2 (hard), 3 (good) or 4 (easy)")
			}
			var card obj
			err := tb.Engine.Call(ctx, req, "anki.reviewCard", obj{"cardId": a.CardID, "rating": a.Rating}, &card)
			return card, err
		})
}

func registerTranscriptionReads(tb *Toolbox) {
	Add(tb, Reads, &sdk.Tool{Name: "transcribe_list", Title: "Transcriptions",
		Description: "The match transcriptions of this database, in progress or finished: id, players, length and state."},
		func(ctx context.Context, req *sdk.CallToolRequest, _ noInput) (any, error) {
			var list []obj
			if err := tb.Engine.Call(ctx, req, "transcriptions.list", nil, &list); err != nil {
				return nil, err
			}
			return obj{"transcriptions": firstN(anySlice(list), maxLimit)}, nil
		})
	type idIn struct {
		ID int64 `json:"id" jsonschema:"a transcription id from transcribe_list"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "transcribe_get", Title: "Read a transcription",
		Description: "One transcription: its match header, games and the moves entered so far."},
		func(ctx context.Context, req *sdk.CallToolRequest, a idIn) (any, error) {
			var t obj
			err := tb.Engine.Call(ctx, req, "transcriptions.get", obj{"id": a.ID}, &t)
			return t, err
		})
	Add(tb, Reads, &sdk.Tool{Name: "transcribe_mat", Title: "Transcription as .mat",
		Description: "One transcription written as a Jellyfish .mat text, the format every analyser reads."},
		func(ctx context.Context, req *sdk.CallToolRequest, a idIn) (any, error) {
			var t obj
			err := tb.Engine.Call(ctx, req, "transcriptions.exportMat", obj{"id": a.ID}, &t)
			return t, err
		})
}

func registerDirectionReads(tb *Toolbox) {
	Add(tb, Reads, &sdk.Tool{Name: "direction_list", Title: "Directed tournaments",
		Description: "The tournaments directed in this database: tournament id, state (draft, running, finished) and the event (Rencontre) each plays in."},
		func(ctx context.Context, req *sdk.CallToolRequest, _ noInput) (any, error) {
			var list []obj
			if err := tb.Engine.Call(ctx, req, "directions.list", nil, &list); err != nil {
				return nil, err
			}
			return obj{"directions": firstN(anySlice(list), maxLimit)}, nil
		})
	type tIn struct {
		TournamentID int64 `json:"tournamentId" jsonschema:"a tournament id from direction_list"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "direction_standings", Title: "Tournament standings",
		Description: "The standings of a directed tournament, section by section, with each player's record and prize. Ties share a rank."},
		func(ctx context.Context, req *sdk.CallToolRequest, a tIn) (any, error) {
			var v obj
			err := tb.Engine.Call(ctx, req, "directions.standings", obj{"tournamentId": a.TournamentID}, &v)
			return v, err
		})
	type seasonIn struct {
		RencontreID int64     `json:"rencontreId,omitempty" jsonschema:"only the tournaments of this event"`
		From        string    `json:"from,omitempty" jsonschema:"first tournament date, YYYY-MM-DD"`
		To          string    `json:"to,omitempty" jsonschema:"last tournament date, YYYY-MM-DD"`
		Points      []float64 `json:"points,omitempty" jsonschema:"points by place, winner first (default 25,18,15,12,10,8,6,4,2,1)"`
		Elo         bool      `json:"elo,omitempty" jsonschema:"add a club Elo replayed over the season's matches"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "direction_season", Title: "Season ranking",
		Description: "The season ranking over several closed tournaments of an event or a period: points by place summed per person, optionally a club Elo (FIBS formula, start 1500)."},
		func(ctx context.Context, req *sdk.CallToolRequest, a seasonIn) (any, error) {
			var v obj
			err := tb.Engine.Call(ctx, req, "rencontres.ranking",
				obj{"rencontreId": a.RencontreID, "from": a.From, "to": a.To, "points": a.Points, "elo": a.Elo}, &v)
			return v, err
		})
}

func anySlice(v []obj) []any {
	out := make([]any, len(v))
	for i := range v {
		out[i] = v[i]
	}
	return out
}

// The club and coach reads (ADR-0063, ADR-0065) span the tenants the proxy
// lists in X-Read-Tenants on the /mcp request; the engine forwards that header
// on their across.* calls and on nothing else. Over stdio, or without the
// header, they read this tenant alone. A Zobrist hash travels as a decimal
// string: it is a 64-bit integer, which a JSON number read as a double loses.

const acrossNote = " Reads this tenant and the tenants the authenticating proxy lists (X-Read-Tenants); every row names its tenant."

func zobristString(z uint64) string { return strconv.FormatUint(z, 10) }

func parseZobrists(in []string) ([]uint64, error) {
	out := make([]uint64, 0, len(in))
	for _, s := range in {
		z, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("zobrist %q: a decimal hash as club_match_positions or club_library_positions returns it", s)
		}
		out = append(out, z)
	}
	return out, nil
}

// acrossPositionRow and its kin decode the across.* wire rows with the hash
// as an integer, before it is turned into a string for the model.
type acrossPositionRow struct {
	Tenant   string           `json:"tenant"`
	Zobrist  uint64           `json:"zobrist"`
	Position *domain.Position `json:"position"`
}

type acrossMoveRow struct {
	Tenant       string                    `json:"tenant"`
	Zobrist      uint64                    `json:"zobrist"`
	MovePosition *domain.MatchMovePosition `json:"movePosition"`
}

type acrossCommentRow struct {
	Tenant     string               `json:"tenant"`
	Zobrist    uint64               `json:"zobrist"`
	PositionID int64                `json:"positionId"`
	Comment    *domain.CommentEntry `json:"comment"`
}

func registerAcrossReads(tb *Toolbox) {
	type matchesIn struct {
		Player   string `json:"player,omitempty" jsonschema:"matches with this player, at either seat (exact name)"`
		DateFrom string `json:"dateFrom,omitempty" jsonschema:"first match date, YYYY-MM-DD"`
		DateTo   string `json:"dateTo,omitempty" jsonschema:"last match date, YYYY-MM-DD"`
		Limit    int    `json:"limit,omitempty" jsonschema:"rows per tenant (default 20, at most 200); at most 200 rows overall"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "club_matches", Title: "Matches across the club",
		Description: "Matches of the read tenants (a coach's students), newest first: tenant, match id, players, length, date, each side's PR." + acrossNote},
		func(ctx context.Context, req *sdk.CallToolRequest, a matchesIn) (any, error) {
			body := obj{"playerName": a.Player, "dateFrom": a.DateFrom, "dateTo": a.DateTo, "limit": clampLimit(a.Limit)}
			rows, err := Stream[obj](ctx, tb.Engine, req, "across.matchesList", body, maxLimit)
			if err != nil {
				return nil, err
			}
			out := make([]obj, 0, len(rows))
			for _, r := range rows {
				m, _ := r["match"].(obj)
				out = append(out, obj{"tenant": r["tenant"], "match": pick(m, "id", "player1_name", "player2_name",
					"match_length", "match_date", "event", "pr", "pr2")})
			}
			return obj{"matches": out}, nil
		})

	type matchIn struct {
		Tenant  string `json:"tenant" jsonschema:"the tenant club_matches named"`
		MatchID int64  `json:"matchId" jsonschema:"the match id in that tenant"`
		Limit   int    `json:"limit,omitempty" jsonschema:"positions to return (default 20, at most 200)"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "club_match_positions", Title: "A club match, move by move",
		Description: "The positions of one match of a read tenant, move by move: the move played, the position summary and its zobrist hash. " +
			"The hash names the same board in every tenant: pass it to club_comments to read the comments written on it." + acrossNote},
		func(ctx context.Context, req *sdk.CallToolRequest, a matchIn) (any, error) {
			rows, err := Stream[acrossMoveRow](ctx, tb.Engine, req, "across.matchMovePositions", obj{"tenant": a.Tenant, "matchId": a.MatchID}, clampLimit(a.Limit))
			if err != nil {
				return nil, err
			}
			out := make([]obj, 0, len(rows))
			for _, r := range rows {
				if r.MovePosition == nil {
					continue
				}
				mp := r.MovePosition
				out = append(out, obj{"tenant": r.Tenant, "zobrist": zobristString(r.Zobrist), "game": mp.GameNumber,
					"move": mp.MoveNumber, "player": mp.PlayerOnRoll, "played": mp.CheckerMove, "cubeAction": mp.CubeAction, "position": summarize(&mp.Position)})
			}
			return obj{"positions": out}, nil
		})

	type commentsIn struct {
		Zobrists []string `json:"zobrists" jsonschema:"board hashes, as club_match_positions or club_library_positions return them (at most 1000)"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "club_comments", Title: "Comments on boards across the club",
		Description: "The comments every read tenant wrote on the given boards, joined by zobrist hash: a coach's notes, written in the coach's own tenant, on a student's positions. " +
			"Each comment says its tenant (who wrote it), the board's id there and its origin (user or the file it came from)." + acrossNote},
		func(ctx context.Context, req *sdk.CallToolRequest, a commentsIn) (any, error) {
			hashes, err := parseZobrists(a.Zobrists)
			if err != nil {
				return nil, err
			}
			rows, err := Stream[acrossCommentRow](ctx, tb.Engine, req, "across.commentsByZobrist", obj{"zobrists": hashes}, maxLimit)
			if err != nil {
				return nil, err
			}
			out := make([]obj, 0, len(rows))
			for _, r := range rows {
				if r.Comment == nil {
					continue
				}
				out = append(out, obj{"tenant": r.Tenant, "zobrist": zobristString(r.Zobrist), "positionId": r.PositionID,
					"text": r.Comment.Text, "origin": r.Comment.Origin, "modifiedAt": r.Comment.ModifiedAt})
			}
			return obj{"comments": out}, nil
		})

	Add(tb, Reads, &sdk.Tool{Name: "club_library", Title: "Shared library",
		Description: "The collections of the read tenants, a shared library among them: tenant, collection id, name, description, position count. Read in place, never copied." + acrossNote},
		func(ctx context.Context, req *sdk.CallToolRequest, _ noInput) (any, error) {
			rows, err := Stream[obj](ctx, tb.Engine, req, "across.collectionsList", nil, maxLimit)
			if err != nil {
				return nil, err
			}
			out := make([]obj, 0, len(rows))
			for _, r := range rows {
				c, _ := r["collection"].(obj)
				out = append(out, obj{"tenant": r["tenant"], "collection": pick(c, "id", "name", "description", "positionCount")})
			}
			return obj{"collections": out}, nil
		})

	type libIn struct {
		Tenant       string `json:"tenant" jsonschema:"the tenant club_library named"`
		CollectionID int64  `json:"collectionId" jsonschema:"the collection id in that tenant"`
		Limit        int    `json:"limit,omitempty" jsonschema:"positions to return (default 20, at most 200)"`
		Offset       int    `json:"offset,omitempty" jsonschema:"positions to skip, for the next page"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "club_library_positions", Title: "Positions of a shared collection",
		Description: "The positions of one collection of a read tenant, in collection order, each with its summary and zobrist hash." + acrossNote},
		func(ctx context.Context, req *sdk.CallToolRequest, a libIn) (any, error) {
			body := obj{"tenant": a.Tenant, "collectionId": a.CollectionID, "limit": clampLimit(a.Limit), "offset": a.Offset}
			rows, err := Stream[acrossPositionRow](ctx, tb.Engine, req, "across.collectionPositions", body, clampLimit(a.Limit))
			if err != nil {
				return nil, err
			}
			out := make([]obj, 0, len(rows))
			for _, r := range rows {
				if r.Position == nil {
					continue
				}
				out = append(out, obj{"tenant": r.Tenant, "zobrist": zobristString(r.Zobrist), "position": summarize(r.Position)})
			}
			return obj{"positions": out}, nil
		})

	type rankingIn struct {
		DateFrom     string `json:"dateFrom,omitempty" jsonschema:"first match date, YYYY-MM-DD"`
		DateTo       string `json:"dateTo,omitempty" jsonschema:"last match date, YYYY-MM-DD"`
		Decision     string `json:"decision,omitempty" jsonschema:"checker, cube or empty for both"`
		MinDecisions int    `json:"minDecisions,omitempty" jsonschema:"leave out players with fewer counted decisions"`
		Players      []struct {
			Tenant string `json:"tenant"`
			Name   string `json:"name"`
		} `json:"players,omitempty" jsonschema:"keep only these players, each named with its tenant"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "club_ranking", Title: "Club ranking",
		Description: "One ranking over the read tenants' players, best PR first, for a period: tenant, rank (ties share it; 0 = no counted decision), name, matches, decisions, PR, errors, blunders. " +
			"A name is never merged across tenants. Not the season ranking of directed tournaments (direction_season)." + acrossNote},
		func(ctx context.Context, req *sdk.CallToolRequest, a rankingIn) (any, error) {
			body := statsFilter{DateFrom: a.DateFrom, DateTo: a.DateTo, Decision: a.Decision}.wire()
			body["minDecisions"] = a.MinDecisions
			body["players"] = a.Players
			var v struct {
				Rows []obj `json:"rows"`
			}
			if err := tb.Engine.Call(ctx, req, "across.clubRanking", body, &v); err != nil {
				return nil, err
			}
			out := make([]obj, 0, len(v.Rows))
			for _, r := range v.Rows {
				p, _ := r["player"].(obj)
				out = append(out, obj{"tenant": r["tenant"], "rank": r["rank"], "player": pick(p, "name", "matches", "wins", "losses",
					"decisions", "pr", "pr_checker", "pr_cube", "errors", "blunders")})
			}
			return obj{"rows": firstN(anySlice(out), maxLimit)}, nil
		})
}
