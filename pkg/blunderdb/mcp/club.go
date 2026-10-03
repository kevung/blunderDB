package mcp

import (
	"context"
	"errors"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// The tools over evaluation, spaced repetition, transcription and tournament
// direction: each one a call to a /v1 route, so a tool reads exactly what the
// HTTP client of the same tenant reads.

func registerClub(tb *Toolbox) {
	registerEvaluate(tb)
	registerAnkiReview(tb)
	registerTranscriptionReads(tb)
	registerDirectionReads(tb)
}

func registerEvaluate(tb *Toolbox) {
	type evalIn struct {
		Text       string `json:"text" jsonschema:"the position as text: XGID, OGID or GNU ID"`
		Ply        int    `json:"ply,omitempty" jsonschema:"search depth, 0 to 2 (default 2)"`
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
			var out obj
			err = tb.Engine.Call(ctx, req, "gammonnet.evaluate", obj{"position": p, "ply": a.Ply, "candidates": a.Candidates}, &out)
			return out, err
		})
}

func registerAnkiReview(tb *Toolbox) {
	type nextIn struct {
		DeckID int64 `json:"deckId" jsonschema:"a deck id from study_decks"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "anki_next", Title: "Next card to review",
		Description: "The next due card of a spaced-repetition deck: its card id and position. Show the position, let the user answer, then grade it with anki_review (on a server that writes). Null when nothing is due."},
		func(ctx context.Context, req *sdk.CallToolRequest, a nextIn) (any, error) {
			var card obj
			if err := tb.Engine.Call(ctx, req, "anki.nextCard", obj{"deckId": a.DeckID}, &card); err != nil {
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
