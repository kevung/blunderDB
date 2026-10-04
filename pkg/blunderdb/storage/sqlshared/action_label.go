package sqlshared

import (
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Action labels — move.move_type, move.cube_action, analysis.best_cube_action —
// are stored as integer codes (domain.ActionCode, ADR-0071). Every read goes
// through ActionLabelSQL and every write through ActionCodeFor, so no query
// spells a code and both backends read back the exact string that was written:
// NULL stays NULL, "" is code 0, a label the fixed list lacks is registered in
// action_label.

// ActionLabelSQL renders the expression that reads the label stored in the
// action-code column col. A fixed code resolves in the CASE; a registered one
// in action_label, which COALESCE only reaches when the CASE gives NULL. It is
// the SQLite form: action_label there belongs to the one library of the file.
func ActionLabelSQL(col string) string {
	return actionLabelSQL(col, "")
}

// TenantActionLabelSQL is ActionLabelSQL for PostgreSQL, where action_label
// holds every tenant's registered labels: the lookup is confined to the tenant
// of the row col belongs to, so a code that crossed tenants (a restored dump,
// a hand edit, a read that bypasses row-level security) reads as no label
// rather than as another tenant's text. col must be qualified (alias.column):
// its qualifier names the row's tenant_id, which an unqualified name inside
// the subquery would resolve to action_label's own.
func TenantActionLabelSQL(col string) string {
	dot := strings.LastIndexByte(col, '.')
	if dot <= 0 {
		panic("sqlshared: TenantActionLabelSQL needs a qualified column, got " + col)
	}
	return actionLabelSQL(col, col[:dot]+".tenant_id")
}

// ActionLabelFor is the label read for dialect d: TenantActionLabelSQL where
// action_label is tenant-scoped, ActionLabelSQL otherwise.
func ActionLabelFor(d Dialect, col string) string {
	if d.ScopeColumn() == "tenant_id" {
		return TenantActionLabelSQL(col)
	}
	return ActionLabelSQL(col)
}

func actionLabelSQL(col, tenantCol string) string {
	var b strings.Builder
	b.WriteString("COALESCE(CASE ")
	b.WriteString(col)
	for code, label := range domain.ActionLabels() {
		fmt.Fprintf(&b, " WHEN %d THEN '%s'", code, strings.ReplaceAll(label, "'", "''"))
	}
	b.WriteString(" END, (SELECT al.label FROM action_label al WHERE al.code = ")
	b.WriteString(col)
	if tenantCol != "" {
		b.WriteString(" AND al.tenant_id = ")
		b.WriteString(tenantCol)
	}
	b.WriteString("))")
	return b.String()
}

// ActionLabelOrEmptySQL is ActionLabelSQL with NULL read as "", the
// COALESCE(col, ”) the text columns were read with.
func ActionLabelOrEmptySQL(col string) string {
	return orEmpty(ActionLabelSQL(col))
}

// TenantActionLabelOrEmptySQL is TenantActionLabelSQL with NULL read as "".
func TenantActionLabelOrEmptySQL(col string) string {
	return orEmpty(TenantActionLabelSQL(col))
}

// ActionLabelOrEmptyFor is ActionLabelFor with NULL read as "".
func ActionLabelOrEmptyFor(d Dialect, col string) string {
	return orEmpty(ActionLabelFor(d, col))
}

func orEmpty(expr string) string { return "COALESCE(" + expr + ", '')" }

// ActionNotEmptySQL renders the predicate COALESCE(col, ”) <> ” of a text
// label column on the action-code column col: "" is code 0.
func ActionNotEmptySQL(col string) string {
	return "COALESCE(" + col + ", 0) <> 0"
}

// ActionCodeOrEmptySQL renders the code of COALESCE(label, ”): NULL and ""
// compare equal, as they did on a text column.
func ActionCodeOrEmptySQL(col string) string {
	return "COALESCE(" + col + ", 0)"
}

// ActionIsSQL renders col = label for a label of the fixed list.
func ActionIsSQL(col, label string) string {
	return col + " = " + fmt.Sprint(fixedActionCode(label))
}

// ActionNotInSQL renders COALESCE(label, ”) NOT IN (labels) for labels of
// the fixed list.
func ActionNotInSQL(col string, labels ...string) string {
	codes := make([]string, len(labels))
	for i, l := range labels {
		codes[i] = fmt.Sprint(fixedActionCode(l))
	}
	return ActionCodeOrEmptySQL(col) + " NOT IN (" + strings.Join(codes, ", ") + ")"
}

// fixedActionCode is the code of a label a query spells; a label outside the
// fixed list is a programming error.
func fixedActionCode(label string) int64 {
	code, ok := domain.ActionCode(label)
	if !ok {
		panic("sqlshared: " + label + " is not a fixed action label")
	}
	return code
}

// RegisterActionLabelSQL inserts label (both placeholders) under the next free
// registered code; a label already present, or a code taken by a concurrent
// registration, inserts nothing and ActionCodeFor tries again.
var RegisterActionLabelSQL = `INSERT INTO action_label (code, label)
SELECT COALESCE(MAX(code), ` + fmt.Sprint(domain.FirstRegisteredActionCode-1) + `) + 1, CAST(? AS TEXT) FROM action_label
WHERE NOT EXISTS (SELECT 1 FROM action_label WHERE label = ?)
ON CONFLICT DO NOTHING`

// LookupActionLabelSQL returns the registered code of a label.
const LookupActionLabelSQL = `SELECT code FROM action_label WHERE label = ?`

// ActionLabelRegistry is what ActionCodeFor needs of a backend: run
// RegisterActionLabelSQL and LookupActionLabelSQL in the caller's transaction.
type ActionLabelRegistry interface {
	RegisterActionLabel(label string) error
	// LookupActionLabel returns false when label is not registered.
	LookupActionLabel(label string) (int64, bool, error)
}

// ActionCodeFor returns the code that stores label, registering it when the
// fixed list lacks it.
func ActionCodeFor(r ActionLabelRegistry, label string) (int64, error) {
	if code, ok := domain.ActionCode(label); ok {
		return code, nil
	}
	for range 3 {
		if code, ok, err := r.LookupActionLabel(label); err != nil || ok {
			return code, err
		}
		if err := r.RegisterActionLabel(label); err != nil {
			return 0, fmt.Errorf("registering action label %q: %w", label, err)
		}
	}
	code, ok, err := r.LookupActionLabel(label)
	if err == nil && !ok {
		err = fmt.Errorf("action label %q could not be registered", label)
	}
	return code, err
}

// ActionLabelCodesSQL renders the SQL that maps a text label column to its
// code during a migration: the fixed codes in a CASE, a registered label
// through action_label (which the migration fills first).
func ActionLabelCodesSQL(col string) string {
	var b strings.Builder
	b.WriteString("CASE ")
	b.WriteString(col)
	for code, label := range domain.ActionLabels() {
		fmt.Fprintf(&b, " WHEN '%s' THEN %d", strings.ReplaceAll(label, "'", "''"), code)
	}
	b.WriteString(" ELSE (SELECT al.code FROM action_label al WHERE al.label = ")
	b.WriteString(col)
	b.WriteString(") END")
	return b.String()
}

// FixedActionLabelsSQL renders the fixed labels as an SQL list, for a
// migration that registers every other label it finds.
func FixedActionLabelsSQL() string {
	labels := domain.ActionLabels()
	parts := make([]string, len(labels))
	for i, l := range labels {
		parts[i] = "'" + strings.ReplaceAll(l, "'", "''") + "'"
	}
	return "(" + strings.Join(parts, ", ") + ")"
}
