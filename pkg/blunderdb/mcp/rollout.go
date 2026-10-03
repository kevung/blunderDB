package mcp

import (
	"context"
	"errors"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const rolloutDescription = `Roll a position of this database out with gammonNet: its plays when it has dice, its cube decision otherwise. ` +
	`Each candidate plays the same dice, the luck of every roll is taken out, and the result gives per candidate the equity, ` +
	`its 95 % interval (ci95), the games played and the JSD (gap to the best in standard deviations; 3 or more separates them). ` +
	`The cube is played inside the games: trust the ranking more than the absolute equity. ` +
	`rollout is "fast" (216 games truncated at 7 half-moves, seconds), "standard" (1296 games truncated at 11, a minute or more), ` +
	`or custom: a preset then key=value among games, min-games, truncation, jsd, ply, candidates, seed (games a multiple of 36), e.g. "standard,ply=1".`

// rolloutArgs is what both forms of the tool take; rolloutStoreArgs adds the
// write, offered only on a server that writes.
type rolloutArgs struct {
	PositionID int64    `json:"positionId" jsonschema:"a position id of this database (save_position adds one given as text)"`
	Rollout    string   `json:"rollout,omitempty" jsonschema:"fast (default), standard, or custom settings such as 'standard,ply=1'"`
	Moves      []string `json:"moves,omitempty" jsonschema:"plays to roll out, in blunderDB notation; empty rolls the best candidates"`
}

type rolloutStoreArgs struct {
	PositionID int64    `json:"positionId" jsonschema:"a position id of this database (save_position adds one given as text)"`
	Rollout    string   `json:"rollout,omitempty" jsonschema:"fast (default), standard, or custom settings such as 'standard,ply=1'"`
	Moves      []string `json:"moves,omitempty" jsonschema:"plays to roll out, in blunderDB notation; empty rolls the best candidates"`
	Store      bool     `json:"store,omitempty" jsonschema:"write the finished rollout on the position (positionId), beside its analysis and never in its place"`
}

// registerRollout offers one tool named rollout: read-only by default, with
// the store argument when the server writes.
func registerRollout(tb *Toolbox) {
	if tb.write {
		Add(tb, Writes, &sdk.Tool{Name: "rollout", Title: "Roll a position out",
			Description: rolloutDescription + ` With store, the rollout is written on the position as a second analysis, beside the one it carries.`},
			func(ctx context.Context, req *sdk.CallToolRequest, a rolloutStoreArgs) (any, error) {
				return tb.rollout(ctx, req, rolloutArgs{a.PositionID, a.Rollout, a.Moves}, a.Store)
			})
		return
	}
	Add(tb, Reads, &sdk.Tool{Name: "rollout", Title: "Roll a position out", Description: rolloutDescription + ` Nothing is stored.`},
		func(ctx context.Context, req *sdk.CallToolRequest, a rolloutArgs) (any, error) {
			return tb.rollout(ctx, req, a, false)
		})
}

// rollout runs on a position of the library, never a bare one: the engine
// behind this server operates on a library and is no evaluator (ADR-0015).
func (tb *Toolbox) rollout(ctx context.Context, req *sdk.CallToolRequest, a rolloutArgs, store bool) (any, error) {
	if a.PositionID <= 0 {
		return nil, errors.New("give positionId: a rollout runs on a position of this database")
	}
	var out obj
	err := tb.Engine.Call(ctx, req, "rollout.position",
		obj{"positionId": a.PositionID, "rollout": a.Rollout, "moves": a.Moves, "store": store}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
