package postgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// A bulk write sends one column per array and lets unnest rebuild the rows:
// the statement text does not depend on the row count, so it is planned once
// and survives in the statement cache, and statement triggers fire once for
// the whole set instead of once per row.
const (
	bulkBigint = "bigint"
	bulkInt    = "int"
	bulkBool   = "boolean"
	bulkBytea  = "bytea"
	bulkText   = "text"
	bulkTime   = "timestamptz"
)

// analyzeFirstOfTenant gives table statistics that know the tenant, when these
// are its first rows there (idColumn below minID: none). A planner that has never
// seen a tenant_id estimates one row for it, and takes any index starting with
// tenant_id for a way to reach a row: each foreign key check, journal trigger and
// statistics query then walks all the tenant's rows, a cost that grows with what
// is being imported, until autovacuum comes by, a minute later at best and never
// for a small tenant in a large table. Paid by a tenant's first import only, in
// its transaction, whose rows it counts; a concurrent ANALYZE is left to finish.
func analyzeFirstOfTenant(ctx context.Context, db execer, table, idColumn string, tenant, minID int64) error {
	var seen bool
	if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM `+table+` WHERE tenant_id = $1 AND `+idColumn+` < $2)`,
		tenant, minID).Scan(&seen); err != nil {
		return fmt.Errorf("postgres: first rows of the tenant in %s: %w", table, err)
	}
	if seen {
		return nil
	}
	if _, err := db.Exec(ctx, "ANALYZE (SKIP_LOCKED) "+table); err != nil {
		return fmt.Errorf("postgres: analyze %s: %w", table, err)
	}
	return nil
}

// unnestSelect renders "SELECT * FROM unnest($1::t1[], $2::t2[], ...)".
func unnestSelect(types []string) string {
	var b strings.Builder
	b.WriteString("SELECT * FROM unnest(")
	for i, t := range types {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("$" + strconv.Itoa(i+1) + "::" + t + "[]")
	}
	b.WriteString(")")
	return b.String()
}

// insertHead is the "INSERT INTO t (cols)" of a single-row insert statement.
func insertHead(sql string) string {
	head, _, ok := strings.Cut(sql, ") VALUES")
	if !ok {
		panic("postgres: not a VALUES insert: " + sql)
	}
	return head + ")"
}

// bulkColumns pivots rows into one typed array per column, NULLs kept, ready
// to bind to unnestSelect(types).
func bulkColumns(types []string, rows [][]any) ([]any, error) {
	out := make([]any, len(types))
	for c, t := range types {
		var err error
		switch t {
		case bulkBigint, bulkInt:
			col := make([]*int64, len(rows))
			for r := range rows {
				if col[r], err = bulkIntValue(rows[r][c]); err != nil {
					return nil, fmt.Errorf("postgres: bulk column %d: %w", c, err)
				}
			}
			out[c] = col
		case bulkBool:
			col := make([]*bool, len(rows))
			for r := range rows {
				switch v := rows[r][c].(type) {
				case nil:
				case bool:
					col[r] = &v
				default:
					return nil, fmt.Errorf("postgres: bulk column %d: %T is not a boolean", c, v)
				}
			}
			out[c] = col
		case bulkBytea:
			col := make([][]byte, len(rows))
			for r := range rows {
				switch v := rows[r][c].(type) {
				case nil:
				case []byte:
					col[r] = v
				default:
					return nil, fmt.Errorf("postgres: bulk column %d: %T is not bytes", c, v)
				}
			}
			out[c] = col
		case bulkText:
			col := make([]*string, len(rows))
			for r := range rows {
				switch v := rows[r][c].(type) {
				case nil:
				case string:
					col[r] = &v
				default:
					return nil, fmt.Errorf("postgres: bulk column %d: %T is not text", c, v)
				}
			}
			out[c] = col
		case bulkTime:
			col := make([]*time.Time, len(rows))
			for r := range rows {
				switch v := rows[r][c].(type) {
				case nil:
				case time.Time:
					col[r] = &v
				default:
					return nil, fmt.Errorf("postgres: bulk column %d: %T is not a time", c, v)
				}
			}
			out[c] = col
		default:
			return nil, fmt.Errorf("postgres: bulk column %d: unknown type %s", c, t)
		}
	}
	return out, nil
}

func bulkIntValue(v any) (*int64, error) {
	var n int64
	switch x := v.(type) {
	case nil:
		return nil, nil
	case int:
		n = int64(x)
	case int8:
		n = int64(x)
	case int16:
		n = int64(x)
	case int32:
		n = int64(x)
	case int64:
		n = x
	case uint8:
		n = int64(x)
	case uint16:
		n = int64(x)
	case uint32:
		n = int64(x)
	default:
		return nil, fmt.Errorf("%T is not an integer", v)
	}
	return &n, nil
}
