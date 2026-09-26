package database

import (
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// Expliquer un blunder, côté base : les règles sont dans engine ; on rend un
// THÈME, pas une phrase, que l'interface écrit dans la langue de l'utilisateur.

// ExplainDecision explains what the played decision cost and why, or returns
// an empty theme when no rule applies confidently.
//
// `played` is the checker move in notation, or a cube action ("nd", "dt",
// "dp"). The rule family is decided by what the analysis holds, not by a
// caller flag that could disagree with it.
func (d *Database) ExplainDecision(positionID int, played string) (engine.Explanation, error) {
	pos, ana, err := d.quizSubject(positionID)
	if err != nil {
		return engine.Explanation{}, err
	}
	if ana == nil {
		return engine.Explanation{}, nil
	}
	if ana.DoublingCubeAnalysis != nil {
		return engine.ExplainCube(ana, played), nil
	}
	return engine.ExplainChecker(pos, ana, played), nil
}
