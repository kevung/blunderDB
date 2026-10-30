package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// failingCommitTx is a pgx.Tx whose commit is refused.
type failingCommitTx struct{ pgx.Tx }

func (failingCommitTx) Commit(context.Context) error   { return errors.New("commit refused") }
func (failingCommitTx) Rollback(context.Context) error { return nil }

type failingCommitDB struct{ execer }

func (failingCommitDB) Begin(context.Context) (pgx.Tx, error) { return failingCommitTx{}, nil }

// The backend name appears once on a refused commit of a shared transaction.
func TestTransact_CommitErrorNamesBackendOnce(t *testing.T) {
	var sh sqlshared.Execer = shared{failingCommitDB{}}
	err := sh.Transact(context.Background(), func(sqlshared.Execer) error { return nil })
	if err == nil {
		t.Fatal("Transact = nil, want the commit error")
	}
	if n := strings.Count(err.Error(), "postgres:"); n != 0 {
		t.Errorf("Transact error %q carries the backend prefix %d times, want 0 (the caller adds it)", err, n)
	}
	err = withTx(context.Background(), failingCommitDB{}, func(execer) error { return nil })
	if err == nil || strings.Count(err.Error(), "postgres:") != 1 {
		t.Errorf("withTx error %q, want one backend prefix", err)
	}
}

func TestIsForeignKeyViolation_ChecksTheConstraint(t *testing.T) {
	fk := func(name string) error {
		return &pgconn.PgError{Code: pgForeignKeyViolation, ConstraintName: name}
	}
	if !isForeignKeyViolation(fk("a_fkey"), "a_fkey") {
		t.Error("the named constraint is not recognised")
	}
	if isForeignKeyViolation(fk("b_fkey"), "a_fkey") {
		t.Error("another foreign key was taken for the expected one")
	}
	if isForeignKeyViolation(errors.New("x"), "a_fkey") {
		t.Error("a plain error was taken for a violation")
	}
}
