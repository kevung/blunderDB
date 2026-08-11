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

- [ ] **Aide intégrée ×9** : retirer `filter, fl` et les migrations mortes ;
      ajouter `tutorial/tour`, `demo`, `blunders/bl`, `stats/st` et les 7
      jetons de filtre manquants. Français d'abord, puis décliner les 8
      autres langues (les libellés de commandes sont identiques, seules les
      descriptions se traduisent).
- [ ] **Test de sync aide↔vocabulaire** : toute entrée de `COMMANDS`
      (`commandVocabulary.js`) apparaît dans la table des commandes de
      `help/fr.js`, et aucune entrée de l'aide n'est absente du vocabulaire.
      Même mécanique pour les jetons de filtres si praticable.
- [ ] **README** : corriger l'exemple (`--error-min`), Go 1.25, METs réelles,
      9 langues ; remplacer la capture d'écran si une session graphique est
      disponible (`make dev` + capture), sinon retirer la capture périmée et
      noter l'action dans FOLLOWUPS.
- [ ] **cli.rst** : sections `identity`, `open`, `epc` ; options manquantes
      d'`export` et `search` (source : CLI_USAGE.md + `internal/cli/`).
- [ ] **manuel.rst** : quatre onglets de configuration (décrire l'onglet
      Bearoff) ; « EPC » → « Bearoff » ; mentionner le bouton « Aller à la
      position ».
- [ ] **mode_headless.rst** : compléter l'énumération (`exports`, `history`,
      `tenant`).
- [ ] **guide_utilisateur.rst** : libellés localisés (citer les libellés FR
      réels), ajouter des renvois vers Anki/vues/diffusion/demo/blunders.
      `annexe_filtres.rst` : libellé FR réel.
- [ ] Vérifier la build Sphinx : `.venv` puis `cd doc && python build.py`
      (ou a minima `sphinx-build -b html` sur le français).

## Critères de fin

- Nouveau test de sync vert ; le rendre rouge en retirant une commande de
  l'aide (vérifié puis restauré).
- `cmd_mode.rst`/`cli.rst`/`manuel.rst` cohérents avec le code au périmètre
  audité ; build Sphinx FR sans erreur.
- README sans affirmation fausse.

## Risques & garde-fous

- Ne toucher que la source française des .rst (les 8 traductions gettext se
  rattrapent à la release, règle projet) — MAIS l'aide intégrée `help/*.js`
  n'est PAS gettext : les 9 fichiers se modifient dans la branche.
- Attention au JA : pas de markup RST inline collé à du CJK (mémoire projet).
