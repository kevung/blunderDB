package database

import (
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// Le mode quiz (#294, fiche J.4), côté base.
//
// Le jugement est dans engine (pur, testé sans base). Ce fichier ne fait que
// deux choses : retrouver la position et son analyse, et appeler le juge. La
// séparation est ce qui permet à l'interface, à la ligne de commande et au
// démon de noter la même réponse de la même façon — l'invariant de parité de
// CLAUDE.md, appliqué à une notion qui aurait très bien pu naître dans le
// frontend et n'y aurait jamais été vérifiable.

// GradeQuizChecker judges a checker answer given by the board the user built.
func (d *Database) GradeQuizChecker(positionID int, played domain.Board) (engine.QuizVerdict, error) {
	pos, ana, err := d.quizSubject(positionID)
	if err != nil {
		return engine.QuizVerdict{}, err
	}
	return engine.GradeCheckerAnswer(pos, ana, played), nil
}

// GradeQuizCheckerMove judges a checker answer given by its notation, which is
// what a keyboard produces. Same grader, same matching — see
// engine.GradeCheckerAnswerNotation.
func (d *Database) GradeQuizCheckerMove(positionID int, notation string) (engine.QuizVerdict, error) {
	pos, ana, err := d.quizSubject(positionID)
	if err != nil {
		return engine.QuizVerdict{}, err
	}
	return engine.GradeCheckerAnswerNotation(pos, ana, notation), nil
}

// GradeQuizCube judges a cube answer ("nd", "dt", "dp").
func (d *Database) GradeQuizCube(positionID int, action string) (engine.QuizVerdict, error) {
	_, ana, err := d.quizSubject(positionID)
	if err != nil {
		return engine.QuizVerdict{}, err
	}
	return engine.GradeCubeAnswer(ana, action), nil
}

// quizSubject reads the position and its analysis. A missing analysis is NOT
// an error: the verdict says "not matched", which is the honest answer for a
// position nobody has evaluated, and lets a quiz keep running through a base
// with gaps instead of stopping on one.
func (d *Database) quizSubject(positionID int) (*domain.Position, *domain.PositionAnalysis, error) {
	pos, err := d.LoadPosition(positionID)
	if err != nil {
		return nil, nil, fmt.Errorf("quiz: position %d: %w", positionID, err)
	}
	ana, err := d.LoadAnalysis(int64(positionID))
	if err != nil {
		return pos, nil, nil
	}
	return pos, ana, nil
}
