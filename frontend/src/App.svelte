<script>
    import { logger } from './utils/logger.js';
    import { onMount, onDestroy, untrack } from 'svelte';
    import { get } from 'svelte/store';
    import { fade } from 'svelte/transition';

    import { WindowGetSize } from '../wailsjs/runtime/runtime.js';
    import { isOnBoard } from './services/boardArea.js';
    import { SaveWindowDimensions, GetLastDatabasePath, SaveLastDatabasePath, GetLanguage } from '../wailsjs/go/main/Config.js';
    import { PathExists, StartupFilePath, CheckForUpdate } from '../wailsjs/go/gui/App.js';
    import { GetCheckForUpdates, GetTrainingSeedSources } from '../wailsjs/go/main/Config.js';
    import { metaStore } from './stores/metaStore.js';
    import { isNewerVersion } from './utils/semver.js';
    import { initLanguage, t, tMsg } from './i18n';
    import { initBoardColors } from './stores/boardColorsStore';
    import { initUIScale } from './stores/uiScaleStore';
    import {
        initPanelPosition,
        effectivePositionStore,
        PANEL_SIDE,
        initPanelSize,
        panelHeightStore,
        panelWidthStore,
        savePanelHeight,
        savePanelWidth,
        DEFAULT_PANEL_HEIGHT,
        DEFAULT_PANEL_WIDTH
    } from './stores/panelLayoutStore';

    import { databasePathStore } from './stores/databaseStore.js';
    import { positionStore, positionsStore, emptyPosition } from './stores/positionStore.js';
    import { analysisStore, emptyAnalysis } from './stores/analysisStore.js';
    import { currentPositionIndexStore, statusBarModeStore, positionReloadTriggerStore, activeTabStore, isAnyModalOpen } from './stores/uiStore.js';
    import { transcriptionWheelStore } from './stores/transcriptionStore.js';

    import { newDatabase, openDatabase, openDatabaseByPath, loadDemoDatabase, exitApp, setStatusBarMessage } from './services/databaseService.js';
    import {
        showPosition,
        loadAllPositions,
        reloadAllPositions,
        loadPositionsByFilters,
        previousPosition,
        nextPosition,
        saveCurrentPosition,
        updatePosition,
        deletePosition,
        toggleAnalysisPanel,
        toggleCommentPanel,
        toggleMetadataPanel,
        toggleMatchPanel,
        toggleCollectionPanelAction,
        toggleEvalMode,
        toggleMatchMode,
        toggleStatsPanel,
        toggleTranscriptionPanel,
        toggleTournamentPanel,
        enterEditMode,
        exitEditMode,
        enterEvalMode,
        exitEvalMode,
        enterTranscribeMode,
        exitTranscribeMode,
        updateEPC,
        handleOpenCollection,
        addSearchToFilterLibrary
    } from './services/positionService.js';
    import { importDatabase, importPosition, handleFileDrop, importIdentifier } from './services/importService.js';
    import { exportDatabase } from './services/exportService.js';
    import { saveSessionState } from './services/sessionService.js';
    import { handleKeyDown, toggleHelpModal, focusSearchTab } from './services/keyboardService.js';
    import { applyTabPanels } from './services/tabHandler.js';
    import { resizable } from './utils/resizeHandle.js';
    import { fileDrop } from './utils/fileDrop.js';
    import { loadWorstBlunders } from './services/positionLoader.js';

    import Toolbar from './components/Toolbar.svelte';
    import CommandPalette from './components/CommandPalette.svelte';
    import Board from './components/Board.svelte';
    import DirectionView from './components/direction/DirectionView.svelte';
    import { directionPageShownStore } from './stores/directionStore';
    import MatchInfoBar from './components/MatchInfoBar.svelte';
    import ViewTabs from './components/ViewTabs.svelte';
    import TabbedPanel from './components/TabbedPanel.svelte';
    import StatusBar from './components/StatusBar.svelte';
    import ModalHost from './components/ModalHost.svelte';
    import { initCommandProcessor, processCommand } from './commandProcessor.js';
    import { searchStructureModeStore } from './stores/searchExcludePositionStore.js';
    import { maybeRunFirstRunTour } from './services/tourService.js';
    import StudyQueueBar from './components/StudyQueueBar.svelte';
    import { startTrainingSession } from './services/trainingTabService.js';
    import { TRAINING_EXERCISES, exerciseForCommand } from './services/trainingTab.js';
    import { showTab, showTrainingPanel } from './services/tabToggles.js';
    import HomeScreen from './components/HomeScreen.svelte';
    import { initFolderWatch } from './services/watchService.js';
    import { initTheme } from './stores/themeStore.js';

    let mainArea;
    let panelHeight = $state(DEFAULT_PANEL_HEIGHT);
    // Hauteur plancher de l'onglet Transcription (ADR-0048 décision 5), mesurée : palette 236 px,
    // barre du brouillon 34, padding 16, barre d'onglets 30. Appliquée sans toucher la valeur
    // stockée, pour qu'un autre onglet retrouve la hauteur choisie.
    const TRANSCRIPTION_MIN_HEIGHT = 320;
    let appliedPanelHeight = $derived($activeTabStore === 'transcription' ? Math.max(panelHeight, TRANSCRIPTION_MIN_HEIGHT) : panelHeight);
    let panelWidth = $state(DEFAULT_PANEL_WIDTH);
    let isSidePanel = $derived($effectivePositionStore === PANEL_SIDE);
    let showDropOverlay = $state(false);
    // Écarté pour la session : le panneau Eval fonctionne sans base.
    let homeDismissed = $state(false);
    let positionCount = 0;
    let saveSessionTimeout = null;
    let tabInitialized = false;
    let previousTab = '';

    // ── Reactive effects (Svelte 5) ────────────────────────────────

    // Eval sync (the EPC measure): re-runs when position OR mode changes (both are tracked deps)
    $effect(() => {
        if ($statusBarModeStore === 'EVAL' && $positionStore) updateEPC($positionStore);
    });

    // Re-fit the board when the effective panel position flips. The rAF lets the flex layout
    // reflow first; a 'resize' that doesn't change the window size is a no-op for
    // windowAspectStore (safe_not_equal), so this can't loop.
    $effect(() => {
        $effectivePositionStore; // tracked dep
        requestAnimationFrame(() => window.dispatchEvent(new Event('resize')));
    });

    // Reload positions when the trigger increments (the only tracked dep).
    $effect(() => {
        $positionReloadTriggerStore; // tracked — fires each time MatchPanel triggers a reload
        untrack(() => {
            if ($databasePathStore) loadAllPositions();
        });
    });

    // Plain .subscribe() is intentional: `positionCount` is never read in the template. App.svelte
    // lives for the app's lifetime; the unsubscribe is still released in onDestroy.
    const unsubscribePositions = positionsStore.subscribe((value) => {
        positionCount = value?.length || 0;
        if (positionCount === 0) {
            positionStore.set(emptyPosition());
            analysisStore.set(emptyAnalysis());
        }
    });

    // Navigate to the current position when the library index changes (`cancelled` guards stale
    // async callbacks). In MATCH mode the board follows the match navigation and the index is
    // stale: without the guard, selecting an analysed move would snap the board to a library
    // position. The mode is read with get() so it is not a dependency; exiting match mode sets
    // NORMAL before the index redraw, so the guard never blocks the return to the library.
    $effect(() => {
        const value = $currentPositionIndexStore;
        let cancelled = false;
        if (get(statusBarModeStore) === 'MATCH') return;
        if (positionCount > 0 && value >= 0 && value < positionCount) {
            // Loads the window around `value` on a miss; a stale callback is dropped.
            positionsStore
                .getPosition(value)
                .then((position) => {
                    if (cancelled || !position) return;
                    return showPosition(position).then(() => {
                        if (cancelled) return;
                        if (saveSessionTimeout) clearTimeout(saveSessionTimeout);
                        saveSessionTimeout = setTimeout(() => saveSessionState(), 500);
                    });
                })
                .catch((e) => logger.error('Error loading position:', e));
        }
        return () => {
            cancelled = true;
        };
    });

    // Tab handler: only $activeTabStore is tracked, the rest is read in untrack(). The first run
    // must not return early (session restore may already be on 'eval' or 'search'); `isFirstRun`
    // skips only the exit paths, which need prevTab.
    $effect(() => {
        const tab = $activeTabStore;
        const isFirstRun = !tabInitialized;
        if (!tabInitialized) {
            tabInitialized = true;
        }
        untrack(() => {
            logger.perf('App:activeTabHandler', () => {
                const prevTab = previousTab;
                previousTab = tab;
                if (!isFirstRun) {
                    if (tab === 'search' && $databasePathStore && $statusBarModeStore !== 'EDIT') enterEditMode();
                    else if (prevTab === 'search' && tab !== 'search' && $statusBarModeStore === 'EDIT') exitEditMode();
                }
                if (tab === 'eval' && $statusBarModeStore !== 'EVAL') logger.perf('App:evalSync', () => enterEvalMode());
                else if (!isFirstRun && prevTab === 'eval' && tab !== 'eval' && $statusBarModeStore === 'EVAL') exitEvalMode();
                // Transcription is a scratch mode too: the board belongs to the draft's Cursor (ADR-0045).
                if (tab === 'transcription' && $statusBarModeStore !== 'TRANSCRIBE') enterTranscribeMode();
                else if (!isFirstRun && prevTab === 'transcription' && tab !== 'transcription' && $statusBarModeStore === 'TRANSCRIBE') exitTranscribeMode();
                applyTabPanels(tab);
            });
        });
    });

    // ── UI event handlers ──────────────────────────────────────────

    // Resize-handle drag (utils/resizeHandle.js).
    function setPanelSize(size, side) {
        if (side) panelWidth = size;
        else panelHeight = size;
    }
    function savePanelSize(size, side) {
        if (side) savePanelWidth(size);
        else savePanelHeight(size);
    }

    // A trackpad fires many wheel events per gesture, each a Wails round trip; 60 ms between
    // navigations keeps one gesture to a handful of steps.
    let lastWheelNavTime = 0;
    function handleWheel(event) {
        if ($isAnyModalOpen || $statusBarModeStore === 'EDIT' || $statusBarModeStore === 'EVAL') return;
        // La page Direction remplace le plateau dans la même zone : la molette y défile.
        if (!isOnBoard(event.target)) return;
        // En TRANSCRIBE, la molette au-dessus du plateau parcourt les candidats (ADR-0048
        // décision 11) au lieu de changer de position, contre laquelle le panneau se battrait.
        if ($statusBarModeStore === 'TRANSCRIBE') {
            const delta = event.deltaY > 0 ? 1 : event.deltaY < 0 ? -1 : 0;
            if (delta === 0) return;
            event.preventDefault();
            transcriptionWheelStore.set({ delta, at: performance.now() });
            return;
        }
        if (positionCount > 0) {
            event.preventDefault();
            const now = performance.now();
            if (now - lastWheelNavTime < 60) return;
            lastWheelNavTime = now;
            if (event.deltaY < 0) previousPosition();
            else if (event.deltaY > 0) nextPosition();
        }
    }

    async function handleResize() {
        try {
            const size = await WindowGetSize();
            if (size) await SaveWindowDimensions(size.w, size.h);
        } catch (err) {
            logger.error('Error getting window dimensions:', err);
        }
    }

    // ── Lifecycle ──────────────────────────────────────────────────

    // Opt-in update notice: fired once, never awaited (a slow network must not delay startup),
    // shown only in the status bar. CheckForUpdate is a no-op on package-managed installs.
    async function maybeCheckForUpdate() {
        try {
            if (!(await GetCheckForUpdates())) return;
            const result = await CheckForUpdate();
            if (result.packageManaged || !result.latestVersion) return;
            const current = get(metaStore).applicationVersion;
            if (isNewerVersion(result.latestVersion, current)) {
                setStatusBarMessage(tMsg('status.updateAvailable', { version: result.latestVersion }));
            }
        } catch (error) {
            logger.error('Error checking for an update:', error);
        }
    }

    // `train` ouvre l'onglet Entraînement ; `train <exercice>` l'ouvre ET démarre (alias dans
    // `exerciseForCommand`). La commande lit la MÊME source mémorisée que le lanceur (ADR-0041
    // règle 2) : le manuel promet la mémoire sans réserve.
    async function startTrainingCommand(drill) {
        const wanted = String(drill || '').trim();
        if (!wanted) {
            // Ouvrir, jamais refermer (`cmd_mode.rst` dit « Ouvre »).
            showTrainingPanel();
            return;
        }
        const exercise = exerciseForCommand(wanted);
        if (!exercise) {
            setStatusBarMessage(tMsg('training.usage', { drills: TRAINING_EXERCISES.map((e) => e.id).join(', ') }));
            return;
        }
        if (!showTab('training')) return;
        let seedSource = '';
        try {
            seedSource = (await GetTrainingSeedSources())?.[exercise] || '';
        } catch (error) {
            logger.error('could not read the remembered training source:', error);
        }
        await startTrainingSession({ exercise, seedSource });
    }

    onMount(async () => {
        maybeCheckForUpdate();

        initCommandProcessor({
            onToggleHelp: toggleHelpModal,
            onNewDatabase: newDatabase,
            onOpenDatabase: openDatabase,
            onLoadDemo: loadDemoDatabase,
            onImportDatabase: importDatabase,
            onExportDatabase: exportDatabase,
            importPosition,
            onImportIdentifier: importIdentifier,
            onTraining: startTrainingCommand,
            onSavePosition: saveCurrentPosition,
            onUpdatePosition: updatePosition,
            onDeletePosition: deletePosition,
            onToggleAnalysis: toggleAnalysisPanel,
            onToggleComment: toggleCommentPanel,
            exitApp,
            onLoadPositionsByFilters: loadPositionsByFilters,
            onLoadAllPositions: reloadAllPositions,
            toggleMetadataPanel,
            focusSearchTab,
            toggleMatchPanel,
            toggleCollectionPanel: toggleCollectionPanelAction,
            toggleEvalMode,
            toggleTranscriptionPanel,
            toggleTournamentPanel,
            toggleMatchMode,
            onToggleStats: () => toggleStatsPanel(),
            onLoadBlunders: loadWorstBlunders
        });
        window.addEventListener('keydown', handleKeyDown);
        mainArea.addEventListener('wheel', handleWheel);
        window.addEventListener('resize', handleResize);

        // The persisted UI language before anything renders; English if the read fails.
        try {
            await initLanguage(await GetLanguage());
        } catch (_e) {
            await initLanguage('en');
        }

        initBoardColors();

        // Les jetons seulement : la palette de l'utilisateur vient d'être chargée et primerait.
        initTheme();

        initUIScale();

        initPanelPosition();

        // One-shot seed of the local $state: the resize-handle drag owns it afterwards.
        initPanelSize().then(() => {
            panelHeight = get(panelHeightStore);
            panelWidth = get(panelWidthStore);
        });

        // First launch only: the guided-tour catalog, also offered by the home screen.
        maybeRunFirstRunTour();

        // The watched folder, fire and forget: a vanished folder must not delay startup.
        initFolderWatch();

        // A database handed on the command line (file association, Exec=blunderDB %f) takes
        // priority over the remembered one.
        try {
            const startupPath = await StartupFilePath();
            if (startupPath && (await PathExists(startupPath))) {
                await openDatabaseByPath(startupPath);
                return;
            }
        } catch (error) {
            logger.error('Error opening the database passed on the command line:', error);
        }

        // Reopen the last database as a host capability (ADR-0004): only a definitively gone path
        // is forgotten, a temporarily unavailable one is kept. Probing first also stops SQLite
        // from silently creating an empty database at a stale path (modernc creates on first use).
        try {
            const lastDbPath = await GetLastDatabasePath();
            if (lastDbPath) {
                if (await PathExists(lastDbPath)) {
                    // Keep the path whatever happens: a transient lock/IO error must not erase it.
                    await openDatabaseByPath(lastDbPath);
                } else {
                    // Definitively gone: forget it (and never recreate an empty database there).
                    logger.log('Last database no longer exists, forgetting path:', lastDbPath);
                    await SaveLastDatabasePath('');
                }
            }
        } catch (error) {
            // Any failure, the probe included, is transient: keep the path.
            logger.error('Error auto-reopening last database (keeping remembered path):', error);
        }
    });

    onDestroy(() => {
        window.removeEventListener('keydown', handleKeyDown);
        mainArea.removeEventListener('wheel', handleWheel);
        window.removeEventListener('resize', handleResize);
        unsubscribePositions();
    });
</script>

<main class="main-container" bind:this={mainArea} use:fileDrop={{ onDrop: handleFileDrop, onOverlayChange: (visible) => (showDropOverlay = visible) }}>
    {#if showDropOverlay}
        <div class="drop-overlay" transition:fade={{ duration: 150 }}>
            <div class="drop-overlay-content">
                <svg class="drop-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="7 10 12 15 17 10" />
                    <line x1="12" y1="15" x2="12" y2="3" />
                </svg>
                <span>{$t('import.dropToImport')}</span>
                <span class="drop-hint">.db &middot; .xg &middot; .sgf &middot; .mat &middot; .bgf &middot; .txt</span>
            </div>
        </div>
    {/if}

    <!-- L'écran d'accueil. Un plateau vide n'est pas une invitation :
         il ne dit ni ce que l'outil sait faire, ni par où commencer. Il
         s'efface dès qu'une base est ouverte, et se laisse écarter pour qui
         veut se servir du panneau Eval sans base. -->
    {#if !$databasePathStore && !homeDismissed}
        <HomeScreen onDismiss={() => (homeDismissed = true)} />
    {/if}

    <Toolbar />

    <ViewTabs />

    <MatchInfoBar />

    <!-- La file d'étude post-import : une bande, pas une fenêtre. Le
         reste de l'application doit rester utilisable pendant le parcours,
         puisque c'est là qu'on commente, qu'on range et qu'on fait une carte. -->
    <StudyQueueBar />

    <div class="body" class:side={isSidePanel}>
        <div class="scrollable-content" data-tour="board" class:exclude-structure-editing={$activeTabStore === 'search' && $searchStructureModeStore === 'exclude'}>
            {#if $activeTabStore === 'search' && $searchStructureModeStore === 'exclude'}
                <div class="exclude-structure-badge">EXCLUDE</div>
            {/if}
            <!-- La seule chose qui remplace le plateau dans la zone principale (ADR-0047) :
                 l'onglet Tournoi actif ET une Direction ouverte. Tout autre onglet ramène le
                 plateau sans rien fermer — la Direction reste ouverte et continue de vivre. -->
            {#if $directionPageShownStore}
                <DirectionView />
            {:else}
                <Board />
            {/if}
        </div>

        <div class="resize-handle" class:side={isSidePanel} use:resizable={{ side: isSidePanel, size: isSidePanel ? panelWidth : panelHeight, onResize: setPanelSize, onCommit: savePanelSize }}></div>

        <div class="panel-wrapper" class:side={isSidePanel} data-tour="panels" style={isSidePanel ? `width: ${panelWidth}px;` : `height: ${appliedPanelHeight}px;`}>
            <TabbedPanel
                onLoadPositionsByFilters={loadPositionsByFilters}
                onCloseAnalysis={toggleAnalysisPanel}
                onCloseComment={toggleCommentPanel}
                onOpenCollection={handleOpenCollection}
                onAddToFilterLibrary={addSearchToFilterLibrary}
            />
        </div>
    </div>

    <ModalHost />

    <CommandPalette />

    <StatusBar onCommand={(cmd) => processCommand(cmd)} />
</main>

<style>
    .main-container {
        display: flex;
        flex-direction: column;
        /* Interface scaling: `zoom` enlarges the whole UI (icons, fonts, panels,
           modals and the SVG board, which stays crisp). The WebKit/WebView
           engines resolve `vw`/`vh` in the zoomed coordinate system, so plain
           100vw/100vh still fill exactly one viewport at any scale — do NOT
           divide them by the scale (that under-sizes the box and leaves gaps
           on the right and below the status bar). */
        zoom: var(--ui-scale, 1);
        height: 100vh;
        width: 100vw;
        padding: 0;
        box-sizing: border-box;
        position: relative;
        overflow: hidden;
    }

    /* Wraps the board, the resize handle and the panel. Column (panel at the
       bottom) by default; row (panel as a vertical column on the right) in
       side mode. flex:1 makes it consume the height between ViewTabs and the
       StatusBar. */
    .body {
        flex: 1;
        min-height: 0;
        display: flex;
        flex-direction: column;
    }
    .body.side {
        flex-direction: row;
    }

    .scrollable-content {
        flex: 1;
        min-height: 0;
        min-width: 0;
        overflow: hidden;
        padding: 0;
        box-sizing: border-box;
        display: flex;
        justify-content: center;
        align-items: center;
        position: relative;
    }

    /* Visual cue while editing the "Except" (exclusion) checker structure. */
    .scrollable-content.exclude-structure-editing {
        outline: 3px solid #c0392b;
        outline-offset: -3px;
    }
    .exclude-structure-badge {
        position: absolute;
        top: 6px;
        right: 10px;
        z-index: 5;
        background: #c0392b;
        color: #fff;
        font-size: var(--font-size-small);
        font-weight: 700;
        letter-spacing: 0.06em;
        padding: 2px 8px;
        border-radius: 3px;
        pointer-events: none;
    }

    .resize-handle {
        flex-shrink: 0;
        height: 2px;
        background: #e0e0e0;
        cursor: ns-resize;
        position: relative;
        z-index: 10;
        transition: background 0.15s;
    }
    .resize-handle.side {
        height: auto;
        width: 2px;
        cursor: ew-resize;
    }

    .resize-handle:hover,
    .resize-handle:active {
        background: #aaa;
    }

    .panel-wrapper {
        flex-shrink: 0;
        overflow: hidden;
        display: flex;
        flex-direction: column;
    }
    /* In side mode the height style is dropped; width is set inline instead. */
    .panel-wrapper.side {
        height: auto;
    }

    .drop-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        z-index: 10000;
        background: rgba(30, 60, 114, 0.85);
        display: flex;
        align-items: center;
        justify-content: center;
        pointer-events: none;
    }

    .drop-overlay-content {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 12px;
        color: #ffffff;
        /* Reuses the dialog-title token rather than a bespoke overlay size — see
           docs/adr/0008. */
        font-size: var(--font-size-dialog-title);
        font-weight: 600;
        text-shadow: 0 1px 4px rgba(0, 0, 0, 0.3);
        border: 3px dashed rgba(255, 255, 255, 0.6);
        border-radius: 16px;
        padding: 40px 60px;
    }

    .drop-icon {
        width: 48px;
        height: 48px;
        color: #ffffff;
        opacity: 0.9;
    }

    .drop-hint {
        font-size: var(--font-size-base);
        font-weight: 400;
        opacity: 0.7;
    }
</style>
