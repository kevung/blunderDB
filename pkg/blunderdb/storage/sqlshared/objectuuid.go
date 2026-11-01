package sqlshared

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

// ObjectUUIDKey is the metadata row recording that every row held before the
// uuid columns existed has its identifier. Never exported.
const ObjectUUIDKey = "object_uuid_backfill"

// ObjectUUIDDeriveFunc is the SQL name of DeriveObjectUUID, a function of the
// database on PostgreSQL and a registered one on SQLite.
const ObjectUUIDDeriveFunc = "blunderdb_derive_uuid"

// ObjectUUIDBatch is how many rows one backfill transaction writes where the
// natural key is unique; the lock it holds lasts one batch.
const ObjectUUIDBatch = 20000

// DeriveObjectUUID is the identifier a row held before the uuid columns
// existed receives: a UUID of version 7 whose 48-bit timestamp is zero (it
// sorts before every row created since) and whose remaining 74 bits are the
// SHA-256 of the object's family, natural key and rank among the rows sharing
// that key. A function of the content alone — no tenant, no row id — so two
// libraries holding the same objects converge on the same identifiers.
// The SQL twin (blunderdb_derive_uuid, migration 054) must agree byte for byte.
func DeriveObjectUUID(family, key string, rank int64) string {
	sum := sha256.Sum256([]byte("blunderdb/uuid/1|" + family + "|" + key + "|" + strconv.FormatInt(rank, 10)))
	h := hex.EncodeToString(sum[:])
	return "00000000-0000-7" + h[0:3] + "-" + string("89ab"[sum[16]&3]) + h[3:6] + "-" + h[6:18]
}

// ObjectUUIDFamily is a kind of object carrying a uuid column, and how its
// natural key reads. The key is a SQL expression over t (the row) and the
// joined tables, written in the portable subset both backends speak.
type ObjectUUIDFamily struct {
	Table  string
	Family string
	// Ranked families may hold several rows under one key (the rank, by row
	// id, tells them apart); the others have a unique key.
	Ranked bool
	// Reads names the tables the backfill statement reads besides its own
	// (a backend that lifts row-level security lifts it on each).
	Reads []string
	joins func(tenanted bool) string
	key   string
}

func tenantOn(tenanted bool, a, b string) string {
	if tenanted {
		return " AND " + a + ".tenant_id = " + b + ".tenant_id"
	}
	return ""
}

func joinPosition(tenanted bool) string {
	return " LEFT JOIN position p ON p.id = t.position_id" + tenantOn(tenanted, "p", "t")
}

const (
	zobristOrID = `COALESCE(CAST(p.zobrist_hash AS TEXT), 'id:' || CAST(t.id AS TEXT))`
	nameKey     = `t.name`
)

// ObjectUUIDFamilies lists the objects that carry a uuid, in the order the
// backfill visits them. An analysis has none: one per position, it is
// named by its position's uuid, and a trigger rewriting the row of its
// blob at every insert would double the cost of the heaviest write.
var ObjectUUIDFamilies = []ObjectUUIDFamily{
	{Table: "position", Family: "position", Reads: []string{"position"}, key: `COALESCE(CAST(t.zobrist_hash AS TEXT), 'id:' || CAST(t.id AS TEXT))`},
	{Table: "comment", Family: "comment", Reads: []string{"comment", "position"}, Ranked: true, joins: joinPosition, key: zobristOrID},
	{Table: "collection", Family: "collection", Reads: []string{"collection"}, Ranked: true, key: nameKey},
	{Table: "match", Family: "match", Reads: []string{"match"}, Ranked: true,
		key: `COALESCE(NULLIF(t.canonical_hash, ''), NULLIF(t.match_hash, ''), 'id:' || CAST(t.id AS TEXT))`},
	{Table: "transcription", Family: "transcription", Reads: []string{"transcription", "match"}, Ranked: true,
		joins: func(tn bool) string { return " LEFT JOIN match m ON m.id = t.match_id" + tenantOn(tn, "m", "t") },
		key:   `COALESCE(m.canonical_hash, '') || '|' || COALESCE(t.label, '')`},
	{Table: "lesson", Family: "lesson", Reads: []string{"lesson"}, Ranked: true, key: nameKey},
	{Table: "anki_card", Family: "anki_card", Reads: []string{"anki_card", "anki_deck", "position"}, Ranked: true,
		joins: func(tn bool) string {
			return " LEFT JOIN anki_deck d ON d.id = t.deck_id" + tenantOn(tn, "d", "t") + joinPosition(tn)
		},
		key: `COALESCE(d.name, '') || '|' || COALESCE(t.kind, '') || '|' || COALESCE(CAST(p.zobrist_hash AS TEXT), t.key, '')`},
}

// NextBoundSQL reads the id closing the next batch of an unranked family
// past after; NULL once none is left. Batches are id ranges so a pass never
// rescans the rows it has already named.
func (f ObjectUUIDFamily) NextBoundSQL(after int64, batch int) string {
	return fmt.Sprintf(`SELECT MAX(id) FROM (SELECT id FROM %s WHERE id > %d ORDER BY id LIMIT %d) b`, f.Table, after, batch)
}

// BackfillSQL is the statement that names the rows of the family still
// without an identifier. An unranked family is written one id range at a time
// (after, upTo]; a ranked one numbers its rows over the whole table, so
// it is written by one statement, ranges ignored (these tables hold names and
// notes, not the positions).
func (f ObjectUUIDFamily) BackfillSQL(tenanted bool, after, upTo int64) string {
	joins := ""
	if f.joins != nil {
		joins = f.joins(tenanted)
	}
	if f.Ranked {
		part := f.key
		if tenanted {
			part = "t.tenant_id, " + f.key
		}
		return fmt.Sprintf(`UPDATE %[1]s SET uuid = %[2]s('%[3]s', s.k, s.rn) FROM (
			SELECT id, k, rn FROM (
				SELECT t.id AS id, t.uuid AS u, %[4]s AS k, row_number() OVER (PARTITION BY %[5]s ORDER BY t.id) AS rn
				FROM %[1]s t%[6]s) x WHERE u IS NULL) s WHERE %[1]s.id = s.id`,
			f.Table, ObjectUUIDDeriveFunc, f.Family, f.key, part, joins)
	}
	return fmt.Sprintf(`UPDATE %[1]s SET uuid = %[2]s('%[3]s', s.k, 1) FROM (
			SELECT t.id AS id, %[4]s AS k FROM %[1]s t%[5]s WHERE t.uuid IS NULL AND t.id > %[6]d AND t.id <= %[7]d) s
			WHERE %[1]s.id = s.id`,
		f.Table, ObjectUUIDDeriveFunc, f.Family, f.key, joins, after, upTo)
}
