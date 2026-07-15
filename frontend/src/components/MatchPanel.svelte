<script>
    import { logger } from '../utils/logger.js';
    import { sortMatches, toDateInputValue, formatDate, formatDiceShort, MATCH_STAT_ROWS } from '../utils/matchTable.js';
    import { nextSort } from '../utils/tableSort.js';
    import { onMount, onDestroy, untrack } from 'svelte';
    import { get } from 'svelte/store';
    import {
        GetAllMatches,
        DeleteMatch,
        UpdateMatch,
        UpdateMatchComment,
        GetMatchMovePositions,
        GetGamesByMatch,
        LoadAnalysis,
        GetAllTournaments,
        SetMatchTournamentByName,
        SwapMatchPlayers,
        SaveLastVisitedPosition,
        GetMatchDetailStats
    } from '../../wailsjs/go/database/Database.js';
    import MergePlayersModal from './MergePlayersModal.svelte';
    import { exportMatchMat } from '../services/exportService.js';
    import { t, tMsg } from '../i18n';
    import { positionStore, matchContextStore, lastVisitedMatchStore } from '../stores/positionStore';
    import {
        statusBarModeStore,
        openPanels,
        PANEL,
        closePanel,
        matchPanelRefreshTriggerStore,
        dbMutationCounterStore,
        positionReloadTriggerStore,
        statusBarTextStore,
        activeTabStore
    } from '../stores/uiStore';
    import { analysisStore, selectedMoveStore } from '../stores/analysisStore';
    import { commentTextStore } from '../stores/uiStore';
    import { tournamentsStore } from '../stores/tournamentStore';
    import { databaseLoadedStore } from '../stores/databaseStore';

    let matches = $state([]);
    let selectedMatch = $state(null);
    const visible = $derived($openPanels.has(PANEL.MATCH));
    const databaseLoaded = $derived($databaseLoadedStore);
    let lastVisitedMatch = $derived($lastVisitedMatchStore);
    let tournaments = $derived($tournamentsStore || []);

    // Detail pane state
    let detailMatch = $state(null); // Match currently shown in detail pane
    let detailMovePositions = $state([]); // MatchMovePosition[] for the detail match
    let detailGames = $state([]); // Game[] for the detail match
    let detailView = $state('transcript'); // 'transcript' | 'metadata' | 'stats'
    let loadingDetail = $state(false);
    let detailStats = $state(null); // MatchDetailStats for the detail match
    let loadingStats = $state(false);

    // Sorting state
    let sortColumn = $state(null); // null | 'player1' | 'player2' | 'date' | 'length' | 'tournament'
    let sortDirection = $state('asc'); // 'asc' | 'desc'

    // Inline tournament editing
    let editingTournamentMatchId = $state(null);
    let editTournamentValue = $state('');
    let showTournamentDropdown = $state(false);
    let filteredTournaments = $state([]);
    let tournamentDropdownStyle = $state('');

    // Inline match editing (player names, date)
    let editingMatchId = $state(null);
    let editPlayer1Value = $state('');
    let editPlayer2Value = $state('');
    let editDateValue = $state('');

    // Merge players modal
    let showMergePlayersModal = $state(false);

    // Inline match comment editing
    let editingDetailComment = $state(false);
    let editDetailCommentText = $state('');

    // Reload matches when a new match is imported (trigger increments from 0)
    $effect(() => {
        const trigger = $matchPanelRefreshTriggerStore;
        if (trigger === 0) return; // skip initial run
        if (untrack(() => !visible || !databaseLoaded)) return;
        loadMatches().then(() => {
            const lvm = lastVisitedMatch;
            if (lvm && lvm.matchID) {
                const m = matches.find((mm) => mm.id === lvm.matchID);
                if (m) {
                    selectedMatch = m;
                    loadMatchDetail(m);
                }
            }
        });
    });

    // Track panel open/close transitions and load data on open. The panel may be
    // open before the DB has finished loading (session restore opens the Matches
    // tab by default, before openDatabaseByPath completes); that case is covered
    // by the matchPanelRefreshTriggerStore bump fired once the DB is open and the
    // session restored — see openDatabaseByPath.
    let _prevVisible = false;
    $effect(() => {
        const opened = visible; // $derived — tracked
        const wasVisible = _prevVisible;
        _prevVisible = opened;
        if (opened && !wasVisible && databaseLoaded) {
            loadMatches().then(() => {
                const lvm = lastVisitedMatch;
                if (lvm && lvm.matchID) {
                    const m = matches.find((mm) => mm.id === lvm.matchID);
                    if (m) {
                        selectedMatch = m;
                        loadMatchDetail(m);
                    } else {
                        selectedMatch = null;
                        detailMatch = null;
                    }
                } else {
                    selectedMatch = null;
                    detailMatch = null;
                }
            });
        } else if (!opened && wasVisible) {
            selectedMatch = null;
            detailMatch = null;
            detailStats = null;
            editingTournamentMatchId = null;
        }
    });

    async function loadMatches() {
        return logger.perf('MatchPanel:loadMatches', async () => {
            try {
                const loadedMatches = await GetAllMatches();
                matches = loadedMatches || [];
                await loadTournaments();
            } catch (error) {
                logger.error('Error loading matches:', error);
                matches = [];
            }
        });
    }

    async function loadTournaments() {
        try {
            const loaded = await GetAllTournaments();
            tournamentsStore.set(loaded || []);
        } catch (error) {
            logger.error('Error loading tournaments:', error);
        }
    }

    function startEditTournament(match, event) {
        event.stopPropagation();
        editingTournamentMatchId = match.id;
        editTournamentValue = match.tournament_name || match.event || '';
        filteredTournaments = tournaments;
        setTimeout(() => {
            const input = document.querySelector('.tournament-edit-input');
            if (input) {
                input.focus();
                computeTournamentDropdownPosition(input);
            }
            showTournamentDropdown = true;
        }, 50);
    }

    function computeTournamentDropdownPosition(inputEl) {
        if (!inputEl) return;
        const rect = inputEl.getBoundingClientRect();
        const spaceBelow = window.innerHeight - rect.bottom;
        const maxH = 120;
        if (spaceBelow < maxH && rect.top > spaceBelow) {
            tournamentDropdownStyle = `position:fixed; bottom:${window.innerHeight - rect.top}px; left:${rect.left}px; width:${rect.width}px; max-height:${Math.min(maxH, rect.top)}px;`;
        } else {
            tournamentDropdownStyle = `position:fixed; top:${rect.bottom}px; left:${rect.left}px; width:${rect.width}px; max-height:${Math.min(maxH, spaceBelow)}px;`;
        }
    }

    function filterTournaments() {
        const val = editTournamentValue.toLowerCase();
        if (!val) {
            filteredTournaments = tournaments;
        } else {
            filteredTournaments = tournaments.filter((t) => t.name.toLowerCase().includes(val));
        }
        const input = document.querySelector('.tournament-edit-input');
        if (input) computeTournamentDropdownPosition(input);
        showTournamentDropdown = true;
    }

    async function selectTournamentOption(name) {
        editTournamentValue = name;
        showTournamentDropdown = false;
        await saveTournamentEdit();
    }

    async function saveTournamentEdit() {
        if (editingTournamentMatchId === null) return;
        try {
            await SetMatchTournamentByName(editingTournamentMatchId, editTournamentValue.trim());
            await loadMatches();
            await loadTournaments();
            statusBarTextStore.set(editTournamentValue.trim() ? tMsg('match.tournamentSet', { name: editTournamentValue.trim() }) : tMsg('match.tournamentCleared'));
        } catch (error) {
            logger.error('Error setting tournament:', error);
            statusBarTextStore.set(tMsg('match.errorSettingTournament'));
        }
        editingTournamentMatchId = null;
        editTournamentValue = '';
        showTournamentDropdown = false;
    }

    function cancelTournamentEdit() {
        editingTournamentMatchId = null;
        editTournamentValue = '';
        showTournamentDropdown = false;
    }

    function handleTournamentKeyDown(event) {
        if (event.key === 'Enter') {
            event.stopPropagation();
            event.preventDefault();
            saveTournamentEdit();
        } else if (event.key === 'Escape') {
            event.stopPropagation();
            event.preventDefault();
            cancelTournamentEdit();
        }
    }

    function startEditMatch(match, ev) {
        ev.stopPropagation();
        editingMatchId = match.id;
        editPlayer1Value = match.player1_name || '';
        editPlayer2Value = match.player2_name || '';
        editDateValue = toDateInputValue(match.match_date);
    }

    async function saveMatchEdit() {
        if (editingMatchId === null) return;
        try {
            await UpdateMatch(editingMatchId, editPlayer1Value, editPlayer2Value, editDateValue);
            await loadMatches();
            statusBarTextStore.set(tMsg('match.matchUpdated'));
        } catch (error) {
            logger.error('Error updating match:', error);
            statusBarTextStore.set(tMsg('match.errorUpdating'));
        }
        editingMatchId = null;
    }

    function cancelMatchEdit() {
        editingMatchId = null;
        editPlayer1Value = '';
        editPlayer2Value = '';
        editDateValue = '';
    }

    function startEditDetailComment() {
        editDetailCommentText = detailMatch.comment || '';
        editingDetailComment = true;
    }

    async function saveDetailComment() {
        if (!detailMatch) return;
        try {
            await UpdateMatchComment(detailMatch.id, editDetailCommentText);
            detailMatch.comment = editDetailCommentText;
            const m = matches.find((x) => x.id === detailMatch.id);
            if (m) m.comment = editDetailCommentText;
            matches = matches;
            statusBarTextStore.set(tMsg('match.commentUpdated'));
        } catch (error) {
            logger.error('Error updating comment:', error);
            statusBarTextStore.set(tMsg('match.errorUpdatingComment'));
        }
        editingDetailComment = false;
    }

    function cancelDetailComment() {
        editingDetailComment = false;
        editDetailCommentText = '';
    }

    function handleDetailCommentKeyDown(event) {
        if (event.key === 'Enter' && !event.shiftKey) {
            event.stopPropagation();
            event.preventDefault();
            saveDetailComment();
        } else if (event.key === 'Escape') {
            event.stopPropagation();
            event.preventDefault();
            cancelDetailComment();
        }
    }

    function handleMatchEditKeyDown(event) {
        if (event.key === 'Enter') {
            event.stopPropagation();
            event.preventDefault();
            saveMatchEdit();
        } else if (event.key === 'Escape') {
            event.stopPropagation();
            event.preventDefault();
            cancelMatchEdit();
        }
    }

    // --- Sorting helpers ---
    function handleSort(column) {
        const next = nextSort(sortColumn, sortDirection, column, { tristate: true });
        sortColumn = next.column;
        sortDirection = next.direction;
    }

    let sortedMatches = $derived.by(() => sortMatches(matches, sortColumn, sortDirection));

    function selectMatch(match) {
        if (selectedMatch && selectedMatch.id === match.id) {
            selectedMatch = null;
            detailMatch = null;
            detailMovePositions = [];
            detailGames = [];
            detailStats = null;
        } else {
            selectedMatch = match;
            loadMatchDetail(match);
        }
    }

    async function loadMatchDetail(match) {
        if (!match) return;
        loadingDetail = true;
        detailMatch = match;
        detailStats = null; // reset stats when switching match
        try {
            const [movePositions, games] = await Promise.all([GetMatchMovePositions(match.id), GetGamesByMatch(match.id)]);
            detailMovePositions = movePositions || [];
            detailGames = games || [];
        } catch (error) {
            logger.error('Error loading match detail:', error);
            detailMovePositions = [];
            detailGames = [];
        }
        loadingDetail = false;
    }

    async function loadMatchStats(match) {
        if (!match || loadingStats) return;
        loadingStats = true;
        try {
            detailStats = await GetMatchDetailStats(match.id);
        } catch (error) {
            logger.error('Error loading match stats:', error);
            detailStats = null;
        }
        loadingStats = false;
    }

    function switchDetailView(view) {
        detailView = view;
        if (view === 'stats' && detailMatch && !detailStats && !loadingStats) {
            loadMatchStats(detailMatch);
        }
    }

    // Group move positions by game number for transcript display
    let transcriptGames = $derived.by(() => {
        if (!detailMovePositions.length) return [];
        // eslint-disable-next-line svelte/prefer-svelte-reactivity -- local temp inside $derived
        const gameMap = new Map();
        for (const mp of detailMovePositions) {
            if (!gameMap.has(mp.game_number)) {
                gameMap.set(mp.game_number, []);
            }
            gameMap.get(mp.game_number).push(mp);
        }
        const result = [];
        for (const [gameNum, moves] of gameMap) {
            // Find corresponding game info
            const gameInfo = detailGames.find((g) => g.game_number === gameNum);
            result.push({ gameNumber: gameNum, moves, gameInfo });
        }
        return result;
    });

    async function navigateToMove(moveIndex) {
        if (!detailMatch || !detailMovePositions.length) return;
        // Enter match mode and navigate to the clicked move
        const movePositions = detailMovePositions;
        const match = detailMatch;

        matchContextStore.set({
            isMatchMode: true,
            matchID: match.id,
            movePositions: movePositions,
            currentIndex: moveIndex,
            player1Name: match.player1_name,
            player2Name: match.player2_name
        });

        const movePos = movePositions[moveIndex];
        positionStore.set(movePos.position);

        let analysis = null;
        try {
            analysis = await LoadAnalysis(movePos.position.id);
        } catch (_error) {
            /* ignored */
        }

        const currentPlayedMove = movePos.checker_move || '';
        const currentPlayedCubeAction = movePos.cube_action || '';

        analysisStore.set({
            positionId: analysis?.positionId || null,
            xgid: analysis?.xgid || '',
            player1: analysis?.player1 || '',
            player2: analysis?.player2 || '',
            analysisType: analysis?.analysisType || '',
            analysisEngineVersion: analysis?.analysisEngineVersion || '',
            checkerAnalysis: analysis?.checkerAnalysis || { moves: [] },
            doublingCubeAnalysis: analysis?.doublingCubeAnalysis || {
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
            playedMove: currentPlayedMove,
            playedCubeAction: currentPlayedCubeAction,
            playedMoves: analysis?.playedMoves || [],
            playedCubeActions: analysis?.playedCubeActions || [],
            creationDate: analysis?.creationDate || '',
            lastModifiedDate: analysis?.lastModifiedDate || ''
        });

        commentTextStore.set('');
        selectedMoveStore.set(null);
        statusBarModeStore.set('MATCH');
        // Player names are shown in the match-info header bar above the board
        // (MatchInfoBar.svelte), so the status bar no longer echoes "P1 vs P2".

        lastVisitedMatchStore.set({
            matchID: match.id,
            currentIndex: moveIndex,
            gameNumber: movePos.game_number
        });

        SaveLastVisitedPosition(match.id, moveIndex).catch((e) => {
            logger.error('Error persisting last visited position:', e);
        });

        // Switch to analysis tab so user sees the analysis
        activeTabStore.set('analysis');
    }

    async function enterMatchMode(match) {
        if (!detailMovePositions.length) {
            // Load if not already loaded
            await loadMatchDetail(match);
        }
        if (!detailMovePositions.length) {
            statusBarTextStore.set(tMsg('match.noMovesFound'));
            return;
        }

        let startIndex = 0;
        if (lastVisitedMatch && lastVisitedMatch.matchID === match.id) {
            if (lastVisitedMatch.currentIndex >= 0 && lastVisitedMatch.currentIndex < detailMovePositions.length) {
                startIndex = lastVisitedMatch.currentIndex;
            }
        } else if (match.last_visited_position >= 0 && match.last_visited_position < detailMovePositions.length) {
            startIndex = match.last_visited_position;
        }

        await navigateToMove(startIndex);
    }

    function handleDoubleClick(match) {
        enterMatchMode(match);
    }

    async function deleteMatchEntry(match, event) {
        event.stopPropagation();
        if (!confirm(`Delete match between ${match.player1_name} and ${match.player2_name}?`)) return;
        try {
            await DeleteMatch(match.id);
            await loadMatches();
            if (selectedMatch && selectedMatch.id === match.id) {
                selectedMatch = null;
                detailMatch = null;
                detailMovePositions = [];
                detailGames = [];
                detailStats = null;
            }
            // Trigger match panel refresh to update all dependent components
            matchPanelRefreshTriggerStore.update((n) => n + 1);
            dbMutationCounterStore.update((n) => n + 1);
            // Trigger position reload to reflect deleted positions
            positionReloadTriggerStore.update((n) => n + 1);
            statusBarTextStore.set(tMsg('match.matchDeleted'));
        } catch (error) {
            logger.error('Error deleting match:', error);
            statusBarTextStore.set(tMsg('match.errorDeleting'));
        }
    }

    async function swapMatchPlayers(match, event) {
        event.stopPropagation();
        try {
            await SwapMatchPlayers(match.id);
            await loadMatches();

            // If we are currently viewing this match in match mode, update context
            const currentContext = get(matchContextStore);
            if (currentContext && currentContext.isMatchMode && currentContext.matchID === match.id) {
                // Reload match positions to reflect swapped players
                const movePositions = await GetMatchMovePositions(match.id);
                if (movePositions && movePositions.length > 0) {
                    const currentIndex = Math.min(currentContext.currentIndex, movePositions.length - 1);
                    matchContextStore.set({
                        isMatchMode: true,
                        matchID: match.id,
                        movePositions: movePositions,
                        currentIndex: currentIndex,
                        player1Name: movePositions[0].player1_name,
                        player2Name: movePositions[0].player2_name
                    });
                    // Update the displayed position
                    positionStore.set(movePositions[currentIndex].position);
                    positionReloadTriggerStore.update((n) => n + 1);
                }
            }

            // Reload detail pane if viewing this match
            if (detailMatch && detailMatch.id === match.id) {
                detailMatch = matches.find((m) => m.id === match.id) || detailMatch;
                await loadMatchDetail(detailMatch);
            }

            statusBarTextStore.set(tMsg('match.swappedPlayers'));
        } catch (error) {
            logger.error('Error swapping match players:', error);
            statusBarTextStore.set(tMsg('match.errorSwapping'));
        }
    }

    function _formatMoveText(mp) {
        if (mp.move_type === 'cube') {
            return mp.cube_action || get(t)('match.cubeAction');
        }
        return mp.checker_move || '—';
    }

    function getPlayerName(mp) {
        return mp.player_on_roll === 0 ? mp.player1_name || get(t)('match.player1') : mp.player2_name || get(t)('match.player2');
    }

    function closeMatchPanel() {
        closePanel(PANEL.MATCH);
    }

    function handleKeyDown(event) {
        if (!visible) return;
        // Don't intercept keys while the merge players modal is open
        if (showMergePlayersModal) return;

        // Let Ctrl+key combos pass through to global handler (e.g. Ctrl+Tab to toggle panel)
        if (event.ctrlKey) return;

        // Let Space pass through so the command line can be opened from global handler
        if (event.code === 'Space') return;

        // Let ? pass through so the help modal can be opened
        if (event.key === '?') return;

        // Stop all keyboard events from propagating to global handlers
        event.stopPropagation();

        if (event.key === 'Escape') {
            if (editingMatchId !== null) {
                cancelMatchEdit();
                event.preventDefault();
            } else if (editingTournamentMatchId !== null) {
                cancelTournamentEdit();
                event.preventDefault();
            } else if (detailMatch) {
                // Close detail pane first
                selectedMatch = null;
                detailMatch = null;
                detailMovePositions = [];
                detailGames = [];
                detailStats = null;
                event.preventDefault();
            } else {
                closeMatchPanel();
            }
            return;
        }

        if (sortedMatches.length > 0) {
            if ((event.key === 'j' || event.key === 'ArrowDown') && !selectedMatch) {
                event.preventDefault();
                selectMatch(sortedMatches[0]);
                setTimeout(() => {
                    const selectedRow = document.querySelector('.match-panel tr.selected');
                    if (selectedRow) selectedRow.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
                }, 0);
                return;
            }
        }

        if (selectedMatch && sortedMatches.length > 0) {
            const currentIndex = sortedMatches.findIndex((m) => m.id === selectedMatch.id);

            if (event.key === 'j' || event.key === 'ArrowDown') {
                event.preventDefault();
                if (currentIndex >= 0 && currentIndex < sortedMatches.length - 1) {
                    selectMatch(sortedMatches[currentIndex + 1]);
                    setTimeout(() => {
                        const selectedRow = document.querySelector('.match-panel tr.selected');
                        if (selectedRow) selectedRow.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
                    }, 0);
                }
            } else if (event.key === 'k' || event.key === 'ArrowUp') {
                event.preventDefault();
                if (currentIndex > 0) {
                    selectMatch(sortedMatches[currentIndex - 1]);
                    setTimeout(() => {
                        const selectedRow = document.querySelector('.match-panel tr.selected');
                        if (selectedRow) selectedRow.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
                    }, 0);
                }
            } else if (event.key === 'Enter') {
                event.preventDefault();
                handleDoubleClick(selectedMatch);
            } else if (event.key === 'Delete') {
                event.preventDefault();
                const syntheticEvent = { stopPropagation: () => {} };
                deleteMatchEntry(selectedMatch, syntheticEvent);
            }
        }
    }

    function handleClickOutside(event) {
        // Don't interfere while the merge players modal is open
        if (showMergePlayersModal) return;
        // Close tournament dropdown if clicking outside
        if (editingTournamentMatchId !== null && !event.target.closest('.tournament-cell-edit')) {
            cancelTournamentEdit();
        }
        // Cancel match edit if clicking outside the editing row
        if (editingMatchId !== null && !event.target.closest('.match-editing-row')) {
            cancelMatchEdit();
        }
        const panel = document.getElementById('matchPanel');
        if (panel && !panel.contains(event.target)) {
            document.activeElement.blur();
        }
    }

    $effect(() => {
        if (visible) {
            const id = setTimeout(() => {
                const panel = document.getElementById('matchPanel');
                if (panel) panel.focus();
                if (selectedMatch) {
                    const selectedRow = document.querySelector('.match-panel tr.selected');
                    if (selectedRow) selectedRow.scrollIntoView({ behavior: 'smooth', block: 'center' });
                }
            }, 100);
            return () => clearTimeout(id);
        }
    });

    onMount(async () => {
        if (visible) await loadMatches();
        document.addEventListener('click', handleClickOutside);
        document.addEventListener('keydown', handleKeyDown);
    });

    onDestroy(() => {
        document.removeEventListener('click', handleClickOutside);
        document.removeEventListener('keydown', handleKeyDown);
    });
</script>

<section class="match-panel" role="dialog" aria-modal="true" aria-label={$t('match.ariaLabel')} id="matchPanel" tabindex="-1">
    <div class="match-panel-content">
        <!-- Match list (left pane) -->
        <div class="match-list-pane" class:has-detail={detailMatch !== null}>
            <div class="match-list-toolbar">
                <button class="toolbar-btn" onclick={() => (showMergePlayersModal = true)} title={$t('match.mergePlayersTitle')} disabled={matches.length === 0}>⇢ {$t('match.mergePlayers')}</button>
            </div>
            <div class="match-table-container">
                <table class="match-table">
                    <thead>
                        <tr>
                            <th class="no-select narrow-col">#</th>
                            <th class="no-select sortable narrow-col" onclick={() => handleSort('date')}
                                >{$t('match.date')}
                                {#if sortColumn === 'date'}<span class="sort-arrow">{sortDirection === 'asc' ? '▲' : '▼'}</span>{/if}</th
                            >
                            <th class="no-select sortable" onclick={() => handleSort('player1')}
                                >{$t('match.player1')}
                                {#if sortColumn === 'player1'}<span class="sort-arrow">{sortDirection === 'asc' ? '▲' : '▼'}</span>{/if}</th
                            >
                            <th class="no-select sortable" onclick={() => handleSort('player2')}
                                >{$t('match.player2')}
                                {#if sortColumn === 'player2'}<span class="sort-arrow">{sortDirection === 'asc' ? '▲' : '▼'}</span>{/if}</th
                            >
                            <th class="no-select sortable narrow-col" onclick={() => handleSort('length')}
                                >{$t('match.pts')}
                                {#if sortColumn === 'length'}<span class="sort-arrow">{sortDirection === 'asc' ? '▲' : '▼'}</span>{/if}</th
                            >
                            <th class="no-select sortable tournament-col" onclick={() => handleSort('tournament')}
                                >{$t('match.tournament')}
                                {#if sortColumn === 'tournament'}<span class="sort-arrow">{sortDirection === 'asc' ? '▲' : '▼'}</span>{/if}</th
                            >
                            <th class="no-select sortable narrow-col" onclick={() => handleSort('pr')}
                                >PR {#if sortColumn === 'pr'}<span class="sort-arrow">{sortDirection === 'asc' ? '▲' : '▼'}</span>{/if}</th
                            >
                            <th class="no-select sortable narrow-col" onclick={() => handleSort('mwc')}
                                >MWC {#if sortColumn === 'mwc'}<span class="sort-arrow">{sortDirection === 'asc' ? '▲' : '▼'}</span>{/if}</th
                            >
                            <th class="no-select actions-col"></th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each sortedMatches as match, index (match.id)}
                            {#if editingMatchId === match.id}
                                <tr class="match-editing-row" class:selected={selectedMatch && selectedMatch.id === match.id}>
                                    <td class="index-cell narrow-col no-select">{index + 1}</td>
                                    <td class="narrow-col">
                                        <input type="date" class="match-edit-input" bind:value={editDateValue} onkeydown={handleMatchEditKeyDown} />
                                    </td>
                                    <td>
                                        <input type="text" class="match-edit-input" bind:value={editPlayer1Value} onkeydown={handleMatchEditKeyDown} placeholder={$t('match.player1')} />
                                    </td>
                                    <td>
                                        <input type="text" class="match-edit-input" bind:value={editPlayer2Value} onkeydown={handleMatchEditKeyDown} placeholder={$t('match.player2')} />
                                    </td>
                                    <td class="narrow-col no-select">{match.match_length}</td>
                                    <td class="tournament-col no-select">{match.tournament_name || ''}</td>
                                    <td class="narrow-col no-select">{match.pr > 0 ? match.pr.toFixed(2) : ''}{match.pr2 > 0 ? ' / ' + match.pr2.toFixed(2) : ''}</td>
                                    <td class="narrow-col no-select">{match.mwc_loss > 0 ? (match.mwc_loss * 100).toFixed(2) + '%' : ''}</td>
                                    <td class="actions-col no-select">
                                        <span class="item-actions editing-actions">
                                            <button
                                                class="icon-btn"
                                                onclick={(e) => {
                                                    e.stopPropagation();
                                                    saveMatchEdit(e);
                                                }}
                                                title={$t('common.save')}>✓</button
                                            >
                                            <button
                                                class="icon-btn"
                                                onclick={(e) => {
                                                    e.stopPropagation();
                                                    cancelMatchEdit(e);
                                                }}
                                                title={$t('common.cancel')}>✕</button
                                            >
                                        </span>
                                    </td>
                                </tr>
                            {:else}
                                <tr class:selected={selectedMatch && selectedMatch.id === match.id} onclick={() => selectMatch(match)} ondblclick={() => handleDoubleClick(match)}>
                                    <td class="index-cell narrow-col no-select">{index + 1}</td>
                                    <td class="narrow-col no-select">{formatDate(match.match_date)}</td>
                                    <td class="no-select">{match.player1_name}</td>
                                    <td class="no-select">{match.player2_name}</td>
                                    <td class="narrow-col no-select">{match.match_length}</td>
                                    <td
                                        class="tournament-col no-select tournament-meta-cell"
                                        onclick={(e) => {
                                            e.stopPropagation();
                                            ((e) => startEditTournament(match, e))(e);
                                        }}
                                    >
                                        {#if editingTournamentMatchId === match.id}
                                            <div class="tournament-cell-edit">
                                                <input
                                                    type="text"
                                                    class="tournament-edit-input"
                                                    bind:value={editTournamentValue}
                                                    oninput={filterTournaments}
                                                    onkeydown={handleTournamentKeyDown}
                                                    onblur={() => setTimeout(cancelTournamentEdit, 200)}
                                                    placeholder={$t('match.tournamentNamePlaceholder')}
                                                />
                                                {#if showTournamentDropdown && filteredTournaments.length > 0}
                                                    <div class="tournament-dropdown" style={tournamentDropdownStyle}>
                                                        {#each filteredTournaments as t (t.name)}
                                                            <div
                                                                class="tournament-dropdown-item"
                                                                onmousedown={(e) => {
                                                                    e.preventDefault();
                                                                    (() => selectTournamentOption(t.name))();
                                                                }}
                                                            >
                                                                {t.name}
                                                            </div>
                                                        {/each}
                                                    </div>
                                                {/if}
                                            </div>
                                        {:else}
                                            <span class="tournament-display" title={$t('match.clickToAssignTournament')}>{match.tournament_name || ''}</span>
                                        {/if}
                                    </td>
                                    <td class="narrow-col no-select stat-col">{match.pr > 0 ? match.pr.toFixed(2) : '—'}{match.pr2 > 0 ? ' / ' + match.pr2.toFixed(2) : ''}</td>
                                    <td class="narrow-col no-select stat-col">{match.mwc_loss > 0 ? (match.mwc_loss * 100).toFixed(2) + '%' : '—'}</td>
                                    <td class="actions-col no-select">
                                        <span class="item-actions">
                                            <button
                                                class="icon-btn"
                                                onclick={(e) => {
                                                    e.stopPropagation();
                                                    exportMatchMat(match);
                                                }}
                                                title={$t('match.exportMat')}>⬇</button
                                            >
                                            <button
                                                class="icon-btn"
                                                onclick={(e) => {
                                                    e.stopPropagation();
                                                    ((e) => swapMatchPlayers(match, e))(e);
                                                }}
                                                title={$t('match.swapPlayers')}>⇄</button
                                            >
                                            <button
                                                class="icon-btn"
                                                onclick={(e) => {
                                                    e.stopPropagation();
                                                    ((e) => startEditMatch(match, e))(e);
                                                }}
                                                title={$t('common.edit')}>✎</button
                                            >
                                            <button
                                                class="icon-btn delete"
                                                onclick={(e) => {
                                                    e.stopPropagation();
                                                    ((e) => deleteMatchEntry(match, e))(e);
                                                }}
                                                title={$t('common.delete')}>×</button
                                            >
                                        </span>
                                    </td>
                                </tr>
                            {/if}
                        {/each}
                    </tbody>
                </table>
                {#if matches.length === 0}
                    <div class="empty-state">{$t('match.noMatchesImported')}</div>
                {/if}
            </div>
        </div>

        <!-- Detail pane (right side, shown when a match is selected) -->
        {#if detailMatch}
            <div class="detail-pane">
                <!-- Match metadata header -->
                <div class="detail-header">
                    <div class="detail-title">
                        <span class="player-name">{detailMatch.player1_name}</span>
                        <span class="vs-label">{$t('match.vs')}</span>
                        <span class="player-name">{detailMatch.player2_name}</span>
                        <span class="match-length-badge">{detailMatch.match_length} pt</span>
                    </div>
                    <div class="detail-meta">
                        {#if detailMatch.match_date && formatDate(detailMatch.match_date) !== '-'}
                            <span class="meta-item" title={$t('match.date')}>{formatDate(detailMatch.match_date)}</span>
                        {/if}
                        {#if detailMatch.tournament_name || detailMatch.event}
                            <span class="meta-item meta-tournament" title={$t('match.tournament')}>{detailMatch.tournament_name || detailMatch.event}</span>
                        {/if}
                        {#if detailMatch.round}
                            <span class="meta-item" title={$t('match.round')}>R{detailMatch.round}</span>
                        {/if}
                        {#if detailMatch.location}
                            <span class="meta-item" title={$t('match.location')}>{detailMatch.location}</span>
                        {/if}
                    </div>
                    <div class="detail-tabs">
                        <button class="detail-tab" class:active={detailView === 'transcript'} onclick={() => switchDetailView('transcript')}>{$t('match.transcript')}</button>
                        <button class="detail-tab" class:active={detailView === 'metadata'} onclick={() => switchDetailView('metadata')}>{$t('match.info')}</button>
                        <button class="detail-tab" class:active={detailView === 'stats'} onclick={() => switchDetailView('stats')}>{$t('match.stats')}</button>
                        <button class="detail-tab export-mat-btn" onclick={() => exportMatchMat(detailMatch)} title={$t('match.exportMat')}>⬇ .mat</button>
                        <button class="detail-tab enter-match-btn" onclick={() => enterMatchMode(detailMatch)} title="{$t('match.enterMatchMode')} (↵)">▶ {$t('match.review')}</button>
                    </div>
                </div>

                <!-- Transcript view -->
                {#if detailView === 'transcript'}
                    <div class="transcript-container">
                        {#if loadingDetail}
                            <div class="loading-state">{$t('common.loading')}</div>
                        {:else if transcriptGames.length === 0}
                            <div class="empty-state">{$t('match.noMovesRecorded')}</div>
                        {:else}
                            {#each transcriptGames as game (game.gameNumber)}
                                <div class="game-section">
                                    <div class="game-header">
                                        <span class="game-title">{$t('match.game', { n: game.gameNumber })}</span>
                                        {#if game.gameInfo}
                                            <span class="game-score">{$t('match.score')}: {game.gameInfo.initial_score[0]}–{game.gameInfo.initial_score[1]}</span>
                                            {#if game.gameInfo.winner >= 0}
                                                <span class="game-result"
                                                    >{$t('match.wonBy', {
                                                        player: game.gameInfo.winner === 0 ? detailMatch.player1_name : detailMatch.player2_name,
                                                        points: game.gameInfo.points_won
                                                    })}</span
                                                >
                                            {/if}
                                        {/if}
                                    </div>
                                    <table class="transcript-table">
                                        <thead>
                                            <tr>
                                                <th class="transcript-num">#</th>
                                                <th class="transcript-player">{$t('match.player')}</th>
                                                <th class="transcript-dice">{$t('match.dice')}</th>
                                                <th class="transcript-move">{$t('match.move')}</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {#each game.moves as mp, mi (mi)}
                                                {@const globalIdx = detailMovePositions.indexOf(mp)}
                                                <tr class="transcript-row" class:cube-row={mp.move_type === 'cube'} onclick={() => navigateToMove(globalIdx)} title={$t('match.clickToReview')}>
                                                    <td class="transcript-num">{mi + 1}</td>
                                                    <td class="transcript-player" class:player1={mp.player_on_roll === 0} class:player2={mp.player_on_roll === 1}>
                                                        {getPlayerName(mp)}
                                                    </td>
                                                    <td class="transcript-dice">
                                                        {#if mp.move_type === 'checker'}
                                                            {formatDiceShort(mp.position.dice)}
                                                        {/if}
                                                    </td>
                                                    <td class="transcript-move">
                                                        {#if mp.move_type === 'cube'}
                                                            <span class="cube-action">{mp.cube_action || $t('match.cube')}</span>
                                                        {:else}
                                                            {mp.checker_move || '—'}
                                                        {/if}
                                                    </td>
                                                </tr>
                                            {/each}
                                        </tbody>
                                    </table>
                                </div>
                            {/each}
                        {/if}
                    </div>
                {/if}

                <!-- Metadata view -->
                {#if detailView === 'metadata'}
                    <div class="metadata-container">
                        <table class="metadata-table">
                            <tbody>
                                <tr><td class="meta-label">{$t('match.player1')}</td><td class="meta-value">{detailMatch.player1_name || '—'}</td></tr>
                                <tr><td class="meta-label">{$t('match.player2')}</td><td class="meta-value">{detailMatch.player2_name || '—'}</td></tr>
                                <tr
                                    ><td class="meta-label">{$t('match.matchLength')}</td><td class="meta-value"
                                        >{detailMatch.match_length > 1 ? $t('match.points', { n: detailMatch.match_length }) : $t('match.point', { n: detailMatch.match_length })}</td
                                    ></tr
                                >
                                <tr><td class="meta-label">{$t('match.games')}</td><td class="meta-value">{detailMatch.game_count || detailGames.length || '—'}</td></tr>
                                <tr><td class="meta-label">{$t('match.date')}</td><td class="meta-value">{formatDate(detailMatch.match_date)}</td></tr>
                                <tr>
                                    <td class="meta-label">{$t('match.comment')}</td>
                                    <td class="meta-value">
                                        {#if editingDetailComment}
                                            <input type="text" class="match-comment-input" bind:value={editDetailCommentText} onkeydown={handleDetailCommentKeyDown} onblur={saveDetailComment} />
                                        {:else}
                                            <!-- svelte-ignore a11y_click_events_have_key_events -->
                                            <!-- svelte-ignore a11y_no_static_element_interactions -->
                                            <span class="match-comment-display" onclick={startEditDetailComment} title={$t('match.clickToAddComment')}>
                                                {detailMatch.comment || $t('match.addComment')}
                                            </span>
                                        {/if}
                                    </td>
                                </tr>
                                <tr>
                                    <td class="meta-label">{$t('match.tournament')}</td>
                                    <td
                                        class="meta-value tournament-meta-cell"
                                        onclick={(e) => {
                                            e.stopPropagation();
                                            ((e) => startEditTournament(detailMatch, e))(e);
                                        }}
                                    >
                                        {#if editingTournamentMatchId === detailMatch.id}
                                            <div class="tournament-cell-edit">
                                                <input
                                                    type="text"
                                                    class="tournament-edit-input"
                                                    bind:value={editTournamentValue}
                                                    oninput={filterTournaments}
                                                    onkeydown={handleTournamentKeyDown}
                                                    onblur={() => setTimeout(cancelTournamentEdit, 200)}
                                                    placeholder={$t('match.tournamentNamePlaceholder')}
                                                />
                                                {#if showTournamentDropdown && filteredTournaments.length > 0}
                                                    <div class="tournament-dropdown" style={tournamentDropdownStyle}>
                                                        {#each filteredTournaments as t (t.name)}
                                                            <div
                                                                class="tournament-dropdown-item"
                                                                onmousedown={(e) => {
                                                                    e.preventDefault();
                                                                    (() => selectTournamentOption(t.name))();
                                                                }}
                                                            >
                                                                {t.name}
                                                            </div>
                                                        {/each}
                                                    </div>
                                                {/if}
                                            </div>
                                        {:else}
                                            <span class="tournament-display" title={$t('match.clickToEdit')}>{detailMatch.tournament_name || detailMatch.event || '—'}</span>
                                        {/if}
                                    </td>
                                </tr>
                                <tr><td class="meta-label">{$t('match.event')}</td><td class="meta-value">{detailMatch.event || '—'}</td></tr>
                                <tr><td class="meta-label">{$t('match.location')}</td><td class="meta-value">{detailMatch.location || '—'}</td></tr>
                                <tr><td class="meta-label">{$t('match.round')}</td><td class="meta-value">{detailMatch.round || '—'}</td></tr>
                                <tr><td class="meta-label">{$t('match.sourceFile')}</td><td class="meta-value source-file">{detailMatch.file_path || '—'}</td></tr>
                                <tr><td class="meta-label">{$t('match.importDate')}</td><td class="meta-value">{formatDate(detailMatch.import_date)}</td></tr>
                                <tr><td class="meta-label">{$t('match.matchId')}</td><td class="meta-value id-value">{detailMatch.id}</td></tr>
                            </tbody>
                        </table>
                    </div>
                {/if}

                <!-- Stats view -->
                {#if detailView === 'stats'}
                    <div class="stats-container">
                        {#if loadingStats}
                            <div class="loading-state">{$t('match.loadingStats')}</div>
                        {:else if !detailStats}
                            <div class="empty-state">{$t('match.noAnalysedPositions')}</div>
                        {:else}
                            {@const p1 = detailStats.player1}
                            {@const p2 = detailStats.player2}
                            {@const p1Name = detailMatch.player1_name || $t('match.player1')}
                            {@const p2Name = detailMatch.player2_name || $t('match.player2')}
                            <table class="stats-table">
                                <thead>
                                    <tr>
                                        <th class="stats-label"></th>
                                        <th class="stats-player">{p1Name}</th>
                                        <th class="stats-player">{p2Name}</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {#each MATCH_STAT_ROWS as row, i (i)}
                                        {#if row.section}
                                            <tr class="stats-section-header"><td colspan="3">{$t(row.section)}</td></tr>
                                        {:else}
                                            <tr>
                                                <td class="stats-label{row.sub ? ' sub-label' : ''}">{row.bullet ? '• ' : ''}{$t(row.label)}</td>
                                                <td class="stats-val{row.valClass ? ' ' + row.valClass : ''}{row.sub ? ' sub-val' : ''}">{row.fmt(p1)}</td>
                                                <td class="stats-val{row.valClass ? ' ' + row.valClass : ''}{row.sub ? ' sub-val' : ''}">{row.fmt(p2)}</td>
                                            </tr>
                                        {/if}
                                    {/each}
                                </tbody>
                            </table>
                        {/if}
                    </div>
                {/if}
            </div>
        {/if}
    </div>
</section>

{#if showMergePlayersModal}
    <MergePlayersModal
        onClose={() => (showMergePlayersModal = false)}
        onMerged={async () => {
            await loadMatches();
            dbMutationCounterStore.update((n) => n + 1);
        }}
    />
{/if}

<style>
    .match-panel {
        width: 100%;
        height: 100%;
        background-color: white;
        outline: none;
        user-select: none;
        -webkit-user-select: none;
    }

    .match-panel-content {
        height: 100%;
        display: flex;
        flex-direction: row;
        overflow: hidden;
    }

    /* --- Match list pane (left) --- */
    .match-list-pane {
        flex: 1;
        min-width: 0;
        height: 100%;
        overflow: hidden;
        display: flex;
        flex-direction: column;
        transition: flex 0.15s;
    }

    .match-list-toolbar {
        flex-shrink: 0;
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 8px;
        border-bottom: 1px solid #e0e0e0;
        background: #fafafa;
    }

    .toolbar-btn {
        background: none;
        border: 1px solid #ccc;
        border-radius: 3px;
        font-size: 11px;
        color: #555;
        cursor: pointer;
        padding: 2px 8px;
        line-height: 1.6;
    }

    .toolbar-btn:hover:not(:disabled) {
        background: #e3f2fd;
        border-color: #1976d2;
        color: #1565c0;
    }

    .toolbar-btn:disabled {
        opacity: 0.4;
        cursor: not-allowed;
    }

    .match-list-pane.has-detail {
        flex: 0 0 45%;
        max-width: 45%;
        border-right: 1px solid #ddd;
    }

    .match-table-container {
        flex: 1;
        overflow-y: auto;
    }

    .match-table {
        width: 100%;
        border-collapse: collapse;
        font-size: 12px;
    }

    .match-table thead {
        position: sticky;
        top: 0;
        background-color: #f5f5f5;
        z-index: 1;
    }

    .match-table th,
    .match-table td {
        padding: 4px 8px;
        text-align: left;
        border-bottom: 1px solid #e0e0e0;
    }

    .match-table th {
        font-weight: 600;
        color: #333;
        font-size: 11px;
    }

    .match-table th.sortable {
        cursor: pointer;
    }

    .match-table th.sortable:hover {
        background-color: #e8e8e8;
    }

    .sort-arrow {
        font-size: 9px;
        margin-left: 3px;
        color: #1976d2;
    }

    .match-table tbody tr {
        transition: background-color 0.1s;
    }

    .match-table tbody tr:hover {
        background-color: #f9f9f9;
    }

    .match-table tbody tr.selected {
        background-color: #e3f2fd;
    }

    .match-table tbody tr.selected:hover {
        background-color: #bbdefb;
    }

    .index-cell {
        text-align: center;
        color: #999;
    }

    .narrow-col {
        width: 1px;
        white-space: nowrap;
        padding-left: 6px;
        padding-right: 6px;
    }

    .actions-col {
        width: 52px;
        min-width: 52px;
        max-width: 52px;
        white-space: nowrap;
        text-align: center;
        padding: 0 4px;
    }

    .tournament-col {
        max-width: 120px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-size: 11px;
        color: #666;
    }

    .stat-col {
        color: #666;
        font-variant-numeric: tabular-nums;
    }

    .item-actions {
        display: inline-flex;
        gap: 2px;
        vertical-align: middle;
    }

    .icon-btn {
        background: none;
        border: none;
        cursor: pointer;
        font-size: 12px;
        color: #666;
        padding: 0 3px;
        line-height: 1;
    }

    .icon-btn:hover {
        color: #000;
    }

    .icon-btn.delete:hover {
        color: #c55;
    }

    .match-edit-input {
        width: 100%;
        padding: 1px 4px;
        border: 1px solid #1976d2;
        border-radius: 2px;
        font-size: 11px;
        box-sizing: border-box;
        outline: none;
    }

    .no-select {
        user-select: none;
        -webkit-user-select: none;
        -moz-user-select: none;
        -ms-user-select: none;
    }

    .empty-state {
        text-align: center;
        color: #999;
        padding: 24px;
        font-size: 12px;
    }

    .loading-state {
        text-align: center;
        color: #999;
        padding: 24px;
        font-size: 12px;
    }

    /* --- Detail pane (right) --- */
    .detail-pane {
        flex: 0 0 55%;
        max-width: 55%;
        height: 100%;
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }

    .detail-header {
        flex-shrink: 0;
        padding: 8px 12px 0 12px;
        border-bottom: 1px solid #e0e0e0;
        background: #fafafa;
    }

    .detail-title {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 13px;
        font-weight: 600;
        color: #222;
        margin-bottom: 4px;
    }

    .vs-label {
        color: #999;
        font-weight: 400;
        font-size: 11px;
    }

    .match-length-badge {
        display: inline-block;
        background: #e3f2fd;
        color: #1565c0;
        font-size: 10px;
        font-weight: 600;
        padding: 1px 6px;
        border-radius: 8px;
        margin-left: 4px;
    }

    .detail-meta {
        display: flex;
        flex-wrap: wrap;
        gap: 4px 12px;
        font-size: 11px;
        color: #666;
        margin-bottom: 6px;
    }

    .meta-item {
        white-space: nowrap;
    }

    .meta-tournament {
        color: #1976d2;
        font-weight: 500;
    }

    .detail-tabs {
        display: flex;
        gap: 0;
        margin: 0 -12px;
        padding: 0 12px;
    }

    .detail-tab {
        background: none;
        border: none;
        border-bottom: 2px solid transparent;
        padding: 4px 12px;
        cursor: pointer;
        font-size: 11px;
        color: #666;
        transition:
            color 0.15s,
            border-color 0.15s;
    }

    .detail-tab:hover {
        color: #333;
    }

    .detail-tab.active {
        color: #1976d2;
        border-bottom-color: #1976d2;
    }

    .enter-match-btn {
        margin-left: auto;
        color: #1976d2;
        font-weight: 600;
    }

    .enter-match-btn:hover {
        color: #0d47a1;
    }

    /* --- Transcript --- */
    .transcript-container {
        flex: 1;
        overflow-y: auto;
        padding: 0;
    }

    .game-section {
        margin-bottom: 2px;
    }

    .game-header {
        position: sticky;
        top: 0;
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 4px 12px;
        background: #f0f4f8;
        font-size: 11px;
        color: #555;
        border-bottom: 1px solid #e0e0e0;
        z-index: 1;
    }

    .game-title {
        font-weight: 600;
        color: #333;
    }

    .game-score {
        color: #777;
    }

    .game-result {
        color: #2e7d32;
        font-style: italic;
    }

    .transcript-table {
        width: 100%;
        border-collapse: collapse;
        font-size: 11px;
    }

    .transcript-table thead th {
        padding: 2px 8px;
        text-align: left;
        font-weight: 600;
        font-size: 10px;
        color: #999;
        border-bottom: 1px solid #eee;
        background: #fafafa;
    }

    .transcript-table tbody td {
        padding: 2px 8px;
        border-bottom: 1px solid #f0f0f0;
    }

    .transcript-row {
        cursor: pointer;
        transition: background-color 0.1s;
    }

    .transcript-row:hover {
        background-color: #e8f4fd;
    }

    .transcript-num {
        width: 28px;
        text-align: center;
        color: #aaa;
    }

    .transcript-player {
        width: 100px;
        max-width: 100px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .transcript-player.player1 {
        color: #333;
    }

    .transcript-player.player2 {
        color: #666;
    }

    .transcript-dice {
        width: 32px;
        text-align: center;
        font-family: monospace;
        font-size: 11px;
        color: #555;
    }

    .transcript-move {
        font-family: monospace;
        font-size: 11px;
        color: #222;
    }

    .cube-row {
        background-color: #fff8e1;
    }

    .cube-row:hover {
        background-color: #fff3cd;
    }

    .cube-action {
        color: #e65100;
        font-weight: 500;
        font-family: inherit;
    }

    /* --- Metadata view --- */
    .metadata-container {
        flex: 1;
        overflow-y: auto;
        padding: 8px 12px;
    }

    .metadata-table {
        width: 100%;
        border-collapse: collapse;
        font-size: 12px;
    }

    .metadata-table td {
        padding: 4px 8px;
        border-bottom: 1px solid #f0f0f0;
        vertical-align: top;
    }

    .meta-label {
        width: 100px;
        color: #888;
        font-size: 11px;
    }

    .meta-value {
        color: #333;
    }

    .source-file {
        font-family: monospace;
        font-size: 10px;
        color: #666;
        word-break: break-all;
    }

    .id-value {
        font-family: monospace;
        font-size: 11px;
        color: #888;
    }

    .tournament-meta-cell {
        cursor: pointer;
    }

    .tournament-cell-edit {
        position: relative;
    }

    .tournament-edit-input {
        width: 100%;
        padding: 2px 4px;
        border: 1px solid #1976d2;
        border-radius: 2px;
        font-size: 11px;
        box-sizing: border-box;
        outline: none;
    }

    .tournament-display {
        color: #666;
        font-size: 11px;
    }

    .tournament-display:hover {
        color: #1976d2;
    }

    .tournament-dropdown {
        overflow-y: auto;
        background: white;
        border: 1px solid #ccc;
        border-radius: 3px;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.18);
        z-index: 9999;
    }

    .tournament-dropdown-item {
        padding: 3px 6px;
        font-size: 11px;
        cursor: pointer;
    }

    .tournament-dropdown-item:hover {
        background: #e3f2fd;
    }

    .match-comment-display {
        cursor: pointer;
        padding: 1px 3px;
        border-radius: 3px;
        min-width: 40px;
        display: inline-block;
        color: #bbb;
        font-style: italic;
    }

    .match-comment-display:hover {
        background: #e8f0fe;
    }

    .match-comment-input {
        width: 100%;
        padding: 1px 3px;
        font-size: 11px;
        border: 1px solid #4a90d9;
        border-radius: 3px;
        outline: none;
        box-sizing: border-box;
    }

    .stats-container {
        flex: 1;
        overflow-y: auto;
        padding: 8px 12px;
    }

    .stats-table {
        width: 100%;
        border-collapse: collapse;
        font-size: 12px;
    }

    .stats-table th {
        text-align: left;
        padding: 4px 8px;
        font-size: 11px;
        color: #555;
        border-bottom: 2px solid #ddd;
        font-weight: 600;
    }

    .stats-table th.stats-player {
        text-align: right;
        min-width: 80px;
    }

    .stats-section-header td {
        background: #f0f4f8;
        padding: 5px 8px;
        font-size: 11px;
        font-weight: 600;
        color: #444;
        text-transform: uppercase;
        letter-spacing: 0.03em;
        border-top: 1px solid #ddd;
    }

    .stats-label {
        padding: 3px 8px;
        color: #555;
        font-size: 12px;
    }

    .sub-label {
        padding-left: 20px;
        color: #888;
        font-size: 11px;
    }

    .stats-val {
        text-align: right;
        padding: 3px 8px;
        font-variant-numeric: tabular-nums;
        color: #222;
        min-width: 80px;
    }

    .sub-val {
        color: #666;
        font-size: 11px;
    }

    .pr-val {
        font-weight: 600;
        color: #1565c0;
    }
</style>
