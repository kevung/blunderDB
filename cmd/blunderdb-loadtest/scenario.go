package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// op is one weighted request kind in a scenario.
type op struct {
	name   string
	weight int
	build  func(rng *rand.Rand) (path string, body []byte)
}

// scenarios maps a scenario name to its weighted op mix. Reads are
// list/search/stats; the only write is positions.save.
var scenarios = map[string][]op{
	"mixed": {
		{"positions.list", 40, buildList},
		{"search.find", 25, buildSearch},
		{"stats.compute", 15, buildStats},
		{"positions.save", 20, buildSave},
	},
	"read-heavy": {
		{"positions.list", 50, buildList},
		{"search.find", 30, buildSearch},
		{"stats.compute", 15, buildStats},
		{"positions.save", 5, buildSave},
	},
	"write-heavy": {
		{"positions.list", 20, buildList},
		{"search.find", 10, buildSearch},
		{"stats.compute", 5, buildStats},
		{"positions.save", 65, buildSave},
	},
}

// picker turns a weighted op list into a fast cumulative chooser.
type picker struct {
	ops   []op
	cum   []int
	total int
}

func newPicker(ops []op) *picker {
	p := &picker{ops: ops}
	sum := 0
	for _, o := range ops {
		sum += o.weight
		p.cum = append(p.cum, sum)
	}
	p.total = sum
	return p
}

func (p *picker) pick(rng *rand.Rand) op {
	r := rng.Intn(p.total)
	for i, c := range p.cum {
		if r < c {
			return p.ops[i]
		}
	}
	return p.ops[len(p.ops)-1]
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func buildList(rng *rand.Rand) (string, []byte) {
	return "/v1/positions.list", mustJSON(map[string]int{"limit": 20, "offset": 0})
}

func buildStats(rng *rand.Rand) (string, []byte) {
	return "/v1/stats.compute", mustJSON(map[string]any{
		"filter": storage.StatsFilter{DecisionType: -1},
	})
}

func buildSearch(rng *rand.Rand) (string, []byte) {
	f := domain.SearchFilters{DecisionTypeFilter: true}
	f.Filter.DecisionType = domain.CheckerAction
	f.Filter.PlayerOnRoll = domain.Black
	return "/v1/search.find", mustJSON(map[string]any{"filters": f})
}

// buildSave generates a position whose Zobrist hash is unique across a large
// range (so writes insert real rows rather than dedup to one), mirroring the
// benchmark generator.
func buildSave(rng *rand.Rand) (string, []byte) {
	i := rng.Int()
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	for k := 0; k < 4; k++ {
		n := (i >> (4 * k)) & 15
		p.Board.Points[1+k] = domain.Point{Checkers: n, Color: domain.White}
	}
	p.Score[0] = (i >> 16) & 63
	p.Score[1] = (i >> 22) & 63
	return "/v1/positions.save", mustJSON(map[string]any{"position": p})
}

// jsonBody wraps a byte slice as a fresh reader for each request (so the body
// can be replayed across keep-alive connections).
func jsonBody(b []byte) *bytes.Reader { return bytes.NewReader(b) }

// tenantHeader formats a tenant id 1..n as the X-Tenant-ID value.
func tenantHeader(n int) string { return fmt.Sprintf("%d", n) }
