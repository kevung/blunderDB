package cli

import (
	"fmt"
	"os"
)

// transcribeImportResult is what --import prints in JSON: the new draft and
// how much of it the replay marks, for a script that corrects before finishing.
type transcribeImportResult struct {
	DraftID         int64 `json:"draftId"`
	Actions         int   `json:"actions"`
	Games           int   `json:"games"`
	Inconsistencies int   `json:"inconsistencies"`
}

// transcribeImport opens a draft on a .mat or a transcription document
// through the panel's own method. A file the import refuses fails the
// command, and no draft is written.
func (cli *CLI) transcribeImport(dbPath, file string, text bool) error {
	if dbPath == "" {
		return fmt.Errorf("--import needs --db")
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading %s: %w", file, err)
	}
	if err := cli.initDatabase(dbPath); err != nil {
		return err
	}
	state, err := cli.db.ImportTranscription(string(raw))
	if err != nil {
		return fmt.Errorf("importing %s: %w", file, err)
	}
	ann := state.Annotated
	out := transcribeImportResult{DraftID: state.ID, Actions: len(ann.Document.Actions), Games: len(ann.Games)}
	for _, a := range ann.Actions {
		out.Inconsistencies += len(a.Inconsistencies)
	}
	if !text {
		return printJSON(out)
	}
	fmt.Printf("draft %d opened: %d action(s), %d game(s), %d inconsistency(ies)\n",
		out.DraftID, out.Actions, out.Games, out.Inconsistencies)
	return nil
}
