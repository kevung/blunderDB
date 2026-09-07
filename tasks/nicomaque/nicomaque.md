# Ce que Nicomaque doit gagner

Le moteur (`github.com/PileOfCells/backgammon-tournoi`, créé par Nicolas Harmand) est
intégré tel quel comme module Go, importé par tag. Ce document liste ce que le cadrage lui
demande, dans l'ordre où blunderDB en a besoin ; chaque point devient une issue **dans le
dépôt Nicomaque** (l'utilisateur y est administrateur), reprise par une issue blunderDB
qui la consomme. Les tests d'invariants par simulation (`sim_test.go`) couvrent chaque
option nouvelle en l'ajoutant à `configs()`. Rien ici ne change l'algorithme d'appariement :
les tirages restent matérialisés dans `EvDraw`, les journaux existants restent rejouables.

## N1 — Codes structurés et version du journal (D9) — avant tout tournoi réel

- `Event.Version` (entier, 1 pour le format actuel avec codes, 0 = journaux antérieurs
  acceptés en lecture).
- Libellés : `Label` devient une structure `{Kind, N, Section}` (`round` + n, `final`,
  `semi`, `quarter`, `round_of` + n, `conso_final`, `grand_final`, `rematch`, `crossed`,
  `playoff`) ; `parseRound` disparaît, le numéro de ronde est un champ de l'événement.
- Notes de classement : `Rank.Note` devient `{Kind, Wins, Lives, Section}` (`winner`,
  `finalist`, `alive`, `eliminated`, `forfeit`, `running`, `awaiting_draw`, `group_wins`).
- Avertissements : `State.Warnings []Warning{Code, Match, Expected, Got, …}`
  (`bracket_wrong_players`, `score_over_length`, `ends_in_break`, `slow_match`).
- Raisons d'attente : `Action.Reason` devient un code (`matches_running`, `no_pairing`,
  `waiting_batch`, `waiting_table`).
- Un rendu français `String()` pour chacun, utilisé par `tournoi-td`, `tournoi-demo` et
  `render/`.
- `sim_test.go` : `m.Label != "Finale"` devient une comparaison de code.

## N2 — Tables (D13)

- `Config.Tables` devient `{Count int, Unavailable []int, Reserved []{Table, Section, Phase}}`.
- `assignTables` saute les indisponibles et les réservées hors de leur usage ; une action
  sans table libre porte `Reason: waiting_table` et `Table: 0`.
- Nouvel événement `table_changed {MatchID, Table}`.

## N3 — Résultat : score libre, forfait d'un match, remarque (D8, D12, D17)

- `EvResult` : scores facultatifs (déjà `omitempty`) ; `Forfeit` désigne le perdant sans
  retrait (déjà le cas) ; nouveau champ `Text` sur le résultat (remarque).
- `EvPlayerWithdrawn` gagne `AfterCurrent bool` : le joueur n'est plus apparié, son match
  en cours se termine normalement.

## N4 — Configuration modifiable en cours (D7)

- `EvConfigChanged {Config}` remplace `EvLengthChanged` (conservé en lecture) : validation
  refusant tout changement de `Kind` d'une phase commencée ou passée, autorisant bascule,
  longueurs, tables, pauses, dotation, phases ajoutées après la courante ; `recompute()`.
- `EvReopened` : annule `Finished`, `Final` est recalculé à la prochaine clôture.

## N5 — Longueurs par tour et fin de suisse

- `PhaseConfig.Lengths []int` (du dernier tour vers le premier) utilisé par
  `bracketSection` ; `LengthLate` + seuil pour allonger la fin d'un suisse.

## N6 — Retardataires après tirage (D12)

- `EvPlayerAdded` gagne `Slot string` : prend une place BYE d'une section non commencée à
  ce tour ; sinon le joueur est inscrit et entre dans la prochaine phase/section admise ;
  jamais de retirage. Un code d'information (`enters_at`) dit où.

## N7 — Micro-rondes et pauses (D11)

- `PhaseConfig.BatchMinutes` : `Propose` n'apparie qu'à l'échéance (horodatage du dernier
  lot dans l'état, dérivé du journal) ; entre deux, `Reason: waiting_batch` avec l'échéance.
  Règle : à l'échéance, tous les joueurs libres d'un même groupe de défaites sont appariés
  au hasard, le reste attend.
- `Config.Breaks []TimeRange` : une action dont la fin attendue tombe dans une pause porte
  l'avertissement `ends_in_break` ; rien n'est bloqué.

## N8 — Réparation proposée après correction (D6)

- Après `result_corrected` ou `match_cancelled` dans un graphe, `Propose` émet les
  `ActCancelMatch` des matchs devenus incohérents puis les `ActStartMatch` corrects, comme
  des propositions ordinaires. Nouveau `ActionKind` `cancel_match`.

## N9 — Dotation et classement par section (D14)

- `Config.Prizes` devient `{EntryFee, Retention{Amount|Percent}, Sections map[string]{Percents|Amounts}}`.
- `SectionRanking(sec)` : classement propre d'une section de tableau (tour atteint) ;
  `Prizes` s'applique par section ; arrondi à l'unité, reste au premier.
- `StandingsCSV` par section.

## N10 — Têtes de série en option (D5)

- `PhaseConfig.Seeding: "" | "rating"` ; `drawSlots` place 1 contre 16, 2 contre 15… quand
  activé ; désactivé par défaut, documenté comme tel (l'étude les écarte).

## N11 — Rendu

- `render/` prend un `Labeler` (fonction code → texte) et une feuille de style injectée :
  la page est autonome (un fichier, CSS embarqué) ; `PairingSheet(st, round)` pour la
  feuille d'appariements ; grille des tables ; consolante à côté du principal ; adversaires
  rencontrés et attente sur le tableau des vies ; crédit en pied de page.
- Golden files SVG/HTML dans `testdata/`.

## N12 — Annuaire CSV

- `players.ToCSV` (l'inverse de `FromCSV`) ; colonnes `nom, club, cote`.

## N13 — Site et crédit

- GitHub Pages du dépôt : documents introductifs, `etude_formats.md`, spécification du
  moteur (journal, événements, formats, propositions), **dans les neuf langues de
  blunderDB** (fr source ; en, de, el, es, fi, it, ja, ru), même chaîne Sphinx + gettext +
  `gh-pages` que blunderDB, avec ses scripts. Le lien est celui du bouton info (F27).

## N14 — Robustesse

- Fuzzing de `Apply` sur des journaux aléatoires ; test de rejeu d'un journal `Version: 0`.
- Bascule Σvies = 2^k en mode `rounds` : test et correction du `budget` (point 4 de
  RESTE_A_FAIRE.md).
- Tag de version à chaque lot ; blunderDB épingle le tag dans `go.mod`.
