package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// runGnubgID handles the gnubgid command: the Position ID and the Match ID GNU
// Backgammon shows for a position given as an XGID or an OGID. Pure
// computation, no database: the same domain encoder serves the GUI and the
// serve daemon (positions.gnubgId).
func (cli *CLI) runGnubgID(args []string) error {
	cmd := flag.NewFlagSet("gnubgid", flag.ContinueOnError)
	format := cmd.String("format", "text", "Output format: text, json")

	cmd.Usage = func() {
		fmt.Println("Usage: blunderdb gnubgid [options] <XGID|OGID>")
		fmt.Println()
		fmt.Println("Print the GNUbg identifier of a position: the Position ID (the board,")
		fmt.Println("from the side of the player on roll) and the Match ID (cube, score,")
		fmt.Println("dice), joined by a colon as GNU Backgammon copies them. The match")
		fmt.Println("length and the scores are derived from the away scores the position")
		fmt.Println("holds, as the XGID does.")
		fmt.Println()
		fmt.Println("Options:")
		cmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb gnubgid 'XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:0:10'")
		fmt.Println("  blunderdb gnubgid --format json '<XGID>'")
	}

	if err := cmd.Parse(args); err != nil {
		return err
	}
	if cmd.NArg() != 1 {
		cmd.Usage()
		return fmt.Errorf("expected exactly one XGID or OGID argument")
	}
	pos, err := domain.DecodePositionID(cmd.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid XGID or OGID: %w", err)
	}
	positionID, err := domain.EncodeGNUbgPositionID(&pos)
	if err != nil {
		return err
	}
	matchID := domain.EncodeGNUbgMatchID(&pos)

	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]string{"position_id": positionID, "match_id": matchID, "id": positionID + ":" + matchID})
	}
	fmt.Printf("%s:%s\n", positionID, matchID)
	return nil
}
