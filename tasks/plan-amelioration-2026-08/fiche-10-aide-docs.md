# Fiche 10 — Aide intégrée et documentation à jour

Branche : `docs/aide-et-docs`

## Objectif

Que l'aide `?` reflète le produit livré (avec un test qui verrouille), que le
README ne mente plus, que `cli.rst` et le manuel rattrapent le code.

## Constats

- Aide intégrée (`frontend/src/i18n/help/*.js`, 9 langues) : documente
  `filter, fl` **supprimée** (l.~741-744 — et `fl` est devenu un jeton de
  filtre au sens différent) ; ignore les commandes `tutorial/tour`, `demo`,
  `blunders/bl`, `stats/st` et les filtres `D1`, `xD65`, `fl`, `co`, `xco`,
  `id`, `pl'nom'` ; liste encore les 3 commandes de migration mortes
  (supprimées en fiche 07).
- Aucun test ne lie l'aide au vocabulaire (le test de sync existant ne couvre
  que `commandVocabulary` ↔ `commandProcessor`).
- `README.md` : exemple `--error` (l'option est `--error-min`), « Go 1.23+ »
  (go.mod exige 1.25.12), MET « Rockwell » inexistante, « French + English »
  (9 langues) ; capture d'écran du 2026-04-20 montrant l'onglet Log supprimé
  et sans l'onglet Stats.
- `doc/source/cli.rst` : 12 commandes sur 15 (manquent `identity`, `open`,
  `epc`) ; options non documentées : `export --password/--watermark/
  --watermark-note`, `search --flagged/--has-comment/--no-comment`.
  (`CLI_USAGE.md`, lui, est à jour — s'en servir de source.)
- `doc/source/manuel.rst:161` : « trois onglets » de configuration alors
  qu'il y en a quatre (Bearoff manquant — la feature phare 0.32.0) ; `:713`
  parle de l'onglet « EPC » renommé « Bearoff ».
- `doc/source/mode_headless.rst` : familles `exports`, `history`, `tenant`
  absentes de l'énumération des endpoints.
- `doc/source/guide_utilisateur.rst:12-38` : libellés de boutons en anglais
  (« New Database »…) alors que l'UI est localisée ; fonctionnalités majeures
  absentes (Anki, vues, diffusion filigranée, demo/tours, blunders).
  `annexe_filtres.rst:47` : « Search in current results » en dur.
- Modal « Aller à la position » (bouton toolbar) absent de toute doc.

## Tâches

- [x] **Aide intégrée ×9** : retirer `filter, fl` et les migrations mortes ;
      ajouter `tutorial/tour`, `demo`, `blunders/bl`, `stats/st` et les 7
      jetons de filtre manquants. Français d'abord, puis décliner les 8
      autres langues (les libellés de commandes sont identiques, seules les
      descriptions se traduisent).
- [x] **Test de sync aide↔vocabulaire** : toute entrée de `COMMANDS`
      (`commandVocabulary.js`) apparaît dans la table des commandes de
      `help/fr.js`, et aucune entrée de l'aide n'est absente du vocabulaire.
      Même mécanique pour les jetons de filtres si praticable.
- [x] **README** : corriger l'exemple (`--error-min`), Go 1.25, METs réelles,
      9 langues ; remplacer la capture d'écran si une session graphique est
      disponible (`make dev` + capture), sinon retirer la capture périmée et
      noter l'action dans FOLLOWUPS.
- [x] **cli.rst** : sections `identity`, `open`, `epc` ; options manquantes
      d'`export` et `search` (source : CLI_USAGE.md + `internal/cli/`).
- [x] **manuel.rst** : quatre onglets de configuration (décrire l'onglet
      Bearoff) ; « EPC » → « Bearoff » ; mentionner le bouton « Aller à la
      position ».
- [x] **mode_headless.rst** : compléter l'énumération (`exports`, `history`,
      `tenant`).
- [x] **guide_utilisateur.rst** : libellés localisés (citer les libellés FR
      réels), ajouter des renvois vers Anki/vues/diffusion/demo/blunders.
      `annexe_filtres.rst` : libellé FR réel.
- [x] Vérifier la build Sphinx : `.venv` puis `cd doc && python build.py`
      (ou a minima `sphinx-build -b html` sur le français).

## Critères de fin

- [x] Nouveau test de sync vert ; le rendre rouge en retirant une commande de
  l'aide (vérifié puis restauré).
- [x] `cmd_mode.rst`/`cli.rst`/`manuel.rst` cohérents avec le code au périmètre
  audité ; build Sphinx FR sans erreur.
- [x] README sans affirmation fausse.

## Notes d'exécution (2026-08-11)

Constats re-vérifiés par grep avant action (les fiches 03/07/09 étaient déjà
mergées dans ce worktree) :

- Les 3 commandes de migration mortes étaient **déjà absentes** de
  `help/*.js` (fiche 07 les avait retirées) — rien à faire sur ce point,
  contrairement à ce que disait le constat initial de la fiche.
- `doc/source/cmd_mode.rst` était **déjà à jour** (tutorial/demo/stats/
  blunders et les 7 jetons de filtre D1/xD65/fl/co/xco/id/pl'nom' y étaient
  tous documentés) — il a servi de source de vérité pour rédiger les
  nouvelles entrées de `help/fr.js`, plutôt que d'être modifié lui-même.
- `printUsage` (main.go) listait déjà `serve`/`call`/`migrate` — rien à
  compléter.
- La sous-commande `vacuum` n'existe pas (fiche 06 non faite) — non
  documentée, conformément à la consigne.

Aide intégrée : suppression de la ligne fantôme `filter, fl` (table
Commandes) dans les 9 langues ; ajout des lignes `tutorial, tour`, `demo`,
`stats, st`, `blunders, bl [n]` (table Commandes) et `D1`, `xD65`, `fl`, `co`,
`xco`, `idx`, `idx,y`, `pl'nom'` (table Filtres) dans les 9 langues, chaque
description traduite indépendamment (pas de recopie de l'anglais) ; espaces
autour du markup latin soignées en japonais (parenthèses pleine largeur,
espace demi-chasse autour des tokens latins, cohérent avec le style déjà en
place dans `help/ja.js`).

Test de sync : `frontend/src/__tests__/helpVocabulary.sync.test.js`, nouveau
fichier. Trois volets : (1) toute entrée de `COMMANDS` apparaît dans la table
Commandes de `help/fr.js` et réciproquement (parsing DOM via jsdom, exclut
`[number]` et `#tag...` comme le fait déjà `commandVocabulary.js`) ; (2) tout
jeton vérifié par un `filters.includes('...')` littéral dans
`commandProcessor.js` apparaît dans la table Filtres de `help/fr.js` — lien
direct au code, pas seulement à `cmd_mode.rst`, donc un futur ajout de jeton
booléen dans le parseur sans mise à jour de l'aide fera échouer ce test ; les
filtres à préfixe/plage (`p>x`, `max`, `idx`, …), matchés par `.startsWith()`/
regex plutôt que `.includes()`, ne sont **pas** couverts mécaniquement — seul
`cmd_mode.rst` en reste la source de vérité relue à la main (documenté en tête
du fichier de test) ; (3) les 8 autres langues sont vérifiées par comptage de
lignes `<tr>` dans leur table Commandes contre `fr.js` (structurel, ne
vérifie pas la qualité de traduction).

Vérification rouge → vert : ligne `blunders, bl [n]` retirée de
`help/fr.js`, `npx vitest run helpVocabulary.sync.test.js` → 9/12 tests
rouges (l'entrée manquante, plus les 8 comparaisons de comptage de lignes qui
suivent l'écart en cascade) ; fichier restauré depuis une copie, tests de
nouveau verts (12/12), diff vérifié identique à l'original.

README : `--error` → `--error-min` (avec la vraie syntaxe, un nombre nu, pas
`">0.05"`) ; Go 1.23+ → Go 1.25+ (`go.mod` exige 1.25.12) ; MET « Kazaross,
Rockwell, … » → « Kazaross-XG2 » (une seule table est embarquée, pas de
Rockwell — vérifié dans `frontend/src/stores/metTable.js` et
`doc/source/manuel.rst`) ; « French + English » → 9 langues. Capture d'écran
retirée (pas de session graphique disponible ; l'image datait du 2026-04-20,
montrait un onglet Log supprimé et pas d'onglet Stats) ; ligne ajoutée à
`tasks/FOLLOWUPS.md` (#7) pour refaire la capture en 0.32+.

cli.rst : sections `identity`, `open`, `epc` ajoutées (contenu aligné sur
`CLI_USAGE.md`/`internal/cli/cli_identity.go`/`cli_epc.go`) ; options
`--password`/`--watermark`/`--watermark-note` ajoutées à `export`,
`--flagged`/`--has-comment`/`--no-comment` ajoutées à `search` (vérifiées
dans `internal/cli/cli_search.go`) ; table récapitulative des commandes mise
à jour (15 commandes). Les changements de sortie CLI de la fiche 03 (version
applicative réelle, messages de connexion sur stderr) ne touchent aucun texte
déjà présent dans `cli.rst` — rien à refléter.

manuel.rst : « trois onglets » → « quatre » avec un nouveau puce/paragraphe
décrivant l'onglet Bearoff (domaine actif TS-06-06/TS-06-11, téléchargement
avec reprise HTTP Range, suppression avec confirmation — fiche 07, sélection
d'un fichier `.bd` externe), lu directement dans `ConfigModal.svelte` ;
« onglet EPC » → « onglet Bearoff » (panneau Bearoff, ~l.725-730) ; mention du
bouton « Aller à la position » de la barre d'outils dans la section
Navigation dans les positions, avec renvoi à `cmd_positions`.

mode_headless.rst : énumération des familles complétée avec « historique »,
« import et export » (au lieu du seul « import »), et le cycle de vie des
tenants (`tenant.purge`) — vérifié contre `internal/server/routes.go` et
`handlers_tenant.go`/`handlers_session.go`/`handlers_imports.go`. Les routes
`stats.tournamentBadges` et `matches.findByHash` (fiche 03) ne sont pas
mentionnées : la doc n'énumère jamais les méthodes individuelles des familles
`stats`/`matches`, seulement à l'échelle famille (déjà couvertes par
« statistiques » et « matchs »).

guide_utilisateur.rst : libellés anglais des boutons toolbar remplacés par
les libellés FR réels (`toolbar.newDatabase`/`openDatabase`/`importDatabase`
dans `frontend/src/i18n/locales/fr.json`) ; nouvelle section « Pour aller plus
loin » avec un renvoi de 2-3 phrases chacun vers Anki (`panneau_anki`), les
vues multiples (`onglets_vues`), la diffusion filigranée
(`diffusion_controlee`), les visites guidées/`demo` (`visites_guidees`) et le
chargement des pires erreurs (`stats`). `annexe_filtres.rst:47` : « Search in
current results » → « Rechercher dans les résultats actuels » (libellé réel
de `search.inResults` dans `fr.json`).

Build Sphinx : `.venv` du checkout principal activé (`source
/home/unger/src/blunderDB/.venv/bin/activate`), `cd doc && python build.py` —
succès (`EXIT=0`) pour les 9 langues (HTML + PDF). Aucun `WARNING:` Sphinx
(HTML) dans le log ; les seuls avertissements sont des `LaTeX Warning: Hyper
reference ... undefined` de première passe, résolus par les reruns
automatiques de latexmk (comportement normal multi-passe, pas une régression
introduite par cette fiche — le PDF final (87 pages, 9 langues) est produit
sans erreur).

Validation frontend avant chaque commit : `npm run lint`, `npm run
format:check`, `npm test -- --run` — tous verts (727 tests, dont les 12 du
nouveau fichier de sync).

## Risques & garde-fous

- Ne toucher que la source française des .rst (les 8 traductions gettext se
  rattrapent à la release, règle projet) — MAIS l'aide intégrée `help/*.js`
  n'est PAS gettext : les 9 fichiers se modifient dans la branche.
- Attention au JA : pas de markup RST inline collé à du CJK (mémoire projet).
