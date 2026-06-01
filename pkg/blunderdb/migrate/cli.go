package migrate

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

const migrateUsage = `blunderdb migrate — copy a SQLite database into PostgreSQL under a tenant.

Usage:
  blunderdb migrate --from sqlite:///path/to/user.db \
                    --to postgres://user:pass@host:5432/db?sslmode=disable \
                    --tenant-id <scope> [--dry-run] [--on-conflict skip]

Copies positions, analyses, comments, matches (games + moves), tournaments and
collections under the given tenant scope, inside a single destination
transaction (atomic; a failed run leaves the destination untouched). App-state
families (anki, filter library, history, session) are not migrated.

Flags:
`

// RunCLI parses the `migrate` subcommand flags, opens both backends and runs the
// migration, streaming NDJSON progress to stdout. args are the arguments after
// "migrate".
func RunCLI(args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, migrateUsage)
		fs.PrintDefaults()
	}
	var (
		from       = fs.String("from", "", "source SQLite database (sqlite:///path or a bare path)")
		to         = fs.String("to", "", "destination PostgreSQL DSN (postgres://…)")
		tenant     = fs.String("tenant-id", "", "destination tenant scope (X-Tenant-ID)")
		dryRun     = fs.Bool("dry-run", false, "count what would be copied without writing")
		onConflict = fs.String("on-conflict", "", "destination-not-empty policy: \"\" (abort) | skip")
		_          = fs.Int("batch-size", 1000, "reserved for future batching (currently unused)")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *from == "" || *to == "" {
		fs.Usage()
		return fmt.Errorf("migrate: --from and --to are required")
	}
	if *tenant == "" && !*dryRun {
		return fmt.Errorf("migrate: --tenant-id is required (the destination scope)")
	}

	ctx := context.Background()

	srcPath := strings.TrimPrefix(strings.TrimPrefix(*from, "sqlite://"), "sqlite:")
	src, err := sqlite.Open(ctx, srcPath, nil)
	if err != nil {
		return fmt.Errorf("migrate: open source %s: %w", srcPath, err)
	}
	defer src.Close()
	if err := src.Migrate(ctx); err != nil {
		return fmt.Errorf("migrate: upgrade source schema: %w", err)
	}

	emit := json.NewEncoder(os.Stdout)
	opts := Options{
		DryRun:     *dryRun,
		OnConflict: *onConflict,
		Progress: func(r Report) {
			_ = emit.Encode(progressEvent{Event: "progress", Report: r})
		},
	}

	if *dryRun {
		// A dry run only reads the source; no destination connection needed.
		rep, err := Run(ctx, src, nil, *tenant, opts)
		if err != nil {
			return err
		}
		return emit.Encode(progressEvent{Event: "dry-run", Report: rep})
	}

	dst, err := postgres.Open(ctx, *to, nil)
	if err != nil {
		return fmt.Errorf("migrate: open destination: %w", err)
	}
	defer dst.Close()
	if err := dst.Migrate(ctx); err != nil {
		return fmt.Errorf("migrate: ensure destination schema: %w", err)
	}

	rep, err := Run(ctx, src, dst, *tenant, opts)
	if err != nil {
		return err
	}
	return emit.Encode(progressEvent{Event: "done", Report: rep})
}

type progressEvent struct {
	Event  string `json:"event"`
	Report Report `json:"report"`
}
