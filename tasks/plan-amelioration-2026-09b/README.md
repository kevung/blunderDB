# Plan d'amélioration blunderDB — second audit du 2026-09-02

État de départ : `main` @ 7ebdcb86, 0.35.0 publiée le matin même, le plan
[`plan-amelioration-2026-09`](../plan-amelioration-2026-09/README.md)
exécuté en entier (lots 0 à 3). Quatre issues ouvertes (#151, #127, #119, #102).

Ce plan repart d'un **audit neuf** mené par sept passes indépendantes (backend,
moteur, frontend, tests/CI/sécurité, serveur/PostgreSQL/GUI Go, docs/
distribution/communauté, produit), chacune vérifiant l'état réel du code
avant de signaler — les rapports ont écarté une douzaine d'items du BACKLOG
déjà faits, et en ont trouvé d'autres que personne n'avait vus. Les chiffres
cités dans les fiches ont été mesurés ce jour ; ils sont à re-vérifier avant
d'être cités ailleurs.

Chaque fiche a son issue GitHub (#155 à #300, numéro dans le titre de la fiche), rangée dans le jalon de son étape.

**Les quatorze recherches externes (P5-P18) ont été rendues le 2026-09-03** et sont
versées sous [`docs/recherche/`](../../docs/recherche/README.md). Quatre d'entre elles
contredisaient une fiche : I.4 et I.5 ont été réécrites (le format OGXM n'existe pas ;
Heroes passe déjà par XG), C.5 est devenue une question de modèle à trancher en amont
(ni gnubg ni Janowski ne font dépendre l'efficacité du videau du propriétaire), B.12
passe à zstd avec dictionnaire. Lire l'index avant d'ouvrir une fiche de recherche.

Format : `[effort S/M/L]` — S ≤ ½ journée, M ≤ 2-3 jours, L = chantier. Une
fiche = une branche = une PR, dans un worktree (`CLAUDE.md`, « Development
Workflow »). Toute fiche visible par l'utilisateur embarque sa doc française
**et ses `.po`** dans le même commit.

---

## 1. Ce que l'audit a trouvé, en une page

**Trois constats critiques, corrigeables en moins d'une journée chacun.**

| | Constat | Où |
|---|---|---|
| 1 | Tout `X-Tenant-ID` non numérique (`alice`, `default`) tombe sur le **tenant 0** : `strconv.ParseInt` sans lecture de l'erreur. L'isolation multi-tenant n'existe pas pour des tenants nommés, et la doc pousse dedans. | `storage/tenant_context.go:24`, `postgres/positions_postgres.go:31` |
| 2 | La table `metadata` est **globale** et exposée en lecture/écriture à tous les tenants (`metadata.load/save/setVersion`) : fuite de l'état de session d'autrui, `/readyz` de toute l'instance cassable par un appel. | `handlers_metadata.go:31`, `sqlshared/metadata.go:41` |
| 3 | Le wrapper `Database` ouvre SQLite avec un **DSN nu** : les PRAGMAs ne s'appliquent qu'à une connexion sur dix, `foreign_keys=OFF` ailleurs ; supprimer un match peut laisser des orphelins. Invisible aux tests `:memory:`. | `database/db.go:223,297` |

**Une dizaine de bugs de données**, tous vérifiés : filtre `E` non
déterministe (map itérée) ; Crawford codé à `false` dans la conversion
MWC→EMG GnuBG (touche l'échelle d'équité, ADR-0019) ; Jacoby/beaver dans le
hash Zobrist mais jamais posés par les importeurs (dédup cassée entre `.xg` et
XGID) ; `analysis(position_id)` sans UNIQUE ; révision Anki hors transaction ;
majeure comparée en chaîne (`"10" > "9"`) ; stats qui avalent une erreur SQL
et affichent un PR partiel ; trois filtres de recherche (`xD`, `id`, `co`)
silencieusement perdus au rejeu depuis l'historique ; XGID copié avec perte ;
huit raccourcis « afficher/cacher » qui ne cachent jamais.

**Trois dettes que le BACKLOG avait mal datées** : `use_cube` à la recherche
est livré (ADR-0023) ; « évaluation refusée = état nommé » est livré ; la
troisième copie des helpers de recherche et `db_session.go` sont déjà réglés.
Ce qui reste vraiment ouvert côté moteur est ailleurs : `terminalValue` n'est
couverte par **aucun** test (aucune position terminale dans les gold), le
gold de recherche ne tourne dans aucun job CI, et **le videau pèse désormais
45 % d'une décision au score** (60 bissections d'une fonction linéaire par
morceaux : forme close à écrire en amont).

**Ce qui coûte en adoption** : le binaire Linux s'installe en `blunderDB`
quand toute la doc dit `blunderdb` ; 96 chaînes en français sont visibles sur
les huit sites traduits ; la base de démo embarque des noms de personnes
réelles dans chaque binaire ; aucune notice tierce pour les tables GNUbg ;
zéro capture d'écran dans le manuel ; l'aide intégrée (11 600 lignes × 9 à la
main) a deux versions de retard ; 65 % du bundle front est de l'i18n de
langues non chargées.

**Ce qui va bien, à ne pas défaire** : `govulncheck` bloquant à 0 ; parité
CLI/GUI/serveur mesurée par réflexion sur 135 méthodes ; invariant Svelte 5
respecté ; ADR-0008 appliquée à 100 % ; 871 clés × 9 langues verrouillées ;
0 injection SQL ; `hostile-smoke` ; discipline « refusé, jamais dégradé » du
moteur tenue de bout en bout ; `EngineVersion` exemplairement documentée.

---

## 2. Les étapes

| Étape | Objectif | Lots | Durée indicative | Sortie |
|---|---|---|---|---|
| **0 — Colmater** | Sécurité, données, promesses écrites | [A](lot-A-urgences.md) | 1 semaine | **0.35.1** |
| **1 — Fiabiliser** | Bugs, tests manquants, CI qui bloque, serveur sûr | B.1-9, C.1-6, D.1-6, E.1-5, G.1-7, H.1-6 | 3-4 semaines | **0.36.0**, schéma **2.16.0** |
| **2 — Consolider** | Perf, dette, design system, API, docs, distribution | B.10-19, C.7-12, D.7-15, E.6-12, G.8-14, H.7-14 | 4-6 semaines | **0.37.0** |
| **3 — Étendre** | Produit : import sans friction, catégorisation courte, pédagogie, partage | [I](lot-I-produit.md) | un trimestre, au fil de la demande | 0.38 → 0.40, schéma **2.17.0** |
| **4 — Fond** | Classification, rollouts, similarité, quiz, web, club | [J](lot-J-chantiers-fond.md) | chacun sa décision | 1.0 ? |

Les lots B à H sont **transverses aux étapes 1 et 2** : chaque fichier de lot
indique, fiche par fiche, à quelle étape elle appartient.

| Lot | Fichier | Fiches | Domaine |
|---|---|---|---|
| A | [lot-A-urgences.md](lot-A-urgences.md) | 14 | tout ce qui ne peut pas attendre |
| B | [lot-B-backend.md](lot-B-backend.md) | 19 | database, storage, ingest, parser, CLI |
| C | [lot-C-moteur.md](lot-C-moteur.md) | 12 + amont | gammonNet, race, EPC, lot d'analyse |
| D | [lot-D-frontend.md](lot-D-frontend.md) | 15 | Svelte 5, a11y, perf, design system |
| E | [lot-E-tests-ci-outillage.md](lot-E-tests-ci-outillage.md) | 12 | tests Go/front, CI, supply chain, outillage |
| G | [lot-G-serveur.md](lot-G-serveur.md) | 14 | serve, PostgreSQL, observabilité, déploiement, GUI Go |
| H | [lot-H-docs-distribution.md](lot-H-docs-distribution.md) | 14 | doc, traductions, distribution, communauté, onboarding |
| I | [lot-I-produit.md](lot-I-produit.md) | 34 | issues verticales S/M |
| J | [lot-J-chantiers-fond.md](lot-J-chantiers-fond.md) | 10 | chantiers L et décisions |
| — | [prompts-deep-search.md](prompts-deep-search.md) | 14 prompts (P5-P18) | à donner tels quels à un moteur de recherche |

(Il n'y a pas de lot F : la sécurité est répartie entre A, E et G plutôt que
mise à part.)

---

## 3. Ordre et dépendances

```
A (semaine 1) ──► 0.35.1
   │
   ├─ A.1 tenant numérique ──► A.2 session scopée ──► G.7 tests d'isolation
   ├─ A.7 .po ──► règle « rst + po même commit » (H.2 CONTRIBUTING)
   └─ A.14 hygiène ──► BACKLOG à jour, #119 fermé

Étape 1
   B.3 + B.5 + A.2 + B.17 ──► une vague de schéma 2.16.0 (triple synchro + G.7 continuité PG)
   E.1 test-os bloquant ──► C.2 filet arm64/-race ──► (#151 NEON, plus tard)
   D.3 parseur JS unique ──► B.18 grammaire Go ──► I.27 intentions
   C.1 + C.2 ──► C.5 (vérif amont) ──► C.7 forme close (amont puis port)
   G.1 compose proxy ──► H.4 tutoriel serve

Étape 2
   B.10 pagination SQL ──► D.8 pagination front ──► G.9 pagination API ──► J.1/J.3 (volume)
   D.9 tokens de couleur ──► I.30 thèmes ──► I.23 rapport imprimable
   C.9 pool réutilisé ──► I.11 matrice du videau
   G.8 OpenAPI ──► I.33 client Python ; H.8 annexe API
   H.6 aide à jour (dernière fois à la main) ──► H.7 aide générée

Étape 3
   I.1 lot d'import ──► I.2 dossier surveillé, I.3 file post-import, I.28 onboarding
   I.8 phase ──► I.10 stats ventilées ──► J.1 classification
   I.14 mesure #127 ──► I.12 PR sans XG, J.2 rollouts
   I.22 rendu SVG unique ──► I.23 rapport, I.31 planche-contact, J.5 web
```

**Deux vagues de schéma seulement** : 2.16.0 (étape 1 : session scopée, hash
sans Jacoby/beaver, UNIQUE analysis, NOT NULL hash, CHECK, FK, colonnes
`analysis_engine`/`analysis_depth`) et 2.17.0 (étape 3 : lot d'import, origine
des commentaires, phase de partie, soft delete). Chaque bump : `DatabaseVersion`,
triple synchro ([[project_schema_triple_sync]]), migration SQLite + PG, test de
continuité des deux côtés (G.7 ajoute celui de PG).

**Deux passages amont gammonNet** à planifier tôt, parce qu'ils bloquent des
fiches d'ici : la forme close de `levelSolve` (C.7) et la question de
l'efficacité miroitée (C.5). Le prompt P6 est à lancer en premier.

---

## 4. Top 20 impact / effort, toutes passes confondues

| # | Fiche | Effort | Pourquoi |
|---|---|---|---|
| 1 | A.1 tenant non numérique → 0 | S | isolation multi-tenant réellement absente |
| 2 | A.2 `metadata` globale exposée | S/M | fuite de session, `/readyz` cassable |
| 3 | A.3 DSN nu, `foreign_keys=OFF` | S | orphelins silencieux sur le chemin GUI/CLI |
| 4 | A.4 dépôt/CI : token `write`, `main` non protégée, 52 actions non pinnées | S | supply chain, et `SECURITY.md` dit le contraire |
| 5 | A.13 filtre `E` non déterministe | S | une recherche répond autrement d'un lancement à l'autre |
| 6 | A.7 + A.10 : 96 chaînes FR publiées, binaire `blunderDB` vs doc `blunderdb` | S | premières minutes d'un nouvel utilisateur Linux |
| 7 | A.8 + A.9 : noms réels dans la démo, aucune notice tierce | S | risque, coût nul |
| 8 | C.1 `terminalValue` jamais testée ; C.2 gold et `-race` absents de la CI | S | la preuve centrale du port n'est exécutée par aucun automate |
| 9 | D.7 charger une seule locale | M | −1 Mo brut, −250 ko gzip (65 % du bundle) |
| 10 | D.3 parseur de recherche unique | M | trois filtres perdus au rejeu, silencieusement |
| 11 | B.2 Crawford GnuBG à `false` | M | équités de videau fausses sur la partie de Crawford |
| 12 | B.3 Jacoby/beaver dans le hash | M | dédup cassée `.xg` vs XGID ; vague 2.16.0 |
| 13 | C.7 forme close de `levelSolve` | M amont + S | 39 % d'une décision au score |
| 14 | C.4 lot : refus retenté à l'infini, `--stale` sans déclencheur | S | trois bumps d'`EngineVersion`, aucun moyen d'en profiter |
| 15 | D.1 six bugs d'ergonomie (Escape, EPC vide, saut de layout, MatchInfoBar, double-clic, sortie de match) | S | irritants quotidiens |
| 16 | E.3 `t.Parallel()` | M | CI ×3 sur le chemin critique |
| 17 | G.1 compose avec proxy authentifiant | S | toute la frontière de sécurité d'ADR-0005 sans exemple |
| 18 | G.13 config non atomique, `.yaml` en JSON, arrêt fatal | S | perte de configuration reproductible |
| 19 | H.5 captures générées ; H.4 tutoriels + « comment progresser » | M | manuel de 1 436 lignes sans image, sans routine |
| 20 | I.1 rapport de fin d'import → I.28 onboarding | M | le moment de vérité du produit se termine sur du vide |

---

## 5. Corrections à porter au BACKLOG (fiche A.14)

À déplacer dans *Historique* (faits, vérifiés le 2026-09-02) : `use_cube` à la
recherche (ADR-0023) ; « évaluation refusée = état nommé » (`gammonnet_eval.go:71-78`,
`cubeDecision.js:33`) ; `cli_import` / provenance (OR collant) ; troisième copie
des helpers (`db_search.go` 59 l.) ; découpage `db_session.go` (254 l.) ;
`rows.Err()` (105/105) ; 8 index PG (`010_search_range_indexes.sql`) ;
assertion Playwright « Eval ne défile pas » ; dédup autocomplétion inline ;
dédup O(n²) de `moves_gen`.

À corriger : `race.Money` — « plus rien ne bloque » ; « valuation du videau
par lot » — précédée par la forme close (C.7) ; « 3 recherches par position »
— fiche C.9 ; « `positionIDsWithStaleGammonNet` décode tout » — B.12/C.4.

Le BACKLOG reste la liste **ouverte** ; ce plan la priorise et s'y substitue
comme « plan courant » (`tasks/BACKLOG.md`, en-tête).

---

## 6. Méthode d'exécution

- **Un worktree, un agent, une fiche.** Fusion depuis la racine après
  `make check-all` (E.5 le crée ; en attendant : `go vet`, `golangci-lint`,
  `go test -race` hors gammonnet, `npm run lint`, `format:check`, vitest).
- **Les fiches S du lot A d'abord, en série** (elles touchent les mêmes
  fichiers de config CI et de schéma). Puis les étapes 1 et 2 par lots en
  parallèle : B et C ne se marchent pas dessus, D est indépendant, E et G
  partagent `build.yml` (les sérialiser).
- **Les mesures se refont avant d'être citées** : couvertures, tailles de
  bundle, profils, temps de CI — tout ce que ce plan chiffre date du
  2026-09-02.
- **Une décision structurante = une ADR avant le code** : B.3 (hash),
  A.1 (tenant entier), C.6 (une règle de verdict), D.9 (palette), H.7 (aide
  générée), et tous les chantiers J.
- **Les prompts de deep search se lancent en amont de la fiche qu'ils
  alimentent** (tableau en tête de `prompts-deep-search.md`) ; les rapports
  se versent sous `docs/recherche/` avec leurs marqueurs de fiabilité.
- **Chaque étape se termine par une release** pilotée par la skill
  `release-blunderdb`, qu'il faut mettre à jour pour : captures régénérées
  (H.5), bloc « dernière version » (H.9), baseline de benchmarks (E.9),
  chiffre des ADR calculé (H.1), tap Homebrew (H.3).

---

## 7. Comptes

| | S | M | L | Total |
|---|---|---|---|---|
| A | 13 | 1 | — | 14 |
| B | 8 | 9 | 2 | 19 |
| C | 8 | 4 | — (amont : 7 décisions) | 12 |
| D | 7 | 8 | — | 15 |
| E | 10 | 2 | — | 12 |
| G | 6 | 8 | — | 14 |
| H | 7 | 6 | 1 | 14 |
| I | 10 | 24 | — | 34 |
| J | — | — | 10 | 10 |
| **Total** | **69** | **62** | **13** | **144** |

À raison de S = ½ j et M = 2,5 j, les étapes 0 à 2 (lots A-H) représentent
environ **125 jours-agent** de travail largement parallélisable, dont une
semaine pour le lot A ; le lot I en ajoute ~65, au fil de la demande ; le lot J
se décide chantier par chantier.
