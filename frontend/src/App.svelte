<script>
    import { logger } from './utils/logger.js';
    import { onMount, onDestroy, untrack } from 'svelte';
    import { get } from 'svelte/store';
    import { fade } from 'svelte/transition';

    // Wails runtime
    import { WindowGetSize, OnFileDrop, OnFileDropOff } from '../wailsjs/runtime/runtime.js';
    import { SaveWindowDimensions, GetLastDatabasePath, SaveLastDatabasePath, GetLanguage } from '../wailsjs/go/main/Config.js';
    import { PathExists } from '../wailsjs/go/gui/App.js';
    import { initLanguage } from './i18n';
    import { initBoardColors } from './stores/boardColorsStore';
    import { initUIScale } from './stores/uiScaleStore';
    import { initPanelPosition, effectivePositionStore, PANEL_SIDE } from './stores/panelLayoutStore';

    // Stores
    import { databasePathStore } from './stores/databaseStore.js';
    import { positionStore, positionsStore } from './stores/positionStore.js';
    import { analysisStore } from './stores/analysisStore.js';
    import {
        currentPositionIndexStore,
        statusBarModeStore,
        showCommandInputStore,
        positionReloadTriggerStore,
        activeTabStore,
        activeModal,
        MODAL,
        closeModal,
        toggleModal,
        isAnyModalOpen
    } from './stores/uiStore.js';
    import {
        showImportProgressModalStore,
        importModalModeStore,
        importAnalysisStore,
        importResultStore,
        showFileImportModalStore,
        fileImportModeStore,
        fileImportTotalFilesStore,
        fileImportCurrentIndexStore,
        fileImportCurrentFileStore,
        fileImportResultsStore
    } from './stores/importModalStore.js';
    import { exportModalModeStore, exportPositionCountStore, exportMetadataStore, exportOptionsStore, exportMatchesStore } from './stores/exportModalStore.js';

    // Services
    import { newDatabase, openDatabase, openDatabaseByPath, loadDemoDatabase, exitApp, closeWarningModal, warningMessageStore } from './services/databaseService.js';
    import {
        showPosition,
        loadAllPositions,
        reloadAllPositions,
        loadPositionsByFilters,
        firstPosition,
        previousPosition,
        nextPosition,
        lastPosition,
        gotoPosition,
        saveCurrentPosition,
        updatePosition,
        deletePosition,
        toggleAnalysisPanel,
        toggleCommentPanel,
        toggleMetadataPanel,
        toggleMatchPanel,
        toggleCollectionPanelAction,
        toggleEPCMode,
        toggleMatchMode,
        toggleStatsPanel,
        enterEditMode,
        exitEditMode,
        enterEPCMode,
        exitEPCMode,
        updateEPC,
        handleOpenCollection,
        addSearchToFilterLibrary,
        togglePipcount,
        loadRandomPosition
    } from './services/positionService.js';
    import {
        importDatabase,
        importPosition,
        importFolder,
        handleImportCommit,
        handleImportCancel,
        handleImportClose,
        handleFileImportCancel,
        handleFileImportClose,
        pastePosition,
        handleFileDrop
    } from './services/importService.js';
    import { exportDatabase, handleExportCommit, handleExportCancel, handleExportClose } from './services/exportService.js';
    import { copyPosition, copyBoardImage } from './services/clipboardService.js';
    import { saveSessionState } from './services/sessionService.js';
    import { handleKeyDown, toggleHelpModal, toggleSearchHistoryPanel } from './services/keyboardService.js';
    import { applyTabPanels } from './services/tabHandler.js';
    import { loadWorstBlunders } from './services/positionLoader.js';

    // Components
    import Toolbar from './components/Toolbar.svelte';
    import Board from './components/Board.svelte';
    import MatchInfoBar from './components/MatchInfoBar.svelte';
    import ViewTabs from './components/ViewTabs.svelte';
    import TabbedPanel from './components/TabbedPanel.svelte';
    import StatusBar from './components/StatusBar.svelte';
    import { initCommandProcessor, processCommand } from './commandProcessor.js';
    import { searchStructureModeStore } from './stores/searchExcludePositionStore.js';
    import HelpModal from './components/HelpModal.svelte';
    import ConfigModal from './components/ConfigModal.svelte';
    import TourCatalogModal from './components/TourCatalogModal.svelte';
    import { maybeRunFirstRunTour } from './services/tourService.js';
    import GoToPositionModal from './components/GoToPositionModal.svelte';
    import MetModal from './components/MetModal.svelte';
    import DataTableModal from './components/DataTableModal.svelte';
    import { takePoint2LastTable } from './stores/takePoint2LastTable';
    import { takePoint2LiveTable } from './stores/takePoint2LiveTable';
    import { takePoint4LastTable } from './stores/takePoint4LastTable';
    import { takePoint4LiveTable } from './stores/takePoint4LiveTable';
    import { gammonValue1Table } from './stores/gammonValue1Table';
    import { gammonValue2Table } from './stores/gammonValue2Table';
    import { gammonValue4Table } from './stores/gammonValue4Table';
    import WarningModal from './components/WarningModal.svelte';
    import ImportProgressModal from './components/ImportProgressModal.svelte';
    import FileImportProgressModal from './components/FileImportProgressModal.svelte';
    import ExportDatabaseModal from './components/ExportDatabaseModal.svelte';

    // Component state
    let mainArea;
    let panelHeight = $state(250);
    let panelWidth = $state(420);
    let isSidePanel = $derived($effectivePositionStore === PANEL_SIDE);
    let _isResizing = false;
    let showDropOverlay = $state(false);
    let dragCounter = 0;
    let positions = [];
    let saveSessionTimeout = null;
    let tabInitialized = false;
    let previousTab = '';

    // ── Reactive effects (Svelte 5) ────────────────────────────────

    // EPC sync: re-runs when position OR mode changes (both are tracked deps)
    $effect(() => {
        if ($statusBarModeStore === 'EPC' && $positionStore) updateEPC($positionStore);
    });

    // Re-fit the board whenever the effective panel position flips (manual mode
    // change, or an auto-mode threshold crossing). The rAF defers the synthetic
    // resize until after the flex layout has reflowed, so two.js measures the
    // new container box. Dispatching 'resize' that doesn't change the window
    // size is a no-op for windowAspectStore (safe_not_equal), so this can't loop.
    $effect(() => {
        $effectivePositionStore; // tracked dep
        requestAnimationFrame(() => window.dispatchEvent(new Event('resize')));
    });

    // Reload positions when trigger increments (positionReloadTriggerStore is
    // the only tracked dep; $databasePathStore is read at call time via $)
    $effect(() => {
        $positionReloadTriggerStore; // tracked — fires each time MatchPanel triggers a reload
        untrack(() => {
            if ($databasePathStore) loadAllPositions();
        });
    });

    // Keep `positions` array in sync with positionsStore.
    // Plain .subscribe() is intentional: `positions` is never read in the
    // template directly, so $state reactivity is not needed here.
    positionsStore.subscribe((value) => {
        positions = Array.isArray(value) ? value : [];
        if (positions.length === 0) {
            positionStore.set({
                id: 0,
                board: { points: Array(26).fill({ checkers: 0, color: -1 }), bearoff: [15, 15] },
                cube: { owner: -1, value: 0 },
                dice: [3, 1],
                score: [-1, -1],
                player_on_roll: 0,
                decision_type: 0,
                has_jacoby: 0,
                has_beaver: 0
            });
            analysisStore.set({
                positionId: null,
                xgid: '',
                player1: '',
                player2: '',
                analysisType: '',
                analysisEngineVersion: '',
                checkerAnalysis: { moves: [] },
                doublingCubeAnalysis: {
                    analysisDepth: '',
                    playerWinChances: 0,
                    playerGammonChances: 0,
                    playerBackgammonChances: 0,
                    opponentWinChances: 0,
                    opponentGammonChances: 0,
                    opponentBackgammonChances: 0,
                    cubelessNoDoubleEquity: 0,
                    cubelessDoubleEquity: 0,
                    cubefulNoDoubleEquity: 0,
                    cubefulNoDoubleError: 0,
                    cubefulDoubleTakeEquity: 0,
                    cubefulDoubleTakeError: 0,
                    cubefulDoublePassEquity: 0,
                    cubefulDoublePassError: 0,
                    bestCubeAction: '',
                    wrongPassPercentage: 0,
                    wrongTakePercentage: 0
                },
                allCubeAnalyses: [],
                playedMove: '',
                playedCubeAction: '',
                playedMoves: [],
                playedCubeActions: [],
                creationDate: '',
                lastModifiedDate: ''
            });
        }
    });

    // Navigate to the current position when the library index changes. The
    // cancelled flag guards against stale async callbacks when the index
    // changes rapidly.
    //
    // Bug 1: in MATCH mode the board is driven directly by the match navigation
    // (matchContextStore + showPosition in positionService), and
    // currentPositionIndexStore is a STALE library index. Selecting an analysed
    // move re-runs this effect via a downstream index notification; without this
    // guard it would call showPosition(positions[value]) and snap the board to
    // the last library position instead of staying on the studied match move.
    // Read the mode with get() (non-reactive) so mode / match-context changes do
    // not add themselves as dependencies of this effect. Exiting match mode sets
    // the mode back to NORMAL *before* the index redraw (see loadAllPositions /
    // exitEditMode), so this guard never blocks the return-to-library redraw.
    $effect(() => {
        const value = $currentPositionIndexStore;
        let cancelled = false;
        if (get(statusBarModeStore) === 'MATCH') return;
        if (positions.length > 0 && value >= 0 && value < positions.length) {
            showPosition(positions[value]).then(() => {
                if (cancelled) return;
                if (saveSessionTimeout) clearTimeout(saveSessionTimeout);
                saveSessionTimeout = setTimeout(() => saveSessionState(), 500);
            });
        }
        return () => {
            cancelled = true;
        };
    });

    // Tab handler: open/close panels and manage mode transitions.
    // Only $activeTabStore is a tracked dependency; everything else is read
    // inside untrack() so that mode/db-path changes alone don't re-run this.
    //
    // On the first run we must NOT return early: session restore may have
    // already set activeTabStore to 'epc' or 'search', and we need to call
    // enterEPCMode / enterEditMode immediately. We use `isFirstRun` to skip
    // the exit paths (which depend on prevTab, unknown on the first call).
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
                if (tab === 'epc' && $statusBarModeStore !== 'EPC') logger.perf('App:epcSync', () => enterEPCMode());
                else if (!isFirstRun && prevTab === 'epc' && tab !== 'epc' && $statusBarModeStore === 'EPC') exitEPCMode();
                applyTabPanels(tab);
            });
        });
    });

    // ── UI event handlers ──────────────────────────────────────────

    function onResizeHandleMouseDown(e) {
        e.preventDefault();
        _isResizing = true;
        // Side panel: drag horizontally to resize its width (it sits to the
        // right of the board, so dragging left grows it). Bottom panel: drag
        // vertically to resize its height (dragging up grows it).
        const side = isSidePanel;
        document.body.style.cursor = side ? 'ew-resize' : 'ns-resize';
        document.body.style.userSelect = 'none';
        const start = side ? e.clientX : e.clientY;
        const startSize = side ? panelWidth : panelHeight;
        function onMouseMove(e) {
            if (side) {
                panelWidth = Math.min(Math.max(150, startSize + (start - e.clientX)), window.innerWidth - 200);
            } else {
                panelHeight = Math.min(Math.max(80, startSize + (start - e.clientY)), window.innerHeight - 160);
            }
            window.dispatchEvent(new Event('resize'));
        }
        function onMouseUp() {
            _isResizing = false;
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
            window.removeEventListener('mousemove', onMouseMove);
            window.removeEventListener('mouseup', onMouseUp);
        }
        window.addEventListener('mousemove', onMouseMove);
        window.addEventListener('mouseup', onMouseUp);
    }

    function handleWheel(event) {
        if ($isAnyModalOpen || $statusBarModeStore === 'EDIT' || $statusBarModeStore === 'EPC') return;
        const boardArea = mainArea?.querySelector('.scrollable-content');
        if (!boardArea || !boardArea.contains(event.target)) return;
        if (positions && positions.length > 0) {
            event.preventDefault();
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

    function handleDragOver(e) {
        e.preventDefault();
        if (!showDropOverlay) {
            dragCounter++;
            showDropOverlay = true;
        }
    }
    function handleDragLeave(_e) {
        dragCounter--;
        if (dragCounter <= 0) {
            dragCounter = 0;
            showDropOverlay = false;
        }
    }
    function handleDragEnd(_e) {
        dragCounter = 0;
        showDropOverlay = false;
    }

    // ── Lifecycle ──────────────────────────────────────────────────

    onMount(async () => {
        initCommandProcessor({
            onToggleHelp: toggleHelpModal,
            onNewDatabase: newDatabase,
            onOpenDatabase: openDatabase,
            onLoadDemo: loadDemoDatabase,
            onImportDatabase: importDatabase,
            onExportDatabase: exportDatabase,
            importPosition,
            onSavePosition: saveCurrentPosition,
            onUpdatePosition: updatePosition,
            onDeletePosition: deletePosition,
            onToggleAnalysis: toggleAnalysisPanel,
            onToggleComment: toggleCommentPanel,
            exitApp,
            onLoadPositionsByFilters: loadPositionsByFilters,
            onLoadAllPositions: reloadAllPositions,
            toggleMetadataPanel,
            toggleSearchHistoryPanel,
            toggleMatchPanel,
            toggleCollectionPanel: toggleCollectionPanelAction,
            toggleEPCMode,
            toggleMatchMode,
            onToggleStats: () => toggleStatsPanel(),
            onLoadBlunders: loadWorstBlunders
        });
        window.addEventListener('keydown', handleKeyDown);
        mainArea.addEventListener('wheel', handleWheel);
        window.addEventListener('resize', handleResize);
        OnFileDrop((x, y, paths) => handleFileDrop(x, y, paths), false);
        window.addEventListener('dragover', handleDragOver);
        window.addEventListener('dragleave', handleDragLeave);
        window.addEventListener('drop', handleDragEnd);

        // Apply the persisted UI language before anything renders; fall back to
        // English if the config read fails.
        try {
            initLanguage(await GetLanguage());
        } catch (_e) {
            initLanguage('en');
        }

        // Load the persisted board palette (falls back to defaults internally).
        initBoardColors();

        // Apply the persisted interface scale (falls back to 100% internally).
        initUIScale();

        // Apply the persisted panel position (falls back to bottom internally).
        initPanelPosition();

        // On first launch only, show the guided-tour catalog once.
        maybeRunFirstRunTour();

        // Reopen the last database, but treat the remembered path as a *host
        // capability* (the filesystem it lives on may be unmounted, the file may
        // be locked by another instance, etc). Only a *definitively* gone path is
        // forgotten; a path that is merely temporarily unavailable is kept so a
        // later launch can reopen it once the condition clears. Probing existence
        // first also stops SQLite from silently recreating an empty database at a
        // stale path (sql.Open is lazy and modernc creates the file on first use).
        try {
            const lastDbPath = await GetLastDatabasePath();
            if (lastDbPath) {
                if (await PathExists(lastDbPath)) {
                    // Present: attempt to reopen. openDatabaseByPath handles and
                    // surfaces its own open errors; we deliberately keep the path
                    // whatever happens, so a transient lock/IO error never erases it.
                    await openDatabaseByPath(lastDbPath);
                } else {
                    // Definitively gone: forget it so we don't keep trying (and
                    // don't recreate an empty database at the old location).
                    logger.log('Last database no longer exists, forgetting path:', lastDbPath);
                    await SaveLastDatabasePath('');
                }
            }
        } catch (error) {
            // Any failure here (including the existence probe itself) is treated as
            // transient: keep the remembered path untouched.
            logger.error('Error auto-reopening last database (keeping remembered path):', error);
        }
    });

    onDestroy(() => {
        window.removeEventListener('keydown', handleKeyDown);
        mainArea.removeEventListener('wheel', handleWheel);
        window.removeEventListener('resize', handleResize);
        window.removeEventListener('dragover', handleDragOver);
        window.removeEventListener('dragleave', handleDragLeave);
        window.removeEventListener('drop', handleDragEnd);
        OnFileDropOff();
    });
</script>

<main class="main-container" bind:this={mainArea}>
    {#if showDropOverlay}
        <div class="drop-overlay" transition:fade={{ duration: 150 }}>
            <div class="drop-overlay-content">
                <svg class="drop-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="7 10 12 15 17 10" />
                    <line x1="12" y1="15" x2="12" y2="3" />
                </svg>
                <span>Drop files to import</span>
                <span class="drop-hint">.db &middot; .xg &middot; .sgf &middot; .mat &middot; .bgf &middot; .txt</span>
            </div>
        </div>
    {/if}

    <Toolbar
        onclick={() => {}}
        onNewDatabase={newDatabase}
        onOpenDatabase={openDatabase}
        onImportDatabase={importDatabase}
        onExportDatabase={exportDatabase}
        onExit={exitApp}
        onImportPosition={importPosition}
        onImportFolder={importFolder}
        onCopyPosition={copyPosition}
        onPastePosition={pastePosition}
        onSavePosition={saveCurrentPosition}
        onUpdatePosition={updatePosition}
        onDeletePosition={deletePosition}
        onFirstPosition={firstPosition}
        onPreviousPosition={previousPosition}
        onNextPosition={nextPosition}
        onLastPosition={lastPosition}
        onGoToPosition={gotoPosition}
        onTogglePipcount={togglePipcount}
        onRandomPosition={loadRandomPosition}
        onCopyBoardImage={copyBoardImage}
        onToggleCommandMode={() => showCommandInputStore.set(true)}
        onToggleHelp={toggleHelpModal}
        onToggleConfig={() => toggleModal(MODAL.CONFIG)}
        onToggleTour={() => toggleModal(MODAL.TOUR)}
        onLoadAllPositions={reloadAllPositions}
        onToggleEPCMode={toggleEPCMode}
    />

    <ViewTabs />

    <MatchInfoBar />

    <div class="body" class:side={isSidePanel}>
        <div class="scrollable-content" data-tour="board" class:exclude-structure-editing={$activeTabStore === 'search' && $searchStructureModeStore === 'exclude'}>
            {#if $activeTabStore === 'search' && $searchStructureModeStore === 'exclude'}
                <div class="exclude-structure-badge">EXCLUDE</div>
            {/if}
            <Board />
        </div>

        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="resize-handle" class:side={isSidePanel} onmousedown={onResizeHandleMouseDown}></div>

        <div class="panel-wrapper" class:side={isSidePanel} data-tour="panels" style={isSidePanel ? `width: ${panelWidth}px;` : `height: ${panelHeight}px;`}>
            <TabbedPanel
                onLoadPositionsByFilters={loadPositionsByFilters}
                onCloseAnalysis={toggleAnalysisPanel}
                onCloseComment={toggleCommentPanel}
                onOpenCollection={handleOpenCollection}
                onAddToFilterLibrary={addSearchToFilterLibrary}
            />
        </div>
    </div>

    <GoToPositionModal visible={$activeModal === MODAL.GO_TO_POSITION} onClose={() => closeModal()} />

    <MetModal visible={$activeModal === MODAL.MET} onClose={() => closeModal()} />
    <DataTableModal visible={$activeModal === MODAL.TAKE_POINT_2_LAST} onClose={() => closeModal()} tables={[{ data: takePoint2LastTable, precision: 1, colCount: 8, colOffset: 2, rowOffset: 2 }]} />
    <DataTableModal visible={$activeModal === MODAL.TAKE_POINT_2_LIVE} onClose={() => closeModal()} tables={[{ data: takePoint2LiveTable, precision: 1, colCount: 8, colOffset: 2, rowOffset: 2 }]} />
    <DataTableModal visible={$activeModal === MODAL.TAKE_POINT_4_LAST} onClose={() => closeModal()} tables={[{ data: takePoint4LastTable, precision: 0, colCount: 7, colOffset: 3, rowOffset: 3 }]} />
    <DataTableModal visible={$activeModal === MODAL.TAKE_POINT_4_LIVE} onClose={() => closeModal()} tables={[{ data: takePoint4LiveTable, precision: 0, colCount: 7, colOffset: 3, rowOffset: 3 }]} />
    <DataTableModal visible={$activeModal === MODAL.GAMMON_VALUE_1} onClose={() => closeModal()} tables={[{ data: gammonValue1Table, precision: 2, colCount: 8, colOffset: 2, rowOffset: 2 }]} />
    <DataTableModal visible={$activeModal === MODAL.GAMMON_VALUE_2} onClose={() => closeModal()} tables={[{ data: gammonValue2Table, precision: 2, colCount: 8, colOffset: 2, rowOffset: 3 }]} />
    <DataTableModal visible={$activeModal === MODAL.GAMMON_VALUE_4} onClose={() => closeModal()} tables={[{ data: gammonValue4Table, precision: 2, colCount: 8, colOffset: 2, rowOffset: 5 }]} />

    <WarningModal message={$warningMessageStore} visible={$activeModal === MODAL.WARNING} onClose={closeWarningModal} />

    <DataTableModal
        visible={$activeModal === MODAL.TAKE_POINT_2}
        onClose={() => closeModal()}
        tables={[
            { title: 'Long Races', data: takePoint2LiveTable, precision: 1, colCount: 8, colOffset: 2, rowOffset: 2 },
            { title: 'Last Roll', data: takePoint2LastTable, precision: 1, colCount: 8, colOffset: 2, rowOffset: 2 }
        ]}
    />
    <DataTableModal
        visible={$activeModal === MODAL.TAKE_POINT_4}
        onClose={() => closeModal()}
        tables={[
            { title: 'Long Races', data: takePoint4LiveTable, precision: 0, colCount: 7, colOffset: 3, rowOffset: 3 },
            { title: 'Last Roll', data: takePoint4LastTable, precision: 0, colCount: 7, colOffset: 3, rowOffset: 3 }
        ]}
    />

    <ImportProgressModal
        visible={$showImportProgressModalStore}
        mode={$importModalModeStore}
        analysis={$importAnalysisStore}
        result={$importResultStore}
        onCancel={handleImportCancel}
        onCommit={handleImportCommit}
        onClose={handleImportClose}
    />

    <FileImportProgressModal
        visible={$showFileImportModalStore}
        mode={$fileImportModeStore}
        totalFiles={$fileImportTotalFilesStore}
        currentIndex={$fileImportCurrentIndexStore}
        currentFile={$fileImportCurrentFileStore}
        results={$fileImportResultsStore}
        onCancel={handleFileImportCancel}
        onClose={handleFileImportClose}
    />

    <ExportDatabaseModal
        visible={$activeModal === MODAL.EXPORT_DATABASE}
        mode={$exportModalModeStore}
        positionCount={$exportPositionCountStore}
        matches={$exportMatchesStore}
        bind:metadata={$exportMetadataStore}
        bind:exportOptions={$exportOptionsStore}
        onCancel={handleExportCancel}
        onExport={handleExportCommit}
        onClose={handleExportClose}
    />

    <HelpModal visible={$activeModal === MODAL.HELP} onClose={toggleHelpModal} handleGlobalKeydown={handleKeyDown} />
    <ConfigModal visible={$activeModal === MODAL.CONFIG} onClose={() => closeModal()} />

    <TourCatalogModal visible={$activeModal === MODAL.TOUR} onClose={() => closeModal()} />

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
        font-size: 11px;
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
        font-size: 1.3rem;
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
        font-size: 0.8rem;
        font-weight: 400;
        opacity: 0.7;
    }
</style>
