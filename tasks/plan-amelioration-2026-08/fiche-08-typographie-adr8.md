# Fiche 08 — Achever la migration typographique (ADR-0008)

Branche : `design/adr8-final`

## Objectif

Terminer la décision « one type scale » : trait global `font: inherit`,
tokens pour les tailles de dialogue, zéro taille absolue inexpliquée.

## Constats

- 11 composants boutons-seuls sans `font: inherit` (`EPCPanel`,
  `FileImportProgressModal`, `HelpModal`, `ImportProgressModal`,
  `TabbedPanel`, `Toolbar`, `TourCatalogModal`, `stats/ContextMenu`,
  `stats/StatsDashboardTab`, `stats/StatsPanel`,
  `stats/StatsProgressionTab`) ; les 19 composants restants déclarent le
  trait localement, en 19 exemplaires.
- `GoToPositionModal.svelte:206-213` : `.input-field { font-size: 18px }`
  écrase son propre `font: inherit` — seule violation directe de la règle.
- L'exception « titre de dialogue » s'est fragmentée en 4 valeurs
  (15px token / 20px / 1.25rem / 12px) et les croix de fermeture en 3
  (24px / 1.5rem / 18px).
- Absolus hors exceptions ADR : `App.svelte:726` (1.3rem, overlay de drop),
  compteurs d'import (`FileImportProgressModal:162`,
  `ImportProgressModal:177`, 28px), `MergePlayersModal:249` (18px),
  `TourCatalogModal:89` (1.25rem).

## Tâches

- [ ] `frontend/src/style.css` : ajouter le trait global
      `input, select, textarea, button { font: inherit; }` puis supprimer les
      19 déclarations locales devenues redondantes (et leurs commentaires
      recopiés).
- [ ] Contrôle visuel des 11 composants nouvellement affectés (onglets
      `TabbedPanel`/`StatsPanel`, boutons `TourCatalogModal`, `HelpModal`) —
      captures avant/après via `make dev` si possible, sinon revue CSS
      attentive des tailles calculées.
- [ ] Corriger `GoToPositionModal:212` (token au lieu de 18px).
- [ ] Nouveaux tokens dans `style.css` : `--font-size-dialog-title` et
      `--font-size-dialog-close` ; convertir les 4 variantes de titres et les
      3 variantes de croix.
- [ ] Ramener aux tokens (ou à défaut nommer en exception) : overlay de drop
      d'`App.svelte:726`, compteurs d'import 28px (les aligner sur l'exception
      « chiffres statistiques » en l'élargissant), croix de
      `MergePlayersModal`.
- [ ] Amender `docs/adr/0008-*.md` : le trait global est posé, les exceptions
      sont re-nommées précisément ; la commande de mesure de l'ADR doit
      retomber à zéro absolu inexpliqué.

## Critères de fin

- La commande de mesure de l'ADR ne liste plus que des exceptions nommées.
- `npm run lint`, `format:check`, vitest verts ; pas de régression visuelle
  flagrante sur les 4 écrans principaux.

## Risques & garde-fous

- Changement purement visuel mais large : commit séparé pour le trait global
  (revert facile) et pour les tokens de dialogue.
- Ne pas toucher aux exceptions légitimes (chiffres stats 28px, chrome).
