package openapigen

import (
	"go/ast"
	"go/token"
	"reflect"
	"strconv"
	"strings"
)

// collectTypeDecls records every top-level `type` declaration in f into
// types, keyed by name. Called once per file; the caller merges across every
// file before any route tries to resolve a type by name (a route in
// handlers_collections.go can reference an alias declared in
// handlers_iters.go).
func collectTypeDecls(f *ast.File, types map[string]typeInfo) {
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			info := typeInfo{name: ts.Name.Name}
			switch t := ts.Type.(type) {
			case *ast.StructType:
				info.fields = fieldsOf(t)
			case *ast.IndexListExpr:
				// type X = iter.Seq2[T, error] — the shape every alias in
				// handlers_iters.go uses.
				if sel, ok := t.X.(*ast.SelectorExpr); ok && sel.Sel.Name == "Seq2" && len(t.Indices) == 2 {
					info.seq2Elem = exprText(t.Indices[0])
				} else {
					info.underlying = exprText(t)
				}
			default:
				info.underlying = exprText(ts.Type)
				if _, ok := ts.Type.(*ast.MapType); ok {
					info.isNamedMap = true
				}
			}
			types[ts.Name.Name] = info
		}
	}
}

// fieldsOf extracts the JSON-relevant fields of a struct type declaration:
// wire name (json tag, or the Go field name lower-cased-first as
// encoding/json would default to when untagged — every Req/Resp struct in
// this codebase tags every field, so the untagged fallback is a safety net,
// not the common case) and Go type text.
func fieldsOf(st *ast.StructType) []Field {
	var fields []Field
	if st.Fields == nil {
		return fields
	}
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			continue // an embedded field; none of this package's Req/Resp structs embed one
		}
		goType := exprText(f.Type)
		_, isPointer := f.Type.(*ast.StarExpr)
		for _, name := range f.Names {
			if !name.IsExported() {
				continue
			}
			wireName := name.Name
			optional := isPointer
			if f.Tag != nil {
				if tagVal, err := strconv.Unquote(f.Tag.Value); err == nil {
					tag := reflect.StructTag(tagVal)
					if jsonTag, ok := tag.Lookup("json"); ok {
						parts := strings.Split(jsonTag, ",")
						if parts[0] == "-" {
							continue
						}
						if parts[0] != "" {
							wireName = parts[0]
						}
						for _, opt := range parts[1:] {
							if opt == "omitempty" {
								optional = true
							}
						}
					}
				}
			}
			fields = append(fields, Field{Name: wireName, GoType: goType, Optional: optional})
		}
	}
	return fields
}
