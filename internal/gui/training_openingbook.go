package gui

import "github.com/kevung/blunderdb/pkg/blunderdb/openingbook"

// The exercise « which play at this score? » (Training tab): drawn from and
// judged against the book embedded in the binary. On *App, not Database: the
// book is data of the binary, not of the library; the CLI's `openingbook`
// and the daemon's /v1/openingbook.* call the same functions.

// OpeningBookStatus says how many lines the book holds and at which scores;
// an empty book is the tab's cue to say the book is not computed yet.
func (a *App) OpeningBookStatus() (openingbook.Status, error) {
	b, err := openingbook.Default()
	if err != nil {
		return openingbook.Status{}, err
	}
	st := b.Status()
	if st.Scores == nil {
		st.Scores = []string{}
	}
	return st, nil
}

// OpeningBookQuestion draws a reply position at a score; Empty when the
// book, or the part of it the request names, holds no line.
func (a *App) OpeningBookQuestion(request openingbook.QuestionRequest) (openingbook.Question, error) {
	b, err := openingbook.Default()
	if err != nil {
		return openingbook.Question{}, err
	}
	return b.Draw(request)
}

// OpeningBookGrade judges the board the user left (or a typed play) and
// returns every candidate against the best.
func (a *App) OpeningBookGrade(answer openingbook.Answer) (openingbook.Verdict, error) {
	b, err := openingbook.Default()
	if err != nil {
		return openingbook.Verdict{}, err
	}
	return b.Grade(answer)
}
