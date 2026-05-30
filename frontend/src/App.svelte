<script>
    import { logger } from './utils/logger.js';
    import { onMount, onDestroy, untrack } from 'svelte';
    import { fade } from 'svelte/transition';

    // Wails runtime
    import { WindowGetSize, OnFileDrop, OnFileDropOff } from '../wailsjs/runtime/runtime.js';
    import { SaveWindowDimensions, GetLastDatabasePath, SaveLastDatabasePath } from '../wailsjs/go/main/Config.js';

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
    import { newDatabase, openDatabase, openDatabaseByPath, exitApp, closeWarningModal, warningMessageStore } from './services/databaseService.js';
    import {
        showPosition,
        loadAllPositions,
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
        toggleFilterLibraryPanel,
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

    // Components
    import Toolbar from './components/Toolbar.svelte';
    import Board from './components/Board.svelte';
    import ViewTabs from './components/ViewTabs.svelte';
    import TabbedPanel from './components/TabbedPanel.svelte';
    import StatusBar from './components/StatusBar.svelte';
    import { initCommandProcessor, processCommand } from './commandProcessor.js';
    import { searchStructureModeStore } from './stores/searchExcludePositionStore.js';
    import HelpModal from './components/HelpModal.svelte';
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

    // Navigate to the current position when index changes. The cancelled flag
    // guards against stale async callbacks when the index changes rapidly.
    $effect(() => {
        const value = $currentPositionIndexStore;
        let cancelled = false;
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
        document.body.style.cursor = 'ns-resize';
        document.body.style.userSelect = 'none';
        const startY = e.clientY;
        const startHeight = panelHeight;
        function onMouseMove(e) {
            panelHeight = Math.min(Math.max(80, startHeight + (startY - e.clientY)), window.innerHeight - 160);
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
            onLoadAllPositions: loadAllPositions,
            toggleFilterLibraryPanel,
            toggleSearchHistoryPanel,
            toggleMatchPanel,
            toggleCollectionPanel: toggleCollectionPanelAction,
            toggleEPCMode,
            toggleMatchMode,
            onToggleStats: () => toggleStatsPanel()
        });
        window.addEventListener('keydown', handleKeyDown);
        mainArea.addEventListener('wheel', handleWheel);
        window.addEventListener('resize', handleResize);
        OnFileDrop((x, y, paths) => handleFileDrop(x, y, paths), false);
        window.addEventListener('dragover', handleDragOver);
        window.addEventListener('dragleave', handleDragLeave);
        window.addEventListener('drop', handleDragEnd);
        try {
            const lastDbPath = await GetLastDatabasePath();
            if (lastDbPath) await openDatabaseByPath(lastDbPath);
        } catch (error) {
            logger.error('Error auto-reopening last database:', error);
            try {
                await SaveLastDatabasePath('');
            } catch (_e) {
                /* ignored */
            }
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
        onLoadAllPositions={loadAllPositions}
        onToggleEPCMode={toggleEPCMode}
    />

    <ViewTabs />

    <div class="scrollable-content" class:exclude-structure-editing={$activeTabStore === 'search' && $searchStructureModeStore === 'exclude'}>
        {#if $activeTabStore === 'search' && $searchStructureModeStore === 'exclude'}
            <div class="exclude-structure-badge">EXCLUDE</div>
        {/if}
        <Board />
    </div>

    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="resize-handle" onmousedown={onResizeHandleMouseDown}></div>

    <div class="panel-wrapper" style="height: {panelHeight}px;">
        <TabbedPanel
            onLoadPositionsByFilters={loadPositionsByFilters}
            onCloseAnalysis={toggleAnalysisPanel}
            onCloseComment={toggleCommentPanel}
            onOpenCollection={handleOpenCollection}
            onAddToFilterLibrary={addSearchToFilterLibrary}
        />
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

    <StatusBar onCommand={(cmd) => processCommand(cmd)} />
</main>

<style>
    .main-container {
        display: flex;
        flex-direction: column;
        height: 100vh;
        padding: 0;
        box-sizing: border-box;
        position: relative;
        overflow: hidden;
        width: 100vw;
    }

    .scrollable-content {
        flex: 1;
        min-height: 0;
        overflow: hidden;
        padding: 0;
        width: 100%;
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
