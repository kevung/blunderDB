package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// runTranscribe is `blunderdb transcribe`: replay a transcription and report
// what the replay found — the CLI's window onto pkg/blunderdb/transcript, the
// pure package the panel and the Wails bindings already use (ADR-0045 §9).
//
// It adds no rule of its own. --check is transcript.Replay's Inconsistencies
// listed with the Action and the game they sit on; --render is
// ingest.RenderMAT over transcript.MatchParts, the same renderer the export
// button calls. Nothing is written to a database, ever: the command reads.
//
// # Why an inconsistency is not an error status
//
// An Inconsistency is a fact about what was written down, not a complaint
// about the input: an illegal play in a .mat is what the players did (or what
// the file's author typed), and ADR-0044 is explicit that nothing is ever
// refused for one — not the save, not the export, and not this. So --check
// REPORTS and exits 0 even when it lists a hundred of them, and a non-zero
// status is kept for what really failed: an unreadable file, a .mat the
// parser cannot make a match of, a database that will not open, an output
// that cannot be written. A script that wants to act on the findings reads
// them, from --format json, rather than reading a status that would conflate
// "this file is broken" with "this game had an illegal move".
func (cli *CLI) runTranscribe(args []string) error {
	transcribeCmd := flag.NewFlagSet("transcribe", flag.ContinueOnError)

	matFile := transcribeCmd.String("mat", "", "Jellyfish/gnubg .mat file to replay")
	dbPath := transcribeCmd.String("db", "", "Database holding the match or the draft to replay")
	matchID := transcribeCmd.Int64("match", 0, "Match id to replay (requires --db)")
	draftID := transcribeCmd.Int64("draft", 0, "Transcription draft id to replay (requires --db)")
	check := transcribeCmd.Bool("check", false, "List the inconsistencies the replay finds (the default)")
	render := transcribeCmd.String("render", "", "Write the transcription back as a .mat file to this path")
	format := transcribeCmd.String("format", "text", "Output format: text or json")

	transcribeCmd.Usage = func() {
		fmt.Println("Usage: blunderdb transcribe [options]")
		fmt.Println()
		fmt.Println("Replay a transcription and report what the replay finds.")
		fmt.Println()
		fmt.Println("The source is a .mat file, a match of a library, or a")
		fmt.Println("transcription draft of one — exactly one of the three.")
		fmt.Println("A match is read through its own .mat rendering, so what")
		fmt.Println("is replayed is what an export of it would contain.")
		fmt.Println()
		fmt.Println("--check lists the inconsistencies: an illegal play, two")
		fmt.Println("turns in a row for the same player, an impossible cube")
		fmt.Println("action, an action past the end of the match, a play that")
		fmt.Println("does not use its own roll. Each is named with the action")
		fmt.Println("number and the game it belongs to.")
		fmt.Println()
		fmt.Println("An inconsistency is REPORTED, never held against the")
		fmt.Println("input: nothing is refused for one, and the exit status is")
		fmt.Println("0 whatever the replay finds. A non-zero status means the")
		fmt.Println("file, the database or the output failed.")
		fmt.Println()
		fmt.Println("--render writes the transcription back as a .mat file,")
		fmt.Println("which is how the round trip is checked on real files.")
		fmt.Println()
		fmt.Println("Options:")
		transcribeCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # List what a .mat file's replay finds")
		fmt.Println("  blunderdb transcribe --mat match.mat --check")
		fmt.Println()
		fmt.Println("  # Same, machine-readable")
		fmt.Println("  blunderdb transcribe --mat match.mat --check --format json")
		fmt.Println()
		fmt.Println("  # Round trip: render it back and compare")
		fmt.Println("  blunderdb transcribe --mat match.mat --render out.mat")
		fmt.Println()
		fmt.Println("  # Replay a match of the library, or a draft being typed")
		fmt.Println("  blunderdb transcribe --db database.db --match 5 --check")
		fmt.Println("  blunderdb transcribe --db database.db --draft 3 --check")
	}

	if err := transcribeCmd.Parse(args); err != nil {
		return err
	}

	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}
	text := formatLower != "json"

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
		return printJSON(result)
	}
	printTranscribeResult(result, *check || *render == "")
	return nil
}

// transcribeSource resolves the one source the command was given into a
// Document. A .mat is read straight off the disk — transcript.FromMAT needs no
// database, exactly as the GnuBG importer's parser does not — and both
// database sources are the Database methods the GUI already calls, so the
// three modes never grow a rule of their own (CLI/GUI/server parity).
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

	// A saved match has no Document of its own — a Match is what a
	// transcription PRODUCES — so it is replayed through the .mat it would
	// export, which is the round trip transcript.FromMAT documents. The plays
	// travel as they were written, so an illegal one is still found.
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

func printTranscribeResult(r transcribeResult, listing bool) {
	if r.MatchLength == 0 {
		fmt.Printf("%s: money session, %d game(s), %d action(s)\n", r.Source, r.Games, r.Actions)
	} else {
		fmt.Printf("%s: %d point match, %d game(s), %d action(s)\n", r.Source, r.MatchLength, r.Games, r.Actions)
	}
	if r.Finished {
		fmt.Printf("  Final score: %d-%d\n", r.Score[0], r.Score[1])
	} else {
		fmt.Printf("  Score so far: %d-%d (the match is unfinished)\n", r.Score[0], r.Score[1])
	}
	if r.Rendered != "" {
		fmt.Printf("  Rendered to %s\n", r.Rendered)
	}
	if !listing {
		return
	}
	if len(r.Inconsistencies) == 0 {
		fmt.Println("Inconsistencies: none")
		return
	}
	fmt.Printf("Inconsistencies (%d):\n", len(r.Inconsistencies))
	for _, bad := range r.Inconsistencies {
		fmt.Printf("  action %d, game %d, player %d (%s): %s — %s\n",
			bad.Action, bad.Game, bad.Player, bad.Kind, bad.Type, bad.Detail)
	}
	fmt.Println("Reported, not refused: a transcription is kept as it was written down,")
	fmt.Println("so this is a finding and the exit status is 0.")
}
