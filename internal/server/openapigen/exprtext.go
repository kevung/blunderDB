package openapigen

import (
	"go/ast"
	"go/types"
)

// exprText renders a type expression to Go source text — "*domain.Position",
// "[]int64", "struct{}" — using go/types.ExprString, a pure syntactic
// pretty-printer that needs no type-checking (this package never builds a
// go/types.Info; the whole point is to work from source text alone).
func exprText(e ast.Expr) string {
	if e == nil {
		return ""
	}
	return types.ExprString(e)
}
