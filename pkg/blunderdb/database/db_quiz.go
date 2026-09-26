package database

import (
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// Le mode quiz, côté base : le jugement est dans engine, pour que GUI, CLI et
// démon notent une réponse de la même façon (parité, CLAUDE.md).

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
// an error: the verdict says "not matched", and the quiz keeps running.
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
