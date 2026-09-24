# Des constats aux lots d'issues

Décisions Q6 et Q7 ([README.md](README.md) § 2). Les issues **se créent après le rapport**,
jamais pendant la mesure ; une issue cite la mesure qui la justifie ; un manque vu dans
trois scénarios est **une** issue. L'exécutant les crée lui-même avec `gh`.

## 1. La convention

Celle du chantier ([plan.md](../plan.md)), prolongée :

- Labels : `tournoi`, `lot:D<n>` avec **n ≥ 5**, plus `enhancement` / `bug` /
  `frontend` / `backend` / `documentation` / `product` selon le cas. Une issue destinée
  au moteur va dans `PileOfCells/backgammon-tournoi` (préfixe `N`, numéros après N14) et
  l'issue blunderDB qui la consomme nomme le tag attendu.
- Titre : `D<n>.<m> — <phrase à l'indicatif> [S|M|L]` ; S ≤ ½ j, M ≤ 2-3 j, L = chantier.
- Corps, quatre sections dans cet ordre : **Ce que la mesure a montré** (scénario, persona,
  opération, gestes comptés, lien vers `rapport/S<n>.md`), **Ce qu'il faut construire**,
  **Critères d'acceptation** (cases à cocher, dont un budget chiffré quand il y en a un et
  la spec Playwright qui le tiendra), **Bloquée par** / **Bloque**.
- Une issue = une branche = un worktree = une PR, verticale (Go → Wails → Svelte → tests →
  doc et huit `.po`). Une fonctionnalité visible **ship avec sa documentation**, la règle
  du dépôt ne change pas pour un lot de plus.
- Un bug trouvé pendant la mesure (décision Q9) est une issue `bug` du lot D5, priorité
  haute si elle a interrompu un scénario.

## 2. Ce qu'une issue n'est pas

- Une préférence sans mesure (« ce serait plus joli à droite »).
- Une réécriture du moteur : un algorithme d'appariement différent se discute chez
  l'auteur de Nicomaque, avec l'étude des formats sous les yeux.
- L'écran des joueurs, autrement que l'issue de cadrage de § 3, lot D9.
- La fermeture de #380.

## 3. Une structure de lots, provisoire

Elle vient de ce que le code montre déjà (§ 4) et des scénarios ; **la mesure la
réordonne** et peut la fusionner ou la couper. L'exécutant garde les numéros libres : un
lot vide n'est pas créé.

| Lot | Nom | Ce qu'il regroupe | Vient de |
|---|---|---|---|
| **D5** | L'échelle | tout ce qui tient à 4 joueurs et casse à 50 : défilements de la grille, de la liste des joueurs, de la file ; filtres ; liste de contrôle de « tout lancer » ; bugs bloquants | S1, S2, S5 |
| **D6** | La Rencontre | l'ADR qui amende 0047 (« une seule entité » reste vrai par épreuve), le terme dans `CONTEXT.md`, le schéma, plusieurs Directions ouvertes, tables et pauses partagées, une page murale de Rencontre, « qui est libre pour l'autre épreuve », le joueur inscrit à deux épreuves ; les doubles (une paire n'est pas un Participant nommé « A / B ») | S3 |
| **D7** | Le championnat | les rondes datées et annoncées à l'avance, l'absence prévue sans retrait (I6), le match en retard entre deux rondes, le classement du mardi sans le portable ; ce qui en revient au moteur (mode `rounds`) devient des N | S4, S5 |
| **D8** | Les sorties | fichier plutôt que presse-papier (CSV, annuaire), page murale par épreuve, feuille d'appariements datée, dossier synchronisé, les réglages de phase sans champ (consolante, réconciliation, recharge, taille des poules, qualifiés, cadence min/point) | S2, S3, S4 |
| **D9** | Le cadrage de l'écran joueur | **une seule issue**, `product` : le nombre d'interruptions « je joue où ? » mesuré par scénario, l'ADR-0047 cité, bloquée par #380 ; aucune construction | S3, S4 |

## 4. Ce que le code montre déjà, à confirmer par la mesure

Relevé le 2026-09-24 en lisant `frontend/src/components/direction/`,
`frontend/src/stores/directionStore.js`, `internal/cli/cli_tournament.go` et le module
`backgammon-tournoi@v0.2.1`. Ce sont des **hypothèses de constat** : l'exécutant les
vérifie dans un scénario avant d'en faire une issue, et en abandonne une que la mesure ne
confirme pas.

| # | Hypothèse | Où | Scénario qui la teste |
|---|---|---|---|
| H1 | Aucune épreuve parallèle : un Tournament = une Direction (1:1), une seule ouverte, en ouvrir une autre ferme la première sans confirmation | `directionStore.js` (`openDirection`), ADR-0047 | S3 |
| H2 | Aucun jour, aucune session, aucune ronde à heure fixe : le moteur ne connaît que des pauses et des micro-rondes ; la feuille d'appariements n'existe que pour des matchs lancés | moteur `horaires.go`, `render.PairingSheet` | S4, S5 |
| H3 | Une absence prévue à une ronde n'a pas de geste : retrait (immédiat ou différé) ou rien | `PlayersView.svelte`, moteur `events.go` | S4 |
| H4 | CSV du classement et de l'annuaire **par le presse-papier seulement**, jamais un fichier ni un dialogue | `DirectoryPanel.svelte`, `StandingsView.svelte` | S2, S4 |
| H5 | Cinq réglages de phase sans champ hors préréglage : `consolation`, `reconciliation`, `recharge`, `group_size`, `qualifiers` ; et `min_per_point` sans champ du tout | `DirectionSettings.svelte` (champs édités : kind, length, lives, target, batch_minutes, length_late, late_threshold, final_length, seeding, lengths) | S2 (consolante ajoutée), S3 |
| H6 | `Ctrl+Y` et `Ctrl+Maj+D` font la même chose (`toggleTab('tournaments')`) ; la doc les présente comme deux raccourcis | `services/tabToggles.js` | S1 (Yanis) |
| H7 | « Clore » ne demande aucune confirmation ; « Rouvrir » passe par un `window.confirm` natif, seul dialogue natif de la vue | `StandingsView.svelte`, `DirectionView.svelte` | S2, S5 |
| H8 | Corriger un résultat ancien renvoie de l'Historique vers la grille : si la table a été reprise par un autre match, où s'ouvre la fiche ? | `HistoryView.svelte`, `correctFromHistory` | S2 (I3), S4 (I3 en semaine 4) |
| H9 | La CLI `tournament` est en lecture seule (`list`, `verify`, `standings`, `page`, `export`) : un résultat reçu par message ne se saisit qu'à l'écran | `cli_tournament.go` | S4 |
| H10 | Un Participant n'est pas une paire : les doubles n'ont ni cote de paire, ni nom de Player valable pour les Matchs | moteur `Player{ID, Name, Club, Rating}` | S3 |
| H11 | Un retardataire prend une place BYE d'une section non commencée, sinon attend la phase suivante — en tableau 16 déjà tiré, il attend donc la consolante ou rien | moteur `retardataire.go` | S2 (I2 à 11 h 15) |
| H12 | Le retrait en plein tableau : sa place devient un forfait pour l'adversaire ; la consolante l'accueille-t-elle ? | moteur `state.go` (`EvPlayerWithdrawn`) | S2 (I1 samedi 22 h) |
| H13 | La fin estimée et le temps écoulé ne connaissent pas le lendemain (« 3 h 12 » sans jour) | `ClockBar.svelte`, `db_direction_clock.go` | S2, S5 |
| H14 | Le placeholder « Cette vue arrive avec son lot » est inatteignable (sept onglets couverts) | `DirectionView.svelte` | nettoyage D5, sans mesure |
| H15 | Le classement d'un joueur retiré est −1 avec la note `forfeit`, quel que soit son parcours : un joueur à 3 victoires parti le mercredi finit dernier | moteur `livesRanking` | S5 (I1) |
| H16 | `Target` (bascule) exige 2 vies : un suisse à 3 vies ne bascule jamais | moteur `format.go` | S5 |

## 5. Après les issues

- Le rapport `README.md` porte, en tête, la liste des lots créés avec leurs numéros, et la
  phrase : « #380 reste ouverte : cette simulation n'a mis aucun joueur réel devant
  l'interface. »
- `tasks/BACKLOG.md` reçoit une ligne par lot, sous « Ouvert — Produit / docs » (il n'y a
  pas de domaine « tournoi » : ne pas en créer un pour cinq lignes).
- La mémoire de session (`project_direction_tournois_cadrage`) est mise à jour : le
  chantier a une suite mesurée, ses lots, et la leçon que le rapport a apprise.
- L'exécutant **s'arrête** : les lots s'implémentent dans des sessions ultérieures
  (décision Q8).
