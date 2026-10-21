package sqlshared

// SetAfterMatchStatsProbe installs f between a reader's completeness probe
// and its read, and returns what restores the previous hook.
func SetAfterMatchStatsProbe(f func()) (restore func()) {
	prev := afterMatchStatsProbe
	afterMatchStatsProbe = f
	return func() { afterMatchStatsProbe = prev }
}

// CubeMultiplierExpr is cubeMultiplierExpr, over aliases p and mv.
var CubeMultiplierExpr = cubeMultiplierExpr

// SetBeforeMatchStatsFill installs f where a fill finds matches to compute,
// and returns what restores the previous hook.
func SetBeforeMatchStatsFill(f func(missing int)) (restore func()) {
	prev := beforeMatchStatsFill
	beforeMatchStatsFill = f
	return func() { beforeMatchStatsFill = prev }
}
