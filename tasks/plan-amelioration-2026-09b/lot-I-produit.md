<!-- Lot du plan tasks/plan-amelioration-2026-09b/README.md. -->

# Lot I — Produit : étape 3 (étendre)

Issues verticales (une fiche = un ticket GitHub = une branche), chacune
livrable seule avec sa doc française et ses `.po`. Effort S/M seulement ; les
chantiers L sont au lot J. Personas : **club** (joueur de club), **compét**
(joueur de compétition), **coach**, **dev** (développeur d'outils).

Carte de départ (vérifiée le 2026-09-02) : import XG/GnuBG/BGF/mat solide ;
~50 filtres ; 4 onglets de stats alignés XG ; Anki/FSRS avec réponse masquée ;
Eval gammonNet 0→2-ply au score, trois régimes de course ; filigrane + `.dbx` ;
CLI/serveur en parité mesurée. Absents : surveillance de dossier, OGXM, Heroes,
catégorisation, similarité, quiz, rapport, thèmes, undo, vérification de
version.

Ordre conseillé : I.1 → I.8 → I.14 → I.17 → I.11 → I.2 → I.9 → I.23 → le
reste selon la demande (Discord).

---

## Import et flux

### I.1 — Rapport de fin d'import [M] — tous — valeur haute (#257)
L'import se termine aujourd'hui sur du vide. Un panneau récapitule : N matchs
importés / M ignorés / K enrichis, PR du lot, 5 pires erreurs, positions
marquées, positions sans analyse (bouton « analyser maintenant »).
- Prérequis : notion de **lot d'import** (`import_batch(id, started_at, source,
  counts)` + `match.import_batch_id`) → bump schéma, à ranger dans la vague
  2.16.0 (lot B) ou la suivante. Le serveur émet le même résumé en NDJSON.
- Recette : import de `testdata/` → panneau avec les cinq blocs ; CLI
  `import --format json` porte le même objet.

### I.2 — Dossier surveillé [M] — compét, coach — valeur haute (#258)
Un dossier déclaré dans la configuration (par défaut celui d'XG) est surveillé
(`fsnotify`, pur Go) ; tout nouveau fichier stable (taille inchangée 2 s) est
importé, dédupliqué, analysé si l'option est cochée ; notification non
bloquante (I.1).
- Prérequis : #108 volet 2 (persister le `MatchHash` du second format, sinon
  re-parse en boucle) ; capacité hôte optionnelle (ADR-0004 : inotify absent
  sur certains partages → repli scrutation 60 s). Doc : `manuel.rst` §
  Configuration.

### I.3 — File d'étude post-import [M] — compét — valeur haute (#259)
L'intention de #110 sans violer ADR-0006 : après un import, une file ordonnée
(blunders du lot, positions marquées XG, décisions serrées) parcourue une fois
avec quatre gestes par position : commenter / collection / carte Anki /
passer. Prérequis : I.1.

### I.4 — Import OGXM et codec OGID (rouvre #114) [M] — compét, dev — valeur haute (#260)
Format MIT entièrement spécifié (HedgeHog/OpenGammon), TLV, jusqu'à 16 blocs
d'analyse par match signés Ed25519. Parseur dans un module séparé (comme
`gnubgparser`), version épinglée ; OGID à côté de XGID dans `parser/` et dans
le collage. Prompt P9 recense la spec et son état.

### I.5 — Presse-papier Heroes (rouvre #61) [S/M] — club — valeur moyenne-haute (#261)
Un parseur de plus dans `pkg/blunderdb/parser/`, détection automatique au
collage. Prérequis : collecter des échantillons réels (Discord) et les
verser en `testdata/` ; parseur contenu derrière un corpus, comme BGF.

### I.6 — Coller un identifiant et enrichir un match depuis un fichier [S] — club, coach (#262)
Champ / commande `import XGID=…|OGID=…` (le serveur a déjà
`positions.fromXGID`) ; bouton « enrichir depuis un fichier » sur la fiche
d'un match (le chemin *enrich* existe, `CanonicalHash` vérifié). Prérequis :
D.11 (XGID sans perte).

### I.7 — Provenance et pluralité des commentaires [M, schéma] — compét, coach (#263)
CONTEXT.md : « un Comment ne porte aucune provenance » ; la GUI n'affiche
qu'un commentaire par position, choisi par l'ordre SQL (perte silencieuse).
Colonne `comment.origin ∈ {user, xg, gnubg, bgf, unknown}`, filtre
`co'user'`, affichage de tous les commentaires (onglets ou liste), la purge
orpheline épargne les commentaires `user`. Vague 2.16.0.

## Catégorisation (étape courte, le classifieur complet est en J.1)

### I.8 — Phase de partie [S/M] — tous — valeur moyenne, effort minime (#264)
Étiquette dérivée et jamais éditable : ouverture / milieu / course / bearoff
(contact, numéro de coup, `no_contact` existant, tous les pions au jan).
Colonne indexée calculée à l'import et à l'ouverture (`repair`), jeton
`ph'course'`, drapeau CLI, route. Donne aussitôt « mon PR en course vs en
contact » (I.10). Aucune taxonomie contestable.

### I.9 — Vocabulaire de tags et panneau Tags [M] — club, coach — valeur haute (#265)
Suite du commentaire de clôture de #60 : jeu de tags recommandés (`#blitz`,
`#prime`, `#holding`, `#backgame`, `#containment`, `#crunch`, `#ace-point`,
`#cube`…) proposés en autocomplétion au `#`, panneau listant les tags de la
base avec cardinalité, cliquables comme filtre ; recherche par délimiteurs
(`#prime` ≠ `#priming`). Zéro schéma. Prompt P5 fournit les définitions de la
littérature pour le vocabulaire.

### I.10 — Stats ventilées : phase, catégorie, score [M] — compét — valeur haute (#266)
Cinquième onglet Stats (ou extension d'*Erreurs*) : PR et taux d'erreur par
phase (I.8), par tag (I.9), et **matrice par score away** (Crawford,
post-Crawford, 2-away/2-away, DMP…), effectifs affichés, cellules grisées
sous un seuil, drill-down. Toutes les données sont en base.

## Analyse et évaluation

### I.11 — Matrice du videau par score [M] — compét, coach — valeur très haute (#267)
Sur la position courante, le verdict à tous les scores d'un match de 5/7/9
(grille away × away). Balayage de l'évaluateur (81 évaluations 2-ply) : 0-ply
au geste, 2-ply au repos, comme #125 ; parallélisme du lot (C.8/C.9 d'abord).
Onglet du panneau Eval ; commande `cm` ; route `/v1/gammonnet.cubeMatrix`.

### I.12 — Un PR sur les matchs sans analyse [M] — compét — valeur très haute (#268)
Le lot connaît la position, pas le coup joué : passer par `game`/`move`,
comparer le coup joué à la recommandation gammonNet (`EquityError` existe),
écrire `move_analysis` étiquetée `gammonNet <version>` sans jamais écraser
une analyse importée (ADR-0013). Un match joué en ligne sans XG obtient un PR.
Préalable de crédibilité : I.14.

### I.13 — Comparaison inter-moteurs sur une position [M] — compét, dev (#269)
Plusieurs moteurs coexistent déjà dans une `Analysis` ; rien ne les montre
côte à côte. Vue « XG dit double/take, gammonNet dit no double » dans le
panneau Analyse (le rendu commun de #124 s'y prête) ; ADR-0017 reste
respectée (une seule Décision dans Eval, la comparaison vit dans Analyse).

### I.14 — Ce que vaut l'évaluateur : #127 et le rapport sur sa propre base [S/M] — compét (#270)
(a) La mesure #127 : gammonNet 2-ply vs table bi-face exacte (taux d'accord du
verdict par distance au point de prise, |Δp|, |Δéquité cubeful|), sur
`gnubg_ts0.bd` puis TS-06-11 (`BLUNDERDB_TS11_PATH`), rapport dans
`docs/recherche/`, page « Que vaut l'évaluateur » dans la doc.
(b) `blunderdb analyze --compare` / onglet Stats : sur les positions analysées
par XG, écart gammonNet (> 0,05 sur x %, concentrés où) — la gate
`integration_gate_test` fait déjà ce calcul.

### I.15 — Drapeaux de règles visibles, MET configurable [S puis M] — compét (#271)
Cube max, Jacoby, beaver affichés à côté du verdict (S, avec C.3). MET
configurable **par base** (lecture des `.xml` gnubg, MET stockée dans
l'analyse pour rester comparable : ADR-0016 refuse un réglage global qui
rend deux analyses incomparables) — M, après décision.

### I.16 — 3-ply mesuré et exposé [S] — compét (#272)
`MaxPly = 4` existe ; `DefaultConfig` a un filtre 3-ply ; aucune mesure
publiée. Mesurer avec le probe (temps, écart d'équité sur 669 décisions),
publier, et exposer le réglage si le rapport temps/gain le justifie.

## Pédagogie

### I.17 — Micro-entraînements : pips, EPC, points de prise [S/M] — club — premier module (#273)
Position, chronomètre, saisie du compte de pips (ou de l'EPC, ou du point de
prise au score), note. Toutes les données et tables (`tp2`, `tp4`, `gv*`,
`met`) sont embarquées. Un onglet « Entraînement » qui préfigure J.4.

### I.18 — Objectifs de progression [M] — compét (#274)
« PR < 5 sur 12 semaines » comme cible sur la courbe de Progression, tendance
et écart ; cible proposée depuis le niveau actuel. Petit stockage de config
par base (`metadata` scopée, A.2).

### I.19 — Statistiques d'étude corrélées au jeu réel [M] — compét (#275)
« 40 positions de backgame révisées ce mois ; PR en backgame 9,2 → 6,8. »
Historique FSRS × catégories (I.8/I.9) × Progression ; formulation prudente.

### I.20 — Cartes de videau chaînées [M] — compét (#276)
Deux cartes liées (doubler ? puis prendre ?) plutôt qu'une carte à deux temps
(ADR-0025 : une carte, une question, une note).

### I.21 — Parcours pédagogiques [M, ADR] — coach, club — valeur haute (#277)
Séquence ordonnée de collections/positions avec un texte d'étape, livrée
comme base filigranée (`.dbx`) : le format de distribution existe. Nouveau
concept dans CONTEXT.md → ADR. Éditeur de parcours en seconde étape.

## Partage

### I.22 — Un seul rendu du plateau, export image fichier [S] — tous (#278)
`clipboardService.js` redessine l'analyse sur canvas avec `formatEquity`
écrit quatre fois et peut diverger de l'écran (BACKLOG) ; pas d'« enregistrer
sous » ni SVG. Un rendu unique (SVG depuis `boardScene`), export SVG/PNG
fichier, presse-papier dérivé. Prompt P12 pour les conventions de diagrammes.

### I.23 — Rapport de session / match / tournoi en HTML [M] — compét, coach (#279)
Document autonome (indicateurs, courbe, 10 pires décisions avec diagramme SVG
en ligne, coup joué vs meilleur coup), imprimable en PDF par le navigateur.
Prérequis : I.22 ; thème « imprimable » (I.30).

### I.24 — Export CSV/Parquet et notebook d'exemple [S/M] — dev, compét (#280)
`list --type positions|moves|analyses --format csv` ; Parquet en option ;
notebook Jupyter versionné (PR dans le temps, distribution des erreurs, top
blunders) exécuté en CI (nightly) pour ne pas se périmer.

### I.25 — Index communautaire de bases filigranées [S] — coach, club (#281)
Une page (site de doc) listant des bases `.dbx` partagées par leurs auteurs
avec leur identité d'émetteur vérifiable ; convention de métadonnées (prompt
P17). Aucun service.

## Recherche et bibliothèque

### I.26 — Collections vivantes, « vue N fois », deux joueurs côte à côte [M] — compét, coach (#282)
Un filtre nommé qui apparaît dans Collections et se réévalue à l'ouverture
(le mécanisme des paquets Anki « recherche » existe) ; jeton `n>3` sur le
nombre de rencontres (chaîne `move → game → match`) ; dans Stats Joueurs,
cocher deux joueurs et superposer leurs indicateurs et les positions où l'un a
bien joué et l'autre non.

### I.27 — Grammaire d'intentions dans la ligne de commande [M] — club (#283)
`s mes blunders de videau au score` traduit en jetons visibles, hors ligne,
déterministe (option 3 de #38). Prérequis : D.3 puis B.18 (une seule
grammaire), sinon c'est un troisième parseur. Anglais + français d'abord.

## Interface et plateforme

### I.28 — Écran d'accueil et onboarding orienté valeur [M] — club — valeur très haute (#284)
Sans base récente : *Visite guidée* / *Base d'exemple* / *Importer mes
matchs* / *Ouvrir*. Le parcours « importer mes matchs » enchaîne I.1 : « voici
votre PR et vos 10 pires erreurs » en deux minutes. Avec H.12.

### I.29 — Corbeille et annulation [M] — tous (#285)
12 `confirmAction` irréversibles. Corbeille des positions (30 jours,
restauration par hash Zobrist), « annuler » sur la suppression d'une carte,
d'une collection, d'un commentaire (soft delete `deleted_at`, vague de
schéma suivante ; purge par `vacuum`).

### I.30 — Thèmes nommés : clair, sombre, contraste élevé, imprimable [M] — tous (#286)
Après D.9 (tokens). Le plateau two.js lit les tokens ; export/import d'un
thème ; `prefers-color-scheme` par défaut.

### I.31 — Confort d'usage quotidien [S chacune] — tous (#287)
Journal d'activité consultable (les 75 `catch` deviennent diagnosticables,
bouton « copier le rapport ») ; palette `Ctrl+K` floue sur commandes,
onglets, matchs, filtres sauvegardés ; badges lisibles dans l'historique de
recherche (`filterTokenHint` existe) ; frise du transcript colorée par
gravité ; compteur « 412 positions · 38 blunders · 5 matchs » cliquable ;
badge de cartes dues sur l'onglet Anki ; filtres favoris épinglés ; pipcount
vivant en mode EDIT ; planche-contact des résultats (grille de
mini-plateaux, après I.22).

### I.32 — Fusionner deux bases, et dire la vérité sur la synchronisation [S] — compét (#288)
Documenter que la base est un fichier unique (Dropbox/Syncthing marchent si
l'on n'ouvre pas des deux côtés ; `serve` est la réponse multi-postes) ;
bouton « fusionner depuis… » qui expose `import_db` comme geste de
synchronisation (la dédup Zobrist fait le merge).

### I.33 — API stable et client minimal [M] — dev (#289)
Après G.8 (OpenAPI) : `clients/python/` généré, exemples, politique de
versionnage de `/v1` écrite ; `pkg/blunderdb/server.Bootstrap` documenté
pour gammonGo.

### I.34 — Capacités serveur-seules remontées dans l'application [M] — compét (#290)
Suspendre / enterrer / retirer une carte, journal de révision, optimisation
FSRS, réparation des analyses : six capacités qui n'existent que sur `/v1`
(G.14). Menu contextuel d'une carte, onglet Anki, Configuration.

---

## Résumé du lot

| Fiches | Effort | Prérequis techniques |
|---|---|---|
| I.6, I.16, I.22, I.25, I.32 | S | D.11 ; — ; — ; — ; — |
| I.5, I.8, I.14, I.17, I.24 | S/M | corpus ; — ; — ; — ; — |
| I.1, I.2, I.3, I.4, I.7, I.9, I.10, I.11, I.12, I.13, I.15, I.18-21, I.23, I.26-31, I.33, I.34 | M | voir fiche |

Bumps de schéma demandés par ce lot : I.1 (lot d'import), I.7 (origine des
commentaires), I.8 (phase), I.29 (soft delete) — à grouper en **une vague
2.17.0** après la 2.16.0 du lot B.
