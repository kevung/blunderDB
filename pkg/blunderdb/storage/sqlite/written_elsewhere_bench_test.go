package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// BenchmarkWatchCost measures what the watch of WrittenElsewhere adds to one
// pooled connection: a read, an autocommit write, a transaction.
func BenchmarkWatchCost(b *testing.B) {
	pools := []struct {
		name string
		open func(path string) (*sql.DB, error)
	}{
		{"plain", func(path string) (*sql.DB, error) { return sql.Open("sqlite", DSN(path)) }},
		{"watched", OpenDB},
	}
	for _, p := range pools {
		b.Run(p.name, func(b *testing.B) {
			db, err := p.open(filepath.Join(b.TempDir(), "bench.db"))
			if err != nil {
				b.Fatal(err)
			}
			defer db.Close()
			db.SetMaxOpenConns(1)
			if _, err := db.Exec(`CREATE TABLE t (x INTEGER)`); err != nil {
				b.Fatal(err)
			}
			b.Run("select", func(b *testing.B) {
				var one int
				for b.Loop() {
					if err := db.QueryRow(`SELECT 1`).Scan(&one); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("insert", func(b *testing.B) {
				for b.Loop() {
					if _, err := db.Exec(`INSERT INTO t (x) VALUES (1)`); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("tx", func(b *testing.B) {
				for b.Loop() {
					tx, err := db.Begin()
					if err != nil {
						b.Fatal(err)
					}
					if _, err := tx.Exec(`INSERT INTO t (x) VALUES (1)`); err != nil {
						b.Fatal(err)
					}
					if err := tx.Commit(); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
