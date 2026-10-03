package storage

import "math/rand/v2"

// StudyWorstGroups is how many of the costliest groups feed a quiz or a deck
// built "from my worst groups".
const StudyWorstGroups = 3

// StudyQuizSize is the default number of positions a worst-groups quiz draws.
const StudyQuizSize = 20

// StudyPositionIDs are the distinct positions of the themed groups picked by
// rank, in the order of the ranking and, inside a group, of decreasing error.
// A rank of 0 takes the StudyWorstGroups costliest groups; a rank n>0 takes the
// n-th alone (1-based). A rank past the end yields nothing.
func (r *RecurringErrors) StudyPositionIDs(rank int) []int64 {
	if r == nil {
		return nil
	}
	groups := r.Groups
	switch {
	case rank > 0 && rank <= len(groups):
		groups = groups[rank-1 : rank]
	case rank > 0:
		return nil
	case len(groups) > StudyWorstGroups:
		groups = groups[:StudyWorstGroups]
	}
	seen := map[int64]bool{}
	ids := []int64{}
	for _, g := range groups {
		for _, id := range g.PositionIDs {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// DrawStudyQuiz draws at most size ids uniformly from ids, without
// replacement. rng makes the draw reproducible in tests; nil uses the global
// source.
func DrawStudyQuiz(ids []int64, size int, rng *rand.Rand) []int64 {
	out := append([]int64{}, ids...)
	shuffle := rand.Shuffle
	if rng != nil {
		shuffle = rng.Shuffle
	}
	shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	if size > 0 && len(out) > size {
		out = out[:size]
	}
	return out
}
