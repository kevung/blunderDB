# Feuille papier oracle

`oracle.py feuille.json` recalcule le classement d'une épreuve à partir de ce que le directeur a
noté, sans code de l'application (Python 3, bibliothèque standard). Tests : `python3 -m unittest
oracle_test` (18 cas faits à la main). Le résultat se compare joueur par joueur (id → rang) : l'ordre
des ex æquo n'a pas de sens. Une feuille incohérente (match absent du tableau, vainqueur absent
du match…) sort en erreur, code 1 ; un écart moins grave va dans `warnings`.

## Schéma de la feuille

```
{ "name", "n26": "repechage"|"moteur",                 (défaut "repechage", voir N26)
  "players": [{"id"?, "name", "rating"?}],             ordre = ordre d'inscription ; id par défaut =
                                                       slug du nom, comme blunderDB (entries.go:361)
  "phases":  [{"kind", "lives"?, "target"?, "group_size"?, "qualifiers"?, "entry"?, "seeding"?}],
  "events":  chronologiques, l'un de :
    {"type":"match", "id"?, "phase", "a", "b", "winner", "forfeit"?, "section"?:"barrage"}
    {"type":"correct", "match", "winner"}   {"type":"cancel", "match"}   {"type":"bye","player"}
    {"type":"withdraw", "player"}           {"type":"late", "player":{…}, "slot"?: place 0-based}
    {"type":"draw", "slots":[id|null…]}  (tableau, places du 1er tour dans l'ordre ; null = exemption)
    {"type":"draw", "groups":[[id…]…]}   (poules A, B, … dans l'ordre affiché)
    {"type":"next_phase", "repechage"?: {"Poule A": id}} }
```
Rencontre : `{"rencontre", "epreuves": [feuille, …]}` ; chaque épreuve est classée séparément.
Sortie : `final` [{rank, player, name, note}], et par phase `ranking`, `qualified`, `pools`,
`barrages`, `expected_byes` {size, sum_lives, extra_byes, bye_pairs, bye_by_lives, slots?}.

Un match d'un tableau ou d'une poule est rattaché au match du graphe qui oppose les deux joueurs
au moment où il est noté : la feuille n'a pas à nommer le tour. Un match en cours au moment d'un
retrait se note comme un match gagné par l'adversaire, `forfeit: true`, **avant** le `withdraw`.

## Règles réimplémentées (S = backgammon-tournoi/docs/specification.md)

- Défauts de configuration : 2 vies, poules de 4, 2 qualifiés, entrée `survivors` — S:517-540.
- Résultat : victoire + défaite ; un forfait est un résultat comme un autre (défaite, vie perdue,
  victoire de l'adversaire) — S:898-922, `onResult` S:1273-1295 ; retrait pendant un match = forfait
  de ce match, retrait général — S:841-856.
- Correction = recalcul complet ; annulation = match ignoré partout ; une correction ressuscite un
  match annulé ; un match garde sa place de graphe — S:898-933, recalcul S:1297-1337.
- Exemption suisse : ni victoire ni défaite — S:935-947.
- Retardataire : avant le tirage (ou suisse non figé) il entre avec toutes ses vies ; dans un tableau
  tiré il prend une exemption libre avec une vie ; sinon il n'entre nulle part (non classé) — S:752-777.
- Suisse à vies : vies restantes = vies − défaites, 0 si retiré — S:1247-1271 ; fin = Σ vies ≤ target
  ou un seul en vie — S:1926-1948 ; classement : en vie = 1000 + vies + victoires, finaliste (dernier
  éliminé quand il reste un seul vivant) = 999, éliminé ou retiré = victoires, ex æquo conservés,
  aucun départage — S:128-137, S:2158-2208, state.go:796-845.
- Passage de phase : `survivors` (suisse : les vivants, avec leurs vies vers un `lives_bracket`),
  `top:N`, `all` ; les retirés à cet instant n'entrent pas — S:1215-1245.
- Tirage du tableau : taille = puissance de 2 ≥ Σ vies ; les joueurs à 2 vies sur les paires paires
  puis impaires, exemptés ; exemptions surnuméraires en 2e place des premières paires libres, donc à
  des joueurs tirés au hasard (l'oracle prédit les paires, pas les joueurs) — S:2344-2382,
  phase_bracket.go:26-80. Têtes de série seulement avec `seeding: "rating"` (éteintes dans tous les
  préréglages) : placement 1-contre-N par cote croissante (PR : plus bas = meilleur), joueurs à
  2 vies d'abord, cote inconnue derrière, égalités par id — S:501, seeding.go:25-82.
- Résolution du graphe : exemption contre BYE, forfait contre un retiré dans un match non lancé —
  graph.go:35-100.
- Classement du tableau : par sa sortie, perdants d'un même tour **ex æquo** (demi-finalistes ex
  æquo) ; vainqueur devant ; retiré classé avec les perdants du match qui l'attendait ; « en cours »
  devant tout le monde — S:2467-2535, phase_bracket.go:213-310. Section principale seule (pas de
  consolante : refusée).
- Poules : victoires dans la poule, forfaits contre un retiré non comptés — S:2589-2618 ; égalité sur
  la coupure → barrage à 2 vies jusqu'à `spots` survivants — S:2620-2676 ; qualifiés +100, retiré
  jamais qualifié, ex æquo — S:2704-2743, phase_rr.go:217-291.
- Classement général : de la dernière phase à la première, décalage = taille du classement de
  phase, rangs 1, 2, 2, 4 — S:2747-2789.

## Hypothèses (règle non écrite)

- **N26** (issue PileOfCells/backgammon-tournoi#26, ouverte, « question de format ») : un qualifié
  de poule qui se retire avant le tableau. Le moteur actuel le garde dans `rrQualified` mais ne le
  fait pas entrer (S:1234-1235) : sa place est perdue, une exemption de plus au tableau (`n26:"moteur"`).
  L'oracle prend par défaut le **repêchage du suivant non retiré de sa poule** (plus de victoires de
  poule ; égalité → avertissement, le directeur note `next_phase.repechage`), qualifié dans le
  classement de poule. À confronter : un écart sur ce point est attendu tant que #26 est ouverte.
- Un match noté contre un joueur déjà éliminé du suisse produit un avertissement, pas une erreur
  (la saisie erronée suivie d'une correction reste lisible).
