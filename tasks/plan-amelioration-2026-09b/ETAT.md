# État d'exécution du plan 2026-09b

Dernière mise à jour : **2026-09-04** (troisième session d'exécution). Ce fichier est
le point de reprise : il dit ce qui est fusionné, ce qui attend dans une branche, et
ce qui est bloqué. Le plan lui-même est dans [README.md](README.md) ; les fiches sont
dans les fichiers de lot, chacune avec le numéro de son issue GitHub.

## Où en est-on

**58 des 145 issues sont fermées** (les neuf dernières le seront au premier `push` : GitHub ne lit les `Closes` qu'à l'arrivée sur la branche par défaut). L'étape 0 (lot A) est entièrement livrée,
l'étape 1 l'est à l'exception de trois fiches, et l'étape 2 est commencée.

Les quatorze recherches externes P5-P18 sont versées sous
[`docs/recherche/`](../../docs/recherche/README.md) ; quatre ont corrigé une fiche
avant qu'un agent n'y travaille (le format OGXM n'existe pas — I.4 et I.5 réécrites ;
l'efficacité du videau est une question de modèle et non un trou de port — C.5 ;
zstd avec dictionnaire — B.12).

## Fusionné sur `main`

| Fiches | Issues | Ce qui a changé |
|---|---|---|
| **Lot A entier** (A.1-A.14) | #155-#168 | Étape 0 : `X-Tenant-ID` non numérique refusé, `metadata` hors `/v1`, PRAGMAs sur toutes les connexions, dépôt et workflows durcis, sept fuites d'erreur serveur, 96 chaînes traduites, base de démo régénérée, notices tierces, conteneur `.dbx` v2, `/healthz` sans stockage, filtre d'erreur déterministe. |
| B.1, B.9 | #169, #177 | Versions majeures comparées numériquement ; migrations réparables ; `verify` diffe le schéma. |
| B.2, B.7 | #170, #175 | Crawford propagé à la conversion GnuBG ; import qui refuse un plateau à plus de quinze pions. |
| B.4 | #172 | Révision Anki transactionnelle ; note hors bornes refusée (elle faisait paniquer le démon). |
| B.6 | #174 | Une trentaine d'erreurs SQL cessent d'être avalées par un `continue` invisible à `rows.Err()`. |
| **B.3, B.5, B.17** | #171, #173, #185 | **Schéma 2.18.0.** Jacoby et beaver quittent l'identité Zobrist (ADR-0028) : seul un XGID les renseignait, donc la même position d'argent entrait par deux portes. `analysis` devient unique par position, `CHECK` de plage, clés étrangères du journal de révision. `zobrist_hash NOT NULL` **refusé**, avec argument. |
| B.8 | #176 | `--format json` sur neuf commandes, plus d'`os.Exit(2)` caché, sous-commande `completion`, `Conn()` dépubliée. |
| C.1 | #188 | Tests moteur manquants ; gold régénéré depuis le C amont. |
| C.2, E.1, E.2 | #189, #217, #218 | `test-os` bloquant, `GOOS=windows go vet`, couverture mesurée juste avec plancher, job `-race` sur le moteur. |
| C.3, C.6 | #190, #193 | Une seule règle de verdict de videau ; « Équité (money) / (match) » ; badges Jacoby/Beaver enfin affichés. |
| C.4 | #191 | Le lot d'analyse ne retente plus l'inévaluable ; `AnalyzeStaleGammonNet` câblée (CLI, GUI, route). |
| C.5 | #192 | **ADR-0029** : ce n'est pas un trou de port, le C fait pareil. Mesuré : 0 verdict basculé sur 604, 0 coup changé sur 60. Le travail est en amont. |
| D.1 | #201 | Six corrections d'ergonomie + régénération de `frontend/wailsjs`. |
| D.2, D.6, D.11 | #202, #206, #211 | Vraie bascule des panneaux, état d'erreur nommé du panneau Eval, XGID encode enfin Jacoby/beaver. |
| D.3, D.4, D.5 | #203, #204, #205 | Parseur de recherche unique, `Tab` rendu à la navigation clavier, plafond de warnings Svelte (35 → 20) câblé en CI. |
| E.4, E.5 | #220, #221 | `t.Skip` silencieux devenus `t.Fatal`, paquet `tests/` redistribué, hooks versionnés, `make check` aligné sur la CI. |
| G.1, G.2 | #229, #230 | Compose avec proxy authentifiant, arguments positionnels refusés, rate limit par défaut. |
| H.1, H.2 | #243, #244 | Page d'installation, `CONTRIBUTING.md`, code de conduite, description et dix sujets du dépôt. |
| H.3 | #245 | Tap Homebrew, manifeste Flatpak constructible en continu, `metainfo.xml` à jour. |
| H.4, H.5 | #246, #247 | Quatre tutoriels de bout en bout, FAQ élargie, et vingt-deux captures réelles régénérables par `make screenshots`. |
| H.6 | #248 | « Bearoff » → « Eval » dans l'aide intégrée et la doc, neuf langues. |
| B.15 | #183 | `find` découpé en `buildWhere`/`scanRows`/`applyGoFilters`. |
| B.18 | #186 | **Une seule grammaire de recherche.** `pkg/blunderdb/searchquery` porte `parseSearchTokens` côté Go, tenu au même corpus que le JS ; `search --query`/`--query-help`, `/v1/search.query` et `/v1/search.parse`. Trouvé en chemin : le filtre joueur `pl"Nom"` ne correspondait à personne. |
| B.19 | #187 | Versions de Go et de Node dites une seule fois par workflow ; gouvernance des dépendances. |
| D.13 | #214 | Fiabilité des tests frontend (dont la régression D.8 : la navigation dans un match ne redessinait plus le plateau). |
| E.3 | #219 | Tests Go parallèles et sharding par index : le paquet `database` de 159 s à 57 s, chaque job dans son budget. |
| H.7 | #249 | L'aide intégrée engendrée depuis les sources Sphinx (ADR-0034). |
| H.10, H.11 | #252, #253 | Page d'accueil du site et négociation de langue ; feuille de route publique et annonces. |
| H.12 | #254 | Trois visites guidées de plus (Eval, Anki, Stats), chacune ouvrant son onglet. Le panneau de bienvenue reste à I.28. |
| — | #304 | Coquilles de la source française. |
| — | — | `.gitattributes` : `* -text`, sans quoi un checkout Windows convertit les fins de ligne de `met_kazaross_xg2.json` et fait paniquer `engine.init` sur son SHA-256. |
| B.15 (suite) | #183 | `StatsStore.Compute` (438 l., onze passes SQL) découpée en `stats_compute.go` ; le plafond `funlen` passe de 438 à 296 lignes. |
| **E.6-E.12** | #222-#228 | **Le lot E est clos.** L'essentiel était déjà livré au fil des autres fiches sans être coché ; l'audit du 2026-09-04 l'a constaté et a fini le reste : crans de complexité remesurés (573/316/462 → 438/180/200), note « clone allégé » et méthode d'édition programmatique des `.po` dans `doc/README.txt`, et `CLAUDE.md` corrigé — il affirmait qu'il n'existait pas de `nightly.yml`. |

Hors fiches, trouvés en chemin :

- **Les traductions sont complètes pour la première fois depuis longtemps** :
  `scripts/doc-i18n-check.sh` finit sur « all translations complete ». Six chaînes
  de `manuel.rst` étaient servies en français sur les huit sites traduits.
- **`msgfmt -c` refusait `en/manuel.po`** depuis plusieurs versions : `msgmerge`
  marque `python-format` toute entrée dont le texte français contient « 93,4 % ».
  Corrigé, et `scripts/doc-po-update.sh` écrit pour que le faux positif — qui revient
  à chaque régénération — soit traité une fois pour toutes.

## Branches en cours

Chacune est un worktree `../blunderDB-<nom>` sur `feat/<nom>`. **Toutes les
branches listées ici au 2026-09-03 ont été fusionnées** ; il n'en reste qu'une.

| Branche | Fiches | Issues |
|---|---|---|
| `feat/g8-g10-g13-serveur` | G.8, G.10, G.13 — contrat d'API, observabilité, GUI Go | #236, #238, #241 |

Cette branche porte une fusion interrompue (`MERGE_HEAD` présent, ~50 commits de
retard sur `main`) : à reprendre dans une passe dédiée, pas au fil d'une autre.

## Ce qui reste

**Étape 1**, une fiche : G.5 (routes d'opération, dépend d'A.2, déjà livrée).

**Étape 2** : B.13, B.14 ; C.7 (forme close de `levelSolve`, à grouper avec le
passage amont d'ADR-0029) ; C.11, C.12 ; D.9, D.10 ; G.8-G.14 (dont trois
attendent dans la branche ci-dessus) ; H.13, H.14.

**Une leçon d'audit** : plusieurs fiches du lot E étaient faites depuis des
jours sans être cochées, parce qu'elles avaient été livrées en passant par
une autre fiche. Avant d'ouvrir un chantier, lire le code plutôt que la case.

**Étape 3** : le lot I, 34 fiches de produit. **Étape 4** : le lot J.

**Une release est due.** Le plan appelait une 0.35.1 en sortie du lot A, mais il y a
maintenant bien plus que des correctifs sur `main` : schéma 2.18.0, conteneur `.dbx`
v2, `--format json`, sous-commande `completion`, tutoriels et captures. C'est une
**0.36.0**, avec sa ligne de changelog. Le skill `release-blunderdb` exige une
confirmation humaine avant de pousser le tag.

## Deux pièges qui ont coûté cher

1. **`git add` un fichier dès qu'il est résolu**, avant toute boucle sur
   `--diff-filter=U`. Une boucle `git checkout --ours` a écrasé une résolution
   manuelle non ajoutée à l'index et livré une `main` dont `TestDatabaseParity`
   échouait. `po_graft.py` a le même piège en interne.
2. **`go test ./... | grep … | head -N; echo $?` ment** : `$?` est celui de `head`,
   et les lignes de journal poussent le `FAIL` hors des N premières. Écrire
   `go test ./... > log 2>&1; echo $?` et relire le journal. `merge.sh` (dans le
   scratchpad de session) a été réécrit en conséquence.

`internal/cli/parity_test.go` est le fichier que tout le monde percute : ajouter la
ligne en même temps que la méthode évite d'en découvrir le trou à la fusion.
