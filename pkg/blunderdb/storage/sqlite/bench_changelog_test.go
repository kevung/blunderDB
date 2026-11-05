package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strconv"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
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

// BenchmarkWriteJournalFamilies prices the change journal on the writes of the
// families the changeset carries beyond the position: the same write with the
// journal triggers in place and dropped (the uuid trigger stays in both). A
// review and a draft are the frequent ones; a deck and a tournament are rare.
func BenchmarkWriteJournalFamilies(b *testing.B) {
	writes := map[string]func(ctx context.Context, s *sqlite.Storage, i int) error{
		"deck": func(ctx context.Context, s *sqlite.Storage, i int) error {
			_, err := s.Anki().CreateDeck(ctx, "", "deck "+strconv.Itoa(i), "", "search", 0, "")
			return err
		},
		"tournament": func(ctx context.Context, s *sqlite.Storage, i int) error {
			_, err := s.Tournaments().Create(ctx, "", "tournament "+strconv.Itoa(i), "", "")
			return err
		},
		"draft": func(ctx context.Context, s *sqlite.Storage, i int) error {
			_, err := s.Transcriptions().Save(ctx, "", &storage.Transcription{FormatVersion: "1", Label: "d", Document: `{"moves":[1,2,3,4,5,6,7,8]}`})
			return err
		},
	}
	for name, write := range writes {
		for _, withJournal := range []bool{true, false} {
			mode := "without"
			if withJournal {
				mode = "with"
			}
			b.Run(name+"/"+mode, func(b *testing.B) {
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
					if err := write(ctx, s, i); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// BenchmarkReviewJournal prices the journal on the frequent Anki write, the
// review of a card, whose trigger journals the card as an update.
func BenchmarkReviewJournal(b *testing.B) {
	for _, withJournal := range []bool{true, false} {
		mode := "without"
		if withJournal {
			mode = "with"
		}
		b.Run(mode, func(b *testing.B) {
			ctx := context.Background()
			path := filepath.Join(b.TempDir(), "bench.db")
			s, err := sqlite.Open(ctx, path, nil)
			if err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() { _ = s.Close() })
			var ids []int64
			for i := 0; i < 200; i++ {
				p := benchPos(i)
				id, err := s.Positions().Save(ctx, "", &p)
				if err != nil {
					b.Fatal(err)
				}
				ids = append(ids, id)
			}
			deck, err := s.Anki().CreateDeck(ctx, "", "bench", "", "search", 0, "")
			if err != nil {
				b.Fatal(err)
			}
			if err := s.Anki().SyncWithPositions(ctx, "", deck, ids); err != nil {
				b.Fatal(err)
			}
			var cards []int64
			for c, err := range s.Anki().DeckCards(ctx, "", deck) {
				if err != nil {
					b.Fatal(err)
				}
				cards = append(cards, c.ID)
			}
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
				if _, err := s.Anki().ReviewCard(ctx, "", cards[i%len(cards)], 3); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
