package sqlshared

import (
	"context"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Les cartes de videau chaînées (#276, fiche I.20).
//
// Une décision de videau est DEUX questions : « double ? », puis « prend ? ».
// blunderDB les enregistre déjà comme deux positions — l'import les sépare, et
// c'est la bonne granularité (ADR-0025 : une carte, une question, une note).
// Ce qui manquait était le lien : les deux moitiés d'une même décision doivent
// se réviser ensemble, sans devenir une carte à deux temps qui recevrait une
// note pour deux réponses.
//
// # Le lien est DÉRIVÉ, pas stocké
//
// Les deux positions sont reconnaissables à un fait des données de match :
// deux lignes de `move` du même jeu, au même numéro de coup, de type "cube".
// C'est ce fait qui les apparie, et le recopier dans une colonne
// `linked_card_id` en aurait fait une seconde vérité — qui se périme au
// réimport, à la suppression d'un match, à la fusion de deux bases. La
// dérivation coûte une jointure sur un index qui existe déjà ; la colonne
// aurait coûté une migration et une maintenance.
//
// # Ce que le chaînage fait, et ce qu'il ne fait pas
//
// Il ORDONNE : après « double ? », si « prend ? » est dans le même paquet et
// due, elle vient tout de suite. Il n'avance AUCUNE échéance : chaque carte
// garde son propre état FSRS, et forcer la seconde hors de son tour
// fausserait l'algorithme pour un effet de mise en scène. Les deux cartes
// naissent ensemble, donc elles sont dues ensemble la première fois — c'est
// là que le chaînage sert, et il sert honnêtement.

// cubeCounterparts returns the position ids that complete a cube decision
// whose other half is in positionIDs, and that are not already there.
//
// A cube decision is two move rows of the same game at the same move number.
// Nothing else in the schema pairs two positions, which is why this is the
// query rather than a column.
func cubeCounterparts(ctx context.Context, db Execer, scope string, positionIDs []int64) ([]int64, error) {
	if len(positionIDs) == 0 {
		return nil, nil
	}
	have := make(map[int64]bool, len(positionIDs))
	args := make([]any, 0, len(positionIDs)+4)
	for _, id := range positionIDs {
		have[id] = true
		args = append(args, id)
	}
	tenantA, argsA := db.TenantFilter("a", scope)
	tenantB, argsB := db.TenantFilter("b", scope)
	args = append(args, argsA...)
	args = append(args, argsB...)

	rows, err := db.Query(ctx,
		`SELECT DISTINCT b.position_id
		 FROM move a
		 INNER JOIN move b ON b.game_id = a.game_id AND b.move_number = a.move_number
		 WHERE a.move_type = 'cube' AND b.move_type = 'cube'
		   AND b.position_id <> a.position_id
		   AND a.position_id IN (`+Placeholders(len(positionIDs))+`)
		   AND `+tenantA+` AND `+tenantB, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if !have[id] {
			have[id] = true
			out = append(out, id)
		}
	}
	return out, rows.Err()
}

// LinkedCard returns the card of the SAME deck holding the other half of
// cardID's cube decision, when that half is due and available.
//
// storage.ErrNotFound means there is nothing to chain — which is the ordinary
// case: a checker decision has no other half, and a cube decision whose
// counterpart is not in the deck, or not due, has nothing to serve.
func (s *AnkiStore) LinkedCard(ctx context.Context, scope string, deckID, cardID int64) (*domain.AnkiReviewCard, error) {
	fail := func(err error) (*domain.AnkiReviewCard, error) {
		return nil, errf(s.DB, fmt.Sprintf("linked card of %d", cardID), err)
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	var positionID int64
	err := s.DB.QueryRow(ctx,
		`SELECT position_id FROM anki_card WHERE id = ? AND deck_id = ? AND `+tenant,
		append([]any{cardID, deckID}, targs...)...).Scan(&positionID)
	if errors.Is(err, ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return fail(err)
	}

	counterparts, err := cubeCounterparts(ctx, s.DB, scope, []int64{positionID})
	if err != nil {
		return fail(err)
	}
	if len(counterparts) == 0 {
		return nil, storage.ErrNotFound
	}

	now := ankiNow()
	ctenant, ctargs := s.DB.TenantFilter("", scope)
	cardArgs := []any{deckID, now}
	for _, id := range counterparts {
		cardArgs = append(cardArgs, id)
	}
	cardArgs = append(cardArgs, now)
	cardArgs = append(cardArgs, ctargs...)
	card, err := scanAnkiCard(s.DB.QueryRow(ctx,
		`SELECT `+s.ankiCardCols()+` FROM anki_card
		 WHERE deck_id = ? AND due <= `+s.DB.TimestampArg()+`
		   AND position_id IN (`+Placeholders(len(counterparts))+`)
		   AND `+s.ankiAvailable()+` AND `+ctenant+`
		 ORDER BY due ASC LIMIT 1`, cardArgs...))
	if errors.Is(err, ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return fail(err)
	}
	pos, err := s.Positions.Load(ctx, scope, card.PositionID)
	if err != nil {
		return fail(err)
	}
	return &domain.AnkiReviewCard{Card: card, Position: *pos}, nil
}

// completeCubePairs extends positionIDs with the counterparts of the cube
// decisions it contains.
//
// Adding them is COMPLETING what the source selected, not adding something
// else: the two halves are one decision, and a deck that asks "double?" and
// never "take?" teaches half a skill. The insert is idempotent
// (ON CONFLICT DO NOTHING) and Sync never deletes, so re-syncing is stable.
func completeCubePairs(ctx context.Context, db Execer, scope string, positionIDs []int64) []int64 {
	extra, err := cubeCounterparts(ctx, db, scope, positionIDs)
	if err != nil {
		// The pairing is a convenience, not the sync's contract: a failure
		// here must not refuse a deck the user asked for. The cards the user
		// selected are still written.
		return positionIDs
	}
	if len(extra) == 0 {
		return positionIDs
	}
	return append(append([]int64{}, positionIDs...), extra...)
}
