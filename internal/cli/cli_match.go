package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/duel"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// runMatch handles the match command
func (cli *CLI) runMatch(args []string) error {
	matchCmd := flag.NewFlagSet("match", flag.ContinueOnError)

	// Define flags
	dbPath := matchCmd.String("db", "", "Path to the database file (required)")
	matchID := matchCmd.Int64("id", 0, "Match ID (required)")
	format := matchCmd.String("format", "json", "Output format: json, text, summary")
	output := matchCmd.String("output", "", "Output file (default: stdout)")

	matchCmd.Usage = func() {
		fmt.Println("Usage: blunderdb match [options]")
		fmt.Println()
		fmt.Println("Display match positions and analysis.")
		fmt.Println()
		fmt.Println("Options:")
		matchCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Display match positions in JSON format")
		fmt.Println("  blunderdb match --db database.db --id 1 --format json")
		fmt.Println()
		fmt.Println("  # Display match summary")
		fmt.Println("  blunderdb match --db database.db --id 1 --format summary")
		fmt.Println()
		fmt.Println("  # Save match positions to file")
		fmt.Println("  blunderdb match --db database.db --id 1 --output match.json")
	}

	if err := matchCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		matchCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	if *matchID == 0 {
		matchCmd.Usage()
		return fmt.Errorf("missing required flag: --id")
	}

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Get match info
	match, err := cli.db.GetMatchByID(*matchID)
	if err != nil {
		return fmt.Errorf("failed to get match: %w", err)
	}

	// Get match positions
	positions, err := cli.db.GetMatchMovePositions(*matchID)
	if err != nil {
		return fmt.Errorf("failed to get match positions: %w", err)
	}

	origin, err := cli.db.GetMatchOrigin(*matchID)
	if err != nil {
		return fmt.Errorf("failed to get match origin: %w", err)
	}

	// Format output based on requested format
	var outputData string
	switch strings.ToLower(*format) {
	case "json":
		outputData, err = cli.formatMatchJSON(match, positions, origin)
	case "text":
		outputData, err = cli.formatMatchText(match, positions, origin)
	case "summary":
		outputData, err = cli.formatMatchSummary(match, positions, origin)
	default:
		return fmt.Errorf("unknown format: %s (must be 'json', 'text', or 'summary')", *format)
	}

	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	// Output results
	if *output != "" {
		err := os.WriteFile(*output, []byte(outputData), 0644)
		if err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Match data written to: %s\n", *output)
	} else {
		fmt.Println(outputData)
	}

	return nil
}

// formatMatchJSON formats match data as JSON
// origin is null for a match not played here.
func (cli *CLI) formatMatchJSON(match *Match, positions []MatchMovePosition, origin *duel.Origin) (string, error) {
	output := map[string]interface{}{
		"match":          match,
		"origin":         origin,
		"positions":      positions,
		"position_count": len(positions),
	}
	// Per Move, in match order; mwc_loss and difficulty are fractions, null
	// when unscored. The summary adds them up per player (ADR-0076).
	if losses := cli.decisionLosses(match.ID); losses != nil {
		output["decision_losses"] = losses
		output["difficulty_summary"] = storage.SummariseDifficulty(losses)
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// writeSourceMetadata writes the lines of what the source file said of the
// match, each only when the file said it.
func writeSourceMetadata(sb *strings.Builder, m *Match) {
	for i, elo := range [2]*float64{m.Player1Elo, m.Player2Elo} {
		exp := [2]*int{m.Player1Experience, m.Player2Experience}[i]
		name := [2]string{m.Player1Name, m.Player2Name}[i]
		switch {
		case elo != nil && exp != nil:
			fmt.Fprintf(sb, "Rating %s: %.0f (experience %d)\n", name, *elo, *exp)
		case elo != nil:
			fmt.Fprintf(sb, "Rating %s: %.0f\n", name, *elo)
		}
	}
	if m.Transcriber != "" {
		fmt.Fprintf(sb, "Transcriber: %s\n", m.Transcriber)
	}
	if m.MatchLength == 0 && (m.HasJacoby != nil || m.HasBeaver != nil) {
		fmt.Fprintf(sb, "Jacoby: %s, Beaver: %s\n", yesNo(m.HasJacoby), yesNo(m.HasBeaver))
	}
	if m.EngineVersion != "" {
		fmt.Fprintf(sb, "Written by: %s\n", m.EngineVersion)
	}
}

// writeOrigin writes how the match came to be when it was played here: the
// revealed seed with its fingerprint, so the rolls can be recomputed without
// trusting the Arbiter (ADR-0072 rule 8), and the rest of its origin.
func writeOrigin(sb *strings.Builder, m *Match, o *duel.Origin) {
	if o == nil {
		sb.WriteString("Origin: not played here\n")
		return
	}
	sb.WriteString("Origin: played here\n")
	if o.Start == "" {
		sb.WriteString("  Start: opening position\n")
	} else {
		fmt.Fprintf(sb, "  Start: %s\n", o.Start)
	}
	fmt.Fprintf(sb, "  Dice seed: %s\n", o.DiceSeed)
	fmt.Fprintf(sb, "  SHA-256 of the seed: %s (compare with the fingerprint published when the Duel was created)\n", o.Fingerprint)
	var overTime string
	if o.OverTime == 1 || o.OverTime == 2 {
		overTime = [2]string{m.Player1Name, m.Player2Name}[o.OverTime-1]
	}
	switch {
	case o.LostOnTime:
		fmt.Fprintf(sb, "  Ended: lost on time (%s)\n", overTime)
	case o.StoppedEarly:
		sb.WriteString("  Ended: stopped before the end\n")
	default:
		sb.WriteString("  Ended: played to the end\n")
	}
	if c := o.CadenceSettings; c != nil {
		fmt.Fprintf(sb, "  Cadence: %s\n", describeCadence(*c))
	}
	if overTime != "" && !o.LostOnTime {
		fmt.Fprintf(sb, "  Reserve ran out first: %s\n", overTime)
	}
	if o.BotLevel != "" {
		fmt.Fprintf(sb, "  Bot: level %s", o.BotLevel)
		if o.BotEngine != "" {
			fmt.Fprintf(sb, ", policy of gammonNet %s", o.BotEngine)
		}
		sb.WriteString("\n")
	}
	if o.RollSeed != "" {
		fmt.Fprintf(sb, "  Combined seed: contributions %q, %q; rolls from %s\n", o.Contributions[0], o.Contributions[1], o.RollSeed)
	}
	for _, b := range o.DeclaredBots {
		fmt.Fprintf(sb, "  Bot declared by player %d (%s): configuration %s, gammonNet %s — not attested\n",
			b.Player, [2]string{m.Player1Name, m.Player2Name}[b.Player-1], b.Configuration, b.Engine)
	}
}

// describeCadence renders a Cadence as its name, then its reserve and delay.
func describeCadence(c duel.Cadence) string {
	var parts []string
	if c.Name != "" {
		parts = append(parts, c.Name)
	}
	if c.ReservePerPoint > 0 {
		parts = append(parts, fmt.Sprintf("reserve %d s per point", c.ReservePerPoint))
	} else {
		parts = append(parts, fmt.Sprintf("reserve %d s", c.Reserve))
	}
	parts = append(parts, fmt.Sprintf("delay %d s", c.Delay))
	if c.TimeOut != "" {
		parts = append(parts, fmt.Sprintf("time out: %s", c.TimeOut))
	}
	return strings.Join(parts, ", ")
}

func yesNo(b *bool) string {
	switch {
	case b == nil:
		return "unknown"
	case *b:
		return "yes"
	default:
		return "no"
	}
}

// formatMatchText formats match data as text
func (cli *CLI) formatMatchText(match *Match, positions []MatchMovePosition, origin *duel.Origin) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Match ID: %d\n", match.ID))
	sb.WriteString(fmt.Sprintf("Players: %s vs %s\n", match.Player1Name, match.Player2Name))
	if match.Event != "" {
		sb.WriteString(fmt.Sprintf("Event: %s\n", match.Event))
	}
	if match.Location != "" {
		sb.WriteString(fmt.Sprintf("Location: %s\n", match.Location))
	}
	sb.WriteString(fmt.Sprintf("Match Length: %d\n", match.MatchLength))
	writeSourceMetadata(&sb, match)
	writeOrigin(&sb, match, origin)
	sb.WriteString(fmt.Sprintf("Total Positions: %d\n\n", len(positions)))

	lossByMove := map[int64]storage.DecisionLoss{}
	for _, d := range cli.decisionLosses(match.ID) {
		if d.MWCLoss != nil {
			lossByMove[d.MoveID] = d
		}
	}
	for i, movePos := range positions {
		sb.WriteString(fmt.Sprintf("Position %d:\n", i+1))
		sb.WriteString(fmt.Sprintf("  Game: %d, Move: %d\n", movePos.GameNumber, movePos.MoveNumber))

		// Handle XG player encoding: -1 = Player1 (X), 1 = Player2 (O)
		var playerName string
		if movePos.PlayerOnRoll == -1 {
			playerName = match.Player1Name
		} else if movePos.PlayerOnRoll == 1 {
			playerName = match.Player2Name
		} else {
			playerName = "Unknown"
		}
		sb.WriteString(fmt.Sprintf("  Player on roll: %d (%s)\n", movePos.PlayerOnRoll, playerName))
		sb.WriteString(fmt.Sprintf("  Score: %d-%d\n", movePos.Position.Score[0], movePos.Position.Score[1]))
		sb.WriteString(fmt.Sprintf("  Cube: %d (owner: %d)\n", movePos.Position.Cube.Value, movePos.Position.Cube.Owner))
		if movePos.Position.Dice[0] != 0 {
			sb.WriteString(fmt.Sprintf("  Dice: %d-%d\n", movePos.Position.Dice[0], movePos.Position.Dice[1]))
		}
		// An unknown time is left out, never printed as zero.
		if movePos.DecisionMS != nil {
			sb.WriteString(fmt.Sprintf("  Decision time: %.1f s\n", float64(*movePos.DecisionMS)/1000))
		}
		if movePos.CubeDecisionMS != nil {
			sb.WriteString(fmt.Sprintf("  Cube decision time: %.1f s\n", float64(*movePos.CubeDecisionMS)/1000))
		}
		// An unscored decision is left out, never printed as zero.
		if d, ok := lossByMove[movePos.MoveID]; ok {
			sb.WriteString(fmt.Sprintf("  MWC loss: %.2f%%\n", *d.MWCLoss*100))
			if d.Difficulty != nil {
				sb.WriteString(fmt.Sprintf("  Difficulty: %.2f%%\n", *d.Difficulty*100))
			}
			if d.Avoidable {
				sb.WriteString("  Avoidable error\n")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// decisionLosses is the per-Move MWC loss of a match, nil when it cannot be read.
func (cli *CLI) decisionLosses(matchID int64) []storage.DecisionLoss {
	if cli.db == nil {
		return nil
	}
	losses, err := cli.db.GetMatchDecisionLosses(matchID)
	if err != nil {
		return nil
	}
	return losses
}

// formatMatchSummary formats match data as a summary
func (cli *CLI) formatMatchSummary(match *Match, positions []MatchMovePosition, origin *duel.Origin) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Match: %s vs %s\n", match.Player1Name, match.Player2Name))
	if match.Event != "" {
		sb.WriteString(fmt.Sprintf("Event: %s\n", match.Event))
	}
	sb.WriteString(fmt.Sprintf("Match Length: %d points\n", match.MatchLength))
	sb.WriteString(fmt.Sprintf("Games: %d\n", match.GameCount))
	writeSourceMetadata(&sb, match)
	writeOrigin(&sb, match, origin)
	sb.WriteString(fmt.Sprintf("Total Positions: %d\n\n", len(positions)))

	// Count positions by game
	gamePositions := make(map[int32]int)
	for _, pos := range positions {
		gamePositions[pos.GameNumber]++
	}

	sb.WriteString("Positions per game:\n")
	for gameNum := int32(1); gameNum <= int32(match.GameCount); gameNum++ {
		count := gamePositions[gameNum]
		sb.WriteString(fmt.Sprintf("  Game %d: %d positions\n", gameNum, count))
	}
	cli.writeLossSummary(&sb, match)
	cli.writeTimeSummary(&sb, match)
	if cli.db == nil {
		return sb.String(), nil
	}
	if detail, err := cli.db.GetMatchDetailStats(match.ID); err == nil && detail != nil {
		sb.WriteString("\nMWC loss, 7-point scale:\n")
		sb.WriteString(fmt.Sprintf("  %s: %s\n", match.Player1Name, formatMWC7(detail.Player1.MWC7)))
		sb.WriteString(fmt.Sprintf("  %s: %s\n", match.Player2Name, formatMWC7(detail.Player2.MWC7)))
	}
	if review, err := cli.db.GetMatchReview(match.ID); err == nil {
		writeMatchReview(&sb, [2]string{match.Player1Name, match.Player2Name}, review)
	}

	return sb.String(), nil
}

// writeMatchReview adds the match's study summary (ADR-0077), per player: PR
// with its interval over the games, the luck-adjusted result, the errors to
// revisit, and the hasty/deliberate split of the errors.
func writeMatchReview(sb *strings.Builder, names [2]string, review storage.MatchReview) {
	sb.WriteString("\nReview:\n")
	for i, name := range names {
		p := review.Players[i]
		fmt.Fprintf(sb, "  %s: PR %.2f %s over %d decisions\n", name, p.PR, formatPRInterval(p.PRInterval), p.Decisions)
		if l := p.Luck; l.Available {
			fmt.Fprintf(sb, "    luck-adjusted result %+.1f%% (result %+.1f%%, net luck %+.1f%%, errors balance %+.1f%%; luck on %d of %d rolls)\n",
				100*l.Adjusted, 100*l.Result, 100*l.Luck, 100*l.ErrorBalance, l.RollsMeasured, l.Rolls)
		}
		for _, d := range p.ToReview {
			fmt.Fprintf(sb, "    to review: game %d move %d (%s), loss %.2f%%, avoidable %.2f%%\n",
				d.GameNumber, d.MoveNumber, d.DecisionType, 100*d.MWCLoss, 100*d.AvoidableLoss)
		}
		if pc := p.Pace; pc.Hasty+pc.Deliberate+pc.Unknown > 0 {
			fmt.Fprintf(sb, "    errors: %d hasty (%.2f%%), %d deliberate (%.2f%%), %d without time\n",
				pc.Hasty, 100*pc.HastyLoss, pc.Deliberate, 100*pc.DeliberateLoss, pc.Unknown)
		}
	}
}

// writeLossSummary adds, per player, the winning chances the match's analysed
// decisions cost (the sum of `match --format text`'s per-decision lines):
// nothing at all when no decision is scored.
func (cli *CLI) writeLossSummary(sb *strings.Builder, match *Match) {
	var total [2]float64
	var n [2]int
	losses := cli.decisionLosses(match.ID)
	for _, d := range losses {
		if d.MWCLoss != nil {
			total[d.Player] += *d.MWCLoss
			n[d.Player]++
		}
	}
	if n[0]+n[1] == 0 {
		return
	}
	sb.WriteString("\nMWC loss:\n")
	for i, name := range [2]string{match.Player1Name, match.Player2Name} {
		fmt.Fprintf(sb, "  %s: %.2f%% over %d decisions\n", name, total[i]*100, n[i])
	}
	writeDifficultySummary(sb, [2]string{match.Player1Name, match.Player2Name}, losses)
}

// writeDifficultySummary adds, per player, the reference player's expected
// loss over the same decisions, the excess over it, the ratio to it and the
// avoidable errors (ADR-0076): nothing when no decision has a difficulty.
func writeDifficultySummary(sb *strings.Builder, names [2]string, losses []storage.DecisionLoss) {
	diff := storage.SummariseDifficulty(losses)
	if diff[0].Decisions+diff[1].Decisions == 0 {
		return
	}
	sb.WriteString("\nDifficulty:\n")
	for i, name := range names {
		s := diff[i]
		ratio := "-"
		if s.Ratio != nil {
			ratio = fmt.Sprintf("%.2f", *s.Ratio)
		}
		fmt.Fprintf(sb, "  %s: difficulty %.2f%%, excess %+.2f%%, ratio %s, %d avoidable errors\n",
			name, s.Difficulty*100, s.Excess*100, ratio, s.Avoidable)
	}
}

// writeTimeSummary adds, per player, the decision times the match recorded:
// nothing at all when it recorded none.
func (cli *CLI) writeTimeSummary(sb *strings.Builder, match *Match) {
	if cli.db == nil {
		return
	}
	sum, err := cli.db.GetMatchTimeSummary(match.ID)
	if err != nil {
		return
	}
	names := [2]string{match.Player1Name, match.Player2Name}
	mean := func(total int64, n int) string {
		if n == 0 {
			return "-"
		}
		return fmt.Sprintf("%.1f s", float64(total)/float64(n)/1000)
	}
	header := false
	for i, p := range sum.Players {
		if p.CheckerCount+p.CubeCount == 0 {
			continue
		}
		if !header {
			sb.WriteString("\nDecision times:\n")
			header = true
		}
		fmt.Fprintf(sb, "  %s: total %.1f s, checker mean %s, cube mean %s", names[i],
			float64(p.TotalMS)/1000, mean(p.CheckerTotalMS, p.CheckerCount), mean(p.CubeTotalMS, p.CubeCount))
		if p.OverTime {
			sb.WriteString(", reserve ran out first")
		}
		sb.WriteString("\n")
	}
}
