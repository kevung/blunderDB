package database

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// lockGuardTimeout bounds one public method call. Every method of Database
// takes d.mu (a sync.RWMutex, which is NOT re-entrant), so a method that
// calls another public method deadlocks on itself. Nothing in the type
// system enforces the "a public method never calls another public method"
// convention: this test does, by calling every exported method on a small
// database and refusing to wait longer than this.
//
// Five seconds is ~100× the slowest legitimate call on these databases, so a
// hit means a wedge, not a slow machine.
const lockGuardTimeout = 5 * time.Second

// lockGuardSkips names the exported methods this guard does not call, with the
// reason. Keep it short: every entry is a method the guard no longer covers.
var lockGuardSkips = map[string]string{
	// Nothing so far: every public method returns on the two databases below
	// once file-path arguments point into a temporary directory (see
	// lockGuardOverrides). Add a method here ONLY when it needs a resource the
	// test cannot provide (network, Wails runtime, an interactive dialog…), and
	// say which one.
}

// lockGuardOverrides supplies explicit arguments for the methods whose
// zero-valued arguments would not exercise the locked section meaningfully
// (an empty path is refused before any lock is taken) or would write outside
// the test's temporary directory.
var lockGuardOverrides = map[string]func(t *testing.T) []any{
	"SetupDatabase": func(*testing.T) []any { return []any{":memory:"} },
	"OpenDatabase":  func(*testing.T) []any { return []any{":memory:"} },
	"ExportDatabase": func(t *testing.T) []any {
		return []any{ExportOptions{
			ExportPath:           filepath.Join(tempDir(t), "export.db"),
			PositionIDs:          []int64{1, 2},
			IncludeAnalysis:      true,
			IncludeComments:      true,
			IncludeFilterLibrary: true,
			IncludePlayedMoves:   true,
			IncludeMatches:       true,
			IncludeCollections:   true,
			CollectionIDs:        []int64{1},
			TournamentIDs:        []int64{1},
		}}
	},
	"ExportCollections": func(t *testing.T) []any {
		return []any{filepath.Join(tempDir(t), "collections.db"), []int64{1}, map[string]string{}, true, true, "", ""}
	},
	"ExportTournaments": func(t *testing.T) []any {
		return []any{filepath.Join(tempDir(t), "tournaments.db"), []int64{1}, map[string]string{}, true, true, "", ""}
	},
	"ExportMatchMAT": func(t *testing.T) []any {
		return []any{int64(1), filepath.Join(tempDir(t), "match.mat")}
	},
}

// TestPublicMethods_ReturnUnderLock calls every exported method of *Database
// with minimal arguments and fails, naming the method, when one does not
// return within lockGuardTimeout. Return values — errors included — are
// ignored: what is under test is that the call comes back, i.e. that no
// method takes d.mu twice.
//
// Each method runs twice, on a fresh database each time: once on an empty
// database (the "nothing found" branches) and once on a copy of a
// populated file (positions, analyses, match, collection, tournament, deck —
// the "found" branches, where a nested call is likelier). Integer arguments
// are 1, so they hit the fixture's first row.
//
// Run it under -race as well: a method that reaches d.db or d.store without
// the lock shows up there, not here.
func TestPublicMethods_ReturnUnderLock(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeOf(&Database{})
	if typ.NumMethod() < 100 {
		t.Fatalf("only %d exported methods found on *Database, reflection is not seeing the whole type", typ.NumMethod())
	}
	for name := range lockGuardSkips {
		if _, ok := typ.MethodByName(name); !ok {
			// A skip entry naming no method is a typo hiding a method the guard
			// should cover.
			t.Errorf("lockGuardSkips names %q, which is not a method of *Database", name)
		}
	}

	// Both fixtures are built once and copied per method: under -race a
	// SetupDatabase (bootstrap, VACUUM, integrity check) costs ~0.1 s, an
	// OpenDatabase of a copied file a third of that, and there are 2 × 139
	// of them.
	variants := []struct {
		name    string
		fixture []byte
	}{
		{"empty", lockGuardEmptyFixture(t)},
		{"populated", lockGuardPopulatedFixture(t)},
	}

	for _, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			for i := 0; i < typ.NumMethod(); i++ {
				method := typ.Method(i)
				t.Run(method.Name, func(t *testing.T) {
					if reason, ok := lockGuardSkips[method.Name]; ok {
						t.Skip(reason)
					}
					path := filepath.Join(tempDir(t), variant.name+".db")
					if err := os.WriteFile(path, variant.fixture, 0o600); err != nil {
						t.Fatal(err)
					}
					db := NewDatabase()
					if err := db.OpenDatabase(path); err != nil {
						t.Fatalf("OpenDatabase(%s): %v", path, err)
					}
					closeOnCleanup(t, db)
					callWithinTimeout(t, db, method)
				})
			}
		})
	}
}

// lockGuardEmptyFixture builds a freshly bootstrapped, empty database once
// and returns its bytes.
func lockGuardEmptyFixture(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join(tempDir(t), "empty.db")
	db := NewDatabase()
	if err := db.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	closeOnCleanup(t, db)
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// lockGuardPopulatedFixture builds the populated database once and returns
// its bytes, so each method can open a private copy (methods mutate:
// DeleteMatch, ResetAnkiDeck, Close…). It extends setupExportTestDB's contents — two
// analysed positions with comments, a match, a collection, a tournament —
// with a synced deck, a saved filter, a session state and some history.
func lockGuardPopulatedFixture(t *testing.T) []byte {
	t.Helper()
	db, dir, _ := setupExportTestDB(t)
	deckID, err := db.CreateAnkiDeck("guard deck", "", "collection", 1, "")
	if err != nil {
		t.Fatalf("CreateAnkiDeck: %v", err)
	}
	if err := db.SyncAnkiDeck(deckID); err != nil {
		t.Fatalf("SyncAnkiDeck: %v", err)
	}
	if err := db.SaveFilter("guard filter", "s"); err != nil {
		t.Fatalf("SaveFilter: %v", err)
	}
	if err := db.SaveSessionState(SessionState{LastSearchCommand: "s", LastPositionIDs: []int64{1, 2}, HasActiveSearch: true}); err != nil {
		t.Fatalf("SaveSessionState: %v", err)
	}
	if err := db.SaveCommand("s"); err != nil {
		t.Fatalf("SaveCommand: %v", err)
	}
	if err := db.SaveSearchHistory("s", "", ""); err != nil {
		t.Fatalf("SaveSearchHistory: %v", err)
	}
	// Close (not the fixture's cleanup) so the advisory file lock is released
	// and the WAL is folded back into the main file before it is copied.
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "source.db"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// callWithinTimeout calls method on db with lockGuardArgs and fails if the
// call has not returned after lockGuardTimeout. A panic inside the method is
// reported as a failure too (it is not the deadlock this guard looks for, but
// a public method that panics on minimal arguments is a bug in its own right).
func callWithinTimeout(t *testing.T, db *Database, method reflect.Method) {
	t.Helper()

	hung := false
	t.Cleanup(func() {
		// Close takes d.mu: on a hung method it would hang too, and take the
		// whole test binary with it (the go test timeout is ten minutes; the
		// failure would lose the method's name). Leak the handle instead.
		if !hung {
			_ = db.Close()
		}
	})

	args := lockGuardArgs(t, method)
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		reflect.ValueOf(db).MethodByName(method.Name).Call(args)
	}()

	select {
	case recovered := <-done:
		if recovered != nil {
			t.Errorf("Database.%s panicked on minimal arguments: %v", method.Name, recovered)
		}
	case <-time.After(lockGuardTimeout):
		hung = true
		t.Fatalf("Database.%s did not return within %s: deadlock on Database.mu (a public method calling another public method?)", method.Name, lockGuardTimeout)
	}
}

// lockGuardArgs builds the argument list for method: the override when one is
// registered, otherwise one value per parameter from lockGuardZero. The
// receiver is not part of the list (the call goes through a bound method
// value).
func lockGuardArgs(t *testing.T, method reflect.Method) []reflect.Value {
	t.Helper()
	ft := method.Type
	numIn := ft.NumIn() - 1 // drop the receiver
	if build, ok := lockGuardOverrides[method.Name]; ok {
		raw := build(t)
		if len(raw) != numIn {
			t.Fatalf("lockGuardOverrides[%q] supplies %d arguments, method takes %d", method.Name, len(raw), numIn)
		}
		args := make([]reflect.Value, numIn)
		for i, v := range raw {
			want := ft.In(i + 1)
			val := reflect.ValueOf(v)
			if !val.Type().AssignableTo(want) {
				t.Fatalf("lockGuardOverrides[%q] argument %d is %s, method takes %s", method.Name, i, val.Type(), want)
			}
			args[i] = val
		}
		return args
	}
	args := make([]reflect.Value, numIn)
	for i := range args {
		args[i] = lockGuardZero(ft.In(i + 1))
	}
	return args
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()

// lockGuardZero returns a minimal, non-nil-where-it-matters value of type typ:
// integers are 1 (a plausible row id rather than "no row"), containers are
// empty but allocated, pointers point at a zero value, funcs are no-ops, a
// context.Context is Background; everything else is the zero value.
func lockGuardZero(typ reflect.Type) reflect.Value {
	switch typ.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v := reflect.New(typ).Elem()
		v.SetInt(1)
		return v
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v := reflect.New(typ).Elem()
		v.SetUint(1)
		return v
	case reflect.Slice:
		return reflect.MakeSlice(typ, 0, 0)
	case reflect.Map:
		return reflect.MakeMap(typ)
	case reflect.Ptr:
		return reflect.New(typ.Elem())
	case reflect.Func:
		return reflect.MakeFunc(typ, func([]reflect.Value) []reflect.Value {
			out := make([]reflect.Value, typ.NumOut())
			for i := range out {
				out[i] = reflect.Zero(typ.Out(i))
			}
			return out
		})
	case reflect.Interface:
		if typ == contextType {
			return reflect.ValueOf(context.Background())
		}
		return reflect.Zero(typ)
	default:
		return reflect.Zero(typ)
	}
}
