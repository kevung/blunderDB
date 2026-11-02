package sqlite

import (
	"context"
	"database/sql"
	"sync"
)

// poolStmts keeps the statement a pool writes one row at a time prepared. SQLite
// compiles every trigger of a table into each statement that writes it, which
// costs more than running the statement; database/sql prepares it on each
// connection once, and SQLite recompiles it by itself when the schema changes.
type poolStmts struct {
	db    *sql.DB
	mu    sync.Mutex
	stmts map[string]*sql.Stmt
}

// exec runs query prepared, on db when p is nil or query cannot be prepared:
// db then reports any error the usual way.
func (p *poolStmts) exec(ctx context.Context, db execer, query string, args ...any) (sql.Result, error) {
	if p != nil {
		if st := p.prepared(ctx, query); st != nil { //nolint:sqlclosecheck // closed by close
			return st.ExecContext(ctx, args...)
		}
	}
	return db.ExecContext(ctx, query, args...)
}

func (p *poolStmts) prepared(ctx context.Context, query string) *sql.Stmt {
	p.mu.Lock()
	defer p.mu.Unlock()
	if st, ok := p.stmts[query]; ok {
		return st
	}
	st, err := p.db.PrepareContext(ctx, query) //nolint:sqlclosecheck // closed by close
	if err != nil {
		return nil
	}
	if p.stmts == nil {
		p.stmts = make(map[string]*sql.Stmt)
	}
	p.stmts[query] = st
	return st
}

func (p *poolStmts) close() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, st := range p.stmts {
		_ = st.Close()
	}
	p.stmts = nil
}
