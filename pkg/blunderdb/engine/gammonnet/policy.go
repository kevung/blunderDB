// SPDX-License-Identifier: MIT

package gammonnet

import (
	"errors"
	"fmt"
)

// The stateless playing policy: a decision goes in, an action comes out —
// gammonNet's gn_policy.c, ported line for line under ADR-0011 and held by the
// upstream reference corpus (testdata/policy_reference.bin). The spec is
// upstream's docs/specs/politique-spec.md; its section numbers are cited
// below. The game, the dice, the clock stay with the caller (ADR-0072 rule 6).

// PolicyEngineVersion names the gammonNet tag that publishes the policy this
// file ports, the one its reference corpus comes from. It is not
// EngineVersion: the policy can move upstream while the network, and so the
// analyses EngineVersion dates, stays the same.
const PolicyEngineVersion = "gammonNet v1.6.0"

// PolicyResignHorizon is H of spec §5.5: how many rolls ahead the certain
// loss is read. Widening it is a measured choice, made upstream.
const PolicyResignHorizon = 2

// ErrPolicyRefused: the decision is refused, never approximated (spec §6).
var ErrPolicyRefused = errors.New("gammonnet: policy refuses the decision")

// Pending is what is waiting to be decided — GnPending (spec §3).
type Pending int

const (
	PendingMove   Pending = iota // dice rolled: the player on roll moves
	PendingCube                  // before rolling: resign, double or roll
	PendingTake                  // the player on roll doubled: take or pass
	PendingResign                // the player on roll offers to resign
)

// ActionKind is what the policy answers — GnActionKind.
type ActionKind int

const (
	ActionMove ActionKind = iota
	ActionRoll
	ActionDouble
	ActionResign
	ActionTake
	ActionPass
	ActionAccept
	ActionReject
)

// PolicyDecision is GnDecision: one referential, the player on roll's
// (Position.Turn). Score and cube owner are seen from them; who decides
// follows from Pending (spec §2, §3).
type PolicyDecision struct {
	Position     Position
	Pending      Pending
	D1, D2       int // PendingMove only
	Cube         int // 1, 2, 4, ...
	CubeOwner    CubeOwner
	Jacoby       bool // money only
	UseMatch     bool
	AwayOnRoll   int
	AwayOpponent int
	Crawford     bool
	ResignValue  int // PendingResign only: 1, 2 or 3
}

// PolicyAction is GnAction (spec §6). EquityA and EquityB are the two numbers
// compared, from the decider's side: the policy's own scale, never stored
// nor shown (ADR-0019).
type PolicyAction struct {
	Kind        ActionKind
	Play        Play // ActionMove only
	ResignValue int  // ActionResign only
	Searched    bool
	EquityA     float64
	EquityB     float64
}

// policyMaxAway is gammonNet's table horizon (GN_MET_MAX_AWAY). This port's
// table reaches further, but a policy past upstream's would answer what the
// reference refuses: one engine on both sides of the corpus.
const policyMaxAway = 25

// policyEfficiency is T34's measurement as T35's loop read it, indexed by the
// cube state seen from the decider (spec §4). The trailing ulp is part of the
// engine T35 measured: these are not DefaultEfficiency's rounded values.
var policyEfficiency = [3]float64{
	CubeCentred:  0.6880000000000001,
	CubeOwned:    0.5660000000000001,
	CubeOpponent: 0.687,
}

// Policy plays. It owns a Searcher and a generator, so it is not
// goroutine-safe: one per goroutine, or behind a lock.
type Policy struct {
	s   *Searcher
	gen Generator
	// One play buffer per recursion level of the certain-loss reading.
	plays [PolicyResignHorizon + 1][]Play
}

// NewPolicy builds a policy over the embedded networks, the pruning one
// included: the "normal" level demands it (spec §4).
func NewPolicy() (*Policy, error) {
	s, err := NewSearcher(SearchConfig{PruneK: 1})
	if err != nil {
		return nil, err
	}
	p := &Policy{s: s}
	for i := range p.plays {
		p.plays[i] = make([]Play, MaxPlays)
	}
	return p, nil
}

// Decide answers d at the named level ("instant", "normal", "thorough"). An
// unknown level or an invalid decision is ErrPolicyRefused.
func (p *Policy) Decide(level string, d *PolicyDecision) (PolicyAction, error) {
	shape, ok := Level(level)
	if !ok {
		return PolicyAction{}, fmt.Errorf("%w: unknown level %q", ErrPolicyRefused, level)
	}
	if !validDecision(d) {
		return PolicyAction{}, fmt.Errorf("%w: invalid decision", ErrPolicyRefused)
	}
	var out PolicyAction
	var ok2 bool
	switch d.Pending {
	case PendingMove:
		ok2 = p.decideMove(&shape, d, &out)
	case PendingCube:
		ok2 = p.decideCube(&shape, d, &out)
	case PendingTake:
		ok2 = p.decideTake(&shape, d, &out)
	case PendingResign:
		ok2 = p.decideResignOffer(&shape, d, &out)
	}
	if !ok2 {
		return PolicyAction{}, ErrPolicyRefused
	}
	return out, nil
}

func (d *PolicyDecision) matchState() MatchState {
	return MatchState{AwayOnRoll: d.AwayOnRoll, AwayOpponent: d.AwayOpponent, Cube: d.Cube, Crawford: d.Crawford}
}

func validDecision(d *PolicyDecision) bool {
	if d == nil || !d.Position.Valid() || d.Position.isOver() {
		return false
	}
	if d.Pending < PendingMove || d.Pending > PendingResign {
		return false
	}
	if d.Cube < 1 || d.Cube&(d.Cube-1) != 0 {
		return false
	}
	if d.CubeOwner < CubeCentred || d.CubeOwner > CubeOpponent {
		return false
	}
	// A turned cube has an owner, an unturned one none (no beaver, spec §7).
	if (d.Cube == 1) != (d.CubeOwner == CubeCentred) {
		return false
	}
	if d.UseMatch {
		if !d.matchState().IsValid() || d.AwayOnRoll > policyMaxAway || d.AwayOpponent > policyMaxAway {
			return false
		}
		// The Crawford game is a 1-away game with no cube in play.
		if d.Crawford && ((d.AwayOnRoll != 1 && d.AwayOpponent != 1) || d.Cube != 1) {
			return false
		}
	}
	if d.Pending == PendingMove && (d.D1 < 1 || d.D1 > 6 || d.D2 < 1 || d.D2 > 6) {
		return false
	}
	if d.Pending == PendingResign && (d.ResignValue < 1 || d.ResignValue > 3) {
		return false
	}
	return true
}

// configure points the searcher at level's shape, money or match at the
// current cube, cubeless — T35's player (spec §5.1). False when the state is
// outside the table.
func (p *Policy) configure(level *SearchLevel, d *PolicyDecision) bool {
	cfg := SearchConfig{Ply: level.Ply, PruneK: level.PruneK}
	if d.UseMatch {
		cfg.UseMatch = true
		cfg.Match = d.matchState()
	}
	copy(cfg.Filter[:], level.Filter)
	copy(cfg.FilterExtra[:], level.FilterExtra)
	copy(cfg.FilterThreshold[:], level.FilterThreshold)
	return p.s.Reconfigure(cfg) == nil
}

func (p *Policy) decideMove(level *SearchLevel, d *PolicyDecision, out *PolicyAction) bool {
	plays := p.plays[0]
	n := p.gen.LegalPlays(&d.Position, d.D1, d.D2, plays)
	if n < 0 {
		return false
	}
	out.Kind = ActionMove
	switch n {
	case 0:
		out.Play = Play{Result: d.Position}
		out.Play.Result.swapTurn()
		return true
	case 1:
		out.Play = plays[0]
		return true
	}
	if !p.configure(level, d) {
		return false
	}
	best, ok, err := p.s.BestPlay(&d.Position, d.D1, d.D2)
	if err != nil || !ok {
		return false
	}
	out.Play = best.Play
	out.Searched = true
	out.EquityA = best.Equity
	return true
}

// preRollProbs is the distribution before the roll at the level, from the
// player on roll's side.
func (p *Policy) preRollProbs(level *SearchLevel, d *PolicyDecision) ([NumOutputs]float32, bool) {
	if !p.configure(level, d) {
		return [NumOutputs]float32{}, false
	}
	return p.s.Probs(&d.Position)
}

// decideCube is spec §5.2. The exact bearoff path of the C exists only with a
// two-sided table installed; this port has none, and the corpus is generated
// without one, so it never takes it.
func (p *Policy) decideCube(level *SearchLevel, d *PolicyDecision, out *PolicyAction) bool {
	out.Kind = ActionRoll

	if value := p.CertainResignation(d); value > 0 {
		out.Kind = ActionResign
		out.ResignValue = value
		return true
	}

	if d.CubeOwner == CubeOpponent {
		return true
	}
	if d.UseMatch && (d.Crawford || d.Cube >= d.AwayOnRoll) {
		return true
	}

	probs, ok := p.preRollProbs(level, d)
	if !ok {
		return false
	}
	var state *MatchState
	if d.UseMatch {
		st := d.matchState()
		state = &st
	}
	decision, ok := Decide(&probs, d.CubeOwner, state, policyEfficiency[d.CubeOwner], d.Jacoby)
	if !ok {
		return false
	}
	out.EquityA = decision.EquityNoDouble
	out.EquityB = decision.EquityDouble
	out.Searched = true

	// Double on double/take and double/pass — except the optional double, a
	// double/pass worth exactly what not doubling is (spec §5.2).
	verdict := decision.Action
	if (verdict == DoubleTake || verdict == DoublePass) &&
		!(verdict == DoublePass && out.EquityB == out.EquityA) {
		out.Kind = ActionDouble
	}
	return true
}

// CertainResignation is the first step of spec §5.2 alone: the value the
// player on roll resigns for, 0 when the loss or its value is not certain.
// An exact reading, never a search: cheap enough to ask before every roll.
func (p *Policy) CertainResignation(d *PolicyDecision) int {
	value := p.certainLoss(&d.Position)
	if value > 0 && !d.UseMatch && d.Jacoby && d.CubeOwner == CubeCentred {
		value = 1 // an unturned cube under Jacoby pays a single game
	}
	return value
}

// decideTake is spec §5.3: the taker compares both branches with the
// efficiency T35 measured, the "opponent" state seen from the doubler.
func (p *Policy) decideTake(level *SearchLevel, d *PolicyDecision, out *PolicyAction) bool {
	if d.CubeOwner == CubeOpponent || (d.UseMatch && d.Crawford) {
		return false
	}
	xTaken := policyEfficiency[CubeOpponent]
	probs, ok := p.preRollProbs(level, d)
	if !ok {
		return false
	}
	var eDT, eDP float64 // the doubler's side, T35's comparison
	if d.UseMatch {
		state := d.matchState()
		doubled := state
		doubled.Cube = 2 * d.Cube
		if eDT, ok = Value(&probs, CubeOpponent, &doubled, xTaken); !ok {
			return false
		}
		cash, ok := metAfter(state, d.Cube, true)
		if !ok {
			return false
		}
		eDP = 2.0*cash - 1.0
	} else {
		in := CubeInputsFromProbs(&probs)
		eDT = 2.0 * janowskiEquity(in.Win, in.WinPoints, in.LosePoints, CubeOpponent, xTaken)
		eDP = 1.0
	}
	out.Searched = true
	if d.UseMatch {
		out.EquityA = (1.0 - eDT) / 2.0
		out.EquityB = (1.0 - eDP) / 2.0
	} else {
		out.EquityA = -eDT
		out.EquityB = -eDP
	}
	out.Kind = ActionPass
	if eDT < eDP {
		out.Kind = ActionTake
	}
	return true
}

// decideResignOffer is spec §5.4: the offer against playing on cubeless,
// accepted when worth at least as much.
func (p *Policy) decideResignOffer(level *SearchLevel, d *PolicyDecision, out *PolicyAction) bool {
	resigner, ok := p.preRollProbs(level, d)
	if !ok {
		return false
	}
	mine := invertProbs(&resigner)

	var accept, playOn float64
	if d.UseMatch {
		// The decider's view: the away scores trade places.
		state := d.matchState().Swap()
		var ok1, ok2 bool
		accept, ok1 = metAfter(state, d.ResignValue*d.Cube, true)
		playOn, ok2 = matchWinningChance(state, &mine)
		if !ok1 || !ok2 {
			return false
		}
	} else {
		accept = float64(d.ResignValue)
		if d.Jacoby && d.CubeOwner == CubeCentred {
			playOn = 2.0*float64(mine[PWin]) - 1.0
		} else {
			playOn = float64(moneyEquity(&mine))
		}
	}
	out.Searched = true
	out.EquityA = accept
	out.EquityB = playOn
	out.Kind = ActionReject
	if accept >= playOn {
		out.Kind = ActionAccept
	}
	return true
}
