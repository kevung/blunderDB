package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// runCapture handles the capture command: the position a screen capture of
// another program's board shows, with what the reading could not settle.
// What no capture shows — the score, the turn when no dice are out — is set
// by --score and --turn. Saving refuses a reading that leaves either unset,
// since its default is a wrong value rather than a blank, and refuses one
// with doubts unless --force says the user has checked the board.
func (cli *CLI) runCapture(args []string) error {
	fs := flag.NewFlagSet("capture", flag.ContinueOnError)
	format := fs.String("format", "text", "Output format: text, json, xgid")
	dbPath := fs.String("db", "", "Database to save the position into (with --save)")
	save := fs.Bool("save", false, "Save the position into --db as an individual position")
	score := fs.String("score", "", "The score the capture does not show: \"money\", or two Away scores \"<player 1>a/<player 2>a\" (e.g. 2a/5a); player 1 plays from the bottom")
	turn := fs.Int("turn", 0, "The player on roll, 1 (bottom) or 2 (top), when no dice show it")
	force := fs.Bool("force", false, "Save even though the reading has doubts (points, bar, cube, dice or orientation it could not settle)")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb capture [options] <image>")
		fmt.Println()
		fmt.Println("Read the position a screen capture of an eXtreme Gammon board shows")
		fmt.Println("(PNG, JPEG or GIF, at any size): checkers, bar, borne-off checkers,")
		fmt.Println("dice, cube and player on roll. The score is not read: set it with")
		fmt.Println("--score, and the turn with --turn when no dice are out. Whatever the")
		fmt.Println("reading cannot settle is listed as a doubt, never guessed.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb capture screenshot.png")
		fmt.Println("  blunderdb capture --format xgid screenshot.png")
		fmt.Println("  blunderdb capture --db my.db --save --score 2a/5a screenshot.png")
		fmt.Println("  blunderdb capture --db my.db --save --score money --turn 2 cube.png")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("expected exactly one image")
	}
	var scoreSet [2]int
	if *score != "" {
		v, err := parseCaptureScore(*score)
		if err != nil {
			return err
		}
		scoreSet = v
	}
	if *turn != 0 && *turn != 1 && *turn != 2 {
		return fmt.Errorf("invalid --turn %d: 1 (bottom) or 2 (top)", *turn)
	}
	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return err
	}
	rd, err := capture.Decode(data)
	if err != nil {
		return err
	}
	if *score != "" {
		rd.SetScore(scoreSet)
	}
	if *turn != 0 {
		rd.SetTurn(map[int]int{1: domain.Black, 2: domain.White}[*turn])
	}

	switch *format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rd); err != nil {
			return err
		}
	case "xgid":
		fmt.Println("XGID=" + domain.EncodeXGID(&rd.Position))
	default:
		fmt.Printf("Software: %s (fit %.2f)\n", rd.Software, rd.Fit)
		fmt.Printf("XGID=%s\n", domain.EncodeXGID(&rd.Position))
		if len(rd.Unread) > 0 {
			fmt.Printf("Not read: %s\n", strings.Join(rd.Unread, ", "))
		}
		for _, d := range rd.Doubts {
			if d.Field == "point" || d.Field == "bar" {
				fmt.Printf("Doubt: %s %d — %s\n", d.Field, d.Point, d.Reason)
			} else {
				fmt.Printf("Doubt: %s — %s\n", d.Field, d.Reason)
			}
		}
	}

	if !*save {
		return nil
	}
	if *dbPath == "" {
		return fmt.Errorf("--save needs --db")
	}
	// An unread score is saved as money play and an unread turn as a guess:
	// both are wrong values, not blanks, so only the user's word settles them.
	if len(rd.Unread) > 0 {
		var opts []string
		for _, f := range rd.Unread {
			opts = append(opts, captureOptionOf[f])
		}
		return fmt.Errorf("the capture does not show %s: set %s", strings.Join(rd.Unread, ", "), strings.Join(opts, ", "))
	}
	if len(rd.Doubts) > 0 && !*force {
		return fmt.Errorf("the reading has %d doubt(s): check the board, then save with --force", len(rd.Doubts))
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}
	res, err := cli.db.SaveIndividualPosition(&rd.Position)
	if err != nil {
		return err
	}
	if *format != "json" {
		fmt.Printf("Saved as position %d (already there: %v)\n", res.ID, res.Existed)
	}
	return nil
}

// captureOptionOf names the option that settles each field a capture leaves
// unread.
var captureOptionOf = map[string]string{"score": "--score", "turn": "--turn"}

// parseCaptureScore reads --score: "money", or "<a>a/<b>a" with player 1's
// Away score first (1 is the Crawford game, 0 one point away after it).
func parseCaptureScore(v string) ([2]int, error) {
	if strings.EqualFold(v, "money") {
		return [2]int{-1, -1}, nil
	}
	bad := fmt.Errorf("invalid --score %q: \"money\" or two Away scores such as 2a/5a", v)
	lo, hi, ok := strings.Cut(v, "/")
	if !ok {
		return [2]int{}, bad
	}
	var out [2]int
	for i, part := range []string{lo, hi} {
		n, err := strconv.Atoi(strings.TrimSuffix(strings.ToLower(strings.TrimSpace(part)), "a"))
		if err != nil || n < 0 {
			return [2]int{}, bad
		}
		out[i] = n
	}
	// Zero away is the post-Crawford side, so the other must still be in the game.
	if (out[0] == 0 && out[1] < 2) || (out[1] == 0 && out[0] < 2) {
		return [2]int{}, bad
	}
	return out, nil
}
