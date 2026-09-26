<script>
    import { onMount, onDestroy } from 'svelte';
    import { logger } from '../utils/logger.js';
    import { focusPanelUnlessTyping } from '../utils/panelFocus.js';
    import { nextSort } from '../utils/tableSort.js';
    import { isLetter, isBareLetter } from '../utils/keys.js';
    import { analysisStore, selectedMoveStore } from '../stores/analysisStore'; // Import analysisStore and selectedMoveStore
    import { positionStore, matchContextStore } from '../stores/positionStore'; // Import positionStore and matchContextStore
    import { playedMovePredicate, playedCubeActionPredicate } from '../utils/playedMarks.js';
    import { t } from '../i18n';
    import { cubeTurnability, isMoneyPosition } from '../utils/cubeDecision.js';
    import { trainingAnalysisHiddenStore } from '../stores/trainingTabStore.js';
    import ExplanationLine from './ExplanationLine.svelte';
    import { canLeaveSubSearchResults } from '../services/positionService.js';

    // Le même remplaçant que le panneau Eval en mode Défi (ADR-0018 règle 6).
    const HIDDEN = '···';
    import AnalysisView from './AnalysisView.svelte';
    import EngineComparison from './EngineComparison.svelte';
    let { onClose } = $props();

    // Read-only mirrors of stores
    let analysisData = $derived($analysisStore);
    let cubeValue = $derived($positionStore.cube.value);
    let onRoll = $derived($positionStore.player_on_roll ?? 0);
    // Cube availability is read off the board, imported records included: an
    // unavailable cube carries no error (ADR-0020 rule 5).
    let turnability = $derived(cubeTurnability($positionStore));
    // Referential and money rule flags (ADR-0016 point 6), read off the position
    // the cube tab switches to.
    let isMoney = $derived(isMoneyPosition($positionStore));
    let jacoby = $derived(isMoney && $positionStore?.has_jacoby === 1);
    let beaver = $derived(isMoney && $positionStore?.has_beaver === 1);
    // The cube ceiling applies at any score: read off the position.
    let maxCube = $derived($positionStore?.max_cube ?? 0);
    let matchCtx = $derived($matchContextStore);

    let activeTab = $state('checker'); // 'checker' or 'cube'

    // Sorting state for checker analysis table
    let sortColumn = $state('equity'); // default sort by equity
    let sortDirection = $state('desc'); // default highest to lowest

    // Auto-switch tab based on current move type in match mode
    $effect(() => {
        const ctx = $matchContextStore;
        if (ctx.isMatchMode && ctx.movePositions.length > 0 && !$selectedMoveStore) {
            const currentMovePos = ctx.movePositions[ctx.currentIndex];
            if (currentMovePos) {
                if (currentMovePos.move_number === 0 || currentMovePos.move_number === 1) {
                    activeTab = 'checker';
                    analysisStore.update((current) => ({
                        ...current,
                        doublingCubeAnalysis: null,
                        playedCubeAction: ''
                    }));
                } else if (currentMovePos.move_type) {
                    activeTab = currentMovePos.move_type;
                }
            }
        }
    });

    // TabbedPanel mounts/destroys this per tab switch: onMount/onDestroy are open/close.
    onMount(() => {
        const ctx = matchCtx;
        if (ctx.isMatchMode) {
            const currentMovePos = ctx.movePositions[ctx.currentIndex];
            if (currentMovePos && currentMovePos.move_type === 'checker') {
                if (currentMovePos.move_number === 0 || currentMovePos.move_number === 1) {
                    analysisStore.update((current) => ({
                        ...current,
                        doublingCubeAnalysis: null,
                        playedCubeAction: ''
                    }));
                } else {
                    loadCubeAnalysisForCurrentPosition().then((hasCubeInGame) => {
                        if (!hasCubeInGame) {
                            analysisStore.update((current) => ({
                                ...current,
                                doublingCubeAnalysis: null,
                                playedCubeAction: ''
                            }));
                        }
                    });
                }
            }
        }
        // Deferred, so the user may be typing by then: never take their field (utils/panelFocus.js).
        setTimeout(() => focusPanelUnlessTyping(document.getElementById('analysisPanel')), 0);
    });

    // Clear the selection: while selectedMoveStore is set, keyboardService
    // withholds position browsing app-wide.
    onDestroy(() => {
        selectedMoveStore.set(null);
    });

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            // Clear selection first if a move is selected. What the panel closes
            // itself, it claims (preventDefault), so the global dispatcher leaves it be.
            if ($selectedMoveStore) {
                event.preventDefault();
                selectedMoveStore.set(null);
            } else if (canLeaveSubSearchResults()) {
                // Nothing to close and `ss` results on screen: let Escape return to the list.
            } else {
                event.preventDefault();
                onClose();
            }
            return;
        }

        // Handle tab switching with 'd' (doubling/cube) key to toggle
        // Only allow if showTabs is true (not first position of game)
        if (showTabs && isLetter(event, 'd')) {
            event.preventDefault();
            const newTab = activeTab === 'checker' ? 'cube' : 'checker';
            handleTabSwitch(newTab);
            return;
        }

        // Handle j/k and arrow keys for move navigation when a move is selected
        // This should work regardless of which tab is active
        if ($selectedMoveStore) {
            if (!sortedMoves || sortedMoves.length === 0) {
                return; // No moves to navigate
            }

            const currentIndex = sortedMoves.findIndex((m) => m.move === $selectedMoveStore);

            if (isBareLetter(event, 'j') || event.key === 'ArrowDown') {
                event.preventDefault();
                if (currentIndex >= 0 && currentIndex < sortedMoves.length - 1) {
                    selectedMoveStore.set(sortedMoves[currentIndex + 1].move);
                }
                return;
            } else if (isBareLetter(event, 'k') || event.key === 'ArrowUp') {
                event.preventDefault();
                if (currentIndex > 0) {
                    selectedMoveStore.set(sortedMoves[currentIndex - 1].move);
                }
                return;
            }
        }
    }

    // Column definitions for sorting
    const sortableColumns = {
        move: { key: 'move', type: 'string' },
        equity: { key: 'equity', type: 'number' },
        error: { key: 'equityError', type: 'number' },
        pw: { key: 'playerWinChance', type: 'number' },
        pg: { key: 'playerGammonChance', type: 'number' },
        pb: { key: 'playerBackgammonChance', type: 'number' },
        ow: { key: 'opponentWinChance', type: 'number' },
        og: { key: 'opponentGammonChance', type: 'number' },
        ob: { key: 'opponentBackgammonChance', type: 'number' },
        depth: { key: 'analysisDepth', type: 'number' },
        engine: { key: 'analysisEngine', type: 'string' }
    };

    function handleSort(column) {
        const next = nextSort(sortColumn, sortDirection, column, {
            // Default direction: desc for numeric, asc for string
            defaultDir: sortableColumns[column].type === 'string' ? 'asc' : 'desc'
        });
        sortColumn = next.column;
        sortDirection = next.direction;
    }

    // Reactive sorted moves array
    let sortedMoves = $derived.by(() => {
        if (!analysisData?.checkerAnalysis?.moves) return [];
        const moves = [...analysisData.checkerAnalysis.moves];
        const col = sortableColumns[sortColumn];
        if (!col) return moves;
        return moves.sort((a, b) => {
            let va = a[col.key];
            let vb = b[col.key];
            if (col.type === 'number') {
                va = va || 0;
                vb = vb || 0;
                return sortDirection === 'asc' ? va - vb : vb - va;
            } else {
                va = va || '';
                vb = vb || '';
                const cmp = va.localeCompare(vb);
                return sortDirection === 'asc' ? cmp : -cmp;
            }
        });
    });

    function handleMoveRowClick(move) {
        // Toggle selection: if clicking the same move, deselect it
        if ($selectedMoveStore === move.move) {
            selectedMoveStore.set(null);
        } else {
            selectedMoveStore.set(move.move);
        }
    }

    // Shared with the Anki review (utils/playedMarks.js, ADR-0025 rule 6).
    let isPlayedMove = $derived(playedMovePredicate(analysisData, { matchMode: matchCtx.isMatchMode }));
    let isPlayedCubeAction = $derived(playedCubeActionPredicate(analysisData, { matchMode: matchCtx.isMatchMode }));

    async function switchTab(tab) {
        activeTab = tab;

        // When switching to cube tab in match mode, load the cube analysis and update position
        if (tab === 'cube' && matchCtx.isMatchMode) {
            await loadCubeAnalysisForCurrentPosition(true); // true = update position to show cube position
        }
    }

    // Load cube analysis for current checker position (find previous cube position)
    // Returns true if cube analysis was found in the same game, false otherwise
    async function loadCubeAnalysisForCurrentPosition(updatePosition = false) {
        if (!matchCtx.isMatchMode) return false;

        const currentIndex = matchCtx.currentIndex;
        const movePositions = matchCtx.movePositions;
        const currentMovePos = movePositions[currentIndex];

        // If we're on the first position of a game (move_number 0 or 1), no cube decision is possible
        if (currentMovePos && (currentMovePos.move_number === 0 || currentMovePos.move_number === 1)) {
            return false;
        }

        const currentGameNumber = movePositions[currentIndex].game_number;

        // Find the most recent cube decision before current position IN THE SAME GAME
        for (let i = currentIndex - 1; i >= 0; i--) {
            // Stop if we've gone to a different game
            if (movePositions[i].game_number !== currentGameNumber) {
                break;
            }

            if (movePositions[i].move_type === 'cube') {
                // Load analysis for this cube position
                try {
                    const { LoadAnalysis } = await import('../../wailsjs/go/database/Database.js');
                    const cubeAnalysis = await LoadAnalysis(movePositions[i].position.id);
                    if (cubeAnalysis && cubeAnalysis.doublingCubeAnalysis) {
                        // Update only the cube analysis part
                        analysisStore.update((current) => ({
                            ...current,
                            doublingCubeAnalysis: cubeAnalysis.doublingCubeAnalysis,
                            playedCubeAction: cubeAnalysis.playedCubeAction || ''
                        }));

                        // Only update position if explicitly requested (when clicking cube tab)
                        if (updatePosition) {
                            const cubePosition = { ...movePositions[i].position };
                            cubePosition.dice = [0, 0];
                            positionStore.set(cubePosition);
                        }
                        return true; // Found cube analysis in current game
                    }
                } catch (error) {
                    logger.error('Error loading cube analysis:', error);
                }
                return false;
            }
        }
        return false; // No cube analysis found in current game
    }

    // When switching back to checker tab, restore checker position
    async function restoreCheckerPosition() {
        if (matchCtx.isMatchMode) {
            const currentMovePos = matchCtx.movePositions[matchCtx.currentIndex];
            if (currentMovePos && currentMovePos.move_type === 'checker') {
                positionStore.set(currentMovePos.position);
            }
        }
    }

    // Enhanced switch with position restore
    async function handleTabSwitch(tab) {
        if (tab === activeTab) return;

        if (tab === 'checker') {
            await restoreCheckerPosition();
        }

        await switchTab(tab);
    }

    // Handle click in analysis content to toggle between checker and cube
    function handleContentClick(event) {
        // Only toggle if both analyses available (MATCH mode)
        if (!showTabs) return;

        // Check if clicking on table header (TH) or on a header row
        const clickedTH = event.target.closest('th');
        const clickedRow = event.target.closest('tr');
        const clickedDataRow = clickedRow && clickedRow.parentElement.tagName === 'TBODY' && !clickedTH;

        // If clicking a sortable checker table header, don't toggle tabs
        if (clickedTH && clickedTH.closest('.checker-table')) {
            return;
        }

        // Toggle if clicking on header OR anywhere outside data rows
        if (clickedTH || !clickedDataRow) {
            // Toggle between checker and cube
            const newTab = activeTab === 'checker' ? 'cube' : 'checker';
            handleTabSwitch(newTab);
        }
        // If clicking on data row (not header), don't toggle - let the row click handler do its job
    }

    // Determine if both analyses are available
    let hasCheckerAnalysis = $derived(analysisData && analysisData.checkerAnalysis && analysisData.checkerAnalysis.moves && analysisData.checkerAnalysis.moves.length > 0);
    // For cube analysis in MATCH mode, it must not be null and have actual data
    // Check both that the object exists and that it has actual analysis content
    let hasCubeAnalysis = $derived(
        analysisData &&
            analysisData.doublingCubeAnalysis !== null &&
            analysisData.doublingCubeAnalysis !== undefined &&
            typeof analysisData.doublingCubeAnalysis === 'object' &&
            (analysisData.doublingCubeAnalysis.bestCubeAction || analysisData.doublingCubeAnalysis.cubefulNoDoubleEquity !== undefined)
    );

    // Check if current position is the first position of a game (no cube decision possible)
    // First position can be move_number 0 or 1
    let isFirstPositionOfGame = $derived(
        matchCtx.isMatchMode &&
            matchCtx.movePositions.length > 0 &&
            (matchCtx.movePositions[matchCtx.currentIndex]?.move_number === 0 || matchCtx.movePositions[matchCtx.currentIndex]?.move_number === 1)
    );
    // Only show tabs in MATCH mode where checker and cube are separate positions
    // BUT not on the first position of a game (cube decision not possible)
    let showTabs = $derived(hasCheckerAnalysis && hasCubeAnalysis && matchCtx.isMatchMode && !isFirstPositionOfGame);

    // The tab decides, else the record's type; here because only the panel knows MATCH mode.
    let viewKind = $derived(showTabs ? activeTab : analysisData.analysisType === 'DoublingCube' ? 'cube' : 'checker');
</script>

<!-- Keyboard delegation on a focus container; no ARIA role fits, hence the ignore. -->
<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<section class="analysis-panel" aria-label={$t('analysis.panelLabel')} id="analysisPanel" tabindex="-1" onkeydown={handleKeyDown}>
    <div class="analysis-content" onclick={handleContentClick} onkeydown={() => {}} role="button" tabindex="-1">
        <!-- Comparaison inter-moteurs ici, pas dans Eval (ADR-0017) ; seulement s'il y en a plusieurs. -->
        {#if $trainingAnalysisHiddenStore}
            <!-- Question de Décision ouverte : la réponse est masquée (ADR-0018 règle 6). -->
            <div class="quiz-masked">{HIDDEN}</div>
        {:else}
            <EngineComparison analysis={analysisData} kind={viewKind} />
            <AnalysisView
                analysis={analysisData}
                kind={viewKind}
                {turnability}
                {cubeValue}
                {onRoll}
                moves={sortedMoves}
                {sortColumn}
                {sortDirection}
                selectedMove={$selectedMoveStore}
                {isPlayedMove}
                {isPlayedCubeAction}
                onSort={handleSort}
                onRowClick={handleMoveRowClick}
                {isMoney}
                {jacoby}
                {beaver}
                {maxCube}
            />
            <!-- Une ligne, et seulement quand une règle est confiante. -->
            <ExplanationLine analysis={analysisData} />
        {/if}
    </div>
</section>

<style>
    /* Plage inerte de la taille du bloc (ADR-0018 règle 6). */
    .quiz-masked {
        display: flex;
        align-items: center;
        justify-content: center;
        min-height: 6em;
        color: var(--color-text-muted);
        letter-spacing: 0.4em;
    }

    .analysis-panel {
        width: 100%;
        height: 100%;
        overflow-y: auto;
        background-color: var(--color-surface);
        padding: 10px;
        box-sizing: border-box;
        outline: none;
        resize: none;
        /* Query container: layout follows the panel's own width. */
        container-type: inline-size;
    }

    .analysis-content {
        font-size: var(--font-size-base); /* Reduce font size */
        color: var(--color-text);
        cursor: default; /* Make analysis content interactive in MATCH mode (toggle on click) */
    }
</style>
