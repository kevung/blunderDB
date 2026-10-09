package domain

// IsResponsePosition reports whether pos is a take/pass decision: a turned
// cube held by no one, which no other decision can carry (a cube above 1 has
// an owner). Importers and transcriptions record the answer to a double
// there, the answerer on roll.
func IsResponsePosition(pos *Position) bool {
	return pos.DecisionType == CubeAction && pos.Cube.Value > 0 && pos.Cube.Owner == None
}
