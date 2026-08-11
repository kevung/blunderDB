package ingest

import "testing"

// bgfADoubleResponse (bgf.go) reads the take/pass response that follows an
// explicit "adouble" move in a BGF match's move list. It has 0% coverage: no
// existing test exercises it, and the .bgf match fixtures under testdata/
// don't happen to contain an explicit "adouble" (BGBlitz usually encodes a
// double as a synthetic "amove" with from[0]==-1, handled by the sibling
// bgfAmoveDoubleResponse). These tests build the minimal []interface{}
// movesData shape it walks directly.
func amoveMap(mtype string) map[string]interface{} {
	return map[string]interface{}{"type": mtype}
}

func toMoves(maps ...map[string]interface{}) []interface{} {
	out := make([]interface{}, len(maps))
	for i, m := range maps {
		out[i] = m
	}
	return out
}

func TestBGFADoubleResponse(t *testing.T) {
	cases := []struct {
		name string
		data []interface{}
		idx  int
		want string
	}{
		{
			"immediately followed by atake",
			toMoves(amoveMap("adouble"), amoveMap("atake")),
			0,
			"Double/Take",
		},
		{
			"immediately followed by apass",
			toMoves(amoveMap("adouble"), amoveMap("apass")),
			0,
			"Double/Pass",
		},
		{
			"adouble is the last move in the game (no response recorded): defaults to pass",
			toMoves(amoveMap("adouble")),
			0,
			"Double/Pass",
		},
		{
			"an intervening amove is skipped while looking for the response",
			toMoves(amoveMap("adouble"), amoveMap("amove"), amoveMap("atake")),
			0,
			"Double/Take",
		},
		{
			"multiple intervening amoves are all skipped",
			toMoves(amoveMap("adouble"), amoveMap("amove"), amoveMap("amove"), amoveMap("apass")),
			0,
			"Double/Pass",
		},
		{
			"an unrecognised move type right after adouble stops the scan and defaults to pass",
			toMoves(amoveMap("adouble"), amoveMap("something-else"), amoveMap("atake")),
			0,
			"Double/Pass",
		},
		{
			"adouble not at index 0: only entries after idx are scanned",
			toMoves(amoveMap("amove"), amoveMap("adouble"), amoveMap("atake")),
			1,
			"Double/Take",
		},
		{
			"malformed entry (not a map) among the moves is skipped, not fatal",
			[]interface{}{amoveMap("adouble"), 42, amoveMap("atake")},
			0,
			"Double/Take",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := bgfADoubleResponse(c.data, c.idx); got != c.want {
				t.Errorf("bgfADoubleResponse(idx=%d) = %q, want %q", c.idx, got, c.want)
			}
		})
	}
}
