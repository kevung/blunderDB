// Command demodb turns a freshly imported database into the sample database
// the GUI embeds (internal/gui/demo.db.gz). It is driven by
// scripts/build-demo-db.sh, which creates the database, imports the source
// matches and runs the gammonNet sweep with the CLI first; this program then
// does what the CLI has no command for:
//
//   - replace the players, events and file names of the imported matches by
//     fictional ones (the fixtures under testdata/ name real people, and a
//     distributed binary must not — see ADR-0007 and issue #162);
//   - group the tournament matches under a fictional tournament;
//   - build three thematic collections from the worst errors of the base;
//   - comment a handful of positions with #blunder / #cube tags;
//   - create an Anki deck from the blunders collection and simulate a few
//     weeks of FSRS reviews so the panel opens on a real journal;
//   - with -only-compact, as the script's last step once every other tool has
//     closed the file: leave it in rollback-journal mode and vacuumed, so the
//     embedded copy is a single self-contained file.
//
// Everything it writes is deterministic for a given input database and
// -now date: positions are picked by analysis ranking, ratings come from a
// seeded generator and FSRS fuzz is disabled for the simulation.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// ankiTimeLayout mirrors the layout storage/sqlite writes in anki_card and
// anki_review_log (the constant is unexported there).
const ankiTimeLayout = "2006-01-02 15:04:05"

// matchFacts is the fictional identity given to one imported source file.
// The key is the base name of the file the CLI imported, so the script and
// this program cannot drift apart silently: an unknown match is an error.
type matchFacts struct {
	player1, player2 string
	event, location  string
	round, date      string
	filePath         string // fictional source name shown by the match panel
	comment          string
	inTournament     bool
}

const (
	tournamentName     = "Open de Saint-Ludovic 2025"
	tournamentLocation = "Saint-Ludovic"
	tournamentDate     = "2025-05-17"
)

var facts = map[string]matchFacts{
	"HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg": {
		player1: "Ada Fairweather", player2: "Sol Marchetti",
		event: tournamentName, location: tournamentLocation, round: "Ronde 4",
		date:         "2025-05-17",
		filePath:     "ronde4-Fairweather-Marchetti-7pt.xg",
		comment:      "Match analysé par eXtreme Gammon. Ada rate plusieurs doubles en fin de partie 3.",
		inTournament: true,
	},
	"test.mat": {
		player1: "Iris Okonkwo", player2: "Ada Fairweather",
		event: tournamentName, location: tournamentLocation, round: "Tableau Or 2",
		date:         "2025-05-18",
		filePath:     "tableau-or-2-Okonkwo-Fairweather-7pt.mat",
		comment:      "Transcription .mat sans analyse à l'import : toutes les positions sont analysées par gammonNet.",
		inTournament: true,
	},
	"TachiAI_V_player_Nov_2__2025__16_55.bgf": {
		player1: "Bram Vesterholt", player2: "Ada Fairweather",
		event: "Partie en ligne", location: "", round: "",
		date:     "2025-11-02",
		filePath: "Vesterholt-Fairweather-en-ligne.bgf",
		comment:  "Partie unique importée depuis BGBlitz.",
	},
}

func main() {
	dbPath := flag.String("db", "", "database to enrich (required)")
	nowFlag := flag.String("now", "", "anchor date of the review journal, YYYY-MM-DD (default: today, UTC)")
	seed := flag.Int64("seed", 1, "seed of the simulated review ratings")
	onlyCompact := flag.Bool("only-compact", false, "skip the enrichment and only compact the file (the last step of the build)")
	flag.Parse()
	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "demodb: -db is required")
		os.Exit(2)
	}
	now := time.Now().UTC().Truncate(24 * time.Hour)
	if *nowFlag != "" {
		t, err := time.Parse("2006-01-02", *nowFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "demodb: -now: %v\n", err)
			os.Exit(2)
		}
		now = t.UTC()
	}
	if *onlyCompact {
		if err := compact(*dbPath); err != nil {
			fmt.Fprintf(os.Stderr, "demodb: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err := run(*dbPath, now, *seed); err != nil {
		fmt.Fprintf(os.Stderr, "demodb: %v\n", err)
		os.Exit(1)
	}
}

func run(path string, now time.Time, seed int64) error {
	d := database.NewDatabase()
	if err := d.OpenDatabase(path); err != nil {
		return err
	}
	if err := enrich(d, now, seed); err != nil {
		_ = d.Close()
		return err
	}
	return d.Close()
}

func enrich(d *database.Database, now time.Time, seed int64) error {
	if err := disguiseMatches(d); err != nil {
		return err
	}
	blunders, cubes, races, err := pickPositions(database.RawConn(d))
	if err != nil {
		return err
	}
	blundersID, err := createCollection(d, "Blunders à revoir",
		"Les vingt plus grosses erreurs de pions de la base, source du paquet Anki.", blunders)
	if err != nil {
		return err
	}
	if _, err := createCollection(d, "Décisions de videau",
		"Les douze décisions de videau les plus coûteuses : doubles manqués, prises et passes.", cubes); err != nil {
		return err
	}
	if _, err := createCollection(d, "Courses et bearoff",
		"Positions sans contact, pour comparer l'analyse au compte de pips effectif (EPC).", races); err != nil {
		return err
	}
	if err := commentPositions(d, blunders, cubes); err != nil {
		return err
	}
	deckID, err := d.CreateAnkiDeck("Blunders à revoir",
		"Les vingt blunders de la collection du même nom, en répétition espacée.",
		"collection", blundersID, "")
	if err != nil {
		return fmt.Errorf("creating anki deck: %w", err)
	}
	if err := d.SyncAnkiDeck(deckID); err != nil {
		return fmt.Errorf("syncing anki deck: %w", err)
	}
	return simulateReviews(database.RawConn(d), deckID, now, seed)
}

// disguiseMatches gives every imported match its fictional identity and
// groups the tournament ones. Names, dates and comments go through the
// Database API; event, location, round and file path have no API and are
// written directly.
func disguiseMatches(d *database.Database) error {
	matches, err := d.GetAllMatches()
	if err != nil {
		return err
	}
	if len(matches) != len(facts) {
		return fmt.Errorf("expected %d imported matches, found %d", len(facts), len(matches))
	}
	tournamentID, err := d.CreateTournament(tournamentName, tournamentDate, tournamentLocation)
	if err != nil {
		return fmt.Errorf("creating tournament: %w", err)
	}
	if err := d.UpdateTournamentComment(tournamentID,
		"Tournoi fictif : les joueurs et le lieu sont inventés, les coups sont réels."); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, m := range matches {
		key := filepath.Base(m.FilePath)
		f, ok := facts[key]
		if !ok {
			return fmt.Errorf("match %d comes from %q, which has no fictional identity in scripts/demodb", m.ID, key)
		}
		seen[key] = true
		if err := d.UpdateMatch(m.ID, f.player1, f.player2, f.date); err != nil {
			return fmt.Errorf("renaming match %d: %w", m.ID, err)
		}
		if _, err := database.RawConn(d).Exec(
			`UPDATE match SET event = ?, location = ?, round = ?, file_path = ? WHERE id = ?`,
			f.event, f.location, f.round, f.filePath, m.ID); err != nil {
			return fmt.Errorf("relabelling match %d: %w", m.ID, err)
		}
		if err := d.UpdateMatchComment(m.ID, f.comment); err != nil {
			return err
		}
		if f.inTournament {
			if err := d.AddMatchToTournament(tournamentID, m.ID); err != nil {
				return err
			}
		}
	}
	for key := range facts {
		if !seen[key] {
			return fmt.Errorf("source %q was not imported", key)
		}
	}
	return nil
}

// pickPositions ranks the analysed positions: the worst checker plays, the
// worst cube decisions, and the contact-free positions where the most equity
// was lost. Errors are stored in millipoints; ties break on id so the choice
// is stable across builds.
func pickPositions(conn *sql.DB) (blunders, cubes, races []int64, err error) {
	const (
		checker = 0
		cube    = 1
	)
	blunders, err = queryIDs(conn, `SELECT p.id FROM position p JOIN analysis a ON a.position_id = p.id
		WHERE p.decision_type = ? ORDER BY a.best_move_equity_error DESC, p.id LIMIT 20`, checker)
	if err != nil {
		return nil, nil, nil, err
	}
	cubes, err = queryIDs(conn, `SELECT p.id FROM position p JOIN analysis a ON a.position_id = p.id
		WHERE p.decision_type = ? ORDER BY a.cube_error DESC, p.id LIMIT 12`, cube)
	if err != nil {
		return nil, nil, nil, err
	}
	races, err = queryIDs(conn, `SELECT p.id FROM position p JOIN analysis a ON a.position_id = p.id
		WHERE p.no_contact = 1 ORDER BY a.cube_error + a.best_move_equity_error DESC, p.id LIMIT 10`)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(blunders) < 20 || len(cubes) < 12 || len(races) < 10 {
		return nil, nil, nil, fmt.Errorf("not enough analysed positions: %d blunders, %d cube decisions, %d races",
			len(blunders), len(cubes), len(races))
	}
	return blunders, cubes, races, nil
}

func queryIDs(conn *sql.DB, query string, args ...any) ([]int64, error) {
	rows, err := conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func createCollection(d *database.Database, name, description string, positionIDs []int64) (int64, error) {
	id, err := d.CreateCollection(name, description)
	if err != nil {
		return 0, fmt.Errorf("creating collection %q: %w", name, err)
	}
	if err := d.AddPositionsToCollection(id, positionIDs); err != nil {
		return 0, fmt.Errorf("filling collection %q: %w", name, err)
	}
	return id, nil
}

// commentPositions leaves ten tagged notes: six on the worst checker blunders,
// four on the worst cube decisions. The wording stays neutral — it states the
// size of the error the analysis recorded and what to look at, never a
// diagnosis the position might contradict.
func commentPositions(d *database.Database, blunders, cubes []int64) error {
	blunderNotes := []string{
		"La plus grosse erreur de pions de la base (%s point). Rejouer la position avant de regarder l'analyse. #blunder",
		"Erreur de %s point. Comparer le coup joué et le meilleur coup : la différence tient à un seul pion. #blunder",
		"%s point perdu. Chercher d'abord les coups qui gardent le contact, puis vérifier avec l'analyse. #blunder",
		"Blunder de %s point. Bon exercice de visualisation : compter les blots après chaque candidat. #blunder #exercice",
		"Erreur de %s point dans une position de course : le compte de pips suffit à trancher. #blunder",
		"%s point. Position à réviser dans le paquet Anki jusqu'à ce que le meilleur coup devienne évident. #blunder #anki",
	}
	cubeNotes := []string{
		"Décision de videau la plus coûteuse de la base (%s point). Vérifier le verdict avec l'onglet Évaluation. #cube",
		"Erreur de videau de %s point. Le score du match change la fenêtre de double : comparer avec la partie en argent. #cube",
		"%s point. Position à la limite de la prise : l'onglet Course donne le verdict exact si le contact est rompu. #cube",
		"Erreur de videau de %s point. Exemple de « trop bon pour doubler » à comparer avec le verdict de gammonNet. #cube #blunder",
	}
	for i, text := range blunderNotes {
		if err := comment(d, blunders[i], "best_move_equity_error", text); err != nil {
			return err
		}
	}
	for i, text := range cubeNotes {
		if err := comment(d, cubes[i], "cube_error", text); err != nil {
			return err
		}
	}
	return nil
}

func comment(d *database.Database, positionID int64, column, format string) error {
	var millipoints int64
	if err := database.RawConn(d).QueryRow(
		`SELECT `+column+` FROM analysis WHERE position_id = ?`, positionID).Scan(&millipoints); err != nil {
		return fmt.Errorf("reading %s of position %d: %w", column, positionID, err)
	}
	text := fmt.Sprintf(format, formatPoint(millipoints))
	if err := d.AddComment(positionID, text); err != nil {
		return fmt.Errorf("commenting position %d: %w", positionID, err)
	}
	return nil
}

// formatPoint renders millipoints the way the French UI does: 0,182.
func formatPoint(millipoints int64) string {
	return fmt.Sprintf("%d,%03d", millipoints/1000, millipoints%1000)
}

// simulateReviews replays a few weeks of FSRS reviews on the deck's cards so
// the demo has a journal, statistics and a forecast. It mirrors what
// storage/sqlite.ReviewCard writes, with the review time chosen instead of
// time.Now(): every card but one in five gets its first review some days ago
// and is reviewed again each time it comes due before the anchor date.
func simulateReviews(conn *sql.DB, deckID int64, now time.Time, seed int64) error {
	cards, err := queryIDs(conn, `SELECT id FROM anki_card WHERE deck_id = ? ORDER BY id`, deckID)
	if err != nil {
		return err
	}
	params := fsrs.DefaultParam()
	params.EnableFuzz = false
	scheduler := fsrs.NewFSRS(params)
	rng := rand.New(rand.NewSource(seed)) // deterministic demo content, not security
	ctx := context.Background()

	for i, cardID := range cards {
		if i%5 == 4 {
			continue // stays a new card
		}
		var positionID int64
		if err := conn.QueryRowContext(ctx, `SELECT position_id FROM anki_card WHERE id = ?`, cardID).Scan(&positionID); err != nil {
			return err
		}
		card := fsrs.NewCard()
		at := now.AddDate(0, 0, -(27 - i)).Add(19 * time.Hour)
		for review := 0; review < 6 && !at.After(now); review++ {
			rating := pickRating(rng)
			info := scheduler.Next(card, at, rating)
			next := info.Card
			if _, err := conn.ExecContext(ctx,
				`INSERT INTO anki_review_log
				 (card_id, deck_id, position_id, rating, state,
				  stability, difficulty, elapsed_days, scheduled_days, reviewed_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				cardID, deckID, positionID, int(rating), int(info.ReviewLog.State),
				next.Stability, next.Difficulty,
				int(info.ReviewLog.ElapsedDays), int(next.ScheduledDays),
				at.Format(ankiTimeLayout)); err != nil {
				return fmt.Errorf("logging review of card %d: %w", cardID, err)
			}
			card = next
			at = next.Due.UTC()
		}
		if _, err := conn.ExecContext(ctx,
			`UPDATE anki_card SET due = ?, stability = ?, difficulty = ?,
			 elapsed_days = ?, scheduled_days = ?, reps = ?, lapses = ?, state = ?, last_review = ?
			 WHERE id = ?`,
			card.Due.UTC().Format(ankiTimeLayout), card.Stability, card.Difficulty,
			card.ElapsedDays, card.ScheduledDays, card.Reps, card.Lapses, int(card.State),
			card.LastReview.UTC().Format(ankiTimeLayout), cardID); err != nil {
			return fmt.Errorf("scheduling card %d: %w", cardID, err)
		}
	}
	return nil
}

// pickRating draws a rating the way a real session goes: mostly Good, a
// forgotten card now and then.
func pickRating(rng *rand.Rand) fsrs.Rating {
	switch r := rng.Intn(100); {
	case r < 15:
		return fsrs.Again
	case r < 35:
		return fsrs.Hard
	case r < 85:
		return fsrs.Good
	default:
		return fsrs.Easy
	}
}

// compact opens the file on a single connection to leave it in
// rollback-journal mode and vacuumed: the embedded copy must be one file,
// and the WAL mode the application pins at every open would otherwise
// persist in the header — which is why this runs after the CLI's last look.
func compact(path string) error {
	raw, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		return err
	}
	defer raw.Close()
	raw.SetMaxOpenConns(1)
	for _, stmt := range []string{"PRAGMA journal_mode=DELETE", "VACUUM", "ANALYZE"} {
		if _, err := raw.Exec(stmt); err != nil {
			return fmt.Errorf("%s: %w", stmt, err)
		}
	}
	return nil
}
