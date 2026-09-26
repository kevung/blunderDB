package openapigen

import (
	"go/ast"
	"go/types"
)

// exprText renders a type expression to Go source text ("*domain.Position",
// "[]int64") syntactically, with no type-checking.
func exprText(e ast.Expr) string {
	if e == nil {
		return ""
	}
	return types.ExprString(e)
}
