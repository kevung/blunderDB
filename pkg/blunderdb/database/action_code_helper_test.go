package database

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// sqlActionLabels is the action-label registry of a test that writes rows
// with raw SQL.
type sqlActionLabels struct{ db *sql.DB }

func (r sqlActionLabels) RegisterActionLabel(label string) error {
	_, err := r.db.Exec(sqlshared.RegisterActionLabelSQL, label, label)
	return err
}

func (r sqlActionLabels) LookupActionLabel(label string) (int64, bool, error) {
	var code int64
	err := r.db.QueryRow(sqlshared.LookupActionLabelSQL, label).Scan(&code)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return code, err == nil, err
}

// testActionCode is the action code a raw-SQL fixture stores for label.
func testActionCode(t testing.TB, db *sql.DB, label string) int64 {
	t.Helper()
	code, err := sqlshared.ActionCodeFor(sqlActionLabels{db}, label)
	if err != nil {
		t.Fatal(err)
	}
	return code
}
