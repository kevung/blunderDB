package openapigen

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// isRouteArrayType reports whether t is "[]route" (routes.go's own
// []route{ {...}, {...} } slice literals).
func isRouteArrayType(t ast.Expr) bool {
	at, ok := t.(*ast.ArrayType)
	if !ok {
		return false
	}
	id, ok := at.Elt.(*ast.Ident)
	return ok && id.Name == "route"
}

// isRouteIdentType reports whether t is the bare "route" identifier — the
// explicit `route{...}` composite literal shape handlers_imports.go uses
// inside append() calls, as opposed to an elided-type element of a
// "[]route{...}" slice literal.
func isRouteIdentType(t ast.Expr) bool {
	id, ok := t.(*ast.Ident)
	return ok && id.Name == "route"
}

// extractRoutes walks f for every route{method, pattern, handler} literal
// with a string-literal pattern (the overwhelming majority) plus, via
// extractLoopRoutes, the one shape in this codebase that registers routes
// from a package-level table in a loop (handlers_imports.go's
// uploadRoutes/ingestRoutes — see collectLoopSlicePatterns).
func extractRoutes(f *ast.File, types map[string]typeInfo, loopPatterns map[string][]string) ([]Route, error) {
	var routes []Route
	var firstErr error

	ast.Inspect(f, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok || cl.Type == nil {
			return true
		}
		switch {
		case isRouteArrayType(cl.Type):
			for _, elt := range cl.Elts {
				eltLit, ok := elt.(*ast.CompositeLit)
				if !ok {
					continue
				}
				r, ok, err := routeFromLit(eltLit, types)
				if err != nil {
					firstErr = err
					return false
				}
				if ok {
					routes = append(routes, r)
				}
			}
			return false
		case isRouteIdentType(cl.Type):
			r, ok, err := routeFromLit(cl, types)
			if err != nil {
				firstErr = err
				return false
			}
			if ok {
				routes = append(routes, r)
			}
			return true
		}
		return true
	})
	if firstErr != nil {
		return nil, firstErr
	}

	routes = append(routes, extractLoopRoutes(f, loopPatterns)...)
	return routes, nil
}

// routeFromLit builds a Route from a 3-element {method, pattern, handler}
// composite literal. ok is false (with no error) for a shape this parser
// deliberately does not resolve here — a non-string pattern (handled by
// extractLoopRoutes instead) or a pattern outside the /v1 domain surface
// (routes.go's own /healthz, /readyz, /metrics registrations, which Paths()
// itself also excludes — see that function's doc comment).
func routeFromLit(lit *ast.CompositeLit, types map[string]typeInfo) (Route, bool, error) {
	if len(lit.Elts) != 3 {
		return Route{}, false, nil
	}
	method := methodFromExpr(lit.Elts[0])
	if method == "" {
		return Route{}, false, nil
	}
	bl, ok := lit.Elts[1].(*ast.BasicLit)
	if !ok || bl.Kind != token.STRING {
		return Route{}, false, nil
	}
	pattern, err := strconv.Unquote(bl.Value)
	if err != nil {
		return Route{}, false, err
	}
	if !isAPIPattern(pattern) {
		return Route{}, false, nil
	}

	family, op := familyOf(pattern)
	r := Route{Method: method, Pattern: pattern, Family: family, Op: op}
	r.Kind, r.ReqType, r.RespType, r.ItemType, r.IdempotencyKeySupported = classifyHandler(lit.Elts[2], types)
	return r, true, nil
}

// methodFromExpr resolves an http.MethodX selector to its string value
// ("POST", "GET", ...) generically: http's Method constants are always
// named "Method"+strings.ToUpper(verb) and hold exactly that uppercase verb.
func methodFromExpr(e ast.Expr) string {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok || id.Name != "http" {
		return ""
	}
	return strings.ToUpper(strings.TrimPrefix(sel.Sel.Name, "Method"))
}

// classifyHandler inspects a route's handler expression and reports its
// Kind, where knowable from source alone its Req/Resp/streamed-item type
// text, and whether it is wrapped with withIdempotency.
func classifyHandler(e ast.Expr, types map[string]typeInfo) (kind, req, resp, item string, idempotencyKey bool) {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return kindCustom, "", "", "", false
	}
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		kind, req, resp, item = classifyRPCCall(fn.Name, call, types)
		return kind, req, resp, item, false
	case *ast.SelectorExpr:
		// s.withIdempotency(INNER) (#236): unwrap one level so the route's
		// real shape is still recognised instead of being masked by the
		// wrapper — this is the only such wrapper in the codebase today.
		if fn.Sel.Name == "withIdempotency" && len(call.Args) == 1 {
			kind, req, resp, item, _ = classifyHandler(call.Args[0], types)
			return kind, req, resp, item, true
		}
		return kindCustom, "", "", "", false
	default:
		return kindCustom, "", "", "", false
	}
}

// classifyRPCCall handles a call whose Fun is a bare identifier: "rpc",
// "rpcVoid", "rpcStream" (handlers_rpc.go's generic builders) each take
// exactly one argument, a func literal — anything else (a hand-written
// package-level function used directly as a route's third element, however
// unlikely) falls back to kindCustom rather than guessing.
func classifyRPCCall(name string, call *ast.CallExpr, types map[string]typeInfo) (kind, req, resp, item string) {
	if len(call.Args) != 1 {
		return kindCustom, "", "", ""
	}
	fn, ok := call.Args[0].(*ast.FuncLit)
	if !ok {
		return kindCustom, "", "", ""
	}
	switch name {
	case "rpc":
		return kindJSON, lastParamType(fn), firstResultType(fn), ""
	case "rpcVoid":
		// rpcVoid's response is always handlers_rpc.go's okResp{"ok":true}
		// — see that file's doc comment.
		return kindJSON, lastParamType(fn), "okResp", ""
	case "rpcStream":
		return kindStream, lastParamType(fn), "", streamItemType(fn, types)
	default:
		return kindCustom, "", "", ""
	}
}

// lastParamType returns the source text of a func literal's last parameter
// type — every rpc/rpcVoid/rpcStream closure in this codebase is
// func(ctx context.Context, scope string, req ReqType) ..., so the last
// parameter is always the request type regardless of its (irrelevant here)
// name.
func lastParamType(fn *ast.FuncLit) string {
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return ""
	}
	last := fn.Type.Params.List[len(fn.Type.Params.List)-1]
	return exprText(last.Type)
}

// firstResultType returns the source text of a func literal's first result
// type — an rpc closure returns (Resp, error), so the first result is
// always the response type.
func firstResultType(fn *ast.FuncLit) string {
	if fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return ""
	}
	return exprText(fn.Type.Results.List[0].Type)
}

// streamItemType resolves an rpcStream closure's single result type — a
// literal iter.Seq2[T, error], or an alias name resolving to one (see
// handlers_iters.go, collectTypeDecls) — to T. Falls back to rendering the
// return type verbatim if it is neither, so a route added in an
// unanticipated shape still shows *something* rather than nothing.
func streamItemType(fn *ast.FuncLit, types map[string]typeInfo) string {
	if fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return ""
	}
	rt := fn.Type.Results.List[0].Type
	if ile, ok := rt.(*ast.IndexListExpr); ok {
		if sel, ok := ile.X.(*ast.SelectorExpr); ok && sel.Sel.Name == "Seq2" && len(ile.Indices) == 2 {
			return exprText(ile.Indices[0])
		}
	}
	if id, ok := rt.(*ast.Ident); ok {
		if info, ok := types[id.Name]; ok && info.seq2Elem != "" {
			return info.seq2Elem
		}
	}
	return exprText(rt)
}

// collectLoopSlicePatterns finds every package-level `var X = []T{ {...},
// {...} }` declaration whose element literals carry at least one string
// literal, and records X -> those strings in out. Deliberately narrow: this
// exists for exactly one shape in this codebase (handlers_imports.go's
// uploadRoutes, `[]struct{ pattern string; format ingest.Format }`) and
// would misfire on an unrelated slice-of-struct that happens to also carry a
// string field — extractLoopRoutes only ever consults an entry here that a
// matching range loop's slice identifier resolves to, so a spurious entry
// for an unrelated variable is harmless (it is simply never looked up).
func collectLoopSlicePatterns(f *ast.File, out map[string][]string) {
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
				continue
			}
			lit, ok := vs.Values[0].(*ast.CompositeLit)
			if !ok {
				continue
			}
			if _, ok := lit.Type.(*ast.ArrayType); !ok {
				continue
			}
			var patterns []string
			for _, elt := range lit.Elts {
				eltLit, ok := elt.(*ast.CompositeLit)
				if !ok {
					continue
				}
				for _, e := range eltLit.Elts {
					bl, ok := e.(*ast.BasicLit)
					if !ok || bl.Kind != token.STRING {
						continue
					}
					if s, err := strconv.Unquote(bl.Value); err == nil {
						patterns = append(patterns, s)
					}
					break
				}
			}
			if len(patterns) > 0 {
				out[vs.Names[0].Name] = patterns
			}
		}
	}
}

// extractLoopRoutes finds `for _, v := range X { ... route{method, v.f,
// handler} ... }` loops whose X is a slice collectLoopSlicePatterns already
// resolved to a set of patterns, and emits one Route per pattern — all
// Kind: kindCustom, since the shared handler's actual behaviour (which
// format it parses) varies per iteration and is not recoverable as a single
// static schema from source alone.
func extractLoopRoutes(f *ast.File, loopPatterns map[string][]string) []Route {
	var routes []Route
	ast.Inspect(f, func(n ast.Node) bool {
		rs, ok := n.(*ast.RangeStmt)
		if !ok {
			return true
		}
		sliceIdent, ok := rs.X.(*ast.Ident)
		if !ok {
			return true
		}
		patterns, ok := loopPatterns[sliceIdent.Name]
		if !ok {
			return true
		}
		valueIdent, ok := rs.Value.(*ast.Ident)
		if !ok {
			return true
		}

		var method string
		ast.Inspect(rs.Body, func(n2 ast.Node) bool {
			lit, ok := n2.(*ast.CompositeLit)
			if !ok || !isRouteIdentType(lit.Type) || len(lit.Elts) != 3 {
				return true
			}
			sel, ok := lit.Elts[1].(*ast.SelectorExpr)
			if !ok {
				return true
			}
			xIdent, ok := sel.X.(*ast.Ident)
			if !ok || xIdent.Name != valueIdent.Name {
				return true
			}
			method = methodFromExpr(lit.Elts[0])
			return false
		})
		if method == "" {
			return true
		}
		for _, p := range patterns {
			if !isAPIPattern(p) {
				continue
			}
			family, op := familyOf(p)
			routes = append(routes, Route{Method: method, Pattern: p, Family: family, Op: op, Kind: kindCustom})
		}
		return true
	})
	return routes
}

// isAPIPattern reports whether a registered pattern belongs to the documented
// surface: the /v1 tenant routes and the /ops/ operator family (G.5, #233).
// The probes (/healthz, /readyz, /metrics) are not — they answer plain text or
// the Prometheus exposition format, neither of which the contract describes.
func isAPIPattern(pattern string) bool {
	return strings.HasPrefix(pattern, "/v1/") || strings.HasPrefix(pattern, "/ops/")
}
