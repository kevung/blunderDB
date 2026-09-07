package sqlshared

import (
	"context"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TrainingStore implements storage.TrainingStore over `training_session` and
// `training_item` (ADR-0040 rule 6). Both are domain tables: every statement
// is confined to the scope's tenant through Dialect.TenantFilter /
// TenantColumns, as the trash and import batches are.
//
// The SQL is the same on both backends — an append-only journal has no
// dialect — which is why it is written once here. The only two facts the
// shared statements ask the dialect for are the tenant columns and how a
// boolean is spelled (`wrong`, `has_deviation`).
type TrainingStore struct{ DB Execer }

var _ storage.TrainingStore = (*TrainingStore)(nil)

// Save appends the session and its items in one transaction. A session with
// no item is still a session: an exercise can be finished after a single
// question was revealed and every number declared right.
func (s *TrainingStore) Save(ctx context.Context, scope string, session storage.TrainingSession) (int64, error) {
	var id int64
	err := s.DB.Transact(ctx, func(tx Execer) error {
		cols, args := tx.TenantColumns(scope)
		cols = append(cols, "exercise", "seed_source", "numbers_asked", "faults", "deviations", "mean_deviation", "median_ms", "pr")
		args = append(args, session.Exercise, session.SeedSource, session.NumbersAsked, session.Faults,
			session.Deviations, session.MeanDeviation, session.MedianMs, session.PR)
		var err error
		id, err = tx.Insert(ctx,
			`INSERT INTO training_session (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`, args...)
		if err != nil {
			return err
		}
		for _, item := range session.Items {
			icols, iargs := tx.TenantColumns(scope)
			icols = append(icols, "session_id", "number_type", "wrong", "has_deviation", "deviation")
			iargs = append(iargs, id, item.NumberType, tx.BoolArg(item.Wrong), tx.BoolArg(item.HasDeviation), item.Deviation)
			if _, err := tx.Exec(ctx,
				`INSERT INTO training_item (`+strings.Join(icols, ", ")+`) VALUES (`+Placeholders(len(icols))+`)`, iargs...); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, errf(s.DB, "save the training session", err)
	}
	return id, nil
}

// Sessions returns the recorded sessions, most recent first. The items are
// deliberately NOT loaded: every reader of this list wants the aggregates,
// and NumberStats answers the per-number question in one query instead of n.
func (s *TrainingStore) Sessions(ctx context.Context, scope, exercise string, limit int) ([]storage.TrainingSession, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	where := tenant
	args := append([]any{}, targs...)
	if exercise != "" {
		where += " AND exercise = ?"
		args = append(args, exercise)
	}
	clause, largs := s.DB.LimitOffset(limit, 0)
	rows, err := s.DB.Query(ctx,
		`SELECT id, exercise, COALESCE(seed_source,''), `+s.DB.TimestampText("created_at")+`,
			numbers_asked, faults, deviations, mean_deviation, median_ms, pr
		 FROM training_session WHERE `+where+` ORDER BY id DESC`+clause,
		append(args, largs...)...)
	if err != nil {
		return nil, errf(s.DB, "list the training sessions", err)
	}
	defer rows.Close()
	var out []storage.TrainingSession
	for rows.Next() {
		var v storage.TrainingSession
		if err := rows.Scan(&v.ID, &v.Exercise, &v.SeedSource, &v.CreatedAt,
			&v.NumbersAsked, &v.Faults, &v.Deviations, &v.MeanDeviation, &v.MedianMs, &v.PR); err != nil {
			return nil, errf(s.DB, "list the training sessions", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "list the training sessions", err)
	}
	return out, nil
}

// NumberStats aggregates the items of one exercise by number type, most-asked
// first. The mean deviation averages only the items that carry one, so a
// question that ran out of time never enters it as a zero error.
func (s *TrainingStore) NumberStats(ctx context.Context, scope, exercise string) ([]storage.TrainingNumberStat, error) {
	itemTenant, itemArgs := s.DB.TenantFilter("i", scope)
	sessionTenant, sessionArgs := s.DB.TenantFilter("s", scope)
	where := itemTenant + " AND " + sessionTenant
	args := append(append([]any{}, itemArgs...), sessionArgs...)
	if exercise != "" {
		where += " AND s.exercise = ?"
		args = append(args, exercise)
	}
	hasDeviation := s.DB.Bool("i.has_deviation", true)
	rows, err := s.DB.Query(ctx,
		`SELECT i.number_type, COUNT(*),
			`+s.DB.Bigint(`SUM(CASE WHEN `+s.DB.Bool("i.wrong", true)+` THEN 1 ELSE 0 END)`)+`,
			`+s.DB.Bigint(`SUM(CASE WHEN `+hasDeviation+` THEN 1 ELSE 0 END)`)+`,
			COALESCE(AVG(CASE WHEN `+hasDeviation+` THEN ABS(i.deviation) END), 0)
		 FROM training_item i
		 JOIN training_session s ON s.id = i.session_id
		 WHERE `+where+`
		 GROUP BY i.number_type
		 ORDER BY COUNT(*) DESC, i.number_type`, args...)
	if err != nil {
		return nil, errf(s.DB, "aggregate the training items", err)
	}
	defer rows.Close()
	var out []storage.TrainingNumberStat
	for rows.Next() {
		var v storage.TrainingNumberStat
		if err := rows.Scan(&v.NumberType, &v.Asked, &v.Faults, &v.Deviations, &v.MeanDeviation); err != nil {
			return nil, errf(s.DB, "aggregate the training items", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "aggregate the training items", err)
	}
	return out, nil
}
