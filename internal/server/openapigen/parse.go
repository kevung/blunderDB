// Package openapigen builds a machine-readable model of blunderDB's /v1 API
// surface (method, pattern, request/response shape) by parsing the Go source
// of internal/server's handlers_*.go files — not by reflecting on a running
// Server, since the generic rpc[Req,Resp]/rpcVoid[Req]/rpcStream[Req,T]
// builders erase their type parameters from a compiled binary's
// runtime-visible function names (Go's dictionary-based generics render
// them as "rpc[...]", not the real instantiated types). Source-level
// analysis is the only way to recover Req/Resp without hand-maintaining a
// second, parallel declaration of every route's types next to the real one.
//
// cmd/openapi-gen drives this package to (re)generate openapi.yaml (repo
// root) and doc/source/api_reference.rst; openapigen_test.go's
// TestGeneratedFilesAreUpToDate is the non-drift guard both files' package
// doc comments point back to.
package openapigen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
)

// Route is one parsed /v1 domain route.
type Route struct {
	Method  string // "GET" | "POST"
	Pattern string // "/v1/positions.save"
	Family  string // "positions"
	Op      string // "save"

	// Kind is "json" (rpc/rpcVoid: decode Req, encode Resp as one JSON
	// object), "stream" (rpcStream: decode Req, stream NDJSON items of
	// ItemType), or "custom" (a hand-written handler — schema unknown from
	// source alone, e.g. a file upload/download or a loop-registered
	// family sharing one handler function).
	Kind string

	// ReqType/RespType/ItemType are Go type expressions rendered to source
	// text (e.g. "*domain.Position", "[]int64", "struct{}"), empty when not
	// applicable to Kind. ItemType is the streamed element type for
	// Kind == "stream".
	ReqType  string
	RespType string
	ItemType string

	// IdempotencyKeySupported is true for the routes whose handler is
	// wrapped with internal/server's withIdempotency (#236: collections.create,
	// tournaments.create, anki.reviewCard, at the time of writing) — detected
	// structurally (the wrapper call itself), not guessed, so this can never
	// silently drift from the real code the way a hand-maintained list would.
	IdempotencyKeySupported bool
}

const (
	kindJSON   = "json"
	kindStream = "stream"
	kindCustom = "custom"
)

// typeInfo is what Parse collects about every top-level type declaration in
// the scanned files: struct fields (for named Req/Resp structs) and, for a
// `type X = iter.Seq2[T, error]` alias, the element type text (for
// resolving an rpcStream route whose return type is the alias name, not the
// literal iter.Seq2[...] expression).
type typeInfo struct {
	name       string
	fields     []Field // nil for a non-struct type (e.g. an iterX alias)
	seq2Elem   string  // non-empty for a `type X = iter.Seq2[T, error]` alias
	isNamedMap bool    // "type X map[K]V" — rendered as an object schema with additionalProperties
	underlying string  // source text of the RHS, used for a non-struct, non-alias named type (e.g. a []string or map[K]V based type)
}

// Field is one struct field relevant to a JSON schema: its wire name (the
// json tag, or the Go field name when untagged) and its Go type rendered to
// source text.
type Field struct {
	Name     string // JSON wire name
	GoType   string // e.g. "*domain.Position", "int64", "[]int64"
	Optional bool   // json tag carried ",omitempty", or the field is a pointer
}

// Model is the fully parsed result: every /v1 route plus every named type
// referenced by one, ready for a renderer.
type Model struct {
	Routes []Route
	Types  map[string]typeInfo
}

// familyOf splits "/v1/positions.save" into ("positions", "save"), and
// "/ops/tenant.purge" into ("tenant", "purge").
func familyOf(pattern string) (family, op string) {
	p := strings.TrimPrefix(strings.TrimPrefix(pattern, "/v1/"), "/ops/")
	if i := strings.Index(p, "."); i >= 0 {
		return p[:i], p[i+1:]
	}
	return p, ""
}

// Parse walks every handlers_*.go file (handlers_*_test.go excluded) under
// dir (internal/server) and returns the parsed Model.
func Parse(dir string) (*Model, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "handlers_*.go"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var srcPaths []string
	for _, p := range paths {
		if !strings.HasSuffix(p, "_test.go") {
			srcPaths = append(srcPaths, p)
		}
	}

	fset := token.NewFileSet()
	files := make(map[string]*ast.File, len(srcPaths))
	for _, p := range srcPaths {
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("openapigen: parse %s: %w", p, err)
		}
		files[p] = f
	}

	types := make(map[string]typeInfo)
	// Pass 1: collect every named type declaration across every file, so a
	// route in one file can reference a struct or alias declared in
	// another (e.g. handlers_iters.go's iterX aliases, handlers_rpc.go's
	// okResp/idResp/idReq).
	for _, f := range files {
		collectTypeDecls(f, types)
	}

	// Pass 2 (also needed before route extraction): resolve
	// uploadRoutes-style "for _, u := range XS { rs = append(rs, route{...,
	// u.pattern, ...}) }" loops, which have no string literal pattern to
	// read directly.
	loopPatterns := make(map[string][]string) // range-variable slice name -> patterns
	for _, f := range files {
		collectLoopSlicePatterns(f, loopPatterns)
	}

	var routes []Route
	for path, f := range files {
		rs, err := extractRoutes(f, types, loopPatterns)
		if err != nil {
			return nil, fmt.Errorf("openapigen: %s: %w", path, err)
		}
		routes = append(routes, rs...)
	}

	sort.Slice(routes, func(i, j int) bool { return routes[i].Pattern < routes[j].Pattern })
	return &Model{Routes: routes, Types: types}, nil
}
