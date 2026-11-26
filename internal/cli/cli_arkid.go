package cli

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// runArkID handles the arkid command: the ArkID of a position given as an
// XGID, an OGID or an ArkID (which it rewrites in canonical form). Pure
// computation, no database: the same domain codec serves the GUI and the serve
// daemon (positions.arkId).
func (cli *CLI) runArkID(args []string) error {
	cmd := flag.NewFlagSet("arkid", flag.ContinueOnError)
	format := cmd.String("format", "text", "Output format: text, json")
	expanded := cmd.Bool("expanded", false, "Write the developed form: every checker of the player on roll, then every checker of the opponent")

	cmd.Usage = func() {
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb arkid [options] <XGID|OGID|ArkID>")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Print the ArkID of a position, blunderDB's readable notation:")
		fmt.Fprintln(cli.stdout(), "[score]dice:cube(gates)rest, written from the side of the player on roll")
		fmt.Fprintln(cli.stdout(), "(lowercase letters; the opponent is uppercase). Two spellings of one")
		fmt.Fprintln(cli.stdout(), "position by blunderDB are identical to the character. An ArkID given as")
		fmt.Fprintln(cli.stdout(), "argument, in either form, comes back canonical.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		cmd.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  blunderdb arkid 'XGID=-b----E-C---eE---c-e----B-:0:0:1:65:0:0:0:0:10'")
		fmt.Fprintln(cli.stdout(), "  blunderdb arkid --expanded '[34]65:x2(ySQmLhfA)S3Qm3L3hf3'")
	}

	if err := cmd.Parse(args); err != nil {
		return err
	}
	if cmd.NArg() != 1 {
		cmd.Usage()
		return fmt.Errorf("expected exactly one XGID, OGID or ArkID argument")
	}
	if *format != "text" && *format != "json" {
		return fmt.Errorf("unknown --format %q: use text or json", *format)
	}
	pos, err := domain.DecodePositionID(cmd.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid XGID, OGID or ArkID: %w", err)
	}
	var id string
	if *expanded {
		id, err = domain.EncodeArkIDExpanded(&pos)
	} else {
		id, err = domain.EncodeArkID(&pos)
	}
	if err != nil {
		return err
	}
	if *format == "json" {
		enc := json.NewEncoder(cli.stdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]string{"arkid": id})
	}
	fmt.Fprintln(cli.stdout(), id)
	return nil
}
