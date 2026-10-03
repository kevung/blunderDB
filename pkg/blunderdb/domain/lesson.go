package domain

// Lesson is an ordered sequence of Steps a coach writes for a student
// (ADR-0066): each Step says something and shows a Collection, a single
// Position, both, or neither. It is authored once and handed over inside an
// exported database, so it travels and is read; the student's own progress
// through it is never recorded (ADR-0007).
type Lesson struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	// StepCount is filled by List, which leaves Steps empty.
	StepCount int `json:"stepCount"`
	// Steps, in reading order; Get fills them.
	Steps []LessonStep `json:"steps"`
}

// LessonStep is one stop of a Lesson. CollectionID and PositionID are 0 when
// the Step shows no Collection or no Position: a Step may be text alone. A
// deleted Collection or Position leaves the Step and its text in place.
type LessonStep struct {
	ID           int64  `json:"id"`
	LessonID     int64  `json:"lessonId"`
	Title        string `json:"title"`
	Text         string `json:"text"`
	CollectionID int64  `json:"collectionId"`
	PositionID   int64  `json:"positionId"`
}
