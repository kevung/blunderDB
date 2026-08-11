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

- [x] `frontend/src/style.css` : ajouter le trait global
      `input, select, textarea, button { font: inherit; }` puis supprimer les
      19 déclarations locales devenues redondantes (et leurs commentaires
      recopiés). Fait — grep confirmait 19 fichiers (pas 11 boutons-seuls
      distincts comme l'ancien constat le disait ; la base avait bougé, voir
      Notes d'exécution).
- [x] Contrôle visuel — pas de `make dev` disponible dans cet environnement ;
      revue CSS attentive à la place (voir Notes d'exécution pour la liste
      des composants à contrôler à l'écran).
- [x] Corriger `GoToPositionModal` `.input-field` (token au lieu de 18px) →
      `var(--font-size-dialog-title)`, avec commentaire justifiant le choix.
- [x] Nouveaux tokens dans `style.css` : `--font-size-dialog-title` (20px) et
      `--font-size-dialog-close` (24px) ; converti les 4 variantes de titres
      (15px token / 20px / 1.25rem / 12px) et les 3 variantes de croix
      (24px / 1.5rem / 18px). `ConfigModal` et `ProtectedCopyModal` gardent
      `--font-size-title` (15px) sur leur titre, délibérément — dialogues
      compacts, documenté dans l'ADR.
- [x] Ramené aux tokens : overlay de drop d'`App.svelte` (1.3rem →
      `--font-size-dialog-title`), compteurs d'import 28px (`FileImportProgressModal`,
      `ImportProgressModal` → nouveau token `--font-size-stat-figure`, exception
      « chiffres statistiques » élargie dans l'ADR), croix de `MergePlayersModal`
      et son titre 12px (tous deux → tokens dialogue).
- [x] Amendé `docs/adr/0008-*.md` : le trait global est posé (2026-08-11), les
      exceptions sont re-nommées précisément ; la commande de mesure de l'ADR
      retombe à zéro absolu inexpliqué (voir sortie avant/après dans l'ADR,
      section Consequences).

## Critères de fin

- La commande de mesure de l'ADR ne liste plus que des exceptions nommées.
- `npm run lint`, `format:check`, vitest verts ; pas de régression visuelle
  flagrante sur les 4 écrans principaux.

## Risques & garde-fous

- Changement purement visuel mais large : commit séparé pour le trait global
  (revert facile) et pour les tokens de dialogue.
- Ne pas toucher aux exceptions légitimes (chiffres stats 28px, chrome).

## Notes d'exécution (2026-08-11)

- Les constats datent d'avant le merge de la fiche 07, qui a supprimé
  `SearchHistoryPanel.svelte` ; ce fichier n'apparaît nulle part dans le
  travail ci-dessous, conformément à la consigne.
- Re-vérification par grep de `font: inherit` : 19 fichiers le déclarent
  localement, chiffre confirmé (liste : `AnkiPanel`, `CollectionPanel`,
  `CommentPanel`, `ConfigModal`, `EPCPanel`, `ExportDatabaseModal`,
  `GoToPositionModal`, `MatchPanel`, `MatchTournamentPickerModal`,
  `MergePlayersModal`, `MetadataPanel`, `MinMaxFilterRow`,
  `ProtectedCopyModal`, `SearchPanel`, `stats/StatsFilterBar`, `StatusBar`,
  `TournamentPanel`, `ViewTabs`, `WarningModal`). Les 11 « composants
  boutons-seuls » cités dans le constat d'origine n'ont pas été revérifiés
  un par un puisque le trait global les couvre désormais sans qu'aucune
  action locale ne soit nécessaire.
- Suppression : les blocs `input, select, textarea, button { font: inherit; }`
  identiques au trait global (avec leur commentaire recopié) ont été retirés
  en entier. Là où `font: inherit` cohabitait avec d'autres propriétés dans
  le même bloc (`ConfigModal .tab`, `EPCPanel .challenge-toggle input` et
  `.link-button`, `MetadataPanel input, textarea`, `WarningModal
  .btn-cancel, .btn-confirm`), seule la ligne redondante a été retirée.
- `MergePlayersModal.svelte` avait aussi un titre à 12px
  (`var(--font-size-base)` sur `.modal-title`) — c'est la 4ᵉ variante « titre
  de dialogue » évoquée dans les constats, pas seulement sa croix à 18px ;
  les deux ont été alignées sur les nouveaux tokens.
- Compteurs d'import 28px : choix fait d'un token partagé
  `--font-size-stat-figure` plutôt que d'élargir seulement le texte de
  l'ADR, pour que la commande de mesure les fasse disparaître de la liste
  des absolus (ils passent par `var(...)`, donc filtrés par le `grep -v
  'var(--font-size'` de la commande).
- `App.svelte` (overlay de drop, 1.3rem ≈ 20.8px) a été raccroché à
  `--font-size-dialog-title` (20px) plutôt que traité en exception séparée —
  la fiche demandait explicitement un token ici. Différence visuelle : environ
  0.8px, imperceptible.
- Composants à contrôler à l'écran (pas de `make dev` dans cet environnement
  d'exécution) : onglets `TabbedPanel`/`StatsPanel` et leurs enfants
  `StatsDashboardTab`/`StatsProgressionTab` (chiffres 28px désormais via
  token), `TourCatalogModal` (titre + croix), `HelpModal` (croix),
  `GoToPositionModal` (champ de saisie agrandi, croix), `MergePlayersModal`
  (titre agrandi 12px→20px, croix), `App.svelte` (overlay de dépôt de
  fichier). Le changement le plus visible est le titre de `MergePlayersModal`
  qui passe de 12px à 20px — à confirmer que la mise en page de son en-tête
  ne déborde pas.
- Sortie de la commande de mesure de l'ADR (`cd frontend/src && grep -rho
  'font-size:[^;]*;' components/ *.svelte | grep -v 'var(--font-size' | sort
  | uniq -c | sort -rn`) :
  - avant : `4×28px, 4×20px, 3×1.5rem, 2×inherit, 2×24px, 2×18px, 2×0.92em,
    1×1.3rem, 1×1.25rem` (21 lignes, 9 valeurs)
  - après : `2×inherit, 2×0.92em` (4 lignes, 2 valeurs, toutes déjà nommées
    par les règles 1 et 4 de l'ADR)
- `go vet`/`go test` non lancés : aucun fichier Go touché par cette fiche.
