package duel

import (
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// OpeningStart is the opening position at an Away score, nobody on roll and
// no roll on it: the Arbiter draws the opening roll.
func OpeningStart(away [2]int) domain.Position {
	return domain.Position{
		Board:        transcript.InitialBoard(),
		Cube:         domain.Cube{Owner: domain.None, Value: 0},
		Score:        away,
		PlayerOnRoll: domain.Black,
		DecisionType: domain.CheckerAction,
	}
}

// pendingDouble reports whether p shows a double offered and not yet
// answered: the cube turned in the middle, or held by the opponent of the
// side on roll, at a cube decision.
func pendingDouble(p *domain.Position) bool {
	if p == nil || p.DecisionType != domain.CubeAction {
		return false
	}
	if p.PlayerOnRoll != domain.Black && p.PlayerOnRoll != domain.White {
		return false
	}
	return (p.Cube.Owner == domain.None && p.Cube.Value > 0) || p.Cube.Owner == 1-p.PlayerOnRoll
}

// resolveStart turns the creation's choices into the Start the draft keeps:
// the Away score replaces the Start's (the opening position when none is
// given), Reroll drops the Start's roll, and a double the Start shows offered
// is either replayed by the Arbiter (doubled reports it, the Start is the
// position before it) or, with AfterCube, taken. Nothing else is corrected:
// the rule machine refuses what the rules do not allow.
func resolveStart(set Settings) (start *domain.Position, doubled bool, err error) {
	if set.Start == nil && set.Away == nil {
		return nil, false, nil
	}
	var p domain.Position
	if set.Start != nil {
		p = *set.Start
	}
	if set.Away != nil {
		if set.MatchLength == 0 {
			return nil, false, &transcript.Refusal{Kind: transcript.RefusedStartScore, Detail: "a money session has no score"}
		}
		if set.Start == nil {
			if *set.Away == [2]int{set.MatchLength, set.MatchLength} {
				// The start of the match: the default Start, kept as none so
				// the Match stays exportable.
				return nil, false, nil
			}
			p = OpeningStart(*set.Away)
		}
		p.Score = *set.Away
	}
	if set.Reroll {
		p.Dice = [2]int{}
	}
	if pendingDouble(&p) {
		if set.AfterCube {
			// After the decision: the double was taken, its taker holds the
			// cube and the side on roll rolls.
			p.Cube.Owner = 1 - p.PlayerOnRoll
			if p.Cube.Value == 0 {
				p.Cube.Value = 1
			}
			p.DecisionType = domain.CheckerAction
		} else {
			// Before the answer: the Start is the position before the double,
			// which the Arbiter plays for the side on roll.
			if p.Cube.Value > 0 {
				p.Cube.Value--
			}
			p.Cube.Owner = domain.None
			if p.Cube.Value > 0 {
				p.Cube.Owner = p.PlayerOnRoll
			}
			doubled = true
		}
	}
	return &p, doubled, nil
}

// beginStart plays what the creation's choices give before the first
// Decision: the double the Start showed offered, or, after the cube
// decision, the side on roll's roll.
func (g *game) beginStart(doubled, afterCube bool) error {
	if doubled {
		return g.apply(transcript.Action{Side: g.doc.Start.PlayerOnRoll, Kind: transcript.KindDouble})
	}
	if !afterCube || g.contributionsPending() || g.doc.Dice != [2]int{} {
		return nil
	}
	if d := g.awaiting(); d != nil && d.Kind == DecideCube {
		return g.roll()
	}
	return nil
}
