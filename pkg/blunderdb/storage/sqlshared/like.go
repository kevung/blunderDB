package sqlshared

import "strings"

// ContainsPattern turns user text into a LIKE pattern matching it anywhere,
// its own %, _ and \ escaped for an ESCAPE '\' clause.
func ContainsPattern(text string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(text) + "%"
}
