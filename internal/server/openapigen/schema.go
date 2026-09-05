package openapigen

import (
	"sort"
	"strconv"
	"strings"
)

// Schema is a minimal JSON-schema-shaped node — just enough of OpenAPI 3's
// vocabulary to describe every Req/Resp/streamed-item shape this codebase's
// Req/Resp structs actually use (named structs, pointers, slices, maps, and
// the handful of Go builtin scalar types), not a general-purpose OpenAPI
// schema builder.
type Schema struct {
	Ref                  string // "#/components/schemas/X"; when set, every other field is ignored by the renderer
	Type                 string // "object" | "array" | "string" | "integer" | "number" | "boolean"
	Format               string // e.g. "int64", "double"
	Nullable             bool
	Items                *Schema
	Properties           map[string]*Schema
	PropOrder            []string // Properties' keys, in field declaration order (maps don't remember it)
	AdditionalProperties *Schema  // set for a map[K]V type
	Description          string
}

// Components accumulates every named schema referenced by at least one
// route, keyed by its OpenAPI component name — sanitized from the Go type
// name (a foreign qualified name like "domain.Position" becomes
// "domain_Position", since "." is not legal in a component name/JSON
// pointer segment the way OpenAPI tooling expects it).
type Components struct {
	schemas map[string]*Schema
}

func newComponents() *Components {
	return &Components{schemas: make(map[string]*Schema)}
}

// Names returns every registered component name, sorted.
func (c *Components) Names() []string {
	names := make([]string, 0, len(c.schemas))
	for n := range c.schemas {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (c *Components) Get(name string) *Schema { return c.schemas[name] }

// componentName sanitizes a Go type name for use as an OpenAPI component
// name: "domain.Position" -> "domain_Position". A bare local type name
// (no ".") is left as-is.
func componentName(goName string) string {
	return strings.ReplaceAll(goName, ".", "_")
}

// resolveSchema converts a Go type's source text (as exprText renders it —
// "*domain.Position", "[]int64", "map[int64]int", "struct{}", "okResp", …)
// into a Schema, registering any named struct it (transitively) references
// into comps. typeText == "" (no request/response at all — a body-less
// call, or a route this parser could not classify) returns nil.
func resolveSchema(typeText string, types map[string]typeInfo, comps *Components) *Schema {
	t := strings.TrimSpace(typeText)
	if t == "" {
		return nil
	}

	if strings.HasPrefix(t, "*") {
		inner := resolveSchema(t[1:], types, comps)
		if inner == nil {
			return nil
		}
		// A $ref cannot carry sibling keywords in strict JSON Schema, but
		// OpenAPI tooling commonly tolerates "nullable" alongside "$ref";
		// simplicity here over strict spec purity — this is a generated
		// reference document, not a schema a validator gates a build on.
		cp := *inner
		cp.Nullable = true
		return &cp
	}
	if strings.HasPrefix(t, "[]") {
		return &Schema{Type: "array", Items: resolveSchema(t[2:], types, comps)}
	}
	if strings.HasPrefix(t, "map[") {
		if end := strings.Index(t, "]"); end > 0 {
			valueType := t[end+1:]
			return &Schema{Type: "object", AdditionalProperties: resolveSchema(valueType, types, comps)}
		}
	}
	if t == "struct{}" {
		return &Schema{Type: "object"}
	}
	if s, ok := primitiveSchema(t); ok {
		return s
	}

	// A qualified name (domain.Position, storage.Collection, race.Eval, …):
	// this package never parses the referenced package, so it is an opaque
	// object stub, registered once.
	if strings.Contains(t, ".") {
		name := componentName(t)
		if comps.Get(name) == nil {
			comps.schemas[name] = &Schema{
				Type:        "object",
				Description: "Opaque: see the Go type " + t + " (defined outside internal/server, not expanded here).",
			}
		}
		return &Schema{Ref: "#/components/schemas/" + name}
	}

	// A bare local identifier: a named struct (positionReq, idResp, …), a
	// type alias, or an unresolved name this parser has never seen (kept as
	// a bare opaque stub rather than failing the whole generation).
	info, known := types[t]
	if !known {
		name := componentName(t)
		if comps.Get(name) == nil {
			comps.schemas[name] = &Schema{Type: "object", Description: "Unresolved local type " + t + "."}
		}
		return &Schema{Ref: "#/components/schemas/" + name}
	}
	if info.underlying != "" {
		return resolveSchema(info.underlying, types, comps)
	}

	name := componentName(t)
	if comps.Get(name) != nil {
		return &Schema{Ref: "#/components/schemas/" + name} // already built (or being built — breaks a cycle)
	}
	// Register a placeholder first so a field that refers back to this same
	// type (however unlikely in this codebase) resolves to a $ref rather
	// than recursing forever.
	comps.schemas[name] = &Schema{Type: "object"}
	props := make(map[string]*Schema, len(info.fields))
	order := make([]string, 0, len(info.fields))
	for _, f := range info.fields {
		fs := resolveSchema(f.GoType, types, comps)
		if fs == nil {
			continue
		}
		if f.Optional {
			cp := *fs
			cp.Nullable = true
			fs = &cp
		}
		props[f.Name] = fs
		order = append(order, f.Name)
	}
	comps.schemas[name] = &Schema{Type: "object", Properties: props, PropOrder: order}
	return &Schema{Ref: "#/components/schemas/" + name}
}

// primitiveSchema maps a Go builtin scalar type name to its OpenAPI
// primitive shape.
func primitiveSchema(t string) (*Schema, bool) {
	switch t {
	case "string":
		return &Schema{Type: "string"}, true
	case "bool":
		return &Schema{Type: "boolean"}, true
	case "float32":
		return &Schema{Type: "number", Format: "float"}, true
	case "float64":
		return &Schema{Type: "number", Format: "double"}, true
	case "int", "int8", "int16", "int32", "uint", "uint8", "uint16", "uint32", "uintptr", "byte", "rune":
		return &Schema{Type: "integer", Format: "int32"}, true
	case "int64", "uint64":
		return &Schema{Type: "integer", Format: "int64"}, true
	case "any", "interface{}":
		return &Schema{Type: "object", Description: "Arbitrary JSON value."}, true
	}
	return nil, false
}

// quoteYAML renders a Go string as a YAML double-quoted scalar — used by
// the openapi.yaml renderer for every free-text value (descriptions can
// legitimately contain ':', '#', newlines, …, all of which are unsafe in
// YAML's unquoted plain scalar style).
func quoteYAML(s string) string {
	return strconv.Quote(s)
}
