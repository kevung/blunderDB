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

// scenario is a weighted op mix plus the way each request names its tenant
// and what answer counts as a success.
type scenario struct {
	ops []op
	// tenant formats the X-Tenant-ID value for tenant number n (1..N).
	tenant func(n int) string
	// ok reports whether a response status is the expected outcome; a request
	// whose status fails it is counted as an error in the report.
	ok func(status int) bool
}

var (
	mixedOps = []op{
		{"positions.list", 40, buildList},
		{"search.find", 25, buildSearch},
		{"stats.compute", 15, buildStats},
		{"positions.save", 20, buildSave},
	}
	readHeavyOps = []op{
		{"positions.list", 50, buildList},
		{"search.find", 30, buildSearch},
		{"stats.compute", 15, buildStats},
		{"positions.save", 5, buildSave},
	}
	writeHeavyOps = []op{
		{"positions.list", 20, buildList},
		{"search.find", 10, buildSearch},
		{"stats.compute", 5, buildStats},
		{"positions.save", 65, buildSave},
	}
)

// scenarios maps a scenario name to its definition. Reads are
// list/search/stats; the only write is positions.save.
//
// The three numeric scenarios expect 2xx. "named-tenants" sends the same
// mixed traffic under tenant NAMES ("tenant-1", "tenant-2", …) and expects
// every request to be refused with 400: a tenant is a positive decimal
// integer (ADR-0005, amendment 2026-09-03), and before that amendment such
// names all landed on tenant 0 and shared its rows. The daemon under test
// silently accepting them would show up as a wall of errors in the report.
var scenarios = map[string]scenario{
	"mixed":         {mixedOps, numericTenant, is2xx},
	"read-heavy":    {readHeavyOps, numericTenant, is2xx},
	"write-heavy":   {writeHeavyOps, numericTenant, is2xx},
	"named-tenants": {mixedOps, namedTenant, isRejected},
}

// scenarioNames lists the scenario keys for --help and error messages.
const scenarioNames = "mixed | read-heavy | write-heavy | named-tenants"

func is2xx(status int) bool      { return status >= 200 && status < 300 }
func isRejected(status int) bool { return status == 400 }

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

// numericTenant formats a tenant id 1..n as the X-Tenant-ID value the daemon
// accepts: the tenant's decimal integer.
func numericTenant(n int) string { return fmt.Sprintf("%d", n) }

// namedTenant formats the value a misconfigured proxy would send — a name
// instead of the tenant's integer — which the daemon must refuse.
func namedTenant(n int) string { return fmt.Sprintf("tenant-%d", n) }
