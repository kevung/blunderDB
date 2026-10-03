package mcp

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
)

// The list sizes a tool returns when the model names none, and the most it may
// ask for: a tool answer lands in a context window, not on a screen.
const (
	defaultLimit = 20
	maxLimit     = 200
)

func clampLimit(n int) int {
	switch {
	case n <= 0:
		return defaultLimit
	case n > maxLimit:
		return maxLimit
	}
	return n
}

// PositionSummary is how a list names a position: enough to choose which one
// to open with get_position, not the board itself.
type PositionSummary struct {
	ID           int64  `json:"id"`
	XGID         string `json:"xgid"`
	Decision     string `json:"decision"`
	Dice         [2]int `json:"dice"`
	Score        [2]int `json:"score"`
	CubeValue    int    `json:"cubeValue"`
	PlayerOnRoll int    `json:"playerOnRoll"`
}

func decisionName(t int) string {
	if t == domain.CubeAction {
		return "cube"
	}
	return "checker"
}

func summarize(p *domain.Position) PositionSummary {
	return PositionSummary{
		ID: p.ID, XGID: domain.EncodeXGID(p), Decision: decisionName(p.DecisionType),
		Dice: p.Dice, Score: p.Score, CubeValue: p.Cube.Value, PlayerOnRoll: p.PlayerOnRoll,
	}
}

func summarizeAll(ps []domain.Position) []PositionSummary {
	out := make([]PositionSummary, 0, len(ps))
	for i := range ps {
		out = append(out, summarize(&ps[i]))
	}
	return out
}

// statsFilter is the player filter shared by the statistics tools. Its wire
// keys are storage.StatsFilter's own (Go field names: the type carries no tags).
type statsFilter struct {
	Player        string   `json:"player" jsonschema:"player name, exactly as list_players returns it"`
	Aliases       []string `json:"aliases,omitempty" jsonschema:"other spellings of the same player, counted with it"`
	TournamentIDs []int64  `json:"tournamentIds,omitempty" jsonschema:"restrict to these tournaments"`
	DateFrom      string   `json:"dateFrom,omitempty" jsonschema:"first match date, YYYY-MM-DD"`
	DateTo        string   `json:"dateTo,omitempty" jsonschema:"last match date, YYYY-MM-DD"`
	Decision      string   `json:"decision,omitempty" jsonschema:"checker, cube or empty for both"`
	MatchLengths  []int    `json:"matchLengths,omitempty" jsonschema:"restrict to these match lengths (0 = money game)"`
}

func (f statsFilter) wire() map[string]any {
	dt := -1
	switch strings.ToLower(f.Decision) {
	case "checker":
		dt = domain.CheckerAction
	case "cube":
		dt = domain.CubeAction
	}
	return map[string]any{"filter": map[string]any{
		"PlayerName": f.Player, "PlayerAliases": f.Aliases, "TournamentIDs": f.TournamentIDs,
		"DateFrom": f.DateFrom, "DateTo": f.DateTo, "DecisionType": dt, "MatchLength": f.MatchLengths,
	}}
}

type obj = map[string]any

func pick(m obj, keys ...string) obj {
	out := obj{}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			out[k] = v
		}
	}
	return out
}

func firstN(v any, n int) any {
	if s, ok := v.([]any); ok && len(s) > n {
		return s[:n]
	}
	return v
}

// groupIDsShown caps the positions listed per recurring-error group: they come
// worst first, and a quiz needs a handful, not the whole group.
const groupIDsShown = 50

type noInput struct{}

type positionRef struct {
	PositionID int64  `json:"positionId,omitempty" jsonschema:"a position id of this database"`
	Text       string `json:"text,omitempty" jsonschema:"or a position given as text: XGID, OGID or GNU ID"`
}

// position resolves a reference to a position: from the database by id, or
// parsed from text by the same parser the GUI's paste uses.
func (tb *Toolbox) position(ctx context.Context, req *sdk.CallToolRequest, ref positionRef) (*domain.Position, error) {
	if ref.PositionID > 0 {
		var p domain.Position
		if err := tb.Engine.Call(ctx, req, "positions.load", obj{"id": ref.PositionID}, &p); err != nil {
			return nil, err
		}
		return &p, nil
	}
	if strings.TrimSpace(ref.Text) == "" {
		return nil, errors.New("give positionId or text")
	}
	var parsed struct {
		Position domain.Position `json:"position"`
	}
	if err := tb.Engine.Call(ctx, req, "positions.parseText", obj{"text": ref.Text}, &parsed); err != nil {
		return nil, err
	}
	return &parsed.Position, nil
}

// analysis loads a position's analysis, nil when it has none.
func (tb *Toolbox) analysis(ctx context.Context, req *sdk.CallToolRequest, id int64) (obj, error) {
	var a obj
	if err := tb.Engine.Call(ctx, req, "analyses.load", obj{"positionId": id}, &a); err != nil {
		var api *APIError
		if errors.As(err, &api) && api.Status == 404 {
			return nil, nil
		}
		return nil, err
	}
	return a, nil
}

// playedOf is the decision actually taken in a position, as the analysis
// recorded it from the imported match.
func playedOf(a obj) string {
	for _, k := range []string{"playedMoves", "playedCubeActions"} {
		if s, ok := a[k].([]any); ok && len(s) > 0 {
			if v, ok := s[0].(string); ok && v != "" {
				return v
			}
		}
	}
	for _, k := range []string{"playedMove", "playedCubeAction"} {
		if v, ok := a[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func registerBuiltins(tb *Toolbox) {
	registerOverview(tb)
	registerSearch(tb)
	registerPositions(tb)
	registerPlayers(tb)
	registerMatches(tb)
	registerStudy(tb)
	registerRollout(tb)
	registerClub(tb)
	registerWrites(tb)
}

func registerOverview(tb *Toolbox) {
	Add(tb, Reads, &sdk.Tool{Name: "database_overview", Title: "Database overview",
		Description: "What this database holds: counts of positions, analyses, matches, games, comments and study cards, the span of match dates, the database schema version, and the most frequent players."},
		func(ctx context.Context, req *sdk.CallToolRequest, _ noInput) (any, error) {
			var counts, version, dates obj
			var players []obj
			if err := tb.Engine.Call(ctx, req, "metadata.counts", nil, &counts); err != nil {
				return nil, err
			}
			if err := tb.Engine.Call(ctx, req, "metadata.version", nil, &version); err != nil {
				return nil, err
			}
			if err := tb.Engine.Call(ctx, req, "stats.dateRange", nil, &dates); err != nil {
				return nil, err
			}
			if err := tb.Engine.Call(ctx, req, "stats.playerNames", nil, &players); err != nil {
				return nil, err
			}
			if len(players) > 10 {
				players = players[:10]
			}
			return obj{"counts": counts, "version": version, "matchDates": dates, "topPlayers": players}, nil
		})
}

var searchDescription = `Find positions with blunderDB's search grammar — the text the application's command bar takes. A query starts with "s", then tokens separated by spaces, all of which must hold:
  s cube p>30 E>50        cube decisions, 30+ pips behind, 50+ millipoints of error
  s m"13/11" T>2026/01/01 played 13/11, imported this year
  s E>100 pl"Alice"       Alice's (either seat) decisions costing more than 0.100

` + searchquery.Reference + `Here the search board is empty, so "cube", "score" and "D" match positions with no cube, no score and no dice: leave them out. An unknown token is refused with its name; fix it rather than dropping it. The answer gives the query's canonical form, a note for any token that had no effect, and position summaries; open one with get_position.`

func registerSearch(tb *Toolbox) {
	type in struct {
		Query  string `json:"query" jsonschema:"the search, e.g. s cube E>50"`
		Limit  int    `json:"limit,omitempty" jsonschema:"rows to return (default 20, at most 200)"`
		Offset int    `json:"offset,omitempty" jsonschema:"rows to skip, for the next page"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "search_positions", Title: "Search positions", Description: searchDescription},
		func(ctx context.Context, req *sdk.CallToolRequest, a in) (any, error) {
			var parsed struct {
				Diags     []obj  `json:"diags"`
				Canonical string `json:"canonical"`
			}
			if err := tb.Engine.Call(ctx, req, "search.parse", obj{"query": a.Query}, &parsed); err != nil {
				return nil, err
			}
			limit := clampLimit(a.Limit)
			rows, err := Stream[domain.Position](ctx, tb.Engine, req, "search.query",
				obj{"query": a.Query, "limit": limit, "offset": a.Offset}, limit)
			if err != nil {
				return nil, err
			}
			return obj{"canonical": parsed.Canonical, "notes": parsed.Diags, "count": len(rows),
				"positions": summarizeAll(rows)}, nil
		})

	type commentsIn struct {
		Query string `json:"query" jsonschema:"words to find in the comments"`
		Limit int    `json:"limit,omitempty" jsonschema:"rows to return (default 20, at most 200)"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "search_comments", Title: "Search comments",
		Description: "Full-text search in the comments written on positions (the user's own and those imported from XG, GNU Backgammon or BGBlitz). Returns the comment, its position id and its origin."},
		func(ctx context.Context, req *sdk.CallToolRequest, a commentsIn) (any, error) {
			rows, err := Stream[obj](ctx, tb.Engine, req, "comments.search", obj{"query": a.Query}, clampLimit(a.Limit))
			if err != nil {
				return nil, err
			}
			return obj{"comments": rows}, nil
		})

	Add(tb, Reads, &sdk.Tool{Name: "saved_searches", Title: "Saved searches",
		Description: "The searches the user saved in the application, each with its name and query: a ready vocabulary for search_positions."},
		func(ctx context.Context, req *sdk.CallToolRequest, _ noInput) (any, error) {
			rows, err := Stream[obj](ctx, tb.Engine, req, "filters.list", nil, maxLimit)
			if err != nil {
				return nil, err
			}
			return obj{"searches": rows}, nil
		})
}

func registerPositions(tb *Toolbox) {
	type getIn struct {
		PositionID int64 `json:"positionId" jsonschema:"the position id"`
		Moves      int   `json:"moves,omitempty" jsonschema:"checker plays of the analysis to include, best first (default 5)"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "get_position", Title: "Read a position",
		Description: "One position with its engine analysis: XGID, decision, dice, score, cube; for a checker decision the best plays with equity and error, for a cube decision the cube analysis and best action; what was played in the imported match; and the user's comment."},
		func(ctx context.Context, req *sdk.CallToolRequest, a getIn) (any, error) {
			p, err := tb.position(ctx, req, positionRef{PositionID: a.PositionID})
			if err != nil {
				return nil, err
			}
			out := obj{"position": summarize(p)}
			ana, err := tb.analysis(ctx, req, a.PositionID)
			if err != nil {
				return nil, err
			}
			if ana != nil {
				n := a.Moves
				if n <= 0 {
					n = 5
				}
				view := pick(ana, "analysisType", "analysisEngineVersion", "doublingCubeAnalysis",
					"playedMoves", "playedCubeActions", "player1", "player2")
				if ca, ok := ana["checkerAnalysis"].(obj); ok {
					view["checkerMoves"] = firstN(ca["moves"], n)
				}
				out["analysis"] = view
			}
			var comment struct {
				Text string `json:"text"`
			}
			if err := tb.Engine.Call(ctx, req, "comments.text", obj{"positionId": a.PositionID}, &comment); err != nil {
				return nil, err
			}
			if comment.Text != "" {
				out["comment"] = comment.Text
			}
			return out, nil
		})

	type explainIn struct {
		PositionID int64  `json:"positionId" jsonschema:"the position id"`
		Played     string `json:"played,omitempty" jsonschema:"the decision to judge (a play like 13/11 6/5, or nd, dt, dp); default: what was played in the match"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "explain_error", Title: "Explain an error",
		Description: "Why a decision was wrong, measured rather than worded: a theme (blots, point, gammon, passive for a checker play; doubletoolate, doubletooearly, taketooloose, passtootight for the cube), its cost in millipoints, the measured differences the theme rests on (blots left, points made, gammon rate) and the best decision. An empty theme means no rule applies confidently or the error is under 60 mp: say so rather than invent a reason."},
		func(ctx context.Context, req *sdk.CallToolRequest, a explainIn) (any, error) {
			played := a.Played
			if played == "" {
				ana, err := tb.analysis(ctx, req, a.PositionID)
				if err != nil {
					return nil, err
				}
				played = playedOf(ana)
				if played == "" {
					return nil, errors.New("nothing was recorded as played here: pass played")
				}
			}
			var ex obj
			if err := tb.Engine.Call(ctx, req, "positions.explain", obj{"positionId": a.PositionID, "played": played}, &ex); err != nil {
				return nil, err
			}
			ex["played"] = played
			return ex, nil
		})

	type similarIn struct {
		PositionID int64 `json:"positionId" jsonschema:"the position id"`
		Limit      int   `json:"limit,omitempty" jsonschema:"rows to return (default 20, at most 200)"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "similar_positions", Title: "Similar positions",
		Description: "Positions of the database whose checker layout is closest to this one, nearest first, with their distance (checkers to move)."},
		func(ctx context.Context, req *sdk.CallToolRequest, a similarIn) (any, error) {
			var rows []struct {
				Position domain.Position `json:"position"`
				Distance int             `json:"distance"`
			}
			if err := tb.Engine.Call(ctx, req, "positions.similar", obj{"positionId": a.PositionID, "limit": clampLimit(a.Limit)}, &rows); err != nil {
				return nil, err
			}
			out := make([]obj, 0, len(rows))
			for i := range rows {
				out = append(out, obj{"position": summarize(&rows[i].Position), "distance": rows[i].Distance})
			}
			return obj{"similar": out}, nil
		})

	Add(tb, Reads, &sdk.Tool{Name: "decode_position", Title: "Decode a position",
		Description: "Read a position given as text (XGID, OGID, GNU ID, or an XG/GNU Backgammon analysis pasted whole) without saving it: its XGID, decision, dice, score and cube, and the analysis when the text carried one."},
		func(ctx context.Context, req *sdk.CallToolRequest, a struct {
			Text string `json:"text" jsonschema:"the position text"`
		}) (any, error) {
			var parsed struct {
				Position domain.Position `json:"position"`
				Analysis obj             `json:"analysis"`
			}
			if err := tb.Engine.Call(ctx, req, "positions.parseText", obj{"text": a.Text}, &parsed); err != nil {
				return nil, err
			}
			out := obj{"position": summarize(&parsed.Position)}
			if parsed.Analysis != nil {
				out["analysis"] = parsed.Analysis
			}
			return out, nil
		})

	Add(tb, Reads, &sdk.Tool{Name: "legal_moves", Title: "Legal plays",
		Description: "Every distinct legal checker play of a position for its dice, in notation. The position comes from the database (positionId) or from text (XGID…)."},
		func(ctx context.Context, req *sdk.CallToolRequest, a positionRef) (any, error) {
			p, err := tb.position(ctx, req, a)
			if err != nil {
				return nil, err
			}
			var plays []struct {
				Notation string `json:"notation"`
			}
			if err := tb.Engine.Call(ctx, req, "positions.legalMoves", obj{"position": p}, &plays); err != nil {
				return nil, err
			}
			out := make([]string, 0, len(plays))
			for _, pl := range plays {
				out = append(out, pl.Notation)
			}
			return obj{"dice": p.Dice, "plays": out}, nil
		})

	Add(tb, Reads, &sdk.Tool{Name: "race_epc", Title: "Effective pip count",
		Description: "Effective pip count (EPC) of both sides of a bear-off/race position, from the exact bear-off database, with each side's checker count. Meaningful only once contact is broken."},
		func(ctx context.Context, req *sdk.CallToolRequest, a positionRef) (any, error) {
			p, err := tb.position(ctx, req, a)
			if err != nil {
				return nil, err
			}
			var epc obj
			if err := tb.Engine.Call(ctx, req, "positions.epc", obj{"position": p}, &epc); err != nil {
				return nil, err
			}
			return obj{"position": summarize(p), "epc": epc}, nil
		})
}

func registerPlayers(tb *Toolbox) {
	Add(tb, Reads, &sdk.Tool{Name: "list_players", Title: "Players",
		Description: "Every player name in the database with how many matches it appears in, most frequent first, an alias counted under its canonical name (player_aliases). Spellings no alias joins yet may still be one person: pass them as aliases to the statistics tools."},
		func(ctx context.Context, req *sdk.CallToolRequest, _ noInput) (any, error) {
			var players []obj
			if err := tb.Engine.Call(ctx, req, "stats.playerNames", nil, &players); err != nil {
				return nil, err
			}
			return obj{"players": players}, nil
		})

	Add(tb, Reads, &sdk.Tool{Name: "player_aliases", Title: "Player aliases",
		Description: "The other spellings recorded for players (alias → canonical name), and the names that differ only by case, accents, punctuation or word order, suggested and never applied. The statistics tools and list_players already read every alias as its canonical name."},
		func(ctx context.Context, req *sdk.CallToolRequest, _ noInput) (any, error) {
			var aliases, suggestions obj
			if err := tb.Engine.Call(ctx, req, "players.alias.list", nil, &aliases); err != nil {
				return nil, err
			}
			if err := tb.Engine.Call(ctx, req, "players.alias.suggest", nil, &suggestions); err != nil {
				return nil, err
			}
			return obj{"aliases": aliases["aliases"], "suggestions": suggestions["suggestions"]}, nil
		})

	Add(tb, Reads, &sdk.Tool{Name: "player_stats", Title: "Player statistics",
		Description: "A player's performance over the matches the filter keeps: PR (performance rating, lower is better) overall, checker and cube, MWC lost, rolling PR over the last 5…250 decisions, and the breakdown by game phase, game type and tournament."},
		func(ctx context.Context, req *sdk.CallToolRequest, f statsFilter) (any, error) {
			var res obj
			if err := tb.Engine.Call(ctx, req, "stats.compute", f.wire(), &res); err != nil {
				return nil, err
			}
			return pick(res, "Totals", "PRGlobal", "PRChecker", "PRCube", "PRRolling", "MWCGlobal",
				"MWCChecker", "MWCCube", "MWCAvailable", "SnowieGlobal", "PerPhase", "PerGameType",
				"PerTournament", "CubeDirections"), nil
		})

	type errorsIn struct {
		statsFilter
		Limit int `json:"limit,omitempty" jsonschema:"blunders to return (default 20, at most 200)"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "recurring_errors", Title: "Recurring errors",
		Description: "Where a player loses equity again and again: the biggest blunders (position id, error in mp, description), the cube actions and comment tags ranked by PR, the histogram of error sizes, and the errors grouped by plan of play and theme (Groups, costliest first, each with at most 50 of its worst PositionIDs: Positions is how many it holds, Truncated says the list is cut). Open a blunder with get_position and explain_error; quiz a group by passing its PositionIDs to quiz_grade."},
		func(ctx context.Context, req *sdk.CallToolRequest, a errorsIn) (any, error) {
			var res obj
			if err := tb.Engine.Call(ctx, req, "stats.compute", a.wire(), &res); err != nil {
				return nil, err
			}
			out := pick(res, "Totals", "CubeActionBreakdown", "PerTag", "ErrorHistogram", "PerScore")
			out["TopBlunders"] = firstN(res["TopBlunders"], clampLimit(a.Limit))
			var recurring obj
			if err := tb.Engine.Call(ctx, req, "stats.recurringErrors", a.wire(), &recurring); err != nil {
				return nil, err
			}
			groups, _ := recurring["Groups"].([]any)
			for _, g := range groups {
				if group, ok := g.(obj); ok {
					all, _ := group["PositionIDs"].([]any)
					group["Positions"] = len(all)
					group["Truncated"] = len(all) > groupIDsShown
					group["PositionIDs"] = firstN(group["PositionIDs"], groupIDsShown)
				}
			}
			out["Groups"] = firstN(groups, clampLimit(a.Limit))
			return out, nil
		})

	type trainingIn struct {
		statsFilter
		Window string `json:"window,omitempty" jsonschema:"calendar window: week (default) or month"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "training_stats", Title: "Training progress",
		Description: "Quiz PR per session and per calendar window, set against the real PR of the same windows, with the Anki retention observed on the same time scale. Use it to tell whether what is drilled shows up in play."},
		func(ctx context.Context, req *sdk.CallToolRequest, a trainingIn) (any, error) {
			wire := a.wire()
			if a.Window != "" {
				wire["window"] = a.Window
			}
			var res obj
			if err := tb.Engine.Call(ctx, req, "stats.training", wire, &res); err != nil {
				return nil, err
			}
			return res, nil
		})
}

func registerMatches(tb *Toolbox) {
	type listIn struct {
		Player       string `json:"player,omitempty" jsonschema:"matches with this player, at either seat (exact name)"`
		TournamentID int64  `json:"tournamentId,omitempty" jsonschema:"matches of this tournament"`
		DateFrom     string `json:"dateFrom,omitempty" jsonschema:"first match date, YYYY-MM-DD"`
		DateTo       string `json:"dateTo,omitempty" jsonschema:"last match date, YYYY-MM-DD"`
		MatchLengths []int  `json:"matchLengths,omitempty" jsonschema:"restrict to these lengths (0 = money game)"`
		Limit        int    `json:"limit,omitempty" jsonschema:"rows to return (default 20, at most 200)"`
		Offset       int    `json:"offset,omitempty" jsonschema:"rows to skip, for the next page"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "list_matches", Title: "Matches",
		Description: "Imported matches, newest first: id, players, length, date, event, tournament, and each side's PR."},
		func(ctx context.Context, req *sdk.CallToolRequest, a listIn) (any, error) {
			body := obj{"playerName": a.Player, "dateFrom": a.DateFrom, "dateTo": a.DateTo,
				"matchLength": a.MatchLengths, "limit": clampLimit(a.Limit), "offset": a.Offset}
			if a.TournamentID > 0 {
				body["tournamentIds"] = []int64{a.TournamentID}
			}
			rows, err := Stream[obj](ctx, tb.Engine, req, "matches.list", body, clampLimit(a.Limit))
			if err != nil {
				return nil, err
			}
			out := make([]obj, 0, len(rows))
			for _, m := range rows {
				out = append(out, pick(m, "id", "player1_name", "player2_name", "match_length", "match_date",
					"event", "tournament_name", "game_count", "pr", "pr2", "comment"))
			}
			return obj{"matches": out}, nil
		})

	Add(tb, Reads, &sdk.Tool{Name: "get_match", Title: "Read a match",
		Description: "One match: players, length, date, event, and each side's detailed performance (decisions, errors, blunders, PR, checker and cube split)."},
		func(ctx context.Context, req *sdk.CallToolRequest, a struct {
			MatchID int64 `json:"matchId" jsonschema:"the match id"`
		}) (any, error) {
			var m, detail obj
			if err := tb.Engine.Call(ctx, req, "matches.get", obj{"id": a.MatchID}, &m); err != nil {
				return nil, err
			}
			if err := tb.Engine.Call(ctx, req, "stats.matchDetail", obj{"matchId": a.MatchID}, &detail); err != nil {
				return nil, err
			}
			return obj{"match": pick(m, "id", "player1_name", "player2_name", "match_length", "match_date",
				"event", "location", "round", "tournament_name", "game_count", "comment"), "performance": detail}, nil
		})

	type pageIn struct {
		Limit  int `json:"limit,omitempty" jsonschema:"rows to return (default 20, at most 200)"`
		Offset int `json:"offset,omitempty" jsonschema:"rows to skip, for the next page"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "list_tournaments", Title: "Tournaments",
		Description: "Tournaments of the database: id, name, date, location, match count, comment. Pass an id to list_matches or player_stats."},
		func(ctx context.Context, req *sdk.CallToolRequest, a pageIn) (any, error) {
			rows, err := Stream[obj](ctx, tb.Engine, req, "tournaments.list", obj{"limit": clampLimit(a.Limit), "offset": a.Offset}, clampLimit(a.Limit))
			if err != nil {
				return nil, err
			}
			return obj{"tournaments": rows}, nil
		})
}

func registerStudy(tb *Toolbox) {
	Add(tb, Reads, &sdk.Tool{Name: "list_collections", Title: "Collections",
		Description: "The user's collections of positions: id, name, description, position count, and the search query of a living collection."},
		func(ctx context.Context, req *sdk.CallToolRequest, _ noInput) (any, error) {
			rows, err := Stream[obj](ctx, tb.Engine, req, "collections.list", nil, maxLimit)
			if err != nil {
				return nil, err
			}
			return obj{"collections": rows}, nil
		})

	type collIn struct {
		CollectionID int64 `json:"collectionId" jsonschema:"the collection id"`
		Limit        int   `json:"limit,omitempty" jsonschema:"rows to return (default 20, at most 200)"`
		Offset       int   `json:"offset,omitempty" jsonschema:"rows to skip, for the next page"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "collection_positions", Title: "Positions of a collection",
		Description: "The positions of one collection, in the collection's order, as summaries."},
		func(ctx context.Context, req *sdk.CallToolRequest, a collIn) (any, error) {
			limit := clampLimit(a.Limit)
			rows, err := Stream[domain.Position](ctx, tb.Engine, req, "collections.positions",
				obj{"collectionId": a.CollectionID, "limit": limit, "offset": a.Offset}, limit)
			if err != nil {
				return nil, err
			}
			return obj{"positions": summarizeAll(rows)}, nil
		})

	Add(tb, Reads, &sdk.Tool{Name: "list_lessons", Title: "Lessons",
		Description: "The lessons a coach wrote: id, name, description and number of steps. Read one with lesson."},
		func(ctx context.Context, req *sdk.CallToolRequest, _ noInput) (any, error) {
			var rows []obj
			if err := tb.Engine.Call(ctx, req, "lessons.list", nil, &rows); err != nil {
				return nil, err
			}
			return obj{"lessons": rows}, nil
		})

	type lessonIn struct {
		LessonID int64 `json:"lessonId" jsonschema:"the lesson id"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "lesson", Title: "A lesson",
		Description: "One lesson with its steps in reading order: each step's title, text, and the collectionId or positionId it shows (0 when none) — read those with collection_positions or the position tools."},
		func(ctx context.Context, req *sdk.CallToolRequest, a lessonIn) (any, error) {
			var l obj
			if err := tb.Engine.Call(ctx, req, "lessons.get", obj{"id": a.LessonID}, &l); err != nil {
				return nil, err
			}
			return l, nil
		})

	Add(tb, Reads, &sdk.Tool{Name: "study_decks", Title: "Study decks",
		Description: "The spaced-repetition (Anki-style) decks: id, name, card count and how many cards are due now."},
		func(ctx context.Context, req *sdk.CallToolRequest, _ noInput) (any, error) {
			rows, err := Stream[obj](ctx, tb.Engine, req, "anki.listDecks", nil, maxLimit)
			if err != nil {
				return nil, err
			}
			out := make([]obj, 0, len(rows))
			for _, d := range rows {
				out = append(out, pick(d, "id", "name", "description", "cardCount", "dueCount", "newCount"))
			}
			return obj{"decks": out}, nil
		})

	type drawIn struct {
		Query string `json:"query,omitempty" jsonschema:"restrict the draw to a search (search_positions grammar); default: any position with an error over 50 mp"`
		Pool  int    `json:"pool,omitempty" jsonschema:"how many matching positions to draw among (default 200)"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "quiz_draw", Title: "Draw a quiz position",
		Description: "Draw one analysed position at random to quiz the user on: its XGID, decision, dice, score and cube — never the answer. Show it, ask for a play (checker) or nd/dt/dp (cube), then call quiz_grade."},
		func(ctx context.Context, req *sdk.CallToolRequest, a drawIn) (any, error) {
			q := strings.TrimSpace(a.Query)
			if q == "" {
				q = "s E>50"
			}
			pool := a.Pool
			if pool <= 0 || pool > maxLimit {
				pool = maxLimit
			}
			rows, err := Stream[domain.Position](ctx, tb.Engine, req, "search.query", obj{"query": q, "limit": pool}, pool)
			if err != nil {
				return nil, err
			}
			rand.Shuffle(len(rows), func(i, j int) { rows[i], rows[j] = rows[j], rows[i] })
			for i := range rows {
				ana, err := tb.analysis(ctx, req, rows[i].ID)
				if err != nil {
					return nil, err
				}
				if ana != nil {
					return obj{"position": summarize(&rows[i]), "poolSize": len(rows)}, nil
				}
			}
			return nil, fmt.Errorf("no analysed position matches %q", q)
		})

	type gradeIn struct {
		PositionID int64  `json:"positionId" jsonschema:"the position the user was quizzed on"`
		Answer     string `json:"answer" jsonschema:"the user's answer: a play in notation (13/11 6/5) for a checker decision, nd, dt or dp for a cube decision"`
	}
	Add(tb, Reads, &sdk.Tool{Name: "quiz_grade", Title: "Grade a quiz answer",
		Description: "Grade the user's answer against the stored analysis — the same numbers the statistics charge — with the verdict, the error of the answer in mp and the best decision. Records nothing."},
		func(ctx context.Context, req *sdk.CallToolRequest, a gradeIn) (any, error) {
			p, err := tb.position(ctx, req, positionRef{PositionID: a.PositionID})
			if err != nil {
				return nil, err
			}
			var verdict obj
			if p.DecisionType == domain.CubeAction {
				err = tb.Engine.Call(ctx, req, "quiz.gradeCube", obj{"positionId": a.PositionID, "action": strings.ToLower(strings.TrimSpace(a.Answer))}, &verdict)
			} else {
				err = tb.Engine.Call(ctx, req, "quiz.gradeCheckerMove", obj{"positionId": a.PositionID, "move": a.Answer}, &verdict)
			}
			if err != nil {
				return nil, err
			}
			return verdict, nil
		})
}

func registerWrites(tb *Toolbox) {
	Add(tb, Writes, &sdk.Tool{Name: "save_position", Title: "Save a position",
		Description: "Add a position given as text (XGID, OGID, GNU ID, or a pasted XG/GNU Backgammon analysis, kept with it) to the database, as a position brought on its own. A position already present is not duplicated. Answers its id and whether it was new."},
		func(ctx context.Context, req *sdk.CallToolRequest, a struct {
			Text string `json:"text" jsonschema:"the position text"`
		}) (any, error) {
			var parsed struct {
				Position domain.Position `json:"position"`
				Analysis obj             `json:"analysis"`
			}
			if err := tb.Engine.Call(ctx, req, "positions.parseText", obj{"text": a.Text}, &parsed); err != nil {
				return nil, err
			}
			// Brought on its own, as a paste in the GUI is: the provenance the
			// search token i reads. The flag is sticky in storage, so a position
			// a match brought first gains it without losing anything.
			parsed.Position.IndividuallyImported = true
			var saved struct {
				ID      int64 `json:"id"`
				Created bool  `json:"created"`
			}
			if err := tb.Engine.Call(ctx, req, "positions.save", obj{"position": parsed.Position}, &saved); err != nil {
				return nil, err
			}
			// An existing analysis is kept: the pasted one only fills a new row.
			if saved.Created && parsed.Analysis != nil {
				if err := tb.Engine.Call(ctx, req, "analyses.save", obj{"positionId": saved.ID, "analysis": parsed.Analysis}, nil); err != nil {
					return nil, err
				}
			}
			return obj{"positionId": saved.ID, "created": saved.Created, "withAnalysis": saved.Created && parsed.Analysis != nil}, nil
		})

	Add(tb, Writes, &sdk.Tool{Name: "comment_position", Title: "Comment a position",
		Description: "Add a comment to a position, beside those it already carries. Tags are words starting with #."},
		func(ctx context.Context, req *sdk.CallToolRequest, a struct {
			PositionID int64  `json:"positionId" jsonschema:"the position id"`
			Text       string `json:"text" jsonschema:"the comment"`
		}) (any, error) {
			var res obj
			if err := tb.Engine.Call(ctx, req, "comments.add", obj{"positionId": a.PositionID, "text": a.Text}, &res); err != nil {
				return nil, err
			}
			return res, nil
		})

	Add(tb, Writes, &sdk.Tool{Name: "create_collection", Title: "Create a collection",
		Description: "Create an empty collection of positions; fill it with add_to_collection."},
		func(ctx context.Context, req *sdk.CallToolRequest, a struct {
			Name        string `json:"name" jsonschema:"the collection name"`
			Description string `json:"description,omitempty" jsonschema:"what it gathers"`
		}) (any, error) {
			var res obj
			if err := tb.Engine.Call(ctx, req, "collections.create", obj{"name": a.Name, "description": a.Description}, &res); err != nil {
				return nil, err
			}
			return res, nil
		})

	Add(tb, Writes, &sdk.Tool{Name: "add_to_collection", Title: "Add to a collection",
		Description: "Add positions to a collection, after those it holds."},
		func(ctx context.Context, req *sdk.CallToolRequest, a struct {
			CollectionID int64   `json:"collectionId" jsonschema:"the collection id"`
			PositionIDs  []int64 `json:"positionIds" jsonschema:"the positions to add"`
		}) (any, error) {
			var res obj
			if err := tb.Engine.Call(ctx, req, "collections.addPositions", obj{"collectionId": a.CollectionID, "positionIds": a.PositionIDs}, &res); err != nil {
				return nil, err
			}
			return res, nil
		})
}
