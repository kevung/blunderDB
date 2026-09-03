<script>
    import { logger } from '../utils/logger.js';
    import { createInlineEdit } from '../utils/inlineEdit.svelte.js';
    import { autofocus } from '../utils/autofocus.js';
    import { onMount, onDestroy } from 'svelte';
    import { createReorder } from '../utils/reorder.js';
    import EntityAutocomplete from './EntityAutocomplete.svelte';
    import PanelTable, { navigationDelta, stepSelection } from './panels/PanelTable.svelte';
    import {
        GetAllTournaments,
        CreateTournament,
        DeleteTournament,
        UpdateTournament,
        GetTournamentMatches,
        RemoveMatchFromTournament,
        GetAllMatches,
        AddMatchToTournament,
        GetMatchMovePositions,
        LoadAnalysis,
        SwapMatchPlayers,
        SaveLastVisitedPosition,
        UpdateMatchComment,
        UpdateTournamentComment,
        ReorderTournamentMatches
    } from '../../wailsjs/go/database/Database.js';
    import { openPanels, PANEL, closePanel, statusBarTextStore, statusBarModeStore } from '../stores/uiStore';
    import { tournamentsStore, selectedTournamentStore, tournamentMatchesStore } from '../stores/tournamentStore';
    import { positionStore, matchContextStore, lastVisitedMatchStore } from '../stores/positionStore';
    import { analysisStore, selectedMoveStore } from '../stores/analysisStore';
    import { commentTextStore } from '../stores/uiStore';
    import { databaseLoadedStore } from '../stores/databaseStore';
    import { panelKeyGuard } from '../services/keyboardService.js';
    import { t, tMsg } from '../i18n';
    import { get } from 'svelte/store';

    // Read-only mirrors of stores
    let tournaments = $derived($tournamentsStore || []);
    let selectedTournament = $derived($selectedTournamentStore);
    let tournamentMatches = $derived($tournamentMatchesStore || []);
    let visible = $derived($openPanels.has(PANEL.TOURNAMENT));

    // Sorting state, cycled by the table header (asc → desc → unsorted)
    let sort = $state({ column: null, direction: 'asc' });
    let listTable = $state(null); // PanelTable of the tournament list, mounted while none is selected
    const sortedTournaments = $derived(sortTournaments(tournaments, sort));

    const tournamentColumns = $derived([
        { key: 'name', label: $t('tournament.name'), sortable: true },
        { key: 'matches', label: $t('tournament.matches'), sortable: true, narrow: true },
        { key: 'date', label: $t('tournament.date'), sortable: true, narrow: true },
        { key: 'location', label: $t('tournament.location'), sortable: true },
        { key: 'pr', label: 'PR', sortable: true, narrow: true },
        { key: 'mwc', label: 'MWC', sortable: true, narrow: true },
        { key: 'actions', actions: true }
    ]);

    const matchColumns = $derived([
        { key: 'index', label: '#', narrow: true },
        { key: 'player1', label: $t('tournament.player1') },
        { key: 'player2', label: $t('tournament.player2') },
        { key: 'length', label: $t('tournament.pts'), narrow: true },
        { key: 'pr', label: 'PR', narrow: true },
        { key: 'mwc', label: 'MWC', narrow: true },
        { key: 'comment', label: $t('tournament.comment'), class: 'comment-col' },
        { key: 'actions', actions: true }
    ]);

    // New tournament form
    let newTournamentName = $state('');
    let newTournamentDate = $state('');
    let newTournamentLocation = $state('');

    // Edit tournament (name, date, location in one row)
    const tournamentEdit = createInlineEdit({
        onSave: async (id, draft) => {
            const name = draft.name.trim();
            if (!name) return false; // a tournament keeps its name until a new one is typed
            try {
                await UpdateTournament(id, name, draft.date, draft.location.trim());
                await loadTournaments();
                if (selectedTournament && selectedTournament.id === id) {
                    selectedTournamentStore.set({ ...selectedTournament, name, date: draft.date, location: draft.location.trim() });
                }
            } catch (error) {
                logger.error('Error updating tournament:', error);
            }
        }
    });

    // Add match to tournament: only matches not yet assigned to one are offered
    let addMatchSearch = $state('');
    let allMatches = $state([]);
    const availableMatches = $derived(allMatches.filter((m) => !m.tournament_id));

    function matchMatchesQuery(m, query) {
        const q = query.trim().toLowerCase();
        if (!q) return true;
        return (m.player1_name || '').toLowerCase().includes(q) || (m.player2_name || '').toLowerCase().includes(q) || String(m.match_length || '').includes(q);
    }

    // Match comment editing (one cell of the matches table)
    const matchCommentEdit = createInlineEdit({
        onSave: async (matchId, comment) => {
            try {
                await UpdateMatchComment(matchId, comment);
                tournamentMatchesStore.set(tournamentMatches.map((m) => (m.id === matchId ? { ...m, comment } : m)));
            } catch (error) {
                logger.error('Error saving match comment:', error);
            }
        }
    });

    // Tournament comment editing (detail header)
    const tournamentCommentEdit = createInlineEdit({
        onSave: async (tournamentId, comment) => {
            if (!selectedTournament || selectedTournament.id !== tournamentId) return;
            try {
                await UpdateTournamentComment(tournamentId, comment);
                selectedTournamentStore.set({ ...selectedTournament, comment });
            } catch (error) {
                logger.error('Error saving tournament comment:', error);
            }
        }
    });

    // Load/unload data when the panel is shown or hidden
    let _prevVisible = false;
    $effect(() => {
        const v = $openPanels.has(PANEL.TOURNAMENT);
        if (v !== _prevVisible) {
            if (v) {
                if ($databaseLoadedStore) loadTournaments();
                selectedTournamentStore.set(null);
                tournamentMatchesStore.set([]);
            } else {
                selectedTournamentStore.set(null);
                tournamentMatchesStore.set([]);
                tournamentEdit.cancel();
                addMatchSearch = '';
            }
            _prevVisible = v;
        }
    });

    async function loadTournaments() {
        try {
            const loaded = await GetAllTournaments();
            tournamentsStore.set(loaded || []);
        } catch (error) {
            logger.error('Error loading tournaments:', error);
        }
    }

    async function loadAllMatches() {
        try {
            allMatches = (await GetAllMatches()) || [];
        } catch (error) {
            logger.error('Error loading matches:', error);
            allMatches = [];
        }
    }

    function sortTournaments(list, { column: sortBy, direction: sortOrder }) {
        if (!sortBy) return list;
        return [...list].sort((a, b) => {
            let valA, valB;
            if (sortBy === 'name') {
                valA = (a.name || '').toLowerCase();
                valB = (b.name || '').toLowerCase();
            } else if (sortBy === 'date') {
                valA = a.date || '';
                valB = b.date || '';
            } else if (sortBy === 'location') {
                valA = (a.location || '').toLowerCase();
                valB = (b.location || '').toLowerCase();
            } else if (sortBy === 'matches') {
                valA = a.matchCount || 0;
                valB = b.matchCount || 0;
            } else if (sortBy === 'pr') {
                valA = a.pr || 0;
                valB = b.pr || 0;
            } else if (sortBy === 'mwc') {
                valA = a.mwc_loss || 0;
                valB = b.mwc_loss || 0;
            } else {
                return 0;
            }
            const cmp = typeof valA === 'number' ? valA - valB : valA.localeCompare(valB);
            return sortOrder === 'asc' ? cmp : -cmp;
        });
    }

    async function selectTournament(tournament) {
        if (selectedTournament && selectedTournament.id === tournament.id) {
            selectedTournamentStore.set(null);
            tournamentMatchesStore.set([]);
            addMatchSearch = '';
            return;
        }
        selectedTournamentStore.set(tournament);
        addMatchSearch = '';
        await loadAllMatches();
        try {
            const matches = await GetTournamentMatches(tournament.id);
            tournamentMatchesStore.set(matches || []);
        } catch (error) {
            logger.error('Error loading tournament matches:', error);
        }
    }

    async function createTournament() {
        if (!newTournamentName.trim()) return;
        try {
            await CreateTournament(newTournamentName.trim(), newTournamentDate, newTournamentLocation.trim());
            await loadTournaments();
            statusBarTextStore.set(tMsg('tournament.created', { name: newTournamentName.trim() }));
            newTournamentName = '';
            newTournamentDate = '';
            newTournamentLocation = '';
        } catch (error) {
            logger.error('Error creating tournament:', error);
            statusBarTextStore.set(tMsg('tournament.errorCreating'));
        }
    }

    async function deleteTournamentEntry(tournament, event) {
        event.stopPropagation();
        if (!confirm(get(t)('tournament.confirmDelete', { name: tournament.name }))) return;
        try {
            await DeleteTournament(tournament.id);
            await loadTournaments();
            if (selectedTournament && selectedTournament.id === tournament.id) {
                selectedTournamentStore.set(null);
                tournamentMatchesStore.set([]);
            }
            statusBarTextStore.set(tMsg('tournament.deleted'));
        } catch (error) {
            logger.error('Error deleting tournament:', error);
        }
    }

    function startEdit(tournament, event) {
        event.stopPropagation();
        tournamentEdit.start(tournament.id, { name: tournament.name, date: tournament.date || '', location: tournament.location || '' });
    }

    async function addMatchToTournament(matchId) {
        if (!selectedTournament) return;
        try {
            await AddMatchToTournament(selectedTournament.id, matchId);
            const matches = await GetTournamentMatches(selectedTournament.id);
            tournamentMatchesStore.set(matches || []);
            await loadTournaments();
            await loadAllMatches();
        } catch (error) {
            logger.error('Error adding match:', error);
        }
    }

    async function removeMatch(matchId) {
        if (!selectedTournament) return;
        try {
            await RemoveMatchFromTournament(matchId);
            const matches = await GetTournamentMatches(selectedTournament.id);
            tournamentMatchesStore.set(matches || []);
            await loadTournaments();
        } catch (error) {
            logger.error('Error removing match:', error);
        }
    }

    // Match reordering: ▲/▼ buttons and pointer drag share one helper.
    const matchOrder = createReorder({
        get: () => (selectedTournament ? tournamentMatches : null),
        set: (next) => tournamentMatchesStore.set(next),
        persist: (next) =>
            ReorderTournamentMatches(
                selectedTournament.id,
                next.map((m) => m.id)
            ),
        label: 'matches'
    });

    async function swapMatchPlayersInTournament(match) {
        try {
            await SwapMatchPlayers(match.id);
            // Reload tournament matches
            if (selectedTournament) {
                const matches = await GetTournamentMatches(selectedTournament.id);
                tournamentMatchesStore.set(matches || []);
            }

            // If we are currently viewing this match in match mode, update context
            const currentContext = get(matchContextStore);
            if (currentContext && currentContext.isMatchMode && currentContext.matchID === match.id) {
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
                    positionStore.set(movePositions[currentIndex].position);
                }
            }

            statusBarTextStore.set(tMsg('tournament.swappedPlayers'));
        } catch (error) {
            logger.error('Error swapping match players:', error);
            statusBarTextStore.set(tMsg('tournament.errorSwapping'));
        }
    }

    async function openMatch(match) {
        try {
            const movePositions = await GetMatchMovePositions(match.id);
            if (!movePositions || movePositions.length === 0) {
                statusBarTextStore.set(tMsg('tournament.noMovesFound'));
                return;
            }

            let startIndex = 0;
            const lastVisitedMatch = get(lastVisitedMatchStore);
            // First check in-memory store for same session, then check DB-persisted value
            if (lastVisitedMatch && lastVisitedMatch.matchID === match.id) {
                if (lastVisitedMatch.currentIndex >= 0 && lastVisitedMatch.currentIndex < movePositions.length) {
                    startIndex = lastVisitedMatch.currentIndex;
                }
            } else if (match.last_visited_position >= 0 && match.last_visited_position < movePositions.length) {
                startIndex = match.last_visited_position;
            }

            matchContextStore.set({
                isMatchMode: true,
                matchID: match.id,
                movePositions: movePositions,
                currentIndex: startIndex,
                player1Name: match.player1_name,
                player2Name: match.player2_name
            });

            const startMovePos = movePositions[startIndex];
            positionStore.set(startMovePos.position);

            let analysis = null;
            try {
                analysis = await LoadAnalysis(startMovePos.position.id);
            } catch (_e) {
                /* ignored */
            }

            const currentPlayedMove = startMovePos.checker_move || '';
            const currentPlayedCubeAction = startMovePos.cube_action || '';

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
            // Player names are shown in the match-info header bar above the
            // board (MatchInfoBar.svelte); no longer echoed in the status bar.
            lastVisitedMatchStore.set({
                matchID: match.id,
                currentIndex: startIndex,
                gameNumber: startMovePos.game_number
            });
            // Persist last visited position to database
            SaveLastVisitedPosition(match.id, startIndex).catch((e) => {
                logger.error('Error persisting last visited position:', e);
            });
            closeTournamentPanel();
        } catch (error) {
            logger.error('Error opening match:', error);
            statusBarTextStore.set(tMsg('tournament.errorOpening'));
        }
    }

    function closeTournamentPanel() {
        closePanel(PANEL.TOURNAMENT);
    }

    function handleKeyDown(event) {
        if (!visible) return;

        // Let Ctrl/Meta combos, Space, '?' and typing in an editable field pass
        // through to the global handler — see keyboardService.panelKeyGuard.
        if (panelKeyGuard(event)) return;

        // Block all other non-Ctrl keys from propagating (prevents position browsing)
        event.stopPropagation();

        if (event.key === 'Escape') {
            event.preventDefault();
            if (tournamentEdit.editingId !== null) {
                tournamentEdit.cancel();
            } else if (addMatchSearch) {
                addMatchSearch = '';
            } else if (selectedTournament) {
                selectedTournamentStore.set(null);
                tournamentMatchesStore.set([]);
            } else {
                closeTournamentPanel();
            }
            return;
        }

        // j/k walk the tournament list, also from the detail view (where the
        // list table is not mounted, hence the module-level helper).
        const delta = navigationDelta(event);
        if (delta !== 0 && sortedTournaments.length > 0) {
            event.preventDefault();
            const next = stepSelection(sortedTournaments, (t) => t.id, selectedTournament?.id, delta);
            if (next) {
                selectTournament(next);
                listTable?.scrollToRow(next);
            }
        }
    }

    $effect(() => {
        if (visible) {
            setTimeout(() => {
                const panel = document.getElementById('tournamentPanel');
                if (panel) panel.focus();
            }, 100);
        }
    });
    onMount(() => {
        document.addEventListener('keydown', handleKeyDown);
    });

    onDestroy(() => {
        document.removeEventListener('keydown', handleKeyDown);
    });
</script>

<section class="tournament-panel" id="tournamentPanel" tabindex="-1" aria-label={$t('tournament.title')}>
    <div class="panel-content">
        {#if !selectedTournament}
            <!-- Tournaments list -->
            <div class="tournament-list-pane">
                <PanelTable
                    bind:this={listTable}
                    rows={sortedTournaments}
                    columns={tournamentColumns}
                    bind:sort
                    sortOptions={{ tristate: true }}
                    rowClass={(tournament) => (tournamentEdit.isEditing(tournament.id) ? 'editing-row' : '')}
                    onSelect={(tournament) => {
                        if (!tournamentEdit.isEditing(tournament.id)) selectTournament(tournament);
                    }}
                    emptyText={$t('tournament.noTournaments')}
                >
                    {#snippet cells(tournament)}
                        {#if tournamentEdit.isEditing(tournament.id)}
                            <td><input class="edit-input" type="text" bind:value={tournamentEdit.draft.name} onkeydown={tournamentEdit.onKeyDown} use:autofocus /></td>
                            <td class="narrow-col"></td>
                            <td class="narrow-col"><input class="edit-input" type="date" bind:value={tournamentEdit.draft.date} onkeydown={tournamentEdit.onKeyDown} /></td>
                            <td><input class="edit-input" type="text" bind:value={tournamentEdit.draft.location} placeholder={$t('tournament.location')} onkeydown={tournamentEdit.onKeyDown} /></td>
                            <td class="narrow-col no-select"></td>
                            <td class="narrow-col no-select"></td>
                            <td class="actions-col no-select">
                                <span class="item-actions editing-actions">
                                    <button class="icon-btn" onclick={() => tournamentEdit.save()} title={$t('common.save')}>✓</button>
                                    <button class="icon-btn" onclick={() => tournamentEdit.cancel()} title={$t('common.cancel')}>✕</button>
                                </span>
                            </td>
                        {:else}
                            <td class="no-select">{tournament.name}</td>
                            <td class="narrow-col no-select count-cell">{tournament.matchCount || 0}</td>
                            <td class="narrow-col no-select">{tournament.date || ''}</td>
                            <td class="no-select">{tournament.location || ''}</td>
                            <td class="narrow-col no-select stat-col" title={tournament.pr > 0 && tournament.ref_player ? $t('tournament.prReferencePlayer', { player: tournament.ref_player }) : ''}
                                >{tournament.pr > 0 ? tournament.pr.toFixed(2) : '—'}</td
                            >
                            <td class="narrow-col no-select stat-col">{tournament.mwc_loss > 0 ? (tournament.mwc_loss * 100).toFixed(2) + '%' : '—'}</td>
                            <td class="actions-col no-select">
                                <span class="item-actions">
                                    <button
                                        class="icon-btn"
                                        onclick={(e) => {
                                            e.stopPropagation();
                                            ((e) => startEdit(tournament, e))(e);
                                        }}
                                        title={$t('common.edit')}>✎</button
                                    >
                                    <button
                                        class="icon-btn delete"
                                        onclick={(e) => {
                                            e.stopPropagation();
                                            ((e) => deleteTournamentEntry(tournament, e))(e);
                                        }}
                                        title={$t('common.delete')}>×</button
                                    >
                                </span>
                            </td>
                        {/if}
                    {/snippet}
                </PanelTable>
                <div class="add-area">
                    <input
                        class="add-input name"
                        type="text"
                        bind:value={newTournamentName}
                        placeholder={$t('tournament.newTournamentPlaceholder')}
                        onkeydown={(e) => {
                            if (e.key === 'Enter') {
                                e.stopPropagation();
                                createTournament();
                            }
                            if (e.key === 'Escape') {
                                e.stopPropagation();
                                e.currentTarget.blur();
                            }
                        }}
                    />
                    <input
                        class="add-input date"
                        type="date"
                        bind:value={newTournamentDate}
                        onkeydown={(e) => {
                            if (e.key === 'Escape') {
                                e.stopPropagation();
                                e.currentTarget.blur();
                            }
                        }}
                    />
                    <input
                        class="add-input loc"
                        type="text"
                        bind:value={newTournamentLocation}
                        placeholder={$t('tournament.location')}
                        onkeydown={(e) => {
                            if (e.key === 'Enter') {
                                e.stopPropagation();
                                createTournament();
                            }
                            if (e.key === 'Escape') {
                                e.stopPropagation();
                                e.currentTarget.blur();
                            }
                        }}
                    />
                </div>
            </div>
        {:else}
            <!-- Matches for selected tournament -->
            <div class="tournament-list-pane">
                <PanelTable rows={tournamentMatches} columns={matchColumns} onActivate={openMatch} onReorder={matchOrder.reorder} emptyText={$t('tournament.noMatches')}>
                    {#snippet header()}
                        <button
                            class="back-btn"
                            onclick={() => {
                                selectedTournamentStore.set(null);
                                tournamentMatchesStore.set([]);
                                addMatchSearch = '';
                                tournamentCommentEdit.cancel();
                            }}
                            title={$t('tournament.backToTournaments')}>←</button
                        >
                        <span class="header-name" title={selectedTournament.name}>{selectedTournament.name}</span>
                        {#if selectedTournament.date || selectedTournament.location}
                            <span class="header-meta">
                                {#if selectedTournament.date}{selectedTournament.date}{/if}
                                {#if selectedTournament.date && selectedTournament.location}
                                    ·
                                {/if}
                                {#if selectedTournament.location}{selectedTournament.location}{/if}
                            </span>
                        {/if}
                        <button
                            class="icon-btn edit-header-btn"
                            onclick={(e) => {
                                e.stopPropagation();
                                // Return to the tournament list first: the inline-editable
                                // row is only rendered in the list view ({#if !selectedTournament}).
                                // Clicking edit here used to set editingTournament while the
                                // detail view stayed visible, so nothing looked editable until
                                // the user manually pressed the back arrow.
                                const t = selectedTournament;
                                selectedTournamentStore.set(null);
                                tournamentMatchesStore.set([]);
                                addMatchSearch = '';
                                tournamentCommentEdit.cancel();
                                startEdit(t, e);
                            }}
                            title={$t('common.edit')}>✎</button
                        >
                        <span class="header-spacer"></span>
                        {#if tournamentCommentEdit.isEditing(selectedTournament.id)}
                            <input
                                class="tournament-comment-inline"
                                type="text"
                                bind:value={tournamentCommentEdit.draft}
                                onkeydown={tournamentCommentEdit.onKeyDown}
                                onblur={tournamentCommentEdit.onBlur}
                                placeholder={$t('tournament.notesPlaceholder')}
                                use:autofocus
                            />
                        {:else}
                            <span
                                class="tournament-comment-text"
                                class:has-comment={selectedTournament.comment}
                                onclick={(e) => {
                                    e.stopPropagation();
                                    tournamentCommentEdit.start(selectedTournament.id, selectedTournament.comment || '');
                                }}
                                title={selectedTournament.comment || $t('tournament.clickToAddNotes')}
                            >
                                {selectedTournament.comment || $t('tournament.notesPlaceholder')}
                            </span>
                        {/if}
                    {/snippet}
                    {#snippet cells(match, index)}
                        <td class="index-cell narrow-col no-select">{index + 1}</td>
                        <td class="no-select">{match.player1_name}</td>
                        <td class="no-select">{match.player2_name}</td>
                        <td class="narrow-col no-select">{match.match_length}</td>
                        <td class="narrow-col no-select stat-col">{match.pr > 0 ? match.pr.toFixed(2) : '—'}</td>
                        <td class="narrow-col no-select stat-col">{match.mwc_loss > 0 ? (match.mwc_loss * 100).toFixed(2) + '%' : '—'}</td>
                        <td class="comment-col no-select">
                            {#if matchCommentEdit.isEditing(match.id)}
                                <input class="edit-input" type="text" bind:value={matchCommentEdit.draft} onkeydown={matchCommentEdit.onKeyDown} onblur={matchCommentEdit.onBlur} use:autofocus />
                            {:else}
                                <span
                                    class="comment-text"
                                    class:has-comment={match.comment}
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        matchCommentEdit.start(match.id, match.comment || '');
                                    }}
                                    title={match.comment || $t('tournament.clickToAddComment')}
                                >
                                    {match.comment || ''}
                                </span>
                            {/if}
                        </td>
                        <td class="actions-col no-select">
                            <span class="item-actions">
                                <button
                                    class="icon-btn"
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        matchOrder.moveUp(index);
                                    }}
                                    disabled={index === 0}
                                    title={$t('tournament.moveUp')}>▲</button
                                >
                                <button
                                    class="icon-btn"
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        matchOrder.moveDown(index);
                                    }}
                                    disabled={index === tournamentMatches.length - 1}
                                    title={$t('tournament.moveDown')}>▼</button
                                >
                                <button
                                    class="icon-btn"
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        (() => swapMatchPlayersInTournament(match))();
                                    }}
                                    title={$t('tournament.swap')}>⇄</button
                                >
                                <button
                                    class="icon-btn delete"
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        (() => removeMatch(match.id))();
                                    }}
                                    title={$t('tournament.remove')}>×</button
                                >
                            </span>
                        </td>
                    {/snippet}
                </PanelTable>
                <div class="add-area">
                    <div class="add-match-wrap">
                        <EntityAutocomplete
                            bind:value={addMatchSearch}
                            items={availableMatches}
                            key={(m) => m.id}
                            label={(m) => `${m.player1_name} ${$t('tournament.vs')} ${m.player2_name}`}
                            filter={matchMatchesQuery}
                            variant="field"
                            placement="above"
                            maxHeight={90}
                            fillOnSelect={false}
                            placeholder={$t('tournament.addMatchPlaceholder')}
                            onFocus={loadAllMatches}
                            onSelect={(m) => addMatchToTournament(m.id)}
                            onCancel={() => (addMatchSearch = '')}
                        >
                            {#snippet item(match)}
                                {match.player1_name}
                                {$t('tournament.vs')}
                                {match.player2_name} <span class="match-pts">{match.match_length}pt</span>
                            {/snippet}
                        </EntityAutocomplete>
                    </div>
                </div>
            </div>
        {/if}
    </div>
</section>

<style>
    .tournament-panel {
        width: 100%;
        height: 100%;
        background: white;
        box-sizing: border-box;
        outline: none;
        overflow: hidden;
        user-select: none;
        -webkit-user-select: none;
    }
    .panel-content {
        font-size: var(--font-size-base);
        color: var(--color-text);
        height: 100%;
        display: flex;
        overflow: hidden;
    }

    .tournament-list-pane {
        flex: 1;
        min-width: 0;
        height: 100%;
        overflow: hidden;
        display: flex;
        flex-direction: column;
    }

    .edit-input {
        width: 100%;
        padding: 1px 4px;
        border: 1px solid var(--color-primary);
        border-radius: 2px;
        font-size: var(--font-size-small);
        box-sizing: border-box;
        outline: none;
    }

    /* Detail header for match view (the strip itself is PanelTable's) */
    .back-btn {
        border: none;
        background: none;
        cursor: pointer;
        font-size: var(--font-size-base);
        color: var(--color-text-muted);
        padding: 0 4px;
        line-height: 1;
        flex-shrink: 0;
    }
    .back-btn:hover {
        color: var(--color-text);
    }
    .header-name {
        font-weight: 600;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .header-meta {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        flex-shrink: 0;
    }
    .edit-header-btn {
        visibility: hidden;
        flex-shrink: 0;
    }
    .tournament-panel :global(.detail-header:hover .edit-header-btn) {
        visibility: visible;
    }
    .header-spacer {
        flex: 1;
    }

    /* Inline tournament comment in header */
    .tournament-comment-inline {
        flex: 1;
        min-width: 80px;
        padding: 1px 4px;
        border: 1px solid var(--color-primary);
        border-radius: 2px;
        font-size: var(--font-size-small);
        box-sizing: border-box;
        outline: none;
    }
    .tournament-comment-text {
        flex-shrink: 1;
        font-size: var(--font-size-small);
        color: #bbb;
        cursor: pointer;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        max-width: 200px;
        font-style: italic;
    }
    .tournament-comment-text.has-comment {
        color: var(--color-text-muted);
    }
    .tournament-comment-text:hover {
        color: var(--color-primary);
    }

    /* Add area */
    .add-area {
        border-top: 1px solid #eee;
        padding: 3px 8px 4px;
        flex-shrink: 0;
        background: #fafafa;
        display: flex;
        gap: 4px;
        align-items: center;
    }
    .add-input {
        padding: 2px 4px;
        border: 1px solid var(--color-border);
        border-radius: 2px;
        font-size: var(--font-size-small);
        outline: none;
        box-sizing: border-box;
    }
    .add-input:focus {
        border-color: #99c;
    }
    .add-input.name {
        flex: 1;
        min-width: 0;
    }
    .add-input.date {
        width: 110px;
        flex-shrink: 0;
    }
    .add-input.loc {
        width: 90px;
        flex-shrink: 0;
    }

    /* Add match dropdown */
    .add-match-wrap {
        position: relative;
        flex: 1;
    }
    .match-pts {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    /* Match comment column, header and cells alike (the header is PanelTable's) */
    .tournament-panel :global(.comment-col) {
        max-width: 140px;
        overflow: hidden;
    }
    .comment-text {
        display: block;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-size: var(--font-size-small);
        color: #aaa;
        cursor: pointer;
        min-height: 16px;
    }
    .comment-text.has-comment {
        color: var(--color-text-muted);
    }
    .comment-text:hover {
        color: var(--color-primary);
    }
</style>
