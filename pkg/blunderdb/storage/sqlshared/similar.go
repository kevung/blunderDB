package sqlshared

import (
	"context"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// La classe d'équivalence d'un classement par similarité (#293, ADR-0043) :
// ce qui restreint l'ENSEMBLE que la distance ordonne, dit en SQL.
//
// Le classement lui-même est dans search.go, avec le reste de la recherche :
// classer, c'est chercher dans un certain ordre, et un second balayage de la
// table des positions aurait été une seconde moitié de grammaire à tenir en
// phase avec la première. Ce fichier ne porte que ce que la classe ajoute à
// la clause WHERE, et de quoi savoir dans quels matchs une cible a été
// rencontrée.
//
// La classe se dit en SQL et non après le calcul de la distance : le décodage
// de `state` est le vrai coût du balayage, donc filtrer sur des colonnes
// indexables avant de décoder paye la classe une fois par ligne au lieu d'une
// fois par vecteur.

// AppendClassSQL writes opts' class into an existing WHERE clause.
func AppendClassSQL(opts storage.SimilarOptions, excludedMatches []int64, where *strings.Builder, args *[]any) {
	if opts.DecisionType != nil {
		where.WriteString(" AND p.decision_type = ?")
		*args = append(*args, *opts.DecisionType)
	}
	if opts.Money != nil {
		// The same test as domain.Position.IsMoney, in SQL: both away scores
		// at the -1 sentinel. similar_regime_test.go holds the two forms
		// together, because a regime that means one thing in Go and another in
		// SQL would split a class in silence.
		if *opts.Money {
			where.WriteString(" AND p.score_1 < 0 AND p.score_2 < 0")
		} else {
			where.WriteString(" AND (p.score_1 >= 0 OR p.score_2 >= 0)")
		}
	}
	if len(excludedMatches) > 0 {
		// "Another match", said once per row against move(position_id), which
		// is indexed. The list is the matches the TARGET was met in — one or
		// two in practice — so the IN stays tiny whatever the library holds.
		where.WriteString(` AND NOT EXISTS (SELECT 1 FROM move mv JOIN game g ON g.id = mv.game_id
			WHERE mv.position_id = p.id AND g.match_id IN (` + Placeholders(len(excludedMatches)) + `))`)
		for _, id := range excludedMatches {
			*args = append(*args, id)
		}
	}
}

// MatchesOfPosition lists the matches a stored position was met in, which is
// what the class's third rule excludes.
//
// Plural, and that is the point: positions are deduplicated by Zobrist hash,
// so one row is reached by the moves of every match that played through it.
// The plies around it are equally uninformative in each of them.
//
// A position that belongs to no match — imported on its own, or a board the
// user has merely drawn — returns nothing, and then nothing is excluded.
func MatchesOfPosition(ctx context.Context, db Execer, scope string, positionID int64) ([]int64, error) {
	if positionID <= 0 {
		return nil, nil
	}
	tenant, args := db.TenantFilter("mv", scope)
	args = append(args, positionID)
	rows, err := db.Query(ctx,
		`SELECT DISTINCT g.match_id FROM move mv JOIN game g ON g.id = mv.game_id
		 WHERE `+tenant+` AND mv.position_id = ?`, args...)
	if err != nil {
		return nil, errf(db, "list the matches a position was met in", err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id *int64
		if err := rows.Scan(&id); err != nil {
			return nil, errf(db, "list the matches a position was met in", err)
		}
		if id != nil {
			out = append(out, *id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, errf(db, "list the matches a position was met in", err)
	}
	return out, nil
}

// LoadTargetPosition reads the position a ranking is taken against.
func LoadTargetPosition(ctx context.Context, db Execer, scope string, id int64) (*domain.Position, error) {
	tenant, args := db.TenantFilter("p", scope)
	args = append(args, id)
	rows, err := db.Query(ctx,
		`SELECT p.id, p.state, p.player_on_roll, p.decision_type, p.score_1, p.score_2
		 FROM position p WHERE `+tenant+` AND p.id = ?`, args...)
	if err != nil {
		return nil, errf(db, "load the position to rank against", err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, errf(db, "load the position to rank against", err)
		}
		return nil, storage.ErrNotFound
	}
	var (
		pid            int64
		state          string
		onRoll, kind   *int64
		score1, score2 *int64
	)
	if err := rows.Scan(&pid, &state, &onRoll, &kind, &score1, &score2); err != nil {
		return nil, errf(db, "load the position to rank against", err)
	}
	p, ok := positionOfState(state)
	if !ok {
		return nil, storage.ErrNotFound
	}
	p.ID = pid
	if onRoll != nil {
		p.PlayerOnRoll = int(*onRoll)
	}
	if kind != nil {
		p.DecisionType = int(*kind)
	}
	// The score decides the regime, so it has to come from the columns rather
	// than from the board blob, which does not carry it.
	p.Score = [2]int{-1, -1}
	if score1 != nil {
		p.Score[0] = int(*score1)
	}
	if score2 != nil {
		p.Score[1] = int(*score2)
	}
	return &p, nil
}
