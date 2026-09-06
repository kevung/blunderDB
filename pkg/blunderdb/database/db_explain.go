package database

import (
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// Expliquer un blunder (#298, fiche J.8), côté base.
//
// Les règles sont dans engine, pures et testées sans base. Ici on retrouve la
// position et son analyse, on choisit la famille de règles d'après ce que
// l'analyse contient, et on rend un THÈME — pas une phrase. La phrase est
// écrite par l'interface, dans la langue de l'utilisateur.

// ExplainDecision explains what the played decision cost and why, or returns
// an empty theme when no rule applies confidently.
//
// `played` is the checker move in notation, or a cube action ("nd", "dt",
// "dp"). Which family of rules runs is decided by what the analysis holds, not
// by a flag the caller passes: an analysis carrying a cube block describes a
// cube decision, and asking the caller to say so again would let the two
// disagree.
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
