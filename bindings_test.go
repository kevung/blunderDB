package main

import (
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/internal/gui"
	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// TestBoundMethodsHaveABindableSignature guards what Wails cannot express and
// does not complain about.
//
// wails/v2 `internal/binding.BoundMethod.Call` switches on the number of
// return values and implements ONE and TWO. A method returning three values
// falls through that switch: the promise resolves with `null`, no error is
// raised anywhere, and the generated `.d.ts` types the call after the first
// return value alone — so the frontend destructures null and the feature is
// dead with a caught exception for a trace. `Config.GetWatchFolder` shipped
// that way in #258, and only svelte-check saw it, from underneath a budget of
// three thousand pre-existing errors.
//
// The two-value case has the same shape of trap: `Call` reads the first value
// and expects the SECOND to be an error, so a `(T, U)` method silently drops
// U while the generated typing promises it.
//
// The bound types are the ones main.go hands to gui.Run: the App, the legacy
// Database wrapper and the persisted Config.
//
// The two exceptions are Go-side helpers that happen to be exported: the
// frontend never calls them, and reshaping their Go signature for a binding
// nobody uses would churn the CLI and the daemon for nothing. They are named
// here rather than tolerated by a rule, so the list cannot grow by accident.
var bindableSignatureExceptions = map[string]string{
	"LoadPositionsByFiltersCore":    "Go-side helper; the GUI calls LoadPositionsByFilters",
	"LoadPositionsByFiltersCoreCtx": "Go-side helper; the GUI calls LoadPositionsByFilters",
}

func TestBoundMethodsHaveABindableSignature(t *testing.T) {
	errType := reflect.TypeOf((*error)(nil)).Elem()

	for _, bound := range []any{&gui.App{}, &database.Database{}, &Config{}} {
		typ := reflect.TypeOf(bound)
		for i := 0; i < typ.NumMethod(); i++ {
			m := typ.Method(i)
			if _, ok := bindableSignatureExceptions[m.Name]; ok {
				continue
			}
			switch n := m.Type.NumOut(); {
			case n <= 1:
			case n == 2:
				if !m.Type.Out(1).Implements(errType) {
					t.Errorf("%s.%s returns (%s, %s): Wails keeps only the first and expects the second to be an error",
						typ, m.Name, m.Type.Out(0), m.Type.Out(1))
				}
			default:
				t.Errorf("%s.%s returns %d values: Wails binds one, or one and an error, and resolves the promise with null for anything else — return a struct",
					typ, m.Name, n)
			}
		}
	}
}
