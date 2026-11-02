package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// BenchmarkSavePositionJournal prices the change journal on the heaviest
// insert: the same Save with the journal triggers in place and with them
// dropped (the uuid trigger stays in both).
func BenchmarkSavePositionJournal(b *testing.B) {
	for _, withJournal := range []bool{true, false} {
		name := "without"
		if withJournal {
			name = "with"
		}
		b.Run(name, func(b *testing.B) {
			ctx := context.Background()
			path := filepath.Join(b.TempDir(), "bench.db")
			s, err := sqlite.Open(ctx, path, nil)
			if err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() { _ = s.Close() })
			if !withJournal {
				raw, err := sql.Open("sqlite", path)
				if err != nil {
					b.Fatal(err)
				}
				if err := sqlite.ForgetChangeLog(ctx, raw); err != nil {
					b.Fatal(err)
				}
				_ = raw.Close()
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := benchPos(i)
				if _, err := s.Positions().Save(ctx, "", &p); err != nil {
					b.Fatalf("Save: %v", err)
				}
			}
		})
	}
}
