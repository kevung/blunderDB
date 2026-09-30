package cli

import (
	"flag"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// The three exits and the one entry of a draft (ADR-0045 §2), over the same
// Database methods as the panel: no rule of their own here.

type transcribeWriteArgs struct {
	dbPath, matFile       string
	matchID, draftID      int64
	finish, abandon, edit bool
	acceptLosses, text    bool
}

// transcribeWriteResult is the --format json shape of a writing option.
type transcribeWriteResult struct {
	Action string `json:"action"`
	// DraftID is the draft finished, abandoned or opened.
	DraftID int64                             `json:"draft_id"`
	Finish  *database.TranscriptionSaveResult `json:"finish,omitempty"`
	Losses  *database.TranscriptionLosses     `json:"losses,omitempty"`
}

func (cli *CLI) transcribeWrite(a transcribeWriteArgs, cmd *flag.FlagSet) error {
	n := 0
	for _, on := range []bool{a.finish, a.abandon, a.edit} {
		if on {
			n++
		}
	}
	switch {
	case n != 1:
		cmd.Usage()
		return fmt.Errorf("name one of --finish, --abandon, --edit")
	case a.matFile != "":
		cmd.Usage()
		return fmt.Errorf("--finish, --abandon and --edit act on a database, not a .mat file")
	case a.dbPath == "":
		cmd.Usage()
		return fmt.Errorf("--finish, --abandon and --edit need --db")
	case a.edit && (a.matchID == 0 || a.draftID != 0):
		cmd.Usage()
		return fmt.Errorf("--edit takes --match, not --draft")
	case !a.edit && (a.draftID == 0 || a.matchID != 0):
		cmd.Usage()
		return fmt.Errorf("--finish and --abandon take --draft, not --match")
	}
	if err := cli.initDatabase(a.dbPath); err != nil {
		return err
	}

	var out transcribeWriteResult
	switch {
	case a.finish:
		res, err := cli.db.FinishTranscription(a.draftID)
		if err != nil {
			return fmt.Errorf("finishing draft %d: %w", a.draftID, err)
		}
		out = transcribeWriteResult{Action: "finish", DraftID: a.draftID, Finish: res}
	case a.abandon:
		if err := cli.db.AbandonTranscription(a.draftID); err != nil {
			return fmt.Errorf("abandoning draft %d: %w", a.draftID, err)
		}
		out = transcribeWriteResult{Action: "abandon", DraftID: a.draftID}
	default:
		losses, err := cli.db.MatchTranscriptionLosses(a.matchID)
		if err != nil {
			return err
		}
		if losses.Lossy() && !a.acceptLosses {
			return fmt.Errorf("match %d was imported: %d analyses and %d comments are not carried by a transcription and may be lost; pass --accept-losses to open a draft on it",
				a.matchID, losses.Analyses, losses.Comments)
		}
		state, err := cli.db.EditMatchTranscription(a.matchID)
		if err != nil {
			return fmt.Errorf("editing match %d: %w", a.matchID, err)
		}
		out = transcribeWriteResult{Action: "edit", DraftID: state.ID, Losses: losses}
	}

	if !a.text {
		return printJSON(out)
	}
	switch out.Action {
	case "finish":
		verb := "created"
		if out.Finish.Replaced {
			verb = "replaced"
		}
		fmt.Printf("draft %d finished: match %d %s, %d game(s), %d move(s), %d position(s) to analyse\n",
			out.DraftID, out.Finish.MatchID, verb, out.Finish.Games, out.Finish.Moves, out.Finish.ToAnalyze)
	case "abandon":
		fmt.Printf("draft %d abandoned\n", out.DraftID)
	default:
		fmt.Printf("draft %d open on match %d\n", out.DraftID, a.matchID)
	}
	return nil
}
