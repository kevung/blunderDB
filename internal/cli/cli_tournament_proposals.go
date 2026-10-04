package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/PileOfCells/backgammon-tournoi/render"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// The engine's queue, read and confirmed from a script: the same proposals the panel shows, and
// the same confirmation it records (Database.ConfirmProposal). A confirmation names a proposal
// by its number in the queue as printed just before; the queue is deterministic, so the number
// designates the same proposal as long as nothing was recorded in between.

// proposalRow is one proposal as printed: its number, the engine's action and its rendering.
type proposalRow struct {
	N      int            `json:"n"`
	Kind   string         `json:"kind"`
	Label  string         `json:"label,omitempty"`
	A      string         `json:"a,omitempty"`
	B      string         `json:"b,omitempty"`
	Table  int            `json:"table,omitempty"`
	Action tournoi.Action `json:"action"`
}

func proposalRows(view *database.DirectionView) []proposalRow {
	names := map[tournoi.PlayerID]string{}
	for _, p := range view.Players {
		names[p.ID] = p.Name
	}
	name := func(id tournoi.PlayerID) string {
		if n := names[id]; n != "" {
			return n
		}
		return string(id)
	}
	l := render.French()
	rows := []proposalRow{}
	for _, a := range view.Proposals {
		if a.Kind == tournoi.ActWait {
			continue
		}
		r := proposalRow{N: len(rows) + 1, Kind: string(a.Kind), Label: l.Label(a.Label), Table: a.Table, Action: a}
		if a.A != "" {
			r.A = name(a.A)
		}
		if a.B != "" {
			r.B = name(a.B)
		}
		rows = append(rows, r)
	}
	return rows
}

func printProposalRows(rows []proposalRow, format string) error {
	if strings.ToLower(format) == "json" {
		return printJSON(rows)
	}
	for _, r := range rows {
		line := fmt.Sprintf("%d\t%s\t%s", r.N, r.Kind, r.Label)
		switch {
		case r.Kind == string(tournoi.ActRepechage):
			// A is the withdrawn qualifier, B the player who would take the place.
			line += fmt.Sprintf("\t%s replaces %s", r.B, r.A)
		case r.A != "" && r.B != "":
			line += fmt.Sprintf("\t%s - %s", r.A, r.B)
		case r.A != "":
			line += "\t" + r.A
		}
		if r.Table > 0 {
			line += fmt.Sprintf("\ttable %d", r.Table)
		}
		fmt.Println(line)
	}
	return nil
}

func (cli *CLI) runTournamentProposals(args []string) error {
	fs, dbPath := tournamentFlagSet("proposals", "Print the engine's proposals for a tournament, numbered for `tournament confirm`.",
		"blunderdb tournament proposals --db base.db --id 3",
		"blunderdb tournament proposals --db base.db --id 3 --format json")
	id := fs.Int64("id", 0, "Tournament ID (required)")
	format := fs.String("format", "text", "Output format: text or json")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *id == 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag: --id")
	}
	view, err := cli.db.GetDirection(*id)
	if err != nil {
		return err
	}
	return printProposalRows(proposalRows(view), *format)
}

func (cli *CLI) runTournamentConfirm(args []string) error {
	fs, dbPath := tournamentFlagSet("confirm", "Confirm one proposal, by its number in `tournament proposals`, and print the queue that follows.",
		"blunderdb tournament confirm --db base.db --id 3 --n 1")
	id := fs.Int64("id", 0, "Tournament ID (required)")
	n := fs.Int("n", 0, "Number of the proposal, as printed by the proposals sub-command (required)")
	format := fs.String("format", "text", "Output format: text or json")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *id == 0 || *n <= 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag: --id and --n are both required")
	}
	view, err := cli.db.GetDirection(*id)
	if err != nil {
		return err
	}
	rows := proposalRows(view)
	if *n > len(rows) {
		return fmt.Errorf("no proposal %d: the queue holds %d", *n, len(rows))
	}
	blob, err := json.Marshal(rows[*n-1].Action)
	if err != nil {
		return err
	}
	if view, err = cli.db.ConfirmProposal(*id, string(blob)); err != nil {
		return err
	}
	return printProposalRows(proposalRows(view), *format)
}
