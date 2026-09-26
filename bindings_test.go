package main

import (
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/internal/gui"
	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// TestBoundMethodsHaveABindableSignature guards what Wails silently mishandles:
// BoundMethod.Call implements only (T) and (T, error). Three returns resolve
// the promise with null; a (T, U) drops U. Checked on App, Database and Config.
//
// The exceptions are exported Go-side helpers the frontend never calls, named
// one by one so the list cannot grow by accident.
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
