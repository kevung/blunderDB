<script>
    import { logger } from '../utils/logger.js';
    import { sortMatches, toDateInputValue, formatDate, formatDiceShort, MATCH_STAT_ROWS } from '../utils/matchTable.js';
    import { createInlineEdit } from '../utils/inlineEdit.svelte.js';
    import { onChange } from '../utils/onChange.js';
    import { onMount, onDestroy, untrack } from 'svelte';
    import { get } from 'svelte/store';
    import { SvelteSet } from 'svelte/reactivity';
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
    import EntityAutocomplete from './EntityAutocomplete.svelte';
    import PanelTable, { navigationDelta } from './panels/PanelTable.svelte';
    import { exportMatchMat } from '../services/exportService.js';
    import { enrichMatchFromFile } from '../services/importService.js';
    import { panelKeyGuard } from '../services/keyboardService.js';
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
    import { commentTextStore, isAnyModalOpen } from '../stores/uiStore';
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

    // Sorting state, cycled by the table header (asc → desc → unsorted)
    let sort = $state({ column: null, direction: 'asc' });
    let table = $state(null); // PanelTable instance: keyboard navigation and scrolling

    // Inline tournament editing (autocomplete over the known tournaments)
    const tournamentEdit = createInlineEdit({
        onSave: async (matchId, value) => {
            const name = value.trim();
            try {
                await SetMatchTournamentByName(matchId, name);
                await loadMatches();
                await loadTournaments();
                statusBarTextStore.set(name ? tMsg('match.tournamentSet', { name }) : tMsg('match.tournamentCleared'));
            } catch (error) {
                logger.error('Error setting tournament:', error);
                statusBarTextStore.set(tMsg('match.errorSettingTournament'));
            }
        }
    });

    // Inline match editing (player names, date) — one draft object per row
    const matchEdit = createInlineEdit({
        onSave: async (matchId, draft) => {
            try {
                await UpdateMatch(matchId, draft.player1, draft.player2, draft.date);
                await loadMatches();
                statusBarTextStore.set(tMsg('match.matchUpdated'));
            } catch (error) {
                logger.error('Error updating match:', error);
                statusBarTextStore.set(tMsg('match.errorUpdating'));
            }
        }
    });

    // Merge players modal
    let showMergePlayersModal = $state(false);

    // Inline match comment editing (detail pane, metadata view)
    const commentEdit = createInlineEdit({
        onSave: async (matchId, text) => {
            try {
                await UpdateMatchComment(matchId, text);
                if (detailMatch && detailMatch.id === matchId) detailMatch.comment = text;
                const m = matches.find((x) => x.id === matchId);
                if (m) m.comment = text;
                matches = matches;
                statusBarTextStore.set(tMsg('match.commentUpdated'));
            } catch (error) {
                logger.error('Error updating comment:', error);
                statusBarTextStore.set(tMsg('match.errorUpdatingComment'));
            }
        }
    });

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
    $effect(
        onChange(
            () => visible, // $derived — tracked
            (opened) => {
                if (opened && databaseLoaded) {
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
                } else if (!opened) {
                    selectedMatch = null;
                    detailMatch = null;
                    detailStats = null;
                    tournamentEdit.cancel();
                }
            },
            false
        )
    );

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
        tournamentEdit.start(match.id, match.tournament_name || match.event || '');
    }

    async function selectTournamentOption(tournament) {
        tournamentEdit.draft = tournament.name;
        await tournamentEdit.save();
    }

    function startEditMatch(match, ev) {
        ev.stopPropagation();
        matchEdit.start(match.id, {
            player1: match.player1_name || '',
            player2: match.player2_name || '',
            date: toDateInputValue(match.match_date)
        });
    }

    let sortedMatches = $derived.by(() => sortMatches(matches, sort.column, sort.direction));

    const columns = $derived([
        { key: 'index', label: '#', narrow: true },
        { key: 'date', label: $t('match.date'), sortable: true, narrow: true },
        { key: 'player1', label: $t('match.player1'), sortable: true },
        { key: 'player2', label: $t('match.player2'), sortable: true },
        { key: 'length', label: $t('match.pts'), sortable: true, narrow: true },
        { key: 'tournament', label: $t('match.tournament'), sortable: true, class: 'tournament-col' },
        { key: 'pr', label: 'PR', sortable: true, narrow: true },
        { key: 'mwc', label: 'MWC', sortable: true, narrow: true },
        { key: 'actions', actions: true }
    ]);

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

    // Group move positions by game number for transcript display. Each move
    // carries its globalIdx (its position in detailMovePositions) precomputed
    // here — the template used to recover it with detailMovePositions.indexOf(mp)
    // inside the {#each} of rows, an O(n) scan per row that made a 500-move
    // match cost ~250 000 comparisons per render (D.8, #208).
    let transcriptGames = $derived.by(() => {
        if (!detailMovePositions.length) return [];
        // eslint-disable-next-line svelte/prefer-svelte-reactivity -- local temp inside $derived
        const gameMap = new Map();
        detailMovePositions.forEach((mp, globalIdx) => {
            if (!gameMap.has(mp.game_number)) {
                gameMap.set(mp.game_number, []);
            }
            gameMap.get(mp.game_number).push({ mp, globalIdx });
        });
        const result = [];
        for (const [gameNum, moves] of gameMap) {
            // Find corresponding game info
            const gameInfo = detailGames.find((g) => g.game_number === gameNum);
            result.push({ gameNumber: gameNum, moves, gameInfo });
        }
        return result;
    });

    // Which games' transcript tables are actually mounted: collapsed games
    // render only their <summary> header, not their (potentially long) move
    // table, so a many-game match keeps most of its transcript out of the DOM
    // until the user opens it (D.8, #208). Mutated in place — the template
    // tracks this one SvelteSet instance, already reactive on its own — and
    // reseeded to just the game holding the current move whenever a
    // different match's moves load.
    const openGames = new SvelteSet();

    $effect(() => {
        const moves = detailMovePositions; // tracked dep: reseed on a new match's moves
        untrack(() => {
            openGames.clear();
            if (!moves.length) return;
            const ctx = get(matchContextStore);
            let targetIndex = null;
            if (ctx.isMatchMode && detailMatch && ctx.matchID === detailMatch.id) {
                targetIndex = ctx.currentIndex;
            } else if (lastVisitedMatch && detailMatch && lastVisitedMatch.matchID === detailMatch.id) {
                targetIndex = lastVisitedMatch.currentIndex;
            }
            const targetMove = targetIndex != null ? moves[targetIndex] : null;
            const defaultGame = targetMove ? targetMove.game_number : moves[moves.length - 1].game_number;
            openGames.add(defaultGame);
        });
    });

    function setGameOpen(gameNumber, isOpen) {
        if (isOpen) openGames.add(gameNumber);
        else openGames.delete(gameNumber);
    }

    // The game the review last stepped into. Reopening is a reaction to
    // *crossing into* a game, not a standing rule that the reviewed game is
    // open: without this, collapsing the game holding the current move
    // reopened it at once — the effect re-ran on the openGames read and put it
    // straight back. Game 1 bore the brunt of it, since a review starts there.
    let lastCrossedGame = null;

    // Keep the transcript following along while reviewing this match in MATCH
    // mode: crossing into a collapsed game reopens it (games are only ever
    // added here, never closed, so a review pass just accumulates the games
    // actually visited).
    $effect(() => {
        const ctx = $matchContextStore;
        if (!ctx.isMatchMode || !detailMatch || ctx.matchID !== detailMatch.id) {
            lastCrossedGame = null;
            return;
        }
        const move = detailMovePositions[ctx.currentIndex];
        if (!move) return;
        if (move.game_number === untrack(() => lastCrossedGame)) return; // still in the same game: leave the user's fold alone
        lastCrossedGame = move.game_number;
        untrack(() => {
            if (!openGames.has(move.game_number)) setGameOpen(move.game_number, true);
        });
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
        if (!confirm(get(t)('match.confirmDelete', { player1: match.player1_name, player2: match.player2_name }))) return;
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

    function getPlayerName(mp) {
        return mp.player_on_roll === 0 ? mp.player1_name || $t('match.player1') : mp.player2_name || $t('match.player2');
    }

    function closeMatchPanel() {
        closePanel(PANEL.MATCH);
    }

    function handleKeyDown(event) {
        if (!visible) return;
        // Don't intercept keys while the merge players modal is open
        if (showMergePlayersModal) return;

        // Let Ctrl/Meta combos, Space, '?' and typing in an editable field pass
        // through to the global handler — see keyboardService.panelKeyGuard.
        if (panelKeyGuard(event)) return;

        // Stop all keyboard events from propagating to global handlers
        event.stopPropagation();

        if (event.key === 'Escape') {
            if (matchEdit.editingId !== null) {
                matchEdit.cancel();
                event.preventDefault();
            } else if (tournamentEdit.editingId !== null) {
                tournamentEdit.cancel();
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

        // j/k walk the list; with no selection, j lands on the first row.
        const delta = navigationDelta(event);
        if (delta !== 0 && sortedMatches.length > 0) {
            event.preventDefault();
            table?.navigate(delta);
            return;
        }

        if (selectedMatch && sortedMatches.length > 0) {
            if (event.key === 'Enter') {
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
        // This listener lives on `document` for the whole life of the panel, so it also sees
        // clicks that have nothing to do with it — including every click inside a modal. It
        // used to blur whatever was focused, which made text fields in the export and
        // settings dialogs impossible to click into: the field took focus on mousedown and
        // lost it on the click that followed. Tab still worked, which is what made the
        // symptom so confusing.
        if (get(isAnyModalOpen)) return;
        // Don't interfere while the merge players modal is open
        if (showMergePlayersModal) return;
        // Close tournament dropdown if clicking outside
        if (tournamentEdit.editingId !== null && !event.target.closest('.tournament-cell-edit')) {
            tournamentEdit.cancel();
        }
        // Cancel match edit if clicking outside the editing row
        if (matchEdit.editingId !== null && !event.target.closest('.match-editing-row')) {
            matchEdit.cancel();
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
                if (selectedMatch) table?.scrollToSelected('center');
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

<section class="match-panel" aria-label={$t('match.ariaLabel')} id="matchPanel" tabindex="-1">
    <div class="match-panel-content">
        <!-- Match list (left pane) -->
        <div class="match-list-pane" class:has-detail={detailMatch}>
            <div class="match-list-toolbar">
                <button class="toolbar-btn" onclick={() => (showMergePlayersModal = true)} title={$t('match.mergePlayersTitle')} disabled={matches.length === 0}>⇢ {$t('match.mergePlayers')}</button>
            </div>
            <PanelTable
                bind:this={table}
                rows={sortedMatches}
                {columns}
                bind:sort
                sortOptions={{ tristate: true }}
                selectedKey={selectedMatch?.id}
                rowClass={(match) => (matchEdit.isEditing(match.id) ? 'match-editing-row' : '')}
                onSelect={(match) => {
                    if (!matchEdit.isEditing(match.id)) selectMatch(match);
                }}
                onActivate={(match) => {
                    if (!matchEdit.isEditing(match.id)) handleDoubleClick(match);
                }}
                emptyText={matches.length === 0 ? $t('match.noMatchesImported') : ''}
            >
                {#snippet cells(match, index)}
                    {#if matchEdit.isEditing(match.id)}
                        <td class="index-cell narrow-col no-select">{index + 1}</td>
                        <td class="narrow-col">
                            <input type="date" class="match-edit-input" bind:value={matchEdit.draft.date} onkeydown={matchEdit.onKeyDown} />
                        </td>
                        <td>
                            <input type="text" class="match-edit-input" bind:value={matchEdit.draft.player1} onkeydown={matchEdit.onKeyDown} placeholder={$t('match.player1')} />
                        </td>
                        <td>
                            <input type="text" class="match-edit-input" bind:value={matchEdit.draft.player2} onkeydown={matchEdit.onKeyDown} placeholder={$t('match.player2')} />
                        </td>
                        <td class="narrow-col no-select">{match.match_length}</td>
                        <td class="tournament-col no-select">{match.tournament_name || match.event || ''}</td>
                        <td class="narrow-col no-select">{match.pr > 0 ? match.pr.toFixed(2) : ''}{match.pr2 > 0 ? ' / ' + match.pr2.toFixed(2) : ''}</td>
                        <td class="narrow-col no-select">{match.mwc_loss > 0 ? (match.mwc_loss * 100).toFixed(2) + '%' : ''}</td>
                        <td class="actions-col no-select">
                            <span class="item-actions editing-actions">
                                <button
                                    class="icon-btn"
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        matchEdit.save();
                                    }}
                                    title={$t('common.save')}>✓</button
                                >
                                <button
                                    class="icon-btn"
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        matchEdit.cancel();
                                    }}
                                    title={$t('common.cancel')}>✕</button
                                >
                            </span>
                        </td>
                    {:else}
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
                            {#if tournamentEdit.isEditing(match.id)}
                                <div class="tournament-cell-edit">
                                    <EntityAutocomplete
                                        bind:value={tournamentEdit.draft}
                                        items={tournaments}
                                        autofocus
                                        blurDelay={200}
                                        placeholder={$t('match.tournamentNamePlaceholder')}
                                        onSelect={selectTournamentOption}
                                        onSubmit={() => tournamentEdit.save()}
                                        onCancel={tournamentEdit.cancel}
                                        onDismiss={tournamentEdit.cancel}
                                    />
                                </div>
                            {:else}
                                <span class="tournament-display" title={$t('match.clickToAssignTournament')}>{match.tournament_name || match.event || ''}</span>
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
                                <!-- Enrichir depuis un fichier (#262). Rien de
                                     nouveau sous ce bouton : réimporter le même
                                     match dans un autre format l'enrichit déjà.
                                     Ce que le bouton apporte, c'est qu'on le
                                     trouve. -->
                                <button
                                    class="icon-btn"
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        enrichMatchFromFile();
                                    }}
                                    title={$t('match.enrichFromFile')}>⊕</button
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
                    {/if}
                {/snippet}
            </PanelTable>
        </div>

        <!-- Detail pane (right side). It exists only while a match is
             selected: an empty pane holding nothing but "select a match" took
             55% of the panel for a sentence, so the list now spans the full
             width until there is something to show. The first click therefore
             narrows the list from 100% to 45% again; that used to move the
             clicked row out from under the cursor before the second click of a
             double-click (#201), which is why the change of width carries no
             transition — and why Enter on the selection and the pane's
             "Review" button are the two other ways to open a match. -->
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
                                {@const isOpen = openGames.has(game.gameNumber)}
                                <details class="game-section" open={isOpen} ontoggle={(e) => setGameOpen(game.gameNumber, e.currentTarget.open)}>
                                    <summary class="game-header">
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
                                    </summary>
                                    {#if isOpen}
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
                                                {#each game.moves as { mp, globalIdx }, mi (globalIdx)}
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
                                    {/if}
                                </details>
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
                                        {#if commentEdit.isEditing(detailMatch.id)}
                                            <input type="text" class="match-comment-input" bind:value={commentEdit.draft} onkeydown={commentEdit.onKeyDown} onblur={commentEdit.onBlur} />
                                        {:else}
                                            <!-- svelte-ignore a11y_click_events_have_key_events -->
                                            <!-- svelte-ignore a11y_no_static_element_interactions -->
                                            <span class="match-comment-display" onclick={() => commentEdit.start(detailMatch.id, detailMatch.comment || '')} title={$t('match.clickToAddComment')}>
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
                                        {#if tournamentEdit.isEditing(detailMatch.id)}
                                            <div class="tournament-cell-edit">
                                                <EntityAutocomplete
                                                    bind:value={tournamentEdit.draft}
                                                    items={tournaments}
                                                    autofocus
                                                    blurDelay={200}
                                                    placeholder={$t('match.tournamentNamePlaceholder')}
                                                    onSelect={selectTournamentOption}
                                                    onSubmit={() => tournamentEdit.save()}
                                                    onCancel={tournamentEdit.cancel}
                                                    onDismiss={tournamentEdit.cancel}
                                                />
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
    /* Fixed split with the detail pane (see the template's note): the list
       never changes width, so a row never moves under the cursor. */
    .match-list-pane {
        flex: 1 1 100%;
        max-width: 100%;
        min-width: 0;
        height: 100%;
        overflow: hidden;
        display: flex;
        flex-direction: column;
    }

    /* No transition on purpose: the row must reach its final width before the
       second click of a double-click, not travel under the cursor (see the
       template comment above the pane). NB: no `#nnn` issue reference inside a
       <style> block — colorTokens.sync.test.js reads it as a hex colour. */
    .match-list-pane.has-detail {
        flex: 0 0 45%;
        max-width: 45%;
        border-right: 1px solid #ddd;
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
        border: 1px solid var(--color-border);
        border-radius: 3px;
        font-size: var(--font-size-small);
        color: #555;
        cursor: pointer;
        padding: 2px 8px;
        line-height: 1.6;
    }

    .toolbar-btn:hover:not(:disabled) {
        background: #e3f2fd;
        border-color: var(--color-primary);
        color: #1565c0;
    }

    .toolbar-btn:disabled {
        opacity: 0.4;
        cursor: not-allowed;
    }

    /* Header and cells of the tournament column alike (the header is PanelTable's). */
    .match-panel :global(.tournament-col) {
        max-width: 120px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .match-edit-input {
        width: 100%;
        padding: 1px 4px;
        border: 1px solid var(--color-primary);
        border-radius: 2px;
        font-size: var(--font-size-small);
        box-sizing: border-box;
        outline: none;
    }

    /* Detail pane placeholders (the list's own empty state is PanelTable's). */
    .empty-state,
    .loading-state {
        text-align: center;
        color: var(--color-text-muted);
        padding: 24px;
        font-size: var(--font-size-base);
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
        font-size: var(--font-size-base);
        font-weight: 600;
        color: #222;
        margin-bottom: 4px;
    }

    .vs-label {
        color: var(--color-text-muted);
        font-weight: 400;
        font-size: var(--font-size-small);
    }

    .match-length-badge {
        display: inline-block;
        background: #e3f2fd;
        color: #1565c0;
        font-size: var(--font-size-small);
        font-weight: 600;
        padding: 1px 6px;
        border-radius: 8px;
        margin-left: 4px;
    }

    .detail-meta {
        display: flex;
        flex-wrap: wrap;
        gap: 4px 12px;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        margin-bottom: 6px;
    }

    .meta-item {
        white-space: nowrap;
    }

    .meta-tournament {
        color: var(--color-primary);
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
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        transition:
            color 0.15s,
            border-color 0.15s;
    }

    .detail-tab:hover {
        color: var(--color-text);
    }

    .detail-tab.active {
        color: var(--color-primary);
        border-bottom-color: var(--color-primary);
    }

    .enter-match-btn {
        margin-left: auto;
        color: var(--color-primary);
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
        font-size: var(--font-size-small);
        color: #555;
        border-bottom: 1px solid #e0e0e0;
        z-index: 1;
        cursor: pointer;
        list-style: none;
    }

    /* .game-header is a <summary>: a game's move table is only mounted while
       its <details> is open (D.8, perf ticket 208), so collapsed games cost
       one row of DOM instead of their whole transcript. Replace the native
       marker with a small disclosure triangle that flips with [open]. */
    .game-header::-webkit-details-marker {
        display: none;
    }

    .game-header::before {
        content: '▸';
        color: var(--color-text-muted);
    }

    .game-section[open] > .game-header::before {
        content: '▾';
    }

    .game-title {
        font-weight: 600;
        color: var(--color-text);
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
        font-size: var(--font-size-small);
    }

    .transcript-table thead th {
        padding: 2px 8px;
        text-align: left;
        font-weight: 600;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
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
        color: var(--color-text);
    }

    .transcript-player.player2 {
        color: var(--color-text-muted);
    }

    .transcript-dice {
        width: 32px;
        text-align: center;
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
        color: #555;
    }

    .transcript-move {
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
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
        font-size: var(--font-size-base);
    }

    .metadata-table td {
        padding: 4px 8px;
        border-bottom: 1px solid #f0f0f0;
        vertical-align: top;
    }

    .meta-label {
        width: 100px;
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .meta-value {
        color: var(--color-text);
    }

    .source-file {
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        word-break: break-all;
    }

    .id-value {
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .tournament-meta-cell {
        cursor: pointer;
    }

    .tournament-cell-edit {
        position: relative;
    }

    .tournament-display {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .tournament-display:hover {
        color: var(--color-primary);
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
        font-size: var(--font-size-small);
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
        font-size: var(--font-size-base);
    }

    .stats-table th {
        text-align: left;
        padding: 4px 8px;
        font-size: var(--font-size-small);
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
        font-size: var(--font-size-small);
        font-weight: 600;
        color: #444;
        text-transform: uppercase;
        letter-spacing: 0.03em;
        border-top: 1px solid #ddd;
    }

    .stats-label {
        padding: 3px 8px;
        color: #555;
        font-size: var(--font-size-base);
    }

    .sub-label {
        padding-left: 20px;
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .stats-val {
        text-align: right;
        padding: 3px 8px;
        font-variant-numeric: tabular-nums;
        color: #222;
        min-width: 80px;
    }

    .sub-val {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .pr-val {
        font-weight: 600;
        color: #1565c0;
    }
</style>
