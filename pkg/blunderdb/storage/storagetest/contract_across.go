// Reads across tenants (storage/across.go, ADR-0061): the union of single-tenant
// reads, each result tagged with its tenant, and no tenant outside the read set
// ever passed to a store.
//
// Like RunTenantIsolationTests, this suite takes the tenants as arguments: a
// real multi-tenant backend (PostgreSQL) passes two read tenants and an
// outsider, and the suite checks that the outsider's rows never appear. SQLite
// has a single tenant and ignores the scope (storage.go's package doc): it
// passes one read tenant and no outsider, and the suite checks the one thing a
// single-tenant read set promises — that it reads exactly what a plain read of
// that tenant reads, tagged with it. A read set never widens SQLite.
package storagetest

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// RunReadAcrossTests runs the read-across checks against a fresh Storage from
// factory. read is the read set's listed tenants (the first one plays the
// writing tenant); outsider, when not empty, is a tenant that holds rows of
// its own and is left out of the set.
func RunReadAcrossTests(t *testing.T, factory func() storage.Storage, read []string, outsider string) {
	t.Helper()
	if len(read) == 0 {
		t.Fatal("RunReadAcrossTests needs at least one read tenant")
	}
	ctx := context.Background()

	s := factory()
	defer s.Close()

	// Every direct call runs under its tenant's context, as the daemon's do,
	// so the suite also holds under PostgreSQL row-level security, where a
	// call without one sees and writes nothing.
	in := func(scope string) context.Context {
		n, err := storage.ParseTenant(scope)
		if err != nil {
			t.Fatalf("tenant %q: %v", scope, err)
		}
		return storage.WithTenant(ctx, n)
	}

	// One match and one position per tenant; the player name says the tenant.
	owners := append(append([]string{}, read...), outsider)
	if outsider == "" {
		owners = owners[:len(read)]
	}
	for _, scope := range owners {
		m := domain.Match{Player1Name: "player-" + scope, Player2Name: "rival", MatchLength: 5}
		if _, err := s.Matches().Save(in(scope), scope, &m); err != nil {
			t.Fatalf("Save match(%s): %v", scope, err)
		}
		p := checkerPos()
		if _, err := s.Positions().Save(in(scope), scope, &p); err != nil {
			t.Fatalf("Save position(%s): %v", scope, err)
		}
	}

	set, err := storage.NewReadTenants(read[0], read[1:])
	if err != nil {
		t.Fatalf("NewReadTenants: %v", err)
	}

	t.Run("MatchesListTagsEachTenant", func(t *testing.T) {
		got := map[string][]string{}
		for tg, err := range storage.StreamAcross(ctx, set, func(ctx context.Context, scope string) iter.Seq2[*domain.Match, error] {
			return s.Matches().List(ctx, scope, storage.MatchListOpts{})
		}) {
			if err != nil {
				t.Fatalf("StreamAcross matches: %v", err)
			}
			got[tg.Tenant] = append(got[tg.Tenant], tg.Item.Player1Name)
		}
		for _, scope := range read {
			want := 0
			for _, err := range s.Matches().List(in(scope), scope, storage.MatchListOpts{}) {
				if err != nil {
					t.Fatalf("List(%s): %v", scope, err)
				}
				want++
			}
			if want == 0 {
				t.Errorf("tenant %s: its own match is not read back", scope)
			}
			if len(got[scope]) != want {
				t.Errorf("tenant %s: %d match(es) across, %d on its own", scope, len(got[scope]), want)
			}
		}
		if outsider != "" {
			if n := len(got[outsider]); n != 0 {
				t.Errorf("outsider %s is read: %d match(es)", outsider, n)
			}
			for tenant, names := range got {
				for _, name := range names {
					if name == "player-"+outsider {
						t.Errorf("tenant %s's page holds the outsider's match", tenant)
					}
				}
			}
		}
	})

	t.Run("PositionsFindTagsEachTenant", func(t *testing.T) {
		got := map[string]int{}
		for tg, err := range storage.StreamAcross(ctx, set, func(ctx context.Context, scope string) iter.Seq2[*domain.Position, error] {
			return s.Search().Find(ctx, scope, domain.SearchFilters{}, storage.ListOpts{})
		}) {
			if err != nil {
				t.Fatalf("StreamAcross positions: %v", err)
			}
			got[tg.Tenant]++
		}
		for tenant := range got {
			if !set.Contains(tenant) {
				t.Errorf("a position tagged %s, outside the read set %v", tenant, set)
			}
		}
		for _, scope := range read {
			if got[scope] == 0 {
				t.Errorf("tenant %s: no position read across", scope)
			}
		}
	})

	t.Run("PlayerNamesPerTenant", func(t *testing.T) {
		got, err := storage.ReadAcross(ctx, set, func(ctx context.Context, scope string) ([]storage.PlayerFrequency, error) {
			return s.Stats().PlayerNames(ctx, scope)
		})
		if err != nil {
			t.Fatalf("ReadAcross: %v", err)
		}
		if len(got) != len(set) {
			t.Fatalf("%d answers for %d read tenants", len(got), len(set))
		}
		for i, tg := range got {
			if tg.Tenant != set[i] {
				t.Errorf("answer %d tagged %s, want %s (the set's order)", i, tg.Tenant, set[i])
			}
			for _, pf := range tg.Item {
				if outsider != "" && pf.Name == "player-"+outsider {
					t.Errorf("tenant %s's player names hold the outsider's player", tg.Tenant)
				}
			}
		}
	})

	t.Run("ReadOneRefusesAnUnlistedTenant", func(t *testing.T) {
		stranger := outsider
		if stranger == "" {
			stranger = "999"
		}
		called := false
		_, err := storage.ReadOne(ctx, set, stranger, func(ctx context.Context, scope string) ([]storage.PlayerFrequency, error) {
			called = true
			return s.Stats().PlayerNames(ctx, scope)
		})
		if !errors.Is(err, storage.ErrInvalid) {
			t.Errorf("ReadOne(%s): err = %v, want ErrInvalid", stranger, err)
		}
		if called {
			t.Errorf("ReadOne(%s) called the store for a tenant outside the set", stranger)
		}
	})
}
