package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// runTranscribe is `blunderdb transcribe`: replay a transcription and report
// what the replay found, over pkg/blunderdb/transcript with no rule of its own
// (ADR-0045 §9). Only --finish, --abandon and --edit write, each through the
// Database method the panel calls.
//
// An Inconsistency is a finding, never a refusal (ADR-0044): --check exits 0
// whatever it lists, and a non-zero status means something really failed
// (unreadable file or database, unwritable output). Scripts read the findings
// from --format json.
func (cli *CLI) runTranscribe(args []string) error {
	transcribeCmd := flag.NewFlagSet("transcribe", flag.ContinueOnError)

	matFile := transcribeCmd.String("mat", "", "Jellyfish/gnubg .mat file to replay")
	dbPath := transcribeCmd.String("db", "", "Database holding the match or the draft to replay")
	matchID := transcribeCmd.Int64("match", 0, "Match id to replay (requires --db)")
	draftID := transcribeCmd.Int64("draft", 0, "Transcription draft id to replay (requires --db)")
	check := transcribeCmd.Bool("check", false, "List the inconsistencies the replay finds (the default)")
	render := transcribeCmd.String("render", "", "Write the transcription back as a .mat file to this path")
	format := transcribeCmd.String("format", "text", "Output format: text or json")
	finish := transcribeCmd.Bool("finish", false, "Finish the --draft: write its match (or replace the one it was opened from) and release the draft")
	abandon := transcribeCmd.Bool("abandon", false, "Abandon the --draft: delete it without a match; a match it was opened from is left untouched")
	edit := transcribeCmd.Bool("edit", false, "Open a draft on the --match (or return the one already open on it), for a correction finished with --finish")
	materialize := transcribeCmd.String("materialize", "", "JSON document of Actions (header and actions, dice included) to write as a match in one step, or nothing (requires --db)")
	importFile := transcribeCmd.String("import", "", "Open a new draft on this .mat or transcription document (JSON), to correct and finish (requires --db)")
	marks := transcribeCmd.String("marks", "", "Write the manifest of the Repères of the local video sources, one JSON line per timed move, to this .jsonl file (requires --db; --match narrows it to one match)")
	yes := transcribeCmd.Bool("yes", false, "With --abandon on a draft that never produced a match: confirm that everything typed in it is lost")
	acceptLosses := transcribeCmd.Bool("accept-losses", false, "With --edit on an imported match: accept that the analyses and comments a .mat cannot carry may be lost")

	transcribeCmd.Usage = func() {
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb transcribe [options]")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Replay a transcription and report what the replay finds.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "The source is a .mat file, a match of a library, or a")
		fmt.Fprintln(cli.stdout(), "transcription draft of one — exactly one of the three.")
		fmt.Fprintln(cli.stdout(), "A match is read through its own .mat rendering, so what")
		fmt.Fprintln(cli.stdout(), "is replayed is what an export of it would contain.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "--check lists the inconsistencies: an illegal play, two")
		fmt.Fprintln(cli.stdout(), "turns in a row for the same player, an impossible cube")
		fmt.Fprintln(cli.stdout(), "action, an action past the end of the match, a play that")
		fmt.Fprintln(cli.stdout(), "does not use its own roll, a play the record does not")
		fmt.Fprintln(cli.stdout(), "carry (gnubg writes it \"???\"), a game score the games")
		fmt.Fprintln(cli.stdout(), "before it do not give. Each is named with the")
		fmt.Fprintln(cli.stdout(), "action number and the game it belongs to.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "An inconsistency is REPORTED, never held against the")
		fmt.Fprintln(cli.stdout(), "input: nothing is refused for one, and the exit status is")
		fmt.Fprintln(cli.stdout(), "0 whatever the replay finds. A non-zero status means the")
		fmt.Fprintln(cli.stdout(), "file, the database or the output failed.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "--render writes the transcription back as a .mat file,")
		fmt.Fprintln(cli.stdout(), "which is how the round trip is checked on real files.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "--import opens a new draft on a .mat or a transcription")
		fmt.Fprintln(cli.stdout(), "document (JSON): its inconsistencies are kept and marked,")
		fmt.Fprintln(cli.stdout(), "to be corrected before --finish writes the match.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Three options write, and only these: --finish writes a")
		fmt.Fprintln(cli.stdout(), "draft's match and releases the draft, --abandon deletes a")
		fmt.Fprintln(cli.stdout(), "draft without a match, --edit opens a draft on an existing")
		fmt.Fprintln(cli.stdout(), "match so that --finish replaces it in place. Editing an")
		fmt.Fprintln(cli.stdout(), "imported match counts what a .mat cannot carry (analyses,")
		fmt.Fprintln(cli.stdout(), "comments) and refuses without --accept-losses. Abandoning a")
		fmt.Fprintln(cli.stdout(), "draft that never produced a match loses everything typed")
		fmt.Fprintln(cli.stdout(), "in it, and asks for --yes.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "--marks writes a manifest, one JSON line per move that")
		fmt.Fprintln(cli.stdout(), "carries a Repère (video instants), for the matches whose")
		fmt.Fprintln(cli.stdout(), "video is a local file: path, match, game, move, dice, play,")
		fmt.Fprintln(cli.stdout(), "board before and after, roll_tick_ms and tick_ms. A video")
		fmt.Fprintln(cli.stdout(), "that is an http(s) URL is left out and counted in the")
		fmt.Fprintln(cli.stdout(), "summary; a local file missing from the disk is named on")
		fmt.Fprintln(cli.stdout(), "the error output and left out. No image is read or written,")
		fmt.Fprintln(cli.stdout(), "and nothing is written to the database.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		transcribeCmd.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  # List what a .mat file's replay finds")
		fmt.Fprintln(cli.stdout(), "  blunderdb transcribe --mat match.mat --check")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "  # Same, machine-readable")
		fmt.Fprintln(cli.stdout(), "  blunderdb transcribe --mat match.mat --check --format json")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "  # Round trip: render it back and compare")
		fmt.Fprintln(cli.stdout(), "  blunderdb transcribe --mat match.mat --render out.mat")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "  # Replay a match of the library, or a draft being typed")
		fmt.Fprintln(cli.stdout(), "  blunderdb transcribe --db database.db --match 5 --check")
		fmt.Fprintln(cli.stdout(), "  blunderdb transcribe --db database.db --draft 3 --check")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "  # Correct a match: open a draft on it, then finish it")
		fmt.Fprintln(cli.stdout(), "  blunderdb transcribe --db database.db --match 5 --edit")
		fmt.Fprintln(cli.stdout(), "  blunderdb transcribe --db database.db --draft 4 --finish")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "  # Write a match played elsewhere from its Actions; the first illegal Action refuses all")
		fmt.Fprintln(cli.stdout(), "  blunderdb transcribe --db database.db --materialize match.json")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "  # Open a draft on a .mat (or a transcription document), then finish it")
		fmt.Fprintln(cli.stdout(), "  blunderdb transcribe --db database.db --import match.mat")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "  # Export the manifest of Repères of the local videos")
		fmt.Fprintln(cli.stdout(), "  blunderdb transcribe --db database.db --marks marks.jsonl")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "  # Drop a draft")
		fmt.Fprintln(cli.stdout(), "  blunderdb transcribe --db database.db --draft 4 --abandon")
	}

	if err := transcribeCmd.Parse(args); err != nil {
		return err
	}

	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}
	text := formatLower != "json"

	if *marks != "" {
		if *materialize != "" || *importFile != "" || *finish || *abandon || *edit || *check || *render != "" || *matFile != "" || *draftID != 0 {
			transcribeCmd.Usage()
			return fmt.Errorf("--marks takes --db, and --match to narrow it, and nothing else to act on")
		}
		return cli.transcribeMarks(*dbPath, *marks, *matchID, text)
	}
	if *importFile != "" {
		if *materialize != "" || *finish || *abandon || *edit || *check || *render != "" || *matFile != "" || *matchID != 0 || *draftID != 0 {
			transcribeCmd.Usage()
			return fmt.Errorf("--import takes --db and nothing else to act on")
		}
		return cli.transcribeImport(*dbPath, *importFile, text)
	}
	if *materialize != "" {
		if *finish || *abandon || *edit || *matFile != "" || *matchID != 0 || *draftID != 0 {
			transcribeCmd.Usage()
			return fmt.Errorf("--materialize takes --db and nothing else to act on")
		}
		return cli.transcribeMaterialize(*dbPath, *materialize, text)
	}
	if *finish || *abandon || *edit {
		return cli.transcribeWrite(transcribeWriteArgs{
			dbPath: *dbPath, matchID: *matchID, draftID: *draftID, matFile: *matFile,
			finish: *finish, abandon: *abandon, edit: *edit, acceptLosses: *acceptLosses, yes: *yes, text: text,
		}, transcribeCmd)
	}

	doc, source, err := cli.transcribeSource(*matFile, *dbPath, *matchID, *draftID, transcribeCmd)
	if err != nil {
		return err
	}

	// One Replay answers both flags: the Inconsistencies --check lists and the
	// games --render needs are derived in the same pass.
	annotated := transcript.Replay(doc, 0)
	result := transcribeResult{
		Source:      source,
		MatchLength: doc.Header.MatchLength,
		Games:       len(annotated.Games),
		Actions:     len(annotated.Actions),
		Finished:    annotated.Finished,
		Score:       annotated.Score,
	}
	for _, info := range annotated.Actions {
		for _, bad := range info.Inconsistencies {
			result.Inconsistencies = append(result.Inconsistencies, transcribeInconsistency{
				Action: info.Index,
				Game:   info.GameNumber,
				Player: info.Side + 1,
				Kind:   string(info.Kind),
				Type:   string(bad.Kind),
				Detail: bad.Detail,
			})
		}
	}

	if *render != "" {
		m, games, moves := transcript.MatchParts(doc)
		if err := os.WriteFile(*render, []byte(ingest.RenderMAT(m, games, moves)), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", *render, err)
		}
		result.Rendered = *render
	}

	if !text {
		return cli.printJSON(result)
	}
	cli.printTranscribeResult(result, *check || *render == "")
	return nil
}

// transcribeSource resolves the command's one source into a Document: a .mat
// straight off the disk, or the Database methods the GUI calls (parity).
func (cli *CLI) transcribeSource(matFile, dbPath string, matchID, draftID int64, cmd *flag.FlagSet) (transcript.Document, string, error) {
	var none transcript.Document

	named := 0
	if matFile != "" {
		named++
	}
	if matchID != 0 {
		named++
	}
	if draftID != 0 {
		named++
	}
	if named != 1 {
		cmd.Usage()
		return none, "", fmt.Errorf("name exactly one source: --mat, --match or --draft")
	}
	if matFile == "" && dbPath == "" {
		cmd.Usage()
		return none, "", fmt.Errorf("--match and --draft need --db")
	}

	if matFile != "" {
		raw, err := os.ReadFile(matFile)
		if err != nil {
			return none, "", fmt.Errorf("reading %s: %w", matFile, err)
		}
		doc, err := transcript.FromMAT(string(raw))
		if err != nil {
			return none, "", fmt.Errorf("replaying %s: %w", matFile, err)
		}
		return doc, matFile, nil
	}

	if err := cli.initDatabase(dbPath); err != nil {
		return none, "", err
	}

	if draftID != 0 {
		state, err := cli.db.OpenTranscription(draftID)
		if err != nil {
			return none, "", fmt.Errorf("opening draft %d: %w", draftID, err)
		}
		return state.Annotated.Document, fmt.Sprintf("draft %d", draftID), nil
	}

	// A saved match has no Document of its own, so it is replayed through the
	// .mat it would export; plays travel as written, so an illegal one is
	// still found.
	mat, err := cli.db.MatchMAT(matchID)
	if err != nil {
		return none, "", fmt.Errorf("reading match %d: %w", matchID, err)
	}
	doc, err := transcript.FromMAT(mat)
	if err != nil {
		return none, "", fmt.Errorf("replaying match %d: %w", matchID, err)
	}
	return doc, fmt.Sprintf("match %d", matchID), nil
}

// transcribeResult is the --format json shape for `transcribe`.
type transcribeResult struct {
	Source string `json:"source"`
	// MatchLength is 0 for a money session, as everywhere else.
	MatchLength     int                       `json:"match_length"`
	Games           int                       `json:"games"`
	Actions         int                       `json:"actions"`
	Finished        bool                      `json:"finished"`
	Score           [2]int                    `json:"score"`
	Inconsistencies []transcribeInconsistency `json:"inconsistencies"`
	Rendered        string                    `json:"rendered,omitempty"`
}

// transcribeInconsistency is one finding, placed: which Action of the
// document, which game of the match, and which player acted.
type transcribeInconsistency struct {
	Action int `json:"action"`
	Game   int `json:"game"`
	Player int `json:"player"`
	// Kind is the Action's own kind (checker, double, take…), Type the
	// inconsistency's (illegal_move, double_turn…).
	Kind   string `json:"kind"`
	Type   string `json:"type"`
	Detail string `json:"detail"`
}

func (cli *CLI) printTranscribeResult(r transcribeResult, listing bool) {
	if r.MatchLength == 0 {
		fmt.Fprintf(cli.stdout(), "%s: money session, %d game(s), %d action(s)\n", r.Source, r.Games, r.Actions)
	} else {
		fmt.Fprintf(cli.stdout(), "%s: %d point match, %d game(s), %d action(s)\n", r.Source, r.MatchLength, r.Games, r.Actions)
	}
	if r.Finished {
		fmt.Fprintf(cli.stdout(), "  Final score: %d-%d\n", r.Score[0], r.Score[1])
	} else {
		fmt.Fprintf(cli.stdout(), "  Score so far: %d-%d (the match is unfinished)\n", r.Score[0], r.Score[1])
	}
	if r.Rendered != "" {
		fmt.Fprintf(cli.stdout(), "  Rendered to %s\n", r.Rendered)
	}
	if !listing {
		return
	}
	if len(r.Inconsistencies) == 0 {
		fmt.Fprintln(cli.stdout(), "Inconsistencies: none")
		return
	}
	fmt.Fprintf(cli.stdout(), "Inconsistencies (%d):\n", len(r.Inconsistencies))
	for _, bad := range r.Inconsistencies {
		fmt.Fprintf(cli.stdout(), "  action %d, game %d, player %d (%s): %s — %s\n",
			bad.Action, bad.Game, bad.Player, bad.Kind, bad.Type, bad.Detail)
	}
	fmt.Fprintln(cli.stdout(), "Reported, not refused: a transcription is kept as it was written down,")
	fmt.Fprintln(cli.stdout(), "so this is a finding and the exit status is 0.")
}

// transcribeMarks writes the manifest of Repères: the lines come from
// Database.TranscriptionMarks, this only spells them as JSON lines and the
// summary beside them.
func (cli *CLI) transcribeMarks(dbPath, out string, matchID int64, text bool) error {
	if dbPath == "" {
		return fmt.Errorf("--marks needs --db")
	}
	if err := cli.initDatabase(dbPath); err != nil {
		return err
	}
	marks, sum, err := cli.db.TranscriptionMarks(matchID)
	if err != nil {
		return fmt.Errorf("reading the Repères: %w", err)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, m := range marks {
		if err := enc.Encode(m); err != nil {
			return err
		}
	}
	if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}

	for _, path := range sum.Missing {
		fmt.Fprintf(os.Stderr, "video not found, match left out: %s\n", path)
	}
	if !text {
		return cli.printJSON(struct {
			Written string `json:"written"`
			database.TranscriptionMarksSummary
		}{out, sum})
	}
	fmt.Fprintf(cli.stdout(), "%d line(s) from %d match(es) written to %s\n", sum.Marks, sum.Matches, out)
	fmt.Fprintf(cli.stdout(), "  %d match(es) left out: their video is a URL\n", sum.URLSources)
	if len(sum.Missing) > 0 {
		fmt.Fprintf(cli.stdout(), "  %d match(es) left out: their video file is not on this disk\n", len(sum.Missing))
	}
	return nil
}
