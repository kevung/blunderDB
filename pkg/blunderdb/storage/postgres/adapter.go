package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// shared adapts the backend's execer (a *pgxpool.Pool or a pgx.Tx) to the
// sqlshared.Execer the backend-agnostic stores run against, and answers the
// dialect questions for PostgreSQL: '?' placeholders rebound to '$N', every
// domain table confined to the scope's tenant_id, BIGINT aggregates cast back
// from NUMERIC, TIMESTAMPTZ dates formatted in UTC.
type shared struct{ db execer }

var _ sqlshared.Execer = shared{}

// shared returns the binder's execer wrapped for the shared stores.
func (b binder) shared() sqlshared.Execer { return shared(b) }

func (shared) Name() string                         { return "postgres" }
func (shared) ScopeColumn() string                  { return "tenant_id" }
func (shared) LibrarySettingsTable() (string, bool) { return "library_settings", true }
func (shared) ScopeArg(scope string) any            { return tenantID(scope) }
func (shared) TenantFilter(alias, scope string) (string, []any) {
	col := "tenant_id"
	if alias != "" {
		col = alias + ".tenant_id"
	}
	return col + " = ?", []any{tenantID(scope)}
}
func (shared) TenantColumns(scope string) ([]string, []any) {
	return []string{"tenant_id"}, []any{tenantID(scope)}
}
func (shared) SameTenant(left, right string) string {
	return left + ".tenant_id = " + right + ".tenant_id"
}
func (shared) Bool(col string, v bool) string {
	if v {
		return col + " = TRUE"
	}
	return col + " = FALSE"
}
func (shared) BoolAsInt(col string) string { return "CASE WHEN " + col + " THEN 1 ELSE 0 END" }
func (shared) BoolArg(v bool) any          { return v }
func (shared) Bigint(expr string) string   { return "CAST(" + expr + " AS BIGINT)" }
func (shared) ILike() string               { return "ILIKE" }

// The array is wrapped in a scalar sub-select so the planner treats it as a
// parameter, not a constant kept by the custom plan (~68 MB per backend at 220k ids).
func (shared) IDsIn(expr string, ids []string) (string, []any) {
	return expr + " = ANY((SELECT ?::bigint[]))", []any{"{" + strings.Join(ids, ",") + "}"}
}
func (shared) ForUpdate() string { return " FOR UPDATE" }
func (shared) Guard(key string) (string, []any) {
	return "SELECT pg_advisory_xact_lock(?)", []any{guardKey(key)}
}
func (shared) LimitOffset(limit, offset int) (string, []any) {
	switch {
	case limit <= 0 && offset <= 0:
		return "", nil
	case limit <= 0:
		return " OFFSET ?", []any{offset}
	case offset <= 0:
		return " LIMIT ?", []any{limit}
	default:
		return " LIMIT ? OFFSET ?", []any{limit, offset}
	}
}
func (shared) TimestampArg() string { return "?::timestamptz" }
func (shared) TimestampTextArg() string {
	return "CAST(? AS TIMESTAMP) AT TIME ZONE 'UTC'"
}
func (shared) InstantArg(t time.Time) (string, any) { return "?", t.UTC() }
func (shared) DateText(col string) string {
	return "COALESCE(TO_CHAR(" + col + " AT TIME ZONE 'UTC','YYYY-MM-DD'),'')"
}
func (shared) TimestampText(col string) string {
	return "COALESCE(TO_CHAR(" + col + " AT TIME ZONE 'UTC','YYYY-MM-DD HH24:MI:SS'),'')"
}
func (shared) Referenced(err error) error { return referenced(err) }

func (a shared) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	tag, err := a.db.Exec(ctx, typedRebind(query, args), args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (a shared) Query(ctx context.Context, query string, args ...any) (sqlshared.Rows, error) {
	rows, err := a.db.Query(ctx, typedRebind(query, args), args...) //nolint:rowserrcheck // the cursor is handed to the caller, which iterates it and checks Err
	if err != nil {
		return nil, err
	}
	return pgRows{rows}, nil
}

// typedRebind is sqlshared.Rebind that declares a Go integer a bigint where it is compared: left to
// infer, it takes the type of its column, and a filter past a smallint would fail to encode instead of
// finding nothing. Only there: a function argument or an operand that wants an integer (substr, date +
// n, jsonb -> n) has no implicit cast from bigint, and is left to infer as before.
func typedRebind(query string, args []any) string {
	if !strings.Contains(query, "?") {
		return query
	}
	if len(args) > 0 {
		if _, option := args[0].(pgx.QueryExecMode); option {
			args = args[1:]
		}
	}
	var b strings.Builder
	b.Grow(len(query) + 16)
	var (
		prev    string // last token: an operator, a parenthesis, a comma or an upper-cased word
		inList  []bool // per open parenthesis, whether it is the list of an IN
		between bool   // a BETWEEN waits for its AND
	)
	n := 0
	for i := 0; i < len(query); {
		c := query[i]
		switch {
		case c == '?':
			b.WriteString("$" + strconv.Itoa(n+1))
			compared := comparison[prev] || prev == "BETWEEN" || prev == "AND+" ||
				((prev == "(" || prev == ",") && len(inList) > 0 && inList[len(inList)-1]) ||
				comparison[nextOperator(query[i+1:])]
			if n < len(args) && isGoInteger(args[n]) && compared && !strings.HasPrefix(query[i+1:], "::") {
				b.WriteString("::bigint")
			}
			n++
			prev = "?"
			i++
			continue
		case c == '(':
			inList = append(inList, prev == "IN")
			prev = "("
		case c == ')':
			if len(inList) > 0 {
				inList = inList[:len(inList)-1]
			}
			prev = ")"
		case c == ',':
			prev = ","
		case isWordByte(c):
			j := i
			for j < len(query) && isWordByte(query[j]) {
				j++
			}
			prev = strings.ToUpper(query[i:j])
			switch {
			case prev == "BETWEEN":
				between = true
			case prev == "AND" && between:
				between, prev = false, "AND+"
			}
			b.WriteString(query[i:j])
			i = j
			continue
		case isOperatorByte(c):
			j := i
			for j < len(query) && isOperatorByte(query[j]) {
				j++
			}
			prev = query[i:j]
			b.WriteString(prev)
			i = j
			continue
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

var comparison = map[string]bool{"=": true, "<>": true, "!=": true, "<": true, ">": true, "<=": true, ">=": true}

// nextOperator is the operator that follows a placeholder, for one written on the left.
func nextOperator(rest string) string {
	rest = strings.TrimLeft(rest, " \t\n")
	j := 0
	for j < len(rest) && isOperatorByte(rest[j]) {
		j++
	}
	return rest[:j]
}

func isWordByte(c byte) bool {
	return c == '_' || c == '.' || c == '$' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isOperatorByte(c byte) bool { return strings.IndexByte("<>=!+-*/%|&~^#@:", c) >= 0 }

func isGoInteger(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint8, uint16, uint32:
		return true
	}
	return false
}

// searchDB plans each statement of a search for its values. A cached statement may keep a generic
// plan, unaware of the LIMIT or of the value, and a search walks or sorts the tenant. Only the search:
// with custom plans, an import into a new tenant, out of the histogram, is estimated at one row and
// nests its loops, seconds per match.
type searchDB struct{ sqlshared.Execer }

func (a searchDB) Query(ctx context.Context, query string, args ...any) (sqlshared.Rows, error) {
	return a.Execer.Query(ctx, query, append([]any{pgx.QueryExecModeDescribeExec}, args...)...)
}

func (a searchDB) QueryRow(ctx context.Context, query string, args ...any) sqlshared.Row {
	return a.Execer.QueryRow(ctx, query, append([]any{pgx.QueryExecModeDescribeExec}, args...)...)
}

func (a shared) QueryRow(ctx context.Context, query string, args ...any) sqlshared.Row {
	return pgRow{a.db.QueryRow(ctx, typedRebind(query, args), args...)}
}

func (a shared) Insert(ctx context.Context, query string, args ...any) (int64, error) {
	var id int64
	if err := a.db.QueryRow(ctx, typedRebind(query, args)+" RETURNING id", args...).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

// FillKey names the scope's tenant within the pool; a transaction has no key.
func (a shared) FillKey(scope string) (any, bool) {
	pool, ok := a.db.(*pgxpool.Pool)
	if !ok {
		return nil, false
	}
	return poolScope{pool, scope}, true
}

// joinOrderSettings are what sqlshared.withJoinOrderPlan needs of the planner.
var joinOrderSettings = []string{"join_collapse_limit", "from_collapse_limit", "enable_hashjoin", "enable_mergejoin", "enable_seqscan", "enable_bitmapscan"}

// PinJoinOrder — see sqlshared.withJoinOrderPlan. Local to the transaction, and
// taken back at the end of the work it was asked for: the transaction may go on
// to other writes.
func (a shared) PinJoinOrder(ctx context.Context) (func() error, error) {
	if _, err := a.db.Exec(ctx, `SELECT set_config('join_collapse_limit', '1', true), set_config('from_collapse_limit', '1', true),
		set_config('enable_hashjoin', 'off', true), set_config('enable_mergejoin', 'off', true), set_config('enable_seqscan', 'off', true),
		set_config('enable_bitmapscan', 'off', true)`); err != nil {
		return nil, fmt.Errorf("postgres: pin join order: %w", err)
	}
	return func() error {
		if _, err := a.db.Exec(ctx, `SELECT set_config(name, reset_val, true) FROM pg_settings WHERE name = ANY($1)`, joinOrderSettings); err != nil {
			return fmt.Errorf("postgres: unpin join order: %w", err)
		}
		return nil
	}, nil
}

type poolScope struct {
	pool  *pgxpool.Pool
	scope string
}

// Transact leaves the backend name off its own errors: sqlshared wraps them
// with the dialect's name.
func (a shared) Transact(ctx context.Context, fn func(sqlshared.Execer) error) error {
	return runTx(ctx, a.db, "", func(tx execer) error { return fn(shared{tx}) })
}

// pgRows gives pgx.Rows the Close() error signature of database/sql.
type pgRows struct{ pgx.Rows }

func (r pgRows) Close() error {
	r.Rows.Close()
	return nil
}

// pgRow translates pgx's no-row sentinel to the shared one.
type pgRow struct{ row pgx.Row }

func (r pgRow) Scan(dest ...any) error {
	err := r.row.Scan(dest...)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlshared.ErrNoRows
	}
	return err
}
