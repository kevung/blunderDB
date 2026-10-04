# Outillage de la simulation 2026-10 : la vraie Direction, mesurée geste par geste

Le vrai front Svelte (Vite, `frontend/`) parle au vrai `*database.Database` Go (vraie SQLite,
vrai moteur Nicomaque) par un shim HTTP. **Rien n'est simulé côté backend** : aucune réponse
n'est fabriquée et `installWailsMock` n'est pas utilisé. Seules les liaisons natives sans
équivalent dans un navigateur (dialogues, fenêtre, presse-papiers) sont figées, et journalisées.

## Lancer

```bash
cd tasks/nicomaque/simulation-2026-10/outils
./run.sh e2e/smoke.spec.js          # ≈ 30 s, doit être vert
SIM_KEEP=1 ./run.sh e2e/t1.spec.js  # garde base, sorties et meter.jsonl dans $SIM_DIR/<nom>
```

`run.sh` relie `node_modules` (ignoré par git) depuis le dépôt principal si le worktree n'en a
pas, fixe `BLUNDERDB_E2E_PORT=5183` (Vite en `--strictPort`, jamais de réutilisation d'un
serveur étranger) et `SIM_DIR=/tmp/blunderdb-sim`. Chromium : `/usr/bin/chromium` (`CHROMIUM`
pour un autre). Un seul worker : chaque spec tient son shim, sur son port, et sa base.

| Fichier | Rôle |
|---|---|
| `shim/main.go` | `//go:build simulation` (hors CI). `POST /call/<Méthode>` par réflexion, `GET /methods`, `GET /warnings` (avertissements de page murale que le GUI met en barre d'état), `GET /stats` (appels par méthode), `POST /savecsv` |
| `e2e/realBackend.js` | `installRealBackend(page, {shimUrl, dbPath, outDir})` ; `shimCall(url, Méthode, …args)` pour l'oracle ; `shimPageWarnings(url)` |
| `e2e/shimProcess.js` | `new Shim({name, port, fresh})` : `start()`, `kill()` (SIGKILL), `restart()`, `dispose({keep})` |
| `e2e/meter.js` | `new Meter(page, {persona, tournament, viewport, input, file})`, `meter.act(meta, g => …)`, `summarize(jsonl)`, `VIEWPORTS`, `zoomViewport` |
| `playwright.config.js`, `package.json`, `run.sh` | configuration jetable, ESM |

## Le shim

- Base absente : `SetupDatabase` puis `Close` ; c'est ensuite **le front** qui l'ouvre
  (`StartupFilePath` → `OpenDatabase`), comme un lancement par association de fichier. Ouvrir
  dans le shim puis laisser le front rouvrir reprendrait le verrou d'écriture deux fois.
- Convention Wails : `(T, error)` résout `T` ou rejette le message ; `error` seul résout `null`.
  Un paramètre `context.Context` reçoit `context.Background()` sans consommer d'argument.
  Une panique est rendue en erreur, le shim survit.
- `kill()` = SIGKILL : ni `Close`, ni `PRAGMA optimize`, verrou et WAL laissés tels quels ;
  `restart()` relance sur la même base, puis `page.reload()` refait l'ouverture du front.
  `new Shim({fresh: false})` reprend une base laissée par une spec précédente.

## Liaisons figées dans le navigateur

- `main.Config` : préférences en mémoire (fr, thème clair, visite déjà vue, MCP et dossier
  surveillé éteints).
- `gui.App` : `StartupFilePath` → la base du shim ; `PathExists` → vrai ;
  `OpenDirectionOutputDialog` → `<SIM_DIR>/<nom>/sortie` (la page murale y est **vraiment**
  écrite, par le backend) ; `SaveCSV(nom, corps)` → écrit par le shim dans ce même dossier ;
  `ShowQuestionDialog` → `window.__questionAnswer` ou le premier bouton (consigné dans
  `window.__dialogs`) ; `ShowAlert` consigné ; `ConfigureMCPHost`, `GetMCPHostStatus` → `{}`.
  Toute autre méthode `App` résout `null` (moteur d'analyse, rollouts, bearoff : hors Direction).
- `runtime` : bus d'événements local, `ClipboardSetText` → `window.__clipboard`,
  `BrowserOpenURL` → `window.__opened` (compté, n'ouvre rien).
- Journal de tous les appels : `window.__calls` (`{ns, name, args, t}`).

## Le mesureur

Une ligne JSONL par action ; champs documentés en tête de `meter.js`. Ce qu'il faut savoir :

- **scrollTop remis à 0** dans tous les conteneurs avant chaque action (leçon de 2026-09).
- `autoScroll` : ce que Playwright a défilé en silence pour atteindre la cible = ce que la
  personne aurait dû faire à la molette (≈ 100 px par cran). `hiddenOnly` : le conteneur défilé
  est en `overflow: hidden/clip`, donc inatteignable à la main.
- Chaque cible cliquée est sondée au viewport du persona, puis à 1366×768 et 1920×1080 (bascule,
  sonde, retour) : hors écran, masquée (`elementFromPoint` au centre, masque nommé), < 24 px,
  texte tronqué. `checkViewports: false` pour accélérer une boucle de saisie répétitive.
- `input: 'keyboard'` : `g.click` devient Tab jusqu'à la cible (Tab comptés, ordre relevé, focus
  visible contrôlé) puis Entrée ; injoignable après 80 Tab → problème noté, repli souris.
- Zoom 150 % : `zoomViewport(vp, 1.5)` (viewport CSS divisé par 1,5, comme Ctrl+ ; le
  `deviceScaleFactor` ne change pas la mise en page et ne mesurerait rien).
- `g.hesitate(raison, fn)` compte un écran ouvert pour rien (novice) ; `g.ecart(texte)` et
  `meta.ecart` portent ce que le directeur voulait et ce que la page proposait.
- `feedback` ne voit que ce qui ressemble à un retour (statut, alerte, `aria-live`, bandeau,
  toast, barre d'état, `direction-last`, confirmations) dans `METER_FEEDBACK_MS` (1500 par
  défaut) : une ligne de tableau qui apparaît n'en est pas un, c'est voulu.

## Inventaire des prises de la Direction (`data-testid`)

- **Entrée** : `tab-tournaments` ; `#tournamentPanel [data-testid="panel-new"]` puis nom +
  Entrée ; la ligne s'ouvre au **double-clic** ; `tournament-direct` crée la Direction avec
  `suisse_tableau` **et l'ouvre** (vue chargée à la demande : l'attendre, ne pas cliquer
  `tournament-direction-toggle`, qui la refermerait).
- **Onglets** : `direction-tab-{direction,players,brackets,slots,standings,history,settings}` ;
  `direction-open-page` (page murale), `direction-close`, `direction-fullscreen-toggle`.
- **Réglages** : `direction-format-{suisse_tableau,elimination,consolante,double_elimination,gsl,poules}`,
  `direction-settings-tables`, `-unavailable`, `-group-size-N`, `-qualifiers-N`, `-apply`,
  `-changes`, `-confirm`, `-output`, `-add-break`.
- **Joueurs** : `direction-player-entry` (1er champ texte = nom, puis club, cote ; Entrée
  inscrit), `direction-player-filter`, lignes `direction-player-{id}` (menu au clic droit ou
  au clavier) : `-withdraw-now`, `-withdraw-later`, `-reinstate`, `-absent*`, `-return`,
  `-correct`, `-slot` (retardataire sur une place libre).
- **File** : `.proposals .queue li:not(.empty)`, bouton `.go` (« Lancer »), `⋯` ;
  `direction-proposals-all`, `-confirm`, `-confirm-all` ; `direction-proposal-ignore` ;
  `direction-manual-open` (apparier à la main).
- **Grille** : `[role="gridcell"]` `direction-table-N`, `.busy` = match en cours ; clic → fiche
  `direction-result-card` : `-winner-a/-b`, `-score-a/-b`, `-forfeit-a/-b`, `-more`, `-move`,
  `-move-table`, `-cancel`, `-error`. Bandeau `direction-last` (`-cancel`, `-correct`).
- **Historique** : `direction-history-{seq}`, `-filter`, `-correct` (fiche `CorrectionPanel`).
- **Classement** : `direction-standings-csv`, `-save`, `-close` (`-confirm-go`), `-reopen`.
- **Rencontre** (Réglages) : `rencontre-name`, `rencontre-tables`, `rencontre-create`,
  `rencontre-attach` (`-confirm`, `-yes`), `rencontre-choose`, `rencontre-open-page` ;
  onglets d'épreuve `epreuve-tab-{id}`, `epreuve-tab-hall`, `hall-proposals-{id}`.
- **Feuilles** : `direction-sheet-print`, `direction-sheet-upcoming` (+ `-announced`, `-print`).

## Constats faits en route (à reprendre par les specs, pas corrigés)

- **C-H1** Toutes les cibles de l'entrée sont sous 24 px de haut à 1920×1080 : onglet Tournois
  40×14, « + Nouveau tournoi » 56×14, « Diriger » 52×12, « Ouvrir la direction » 56×12, onglets
  de la Direction 34×22, champ nom de l'inscription 120×13, « Lancer » 32×13. Suspects :
  `TournamentPanel.svelte` (`.direction-btn`), `NewButton.svelte`, `DirectionView.svelte`
  (onglets), `PlayersView.svelte`, `ProposalList.svelte` (`.go`).
- **C-H2** Focus perdu (sur `body`) après « Lancer » et après la saisie d'un vainqueur : le
  clavier repart du début de la page. Suspects : `ProposalList.svelte`, `ResultCard.svelte`.
- **C-H3** « Diriger » ouvre la Direction ; le bouton voisin « Ouvrir la direction » la
  referme si on clique avant que la vue (chargée à la demande) n'apparaisse — un double geste
  du novice ferme ce qu'il vient d'ouvrir. Suspect : `TournamentPanel.svelte` (`toggleDirection`).
- **C-H4** Ouvrir un tournoi demande un double-clic, sans indice visuel ; « Diriger » coûte
  donc 3 clics. Suspect : `TournamentPanel.svelte` (`onActivate`).
- **C-H5** Avec 4 inscrits et `suisse_tableau` (cible 16), la tête de file est « Phase
  suivante » puis « Tirage : Tableau de 8 places » : la suisse est sautée sans un mot. À
  confirmer à 16 joueurs (T1) ; suspects : `directionStore.js` (préréglage, `target`), moteur.
- **C-H6** L'inscription d'un joueur ne donne aucun retour hors la ligne ajoutée (pas de statut
  ni d'annonce `aria-live`).
