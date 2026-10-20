package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// runElsewhere is `blunderdb elsewhere`: one board looked up by its Zobrist
// hash in other database files, judged with the thresholds and match equity
// table of --db (ADR-0103). The daemon serves the same read, across tenants,
// as /v1/across.positionElsewhere.
func (cli *CLI) runElsewhere(args []string) error {
	fs := flag.NewFlagSet("elsewhere", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Database the board is judged from: its thresholds and match equity table (required)")
	var others moveList
	fs.Var(&others, "other", "Another database file to look in (repeatable, at least one)")
	position := fs.Int64("position", 0, "The board, as a position id of --db")
	xgid := fs.String("xgid", "", "The board, as an XGID")
	zobrist := fs.String("zobrist", "", "The board, as its Zobrist hash (decimal); neighbours are then ranked only where it is held")
	neighbours := fs.Int("neighbours", storage.ElsewhereDefaultNeighbours, fmt.Sprintf("Nearest boards listed per database (0: none, at most %d)", storage.ElsewhereMaxNeighbours))
	maxDistance := fs.Int("max-distance", 0, "Leave out neighbours farther than this many checker-pips (0: no ceiling)")
	format := fs.String("format", "text", "Output format: text or json")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb elsewhere --db <file> --other <file> [--other <file>...] (--position <id> | --xgid <XGID> | --zobrist <hash>) [options]")
		fmt.Println()
		fmt.Println("What other databases know of one board: their analysis, their comments, the")
		fmt.Println("matches where it was played and how, and their boards nearest to it. A play's")
		fmt.Println("cost is judged with the error and blunder thresholds of --db; an analysis")
		fmt.Println("valued with another match equity table than the one of --db is flagged, and")
		fmt.Println("its match-play costs are shown but not judged. The other files are opened")
		fmt.Println("read-only: nothing is written to any of them.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb elsewhere --db mine.db --other club.db --position 412")
		fmt.Println("  blunderdb elsewhere --db mine.db --other a.db --other b.db --xgid \"XGID=-b----E-C---eE---c-e----B-:0:0:1:52:0:0:0:0:10\"")
		fmt.Println("  blunderdb elsewhere --db mine.db --other club.db --zobrist 1234567890 --format json")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --db")
	}
	if len(others) == 0 {
		return fmt.Errorf("missing required flag: --other")
	}
	given := 0
	for _, set := range []bool{*position != 0, *xgid != "", *zobrist != ""} {
		if set {
			given++
		}
	}
	if given != 1 {
		return fmt.Errorf("name the board with exactly one of --position, --xgid or --zobrist")
	}
	if *neighbours < 0 || *neighbours > storage.ElsewhereMaxNeighbours {
		return fmt.Errorf("--neighbours must be between 0 and %d", storage.ElsewhereMaxNeighbours)
	}
	if f := strings.ToLower(*format); f != "text" && f != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}

	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}
	here, _ := database.CurrentStore(cli.db)
	if here == nil {
		return fmt.Errorf("no database open")
	}
	ctx := context.Background()
	q := storage.ElsewhereQuery{Neighbours: *neighbours, MaxDistance: *maxDistance}
	if *neighbours == 0 {
		q.Neighbours = -1
	}
	switch {
	case *position != 0:
		p, err := here.Positions().Load(ctx, "", *position)
		if err != nil {
			return fmt.Errorf("position %d: %w", *position, err)
		}
		q.Target = p
	case *xgid != "":
		p, err := domain.DecodeXGID(*xgid)
		if err != nil {
			return fmt.Errorf("--xgid: %w", err)
		}
		q.Target = &p
	default:
		z, err := strconv.ParseUint(*zobrist, 10, 64)
		if err != nil {
			return fmt.Errorf("--zobrist: %w", err)
		}
		q.Zobrist = z
	}
	judge, err := storage.JudgeOf(ctx, here, "")
	if err != nil {
		return err
	}

	sources := make([]storage.Source, 0, len(others))
	for _, path := range others {
		st, err := openReadOnly(ctx, path)
		if err != nil {
			return err
		}
		defer st.Close()
		sources = append(sources, storage.Source{Name: path, Storage: st})
	}
	reports, err := storage.Elsewhere(ctx, sources, q, judge)
	if err != nil {
		return err
	}
	if q.Target != nil {
		q.Zobrist = storage.ZobristOf(q.Target)
	}
	if strings.ToLower(*format) == "json" {
		type fileReport struct {
			File string `json:"file"`
			storage.ElsewhereReport
		}
		out := struct {
			Zobrist uint64        `json:"zobrist"`
			Judge   storage.Judge `json:"judge"`
			Results []fileReport  `json:"results"`
		}{Zobrist: q.Zobrist, Judge: judge, Results: make([]fileReport, len(reports))}
		for i, r := range reports {
			out.Results[i] = fileReport{File: r.Tenant, ElsewhereReport: r.Item}
		}
		return printJSON(out)
	}
	printElsewhere(q.Zobrist, judge, reports)
	return nil
}

// openReadOnly opens another database file without writing to it: it must
// exist and be at the current schema, since a migration is a write.
func openReadOnly(ctx context.Context, path string) (*sqlite.Storage, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	st, err := sqlite.OpenReadOnly(ctx, path)
	if err != nil {
		return nil, err
	}
	v, err := st.Version(ctx)
	if err != nil {
		st.Close()
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := schemaReadable(path, v); err != nil {
		st.Close()
		return nil, err
	}
	return st, nil
}

// schemaReadable refuses a file at another schema than this blunderDB's, and
// says which side has to move.
func schemaReadable(path, version string) error {
	cmp, err := database.CompareVersions(version, domain.DatabaseVersion)
	switch {
	case err != nil:
		return fmt.Errorf("%s: unreadable database version %q: %w", path, version, err)
	case cmp < 0:
		return fmt.Errorf("%s: database version %s is older than %s: open it once with blunderDB to upgrade it", path, version, domain.DatabaseVersion)
	case cmp > 0:
		return fmt.Errorf("%s: database version %s is newer than %s: update blunderDB to read it", path, version, domain.DatabaseVersion)
	}
	return nil
}

func printElsewhere(zobrist uint64, judge storage.Judge, reports []storage.Tagged[storage.ElsewhereReport]) {
	fmt.Printf("Board %d, judged at error >= %d mp, blunder >= %d mp\n",
		zobrist, judge.Settings.ErrorThresholdMP, judge.Settings.BlunderThresholdMP)
	for _, r := range reports {
		rep := r.Item
		fmt.Println()
		fmt.Printf("== %s\n", r.Tenant)
		if rep.PositionID == 0 {
			fmt.Println("Not held.")
		} else {
			fmt.Printf("Position %d", rep.PositionID)
			switch {
			case rep.Analysis == nil:
				fmt.Print(", no analysis")
			case rep.OtherMET:
				name := rep.METName
				if name == "" {
					name = "built-in"
				}
				fmt.Printf(", analysed with another match equity table (%s)", name)
			default:
				fmt.Print(", analysed")
			}
			fmt.Println()
			for _, c := range rep.Comments {
				author := c.Author
				if author == "" {
					author = "-"
				}
				fmt.Printf("  comment (%s): %s\n", author, strings.ReplaceAll(c.Text, "\n", " "))
			}
			if len(rep.Plays) > 0 {
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "  Match\tPlayers\tLength\tDate\tPlayed\tError (mp)\tVerdict")
				for _, p := range rep.Plays {
					cost := "-"
					if p.ErrorMP != nil {
						cost = strconv.Itoa(int(*p.ErrorMP))
					}
					fmt.Fprintf(w, "  %d\t%s - %s\t%d\t%s\t%s\t%s\t%s\n", p.MatchID, p.Player1, p.Player2, p.MatchLength, p.MatchDate, p.Played, cost, p.Verdict)
				}
				w.Flush()
			}
			if rep.PlaysTruncated {
				fmt.Printf("  %d of %d matches shown, the latest imported\n", len(uniqueMatches(rep.Plays)), rep.MatchesTotal)
			}
		}
		if len(rep.Neighbours) > 0 {
			fmt.Println("  Nearest boards:")
			for _, n := range rep.Neighbours {
				fmt.Printf("    position %d at %d checker-pips (%s)\n", n.Position.ID, n.Distance, domain.EncodeXGID(&n.Position))
			}
		}
	}
}

func uniqueMatches(plays []storage.ElsewherePlay) map[int64]bool {
	out := map[int64]bool{}
	for _, p := range plays {
		out[p.MatchID] = true
	}
	return out
}
