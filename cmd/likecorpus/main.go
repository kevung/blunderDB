// Command likecorpus produces the corpus the J.3 sheet asked for and never
// got: N targets drawn from a real library, their k nearest neighbours, and a
// column for a player to write "same problem: yes / no" in.
//
// # Why a tool rather than a test
//
// The check J.3 wanted cannot be automated, and that is the point of it. The
// metric was chosen from a report (P7) and the equivalence class from an
// interview (ADR-0043); both are reasoning, and reasoning about similarity is
// exactly what a player's eye is there to contradict. So this produces the
// material and stops: it never scores itself, and it has no pass/fail.
//
// The threshold is stated before the measurement, in the issue and in the note
// this fills: if fewer than two thirds of the FIRST neighbours are judged "same
// problem", the class rules are revised before the token's documentation is
// trusted. Writing the bar down first is what keeps the exercise honest.
//
// # Usage
//
//	go run ./cmd/likecorpus -db library.db -targets 30 -k 5 > corpus.md
//
// The output is Markdown: one section per target, each row a neighbour with
// its distance, its compact board and an empty verdict cell. The judging is
// done WITH the application open — every row names a position id, and typing
// that id in the command bar puts the board on screen — because two positions
// cannot be told apart from two strings, which is the whole reason the check
// needs a player rather than a test.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func main() {
	dbPath := flag.String("db", "", "the library to draw targets from (required)")
	targets := flag.Int("targets", 30, "how many targets to draw")
	k := flag.Int("k", 5, "how many neighbours per target")
	seed := flag.Int64("seed", 0, "random seed; 0 uses the clock. State it in the note so the draw can be repeated")
	flag.Parse()

	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "likecorpus: -db is required")
		os.Exit(2)
	}
	if *seed == 0 {
		*seed = time.Now().UnixNano()
	}

	db := database.NewDatabase()
	if err := db.OpenDatabase(*dbPath); err != nil {
		fail(err)
	}
	defer func() { _ = db.Close() }()

	ids, err := db.ListPositionIDs()
	if err != nil {
		fail(err)
	}
	if len(ids) == 0 {
		fail(fmt.Errorf("the library holds no position"))
	}

	rng := rand.New(rand.NewSource(*seed))
	rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
	if *targets < len(ids) {
		ids = ids[:*targets]
	}

	fmt.Printf("# Corpus de jugement — « une voisine est-elle le même problème ? »\n\n")
	fmt.Printf("Base : `%s` — %d cibles, %d voisines chacune, graine %d.\n\n", *dbPath, len(ids), *k, *seed)
	fmt.Printf("Remplir la colonne **Même problème** par oui ou non, blunderDB ouvert sur\n")
	fmt.Printf("la même base : taper l'indice d'une position dans la barre de commande\n")
	fmt.Printf("amène son plateau à l'écran, et deux plateaux ne se comparent pas de tête.\n")
	fmt.Printf("Le seuil est posé d'avance : si moins des deux tiers des\n")
	fmt.Printf("PREMIÈRES voisines sont jugées « oui », les règles de classe de l'ADR-0043\n")
	fmt.Printf("sont à revoir. Le jugement porte sur la CLASSE, pas sur le choix de la\n")
	fmt.Printf("métrique, qui reste celui du rapport P7.\n\n")

	empty := 0
	for _, id := range ids {
		target, err := db.LoadPosition(int(id))
		if err != nil {
			continue
		}
		neighbours, err := db.SimilarPositions(int(id), *k)
		if err != nil {
			fail(err)
		}
		fmt.Printf("## Cible %d — %s\n\n", id, describe(target))
		fmt.Printf("Plateau cible : `%s`\n\n", domain.EncodeXGIDBoard(target))
		if len(neighbours) == 0 {
			// An empty ranking is a result too: it says the class left nothing
			// in this library, which is worth counting rather than skipping.
			fmt.Printf("*Aucune voisine dans sa classe.*\n\n")
			empty++
			continue
		}
		fmt.Printf("| Rang | Position | Distance | Même problème | Plateau |\n")
		fmt.Printf("|---|---|---|---|---|\n")
		for i, n := range neighbours {
			fmt.Printf("| %d | %d | %d | | `%s` |\n", i+1, n.Position.ID, n.Distance, domain.EncodeXGIDBoard(&n.Position))
		}
		fmt.Println()
	}
	fmt.Printf("---\n\nCibles sans voisine : %d sur %d.\n", empty, len(ids))
}

func describe(p *domain.Position) string {
	kind := "coup de pions"
	if p.DecisionType == domain.CubeAction {
		kind = "décision de videau"
	}
	regime := fmt.Sprintf("%d-%d", p.Score[0], p.Score[1])
	if p.IsMoney() {
		regime = "argent"
	}
	return fmt.Sprintf("%s, %s", kind, regime)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "likecorpus:", err)
	os.Exit(1)
}
