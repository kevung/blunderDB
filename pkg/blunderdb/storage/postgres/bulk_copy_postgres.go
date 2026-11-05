package postgres

import (
	"context"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// copyTx is a transaction whose bulk writes travel by COPY. COPY FROM is
// refused on a table under row-level security, so the rows go to a session
// temporary table, outside any policy and gone with the session, and the same
// INSERT ... SELECT as the unnest path moves them under the policy: what a
// row may be, its conflicts and what it returns do not change.
type copyTx struct {
	pgx.Tx
}

type copier interface {
	CopyFrom(ctx context.Context, table pgx.Identifier, columns []string, src pgx.CopyFromSource) (int64, error)
}

// WriteByUnnest returns a loader to the ordinary statements, to compare.
func (s *Storage) WriteByUnnest() { s.copyWrites = false }

// bulkStatement gives the statement and arguments that write rows by sql, an
// insert from unnestSelect(types). Under COPY the rows are staged first and
// the statement reads the stage instead of the arrays.
func bulkStatement(ctx context.Context, db execer, sql string, types []string, rows [][]any) (string, []any, error) {
	cols, err := bulkColumns(types, rows)
	if err != nil {
		return "", nil, err
	}
	c, ok := db.(copier)
	if !ok {
		return sql, cols, nil
	}
	unnest := unnestSelect(types)
	if !strings.Contains(sql, unnest) {
		return "", nil, fmt.Errorf("postgres: bulk statement does not read %s", unnest)
	}
	stage, names := stageTable(types)
	if _, err := db.Exec(ctx, stageDDL(stage, names, types)); err != nil {
		return "", nil, fmt.Errorf("postgres: bulk stage: %w", err)
	}
	if _, err := db.Exec(ctx, `TRUNCATE pg_temp.`+stage); err != nil {
		return "", nil, fmt.Errorf("postgres: bulk stage: %w", err)
	}
	if _, err := c.CopyFrom(ctx, pgx.Identifier{"pg_temp", stage}, names, pgx.CopyFromRows(transpose(cols, len(rows)))); err != nil {
		return "", nil, fmt.Errorf("postgres: bulk copy: %w", outOfRange(err))
	}
	return strings.Replace(sql, unnest, `SELECT * FROM pg_temp.`+stage, 1), nil, nil
}

// stageTable names the stage after its column types: two statements with the
// same types share one, since each empties it before use.
func stageTable(types []string) (string, []string) {
	h := fnv.New32a()
	h.Write([]byte(strings.Join(types, ",")))
	names := make([]string, len(types))
	for i := range types {
		names[i] = "c" + strconv.Itoa(i+1)
	}
	return "bulk_" + strconv.FormatUint(uint64(h.Sum32()), 16), names
}

func stageDDL(stage string, names, types []string) string {
	var b strings.Builder
	b.WriteString("CREATE TEMP TABLE IF NOT EXISTS " + stage + " (")
	for i, t := range types {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(names[i] + " " + t)
	}
	b.WriteString(")")
	return b.String()
}

// transpose turns bulkColumns' arrays back into rows, so COPY and unnest see
// the same converted values.
func transpose(cols []any, n int) [][]any {
	out := make([][]any, n)
	for r := range out {
		out[r] = make([]any, len(cols))
	}
	for c, col := range cols {
		for r := 0; r < n; r++ {
			switch v := col.(type) {
			case []*int64:
				out[r][c] = v[r]
			case []*bool:
				out[r][c] = v[r]
			case [][]byte:
				out[r][c] = v[r]
			case []*string:
				out[r][c] = v[r]
			case []*time.Time:
				out[r][c] = v[r]
			}
		}
	}
	return out
}
