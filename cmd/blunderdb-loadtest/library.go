package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"
)

// The "library" scenario drives the calls a player makes on a real library:
// open a position and its analysis, search, page the match list, read the
// statistics, and now and then comment or re-save an analysis. Its requests
// name rows of the served base, so it reads a sample of them first
// (librarySetup); the static scenarios work on any base, empty included.

// libraryPool holds the rows of the served base the scenario's requests name.
type libraryPool struct {
	positionIDs []int64
	players     []string
	// analyses are analyses.save bodies, each re-saving a position's own
	// analysis as it was read: a write of real size that leaves the figures
	// as they were.
	analyses   [][]byte
	maxMatches int
}

// librarySearches are the filter sets a GUI search sends most: blunders,
// cube decisions, a player's errors.
var librarySearches = []map[string]any{
	{"moveErrorFilter": "E>100"},
	{"decisionTypeFilter": true, "filter": map[string]any{"decision_type": 1}},
	{"moveErrorFilter": "E>50", "decisionTypeFilter": true, "filter": map[string]any{"decision_type": 0}},
}

func libraryOps(p *libraryPool) []op {
	pid := func(rng *rand.Rand) int64 { return p.positionIDs[rng.Intn(len(p.positionIDs))] }
	player := func(rng *rand.Rand) string { return p.players[rng.Intn(len(p.players))] }
	return []op{
		{"positions.load", 34, func(rng *rand.Rand) (string, []byte) {
			return "/v1/positions.load", mustJSON(map[string]int64{"id": pid(rng)})
		}},
		{"analyses.load", 12, func(rng *rand.Rand) (string, []byte) {
			return "/v1/analyses.load", mustJSON(map[string]int64{"positionId": pid(rng)})
		}},
		{"search.find", 12, func(rng *rand.Rand) (string, []byte) {
			f := librarySearches[rng.Intn(len(librarySearches))]
			return "/v1/search.find", mustJSON(map[string]any{"filters": f, "limit": 50})
		}},
		{"search.count", 5, func(rng *rand.Rand) (string, []byte) {
			return "/v1/search.count", mustJSON(map[string]any{"filters": librarySearches[0]})
		}},
		{"matches.list", 12, func(rng *rand.Rand) (string, []byte) {
			req := map[string]any{"limit": 100, "offset": rng.Intn(p.maxMatches)}
			if rng.Intn(2) == 0 {
				req = map[string]any{"limit": 100, "playerName": player(rng)}
			}
			return "/v1/matches.list", mustJSON(req)
		}},
		{"stats.compute.player", 12, func(rng *rand.Rand) (string, []byte) {
			return "/v1/stats.compute", mustJSON(map[string]any{
				"filter": map[string]any{"DecisionType": -1, "PlayerName": player(rng)},
			})
		}},
		{"stats.compute.all", 3, func(rng *rand.Rand) (string, []byte) {
			return "/v1/stats.compute", mustJSON(map[string]any{"filter": map[string]any{"DecisionType": -1}})
		}},
		{"comments.add", 7, func(rng *rand.Rand) (string, []byte) {
			return "/v1/comments.add", mustJSON(map[string]any{"positionId": pid(rng), "text": "loadtest"})
		}},
		{"analyses.save", 3, func(rng *rand.Rand) (string, []byte) {
			return "/v1/analyses.save", p.analyses[rng.Intn(len(p.analyses))]
		}},
	}
}

// librarySetup reads the pool of the "library" scenario from the daemon.
func librarySetup(target, tenant string, maxMatches int) (scenario, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	call := func(path string, body any, out any) error {
		req, err := http.NewRequest(http.MethodPost, target+path, bytes.NewReader(mustJSON(body)))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", tenant)
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("%s: %d %s", path, resp.StatusCode, strings.TrimSpace(string(data)))
		}
		return json.Unmarshal(data, out)
	}

	p := &libraryPool{maxMatches: max(maxMatches, 1)}
	var blunders []int64
	if err := call("/v1/search.ids", map[string]any{"filters": librarySearches[0], "limit": 1000}, &blunders); err != nil {
		return scenario{}, err
	}
	p.positionIDs = append(p.positionIDs, blunders...)
	for _, offset := range []int{0, 50_000, 150_000, 250_000} {
		var ids []int64
		if err := call("/v1/positions.listIds", map[string]int{"limit": 500, "offset": offset}, &ids); err != nil {
			return scenario{}, err
		}
		p.positionIDs = append(p.positionIDs, ids...)
	}
	if len(p.positionIDs) == 0 {
		return scenario{}, fmt.Errorf("library scenario: the served base has no position")
	}

	var names []struct {
		Name  string
		Count int
	}
	if err := call("/v1/stats.playerNames", struct{}{}, &names); err != nil {
		return scenario{}, err
	}
	sort.SliceStable(names, func(i, j int) bool { return names[i].Count > names[j].Count })
	for _, n := range names[:min(len(names), 50)] {
		p.players = append(p.players, n.Name)
	}
	if len(p.players) == 0 {
		return scenario{}, fmt.Errorf("library scenario: the served base has no player")
	}

	for _, id := range p.positionIDs[:min(len(p.positionIDs), 60)] {
		var a json.RawMessage
		if err := call("/v1/analyses.load", map[string]int64{"positionId": id}, &a); err != nil {
			continue
		}
		if len(a) > 0 && string(a) != "null" {
			p.analyses = append(p.analyses, mustJSON(map[string]any{"positionId": id, "analysis": a}))
		}
	}
	if len(p.analyses) == 0 {
		return scenario{}, fmt.Errorf("library scenario: none of the sampled positions has an analysis")
	}
	return scenario{ops: libraryOps(p), tenant: numericTenant, ok: is2xx}, nil
}
