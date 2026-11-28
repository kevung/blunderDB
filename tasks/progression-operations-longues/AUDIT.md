# Opérations longues : progression, annulation, blocage (audit du 2026-11-27)

Déclencheur : l'archive papier n'affiche aucune progression et bloque l'utilisateur.
`Database.mu` est le vrai point de gel : un `Lock` long gèle tout ; un `RLock` long aussi
dès qu'un écrivain attend (le RWMutex de Go fait passer l'écrivain avant les lecteurs).

| Opération | Entrée frontend | Go | Progression | Annulation | Blocage / verrou | Gravité |
|---|---|---|---|---|---|---|
| Export de base | ExportDatabaseModal, exportService.js:325 | ExportDatabase → ExportDatabaseCtx(Background) | spinner | **fausse** : handleExportCancel ferme la boîte, l'export continue | Lock en écriture tout l'export | critique |
| Archive papier, export | PaperArchiveModal, paperArchiveService.js:24 | ExportPaperArchive (db_paper_archive.go:26) | non | non (Background, alors qu'ingest.ExportArchive teste ctx) | boîte busy ; RLock pendant l'encodage QR ; toute la base sans plafond | haute |
| Archive papier, import | PaperArchiveModal, paperArchiveService.js:45 | ImportPaperArchive | non | moteur oui, bouton non | boîte busy ; Lock pour les écritures ; absent du panneau Calculs | haute |
| Import de document | DocumentImportModal, documentImportService.js:72 | AnalyzeDocument (db_document.go:36) | non | moteur oui (CancelImport), boutons désactivés pendant busy | modale verrouillée des minutes | haute |
| Liste de positions / collection (≤ 3000) | PositionPrintModal, positionSheetService.js:88 | PositionSheetHTML | non | non | busy ; RLock | moyenne-haute |
| Feuille de matchs (≤ 100) | MatchPrintModal, matchSheetService.js:41 | MatchSheetHTML | non | non | busy ; RLock | moyenne |
| Aperçu de l'export | ExportDatabaseModal | PlanExportCtx(Background) | non | non | RLock | moyenne |
| Export .apkg | AnkiPanel, ankiService.js:567 | ExportAnkiPackage | non | non | RLock ; double clic possible | moyenne |
| Import .db, analyse | importService.js:166 | AnalyzeImportDatabase | non | non | modale ; RLock | moyenne |
| Import .db, validation | importService.js:217/264 | CommitImportDatabase (tracked) | spinner | oui | Lock tout l'import | moyenne |
| Rapport de stats | reportService.js | ComputeStats + StatsReportHTML (Ctx avec Background) | non | non | RLock | moyenne |
| Stats / rolling stats | StatsPanel, onglets | ComputeStats, PRByWindow, HeadToHead, PlayerRanking | « chargement » | non : un filtre changé laisse l'ancien calcul tourner | RLock | moyenne |
| Transfert entre bibliothèques | transferService.js:67 | App.TransferToGroup | tâche sans Progress | oui | RLock source / Lock destination | moyenne |
| Vider la corbeille | TrashModal | EmptyTrash | non | non | busy ; Lock | faible-moyenne |
| Réparation vidéo | VideoPane | App.RepairVideoMux | non | en fermant le volet | volet seul | faible |

Déjà corrects : import de fichiers, maintenance/recalcul, GammonNet par lot, rollouts, bearoff,
migration à l'ouverture, recomptage des gaffes, synchro.

## Modèle à généraliser

`App.StartMaintenance` (internal/gui/maintenance.go) et gammonnet_batch.go : retour immédiat,
`context.WithCancel(a.batchCtx())`, `a.jobs.Start(jobs.Spec{Kind, Label, Library, Cancel})`,
goroutine `task.Turn(ctx)` → `Database.XCtx(ctx, …, onProgress)` → `task.Progress` + événement
limité (~4/s) → `x:done|cancelled|error` + `task.Finish(err)`. `jobs:changed` alimente déjà
jobsStore, JobsModal (`<progress>`) et la pastille de la barre d'état : ajouter le genre dans
jobs.go et `KIND_KEYS` de JobsModal. Côté moteur : propager ctx au lieu de Background
(ingest.ExportArchive, printsheet.Build*, apkg.Export, ExportSQLite), progression par match/lot,
`d.mu` par lot plutôt que pour toute l'opération.

Variante légère pour les appels directs : `tracked(d, kind, label, d.CancelImport, f)`
(task_tracker.go) et un bouton Annuler actif pendant busy. Boîte de référence avec barre,
vitesse et temps restant : FileImportProgressModal.svelte, alimentée par `import-files:progress`.
