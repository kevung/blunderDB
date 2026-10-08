package gammonnet

import (
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// legalScratch is a generator and its play buffer, pooled: a buffer holds
// MaxPlays plays, too large to allocate per call on an import's hot path.
type legalScratch struct {
	gen   Generator
	plays []Play
}

var legalPool = sync.Pool{New: func() any {
	return &legalScratch{plays: make([]Play, MaxPlays)}
}}

func init() { engine.RegisterLegalPlayCounter(CountLegalPlays) }

// CountLegalPlays returns how many distinct legal checker plays the player on
// roll has in p with its dice, or engine.LegalPlaysUnknown when p carries no
// dice or its board cannot be generated from. It is what tells a forced play
// from a decision (engine.IsForcedChecker).
func CountLegalPlays(p *domain.Position) int {
	if p == nil {
		return engine.LegalPlaysUnknown
	}
	d1, d2 := p.Dice[0], p.Dice[1]
	if d1 < 1 || d1 > 6 || d2 < 1 || d2 > 6 {
		return engine.LegalPlaysUnknown
	}
	pos, err := FromDomain(p)
	if err != nil {
		return engine.LegalPlaysUnknown
	}
	s := legalPool.Get().(*legalScratch)
	defer legalPool.Put(s)
	n := s.gen.LegalPlays(&pos, d1, d2, s.plays)
	if n < 0 {
		return engine.LegalPlaysUnknown
	}
	return n
}
