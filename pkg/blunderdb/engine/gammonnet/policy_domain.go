// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The policy's domain edge (ADR-0011): a domain.Position in, a
// domain.CheckerStep list out. Nothing below the edge sees a domain type.

// PolicyDecisionFromDomain is the decision pending on pos, from pos's player
// on roll: the dice for PendingMove, the cube and score as pos states them.
// For PendingTake pos is the doubler's, on roll before the double — the cube
// at its value before the turn. resign is PendingResign's offered value.
func PolicyDecisionFromDomain(pos *domain.Position, pending Pending, resign int) (PolicyDecision, error) {
	gp, err := FromDomain(pos)
	if err != nil {
		return PolicyDecision{}, err
	}
	d := PolicyDecision{
		Position:    gp,
		Pending:     pending,
		Cube:        1 << uint(pos.Cube.Value),
		CubeOwner:   CubeOwnerOf(pos),
		ResignValue: resign,
	}
	if pending == PendingMove {
		d.D1, d.D2 = pos.Dice[0], pos.Dice[1]
	}
	if pos.IsMoney() {
		d.Jacoby = pos.HasJacoby != 0
		return d, nil
	}
	state, ok := MatchStateFromPosition(pos)
	if !ok {
		return PolicyDecision{}, fmt.Errorf("%w: match score %v", ErrNotEvaluable, pos.Score)
	}
	d.UseMatch = true
	d.AwayOnRoll, d.AwayOpponent, d.Crawford = state.AwayOnRoll, state.AwayOpponent, state.Crawford
	return d, nil
}

// PolicyPlaySteps is play, a play the policy chose on pos, as the domain's
// steps: the legal play of pos that reaches the same board. A play that
// reaches none is an error, never a guessed move.
func PolicyPlaySteps(pos *domain.Position, play *Play) ([]domain.CheckerStep, error) {
	if play.NumMoves == 0 {
		return nil, nil
	}
	opponent := domain.White
	if pos.PlayerOnRoll == domain.White {
		opponent = domain.Black
	}
	for _, legal := range domain.LegalMoves(pos) {
		res := legal.Result
		res.PlayerOnRoll = opponent // Play.Result already has the turn passed
		g, err := FromDomain(&res)
		if err == nil && g == play.Result {
			return legal.Steps, nil
		}
	}
	return nil, fmt.Errorf("gammonnet: the policy's play reaches no legal board of the position")
}
