<script>
    /** @typedef {import('../../wailsjs/go/models').domain.Match} Match */
    import { confirmModalStore } from '../services/confirmService.js';
    import { trashMatchEntry } from '../services/matchTrash.js';
    import { loadTournaments } from '../services/tournamentLoader.js';
    import { logger } from '../utils/logger.js';
    import { swapPlayersErrorKey } from '../utils/swapPlayersError.js';
    import { focusPanelUnlessTyping } from '../utils/panelFocus.js';
    import { toDateInputValue, formatDate } from '../utils/matchTable.js';
    import { createInlineEdit } from '../utils/inlineEdit.svelte.js';
    import { onChange } from '../utils/onChange.js';
    import { fmtMwc7, mwc7Tooltip } from '../utils/mwc7.js';
    import { onMount, onDestroy, untrack, tick } from 'svelte';
    import { get } from 'svelte/store';
    import { noteSelection, forgetSelection } from '../stores/exchangeObjectStore.js';
    import { GetMatchByID, UpdateMatch, GetMatchMovePositions, SetMatchTournamentByName, SwapMatchPlayers } from '../../wailsjs/go/database/Database.js';
    import ManagePlayersModal from './ManagePlayersModal.svelte';
    import MatchSheet from './MatchSheet.svelte';
    import EntityAutocomplete from './EntityAutocomplete.svelte';
    import PanelTable, { navigationDelta } from './panels/PanelTable.svelte';
    import { exportMatchMat } from '../services/exportService.js';
    import { menuFor } from '../services/actionCatalog.js';
    import { sendContext } from '../services/sendActions.js';

    /** The send group of a match row: its copies to the other bases, and the lot. @param {any} match */
    const sendMatch = (match) => sendContext('match', match.id, `${match.player1_name ?? ''} – ${match.player2_name ?? ''}`);
    import { panelKeyGuard } from '../services/keyboardService.js';
    import { registerKeys } from '../services/keyDispatch.js';
    import { editMatchTranscription } from '../services/transcriptionSave.js';
    import { enrichMatchFromFile } from '../services/importService.js';
    import { t, tMsg } from '../i18n';
    import { positionStore, matchContextStore, lastVisitedMatchStore } from '../stores/positionStore';
    import {
        openPanels,
        PANEL,
        closePanel,
        matchPanelRefreshTriggerStore,
        dbMutationCounterStore,
        positionReloadTriggerStore,
        statusBarTextStore,
        matchOpenRequestStore,
        isAnyModalOpen
    } from '../stores/uiStore';
    import { tournamentsStore } from '../stores/tournamentStore';
    import { matchListStore } from '../stores/matchListStore.js';
    import { databaseLoadedStore, databasePathStore } from '../stores/databaseStore';
    import { libraryDragAttrs } from '../utils/libraryDrag.js';
    import { activeGroupId, libraryMark, writingOnLibrary } from '../stores/groupActivityStore.js';

    /** @type {any[]} */
    // The page(s) of the shared match list, filtered and ordered by SQL.
    const matches = $derived($matchListStore.rows);
    const matchTotal = $derived($matchListStore.total);
    let filterText = $state('');
    /** @type {HTMLInputElement | null} */
    let filterInput = $state(null);
    /** @type {ReturnType<typeof setTimeout> | null} */
    /** @type {ReturnType<typeof setTimeout> | undefined} */
    let filterTimer;
    /** @type {any} */
    let selectedMatch = $state(null);
    // The « Autres bases » panel speaks of the match the user selected: noted on the gesture, not
    // on selectedMatch, which hiding the tab and restoring the last visited match also move.
    /** @param {any} match */
    const noteMatch = (match) => noteSelection('match', { id: match.id, label: `${match.player1_name} – ${match.player2_name}` });
    // A match requested from the command palette is being opened.
    let openingRequested = false;
    const visible = $derived($openPanels.has(PANEL.MATCH));
    const databaseLoaded = $derived($databaseLoadedStore);
    let lastVisitedMatch = $derived($lastVisitedMatchStore);
    let tournaments = $derived($tournamentsStore || []);

    // Sorting state, cycled by the table header (asc → desc → unsorted)
    let sort = $state(/** @type {{ column: string | null, direction: string }} */ ({ column: null, direction: 'asc' }));
    /** @type {any} */
    let table = $state(null); // PanelTable instance: keyboard navigation and scrolling

    // Inline tournament editing (autocomplete over the known tournaments)
    const tournamentEdit = createInlineEdit({
        onSave: async (matchId, value, mark) => {
            const name = value.trim();
            try {
                if (!(await writingOnLibrary(mark, async () => (await SetMatchTournamentByName(matchId, name), true)))) return;
                await matchListStore.refreshRow(matchId);
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
        onSave: async (matchId, draft, mark) => {
            try {
                if (!(await writingOnLibrary(mark, async () => (await UpdateMatch(matchId, draft.player1, draft.player2, draft.date), true)))) return;
                await matchListStore.refreshRow(matchId);
                statusBarTextStore.set(tMsg('match.matchUpdated'));
            } catch (error) {
                logger.error('Error updating match:', error);
                statusBarTextStore.set(tMsg('match.errorUpdating'));
            }
        }
    });

    // Merge players modal
    let showManagePlayersModal = $state(false);

    // Reload matches when a new match is imported, or a database opened. The value present
    // at mount is not news: the panel mounting with a database open loads through the
    // `visible` effect below, and a second load would run the same costly list again.
    const triggerAtMount = untrack(() => $matchPanelRefreshTriggerStore);
    $effect(() => {
        const trigger = $matchPanelRefreshTriggerStore;
        if (trigger === triggerAtMount) return;
        if (untrack(() => !visible || !databaseLoaded)) return;
        loadMatches().then(async () => {
            const lvm = lastVisitedMatch;
            if (openingRequested) return; // the command palette's match wins
            if (lvm && lvm.matchID) {
                const m = await findMatch(lvm.matchID);
                if (m) {
                    selectedMatch = m;
                    showSheet(m);
                }
            }
        });
    });

    // Load on open. Opening before the DB is ready (session restore) is covered
    // by openDatabaseByPath's matchPanelRefreshTriggerStore bump.
    $effect(
        onChange(
            () => visible, // $derived — tracked
            (/** @type {boolean} */ opened) => {
                if (opened) {
                    // Kept mounted while hidden (TabbedPanel), the panel may come back to a list
                    // another library group put in place: the header and the field follow it.
                    untrack(() => {
                        const list = get(matchListStore);
                        if (sort.column !== list.column || sort.direction !== list.direction) sort = { column: list.column, direction: list.direction };
                        if (filterText.trim() !== list.text) {
                            if (filterTimer) clearTimeout(filterTimer);
                            filterText = list.text;
                        }
                        // Back with no database: the rows of the one closed meanwhile must not show.
                        if (!databaseLoaded) matchListStore.set({ ...list, rows: [], total: 0, loading: false, loaded: false });
                    });
                }
                if (opened && databaseLoaded) {
                    loadMatches().then(async () => {
                        const lvm = lastVisitedMatch;
                        if (openingRequested) return; // the command palette's match wins
                        if (lvm && lvm.matchID) {
                            const m = await findMatch(lvm.matchID);
                            if (m) {
                                selectedMatch = m;
                                showSheet(m);
                            } else {
                                selectedMatch = null;
                                sheetMatchId = null;
                            }
                        } else {
                            selectedMatch = null;
                            sheetMatchId = null;
                        }
                    });
                } else if (!opened) {
                    selectedMatch = null;
                    sheetMatchId = null;
                    tournamentEdit.cancel();
                    sheet?.cancelTournamentEdit();
                }
            },
            false
        )
    );

    async function loadMatches() {
        return logger.perf('MatchPanel:loadMatches', async () => {
            try {
                // The tournaments only feed the edit autocomplete, and their badges cost a full
                // aggregate on a large library: the list must not wait for them.
                loadTournaments();
                await matchListStore.reload();
            } catch (error) {
                logger.error('Error loading matches:', error);
            }
        });
    }

    function startEditTournament(/** @type {Match} */ match, /** @type {Event} */ event) {
        event.stopPropagation();
        tournamentEdit.start(match.id, match.tournament_name || match.event || '');
    }

    async function selectTournamentOption(/** @type {{ name: string }} */ tournament) {
        tournamentEdit.draft = tournament.name;
        await tournamentEdit.save();
    }

    function startEditMatch(/** @type {Match} */ match, /** @type {Event | undefined} */ ev) {
        ev?.stopPropagation();
        matchEdit.start(match.id, {
            player1: match.player1_name || '',
            player2: match.player2_name || '',
            date: toDateInputValue(match.match_date)
        });
    }

    // Order is the database's (or the store's, for PR/MWC): follow the header.
    $effect(() => {
        const { column, direction } = sort;
        untrack(() => matchListStore.setSort(column, direction));
    });

    function onFilterInput() {
        clearTimeout(filterTimer);
        filterTimer = setTimeout(() => matchListStore.setText(filterText.trim()), 250);
    }

    function clearFilter() {
        clearTimeout(filterTimer);
        filterText = '';
        matchListStore.setText('');
    }

    const sortedMatches = $derived(matches);

    const columns = $derived([
        { key: 'index', label: '#', narrow: true },
        { key: 'date', label: $t('match.date'), sortable: true, narrow: true },
        { key: 'player1', label: $t('match.player1'), sortable: true },
        { key: 'player2', label: $t('match.player2'), sortable: true },
        { key: 'length', label: $t('match.pts'), sortable: true, narrow: true, align: 'right' },
        { key: 'tournament', label: $t('match.tournament'), sortable: true, class: 'tournament-col' },
        { key: 'pr', label: 'PR', sortable: true, narrow: true, align: 'right' },
        { key: 'mwc', label: 'MWC', sortable: true, narrow: true, align: 'right' },
        { key: 'mwc7', label: $t('mwc7.short'), sortable: true, narrow: true, align: 'right', title: $t('mwc7.name') },
        { key: 'actions', actions: true }
    ]);

    /** A match by id: from the loaded rows, else read on its own (it may lie beyond the first page). */
    async function findMatch(/** @type {number} */ id) {
        const loaded = matches.find((m) => m.id === id);
        if (loaded) return loaded;
        try {
            return (await GetMatchByID(id)) || null;
        } catch {
            return null;
        }
    }

    // The match whose sheet sits beside the list, and the sheet itself (its decisions, its video, its keys).
    /** @type {number | null} */
    let sheetMatchId = $state(null);
    /** @type {any} */
    let sheet = $state(null);

    /** Shows the sheet of a match, read again when it already shows it. */
    function showSheet(/** @type {Match} */ match) {
        if (sheetMatchId === match.id) sheet?.reload();
        else sheetMatchId = match.id;
    }

    function closeSheet() {
        selectedMatch = null;
        forgetSelection('match');
        sheetMatchId = null;
    }

    function selectMatch(/** @type {Match} */ match) {
        if (selectedMatch && selectedMatch.id === match.id) {
            closeSheet();
        } else {
            selectedMatch = match;
            noteMatch(match);
            showSheet(match);
        }
    }

    async function enterMatchMode(/** @type {Match} */ match) {
        if (sheetMatchId !== match.id) {
            sheetMatchId = match.id;
            await tick();
        }
        await sheet?.enterMatchMode();
    }

    function handleDoubleClick(/** @type {Match} */ match) {
        enterMatchMode(match);
    }

    // A match requested by the command palette opens like a double-click once
    // the list is loaded; the request is consumed even if the match is gone.
    $effect(() => {
        const requested = $matchOpenRequestStore;
        if (requested == null || !visible || !$matchListStore.loaded) return;
        untrack(() => {
            matchOpenRequestStore.set(null);
            // Held while the match is fetched: the list's own reselect must not run meanwhile.
            openingRequested = true;
            const target = typeof requested === 'object' ? requested : { id: requested };
            findMatch(target.id)
                .then(async (match) => {
                    if (!match) {
                        statusBarTextStore.set(tMsg('palette.matchGone'));
                        return;
                    }
                    selectedMatch = match;
                    noteMatch(match);
                    showSheet(match);
                    await tick();
                    await sheet?.whenLoaded();
                    if (target.moveId && (await sheet?.jumpToMoveId(target.moveId))) return;
                    await sheet?.enterMatchMode();
                    if (target.moveId) statusBarTextStore.set(tMsg('status.moveGone'));
                })
                .finally(() => (openingRequested = false));
        });
    });

    // The sheet trashed its match (the list is already read again): the sheet closes.
    function onSheetDeleted(/** @type {Match} */ match) {
        loadTournaments();
        if (sheetMatchId === match.id || selectedMatch?.id === match.id) closeSheet();
    }

    async function deleteMatchEntry(/** @type {Match} */ match, /** @type {Event | undefined} */ event) {
        event?.stopPropagation();
        await trashMatchEntry(match, async () => {
            await loadMatches();
            if (sheetMatchId === match.id || selectedMatch?.id === match.id) closeSheet();
        });
    }

    async function swapMatchPlayers(/** @type {Match} */ match, /** @type {Event | undefined} */ event) {
        event?.stopPropagation();
        const mark = libraryMark();
        try {
            if (!(await writingOnLibrary(mark, async () => (await SwapMatchPlayers(match.id), true)))) return;
            await matchListStore.refreshRow(match.id);

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

            if (sheetMatchId === match.id) await sheet?.reload();

            statusBarTextStore.set(tMsg('match.swappedPlayers'));
        } catch (error) {
            logger.error('Error swapping match players:', error);
            statusBarTextStore.set(tMsg(swapPlayersErrorKey(error, 'match.errorSwapping')));
        }
    }

    function closeMatchPanel() {
        closePanel(PANEL.MATCH);
    }

    function handleKeyDown(/** @type {KeyboardEvent} */ event) {
        if (!visible) return;
        // Already handled: a dialog delegated on the app root runs first and claims its keys this way.
        if (event.defaultPrevented) return;
        // A confirmation is up: Enter / Escape / Delete belong to it.
        if (get(confirmModalStore)) return;
        // Don't intercept keys while the merge players modal is open
        if (showManagePlayersModal) return;

        // [ and ] set the open video's speed, and `v` shows the reviewed decision in it. Read before
        // panelKeyGuard: AltGr, which types them on AZERTY, arrives as Ctrl+Alt on Windows.
        if (sheet?.handleSheetKey(event)) return;

        // Let Ctrl/Meta combos, Space, '?' and typing in an editable field pass
        // through to the global handler — see keyboardService.panelKeyGuard.
        if (panelKeyGuard(event)) return;

        // Only the keys this panel owns are claimed (shortcutMap.js, matchPanel): the others
        // browse the board through the global handler.
        if (event.key === 'Escape') {
            event.stopPropagation();
            if (matchEdit.editingId !== null) {
                matchEdit.cancel();
                event.preventDefault();
            } else if (tournamentEdit.editingId !== null || sheet?.tournamentEditing()) {
                tournamentEdit.cancel();
                sheet?.cancelTournamentEdit();
                event.preventDefault();
            } else if (sheetMatchId != null) {
                // Close the sheet first
                closeSheet();
                event.preventDefault();
            } else {
                closeMatchPanel();
            }
            return;
        }

        // `/` goes to the filter field.
        if (event.key === '/' && !event.altKey) {
            event.stopPropagation();
            event.preventDefault();
            filterInput?.focus();
            return;
        }

        // j/k walk the list; with no selection, j lands on the first row.
        const delta = navigationDelta(event);
        if (delta !== 0) {
            event.stopPropagation();
            if (sortedMatches.length > 0) {
                event.preventDefault();
                table?.navigate(delta);
            }
            return;
        }

        // Claimed with or without a selection: a Delete here never reaches the board's position.
        if (event.key === 'Enter' || event.key === 'Delete') event.stopPropagation();

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

    function handleClickOutside(/** @type {MouseEvent} */ event) {
        // Kept mounted behind another tab (TabbedPanel): the other panel's clicks are not ours to blur.
        if (!visible) return;
        // A `document` listener sees every click, modals included: it must never
        // blur the focused element, or modal text fields become unclickable.
        if (get(isAnyModalOpen)) return;
        // Don't interfere while the merge players modal is open
        if (showManagePlayersModal) return;
        // Close tournament dropdown if clicking outside
        if ((tournamentEdit.editingId !== null || sheet?.tournamentEditing()) && !event.target.closest('.tournament-cell-edit')) {
            tournamentEdit.cancel();
            sheet?.cancelTournamentEdit();
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
                // Deferred, so the user may be typing by then: never take their field (utils/panelFocus.js).
                focusPanelUnlessTyping(document.getElementById('matchPanel'));
                if (selectedMatch) table?.scrollToSelected('center');
            }, 100);
            return () => clearTimeout(id);
        }
    });

    /** @type {(() => void) | null} */
    let unregisterKeys = null;

    onMount(async () => {
        // With a database open, the `visible` effect has this mount's load already.
        if (visible && !databaseLoaded) await loadMatches();
        document.addEventListener('click', handleClickOutside);
        unregisterKeys = registerKeys('matchPanel', handleKeyDown);
    });

    onDestroy(() => {
        document.removeEventListener('click', handleClickOutside);
        unregisterKeys?.();
    });
</script>

<section class="match-panel" aria-label={$t('match.ariaLabel')} id="matchPanel" tabindex="-1">
    <div class="match-panel-content">
        <!-- Match list (left pane) -->
        <div class="match-list-pane" class:has-detail={sheetMatchId != null}>
            <div class="match-list-toolbar">
                <input
                    bind:this={filterInput}
                    bind:value={filterText}
                    oninput={onFilterInput}
                    onkeydown={(e) => {
                        if (e.key === 'Escape' && filterText) {
                            e.stopPropagation();
                            clearFilter();
                        }
                    }}
                    class="match-filter"
                    type="search"
                    placeholder={$t('match.filterPlaceholder')}
                    aria-label={$t('match.filterAria')}
                />
                {#if $matchListStore.ids}
                    <button type="button" class="toolbar-btn match-ids-chip" data-testid="match-ids-filter" onclick={() => matchListStore.setIDs(null)} title={$t('match.idsFilterClear')}
                        >{$t('match.idsFilter', { n: $matchListStore.ids.length })} ×</button
                    >
                {/if}
                {#if matches.length > 0 && matches.length < matchTotal}<span class="match-count">{$t('match.countOfTotal', { shown: matches.length, total: matchTotal })}</span>{/if}
                <button class="toolbar-btn" onclick={() => (showManagePlayersModal = true)} title={$t('match.mergePlayersTitle')} disabled={matches.length === 0}>⇢ {$t('match.mergePlayers')}</button>
            </div>
            <PanelTable
                bind:this={table}
                rows={sortedMatches}
                {columns}
                bind:sort
                sortOptions={{ tristate: true }}
                selectedKey={selectedMatch?.id}
                rowClass={(/** @type {Match} */ match) => (matchEdit.isEditing(match.id) ? 'match-editing-row' : '')}
                rowAttrs={(/** @type {Match} */ match) =>
                    libraryDragAttrs(() =>
                        matchEdit.isEditing(match.id) ? null : { group: $activeGroupId, kind: 'match', id: match.id, label: `${match.player1_name || '?'} – ${match.player2_name || '?'}` }
                    )}
                onSelect={(/** @type {Match} */ match) => {
                    if (!matchEdit.isEditing(match.id)) selectMatch(match);
                }}
                onActivate={(/** @type {Match} */ match) => {
                    if (!matchEdit.isEditing(match.id)) handleDoubleClick(match);
                }}
                onNearEnd={matchListStore.loadMore}
                rowMenu={(/** @type {Match} */ match) =>
                    matchEdit.isEditing(match.id)
                        ? []
                        : menuFor(
                              'match',
                              {
                                  ...sendMatch(match),
                                  do: {
                                      ...sendMatch(match).do,
                                      open: () => handleDoubleClick(match),
                                      exportMat: () => exportMatchMat(match),
                                      editTranscription: () => editMatchTranscription(match.id),
                                      swapPlayers: () => swapMatchPlayers(match, undefined),
                                      edit: () => startEditMatch(match, undefined),
                                      trash: () => deleteMatchEntry(match, undefined)
                                  }
                              },
                              $t
                          )}
                emptyActions
                emptyClear={filterText.trim() ? { label: $t('emptyState.clearFilter'), onClick: clearFilter } : null}
                emptyText={matches.length === 0 && ($matchListStore.loaded || !$databasePathStore) ? (filterText.trim() ? $t('match.noMatchesFiltered') : $t('match.noMatchesImported')) : ''}
            >
                {#snippet cells(/** @type {Match} */ match, /** @type {number} */ index)}
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
                        <td class="narrow-col align-right no-select">{match.match_length}</td>
                        <td class="tournament-col no-select">{match.tournament_name || match.event || ''}</td>
                        <td class="narrow-col align-right no-select">{match.pr > 0 ? match.pr.toFixed(2) : ''}{match.pr2 > 0 ? ' / ' + match.pr2.toFixed(2) : ''}</td>
                        <td class="narrow-col align-right no-select">{match.mwc_loss > 0 ? (match.mwc_loss * 100).toFixed(2) + '%' : ''}</td>
                        <td class="narrow-col align-right no-select">{fmtMwc7(match.mwc7)}</td>
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
                        <td class="narrow-col align-right no-select">{match.match_length}</td>
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
                        <td class="narrow-col align-right no-select stat-col">{match.pr > 0 ? match.pr.toFixed(2) : '—'}{match.pr2 > 0 ? ' / ' + match.pr2.toFixed(2) : ''}</td>
                        <td class="narrow-col align-right no-select stat-col">{match.mwc_loss > 0 ? (match.mwc_loss * 100).toFixed(2) + '%' : '—'}</td>
                        <td class="narrow-col align-right no-select stat-col" title={mwc7Tooltip(match.mwc7, $t)}
                            >{fmtMwc7(match.mwc7)}{match.mwc7_p2?.available ? ' / ' + fmtMwc7(match.mwc7_p2) : ''}</td
                        >
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
                                        editMatchTranscription(match.id);
                                    }}
                                    title={$t('match.editTranscriptionTooltip')}
                                    aria-label={$t('match.editTranscription')}>⌨</button
                                >
                                <button
                                    class="icon-btn"
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        ((e) => swapMatchPlayers(match, e))(e);
                                    }}
                                    title={$t('match.swapPlayers')}>⇄</button
                                >
                                <!-- Enrichir : le même réimport qu'ailleurs, rendu trouvable. -->
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

        <!-- Detail pane, only while a match is selected (the list spans the full
             width otherwise). The width change is not animated, so the clicked row
             stays under the cursor for the second click of a double-click. -->
        <!-- Sheet, only while a match is selected (the list spans the full width otherwise).
             The width change is not animated, so the clicked row stays under the cursor for the
             second click of a double-click. -->
        {#if sheetMatchId != null}
            <div class="sheet-slot">
                <MatchSheet bind:this={sheet} matchId={sheetMatchId} onDeleted={onSheetDeleted} />
            </div>
        {/if}
    </div>
</section>

{#if showManagePlayersModal}
    <ManagePlayersModal
        onClose={() => (showManagePlayersModal = false)}
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
        background-color: var(--color-surface);
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
    /* Full width until a match is selected, then 45 % beside the detail pane. */
    .match-list-pane {
        flex: 1 1 100%;
        max-width: 100%;
        min-width: 0;
        height: 100%;
        overflow: hidden;
        display: flex;
        flex-direction: column;
    }

    /* No transition: the row must not travel under the cursor between the two
       clicks of a double-click. No `#nnn` in <style>: colorTokens.sync.test.js
       reads it as a hex colour. */
    .match-list-pane.has-detail {
        flex: 0 0 45%;
        max-width: 45%;
        border-right: 1px solid var(--color-border);
    }

    /* Drawn as PanelHeader's strip, so the list and the match beside it share one band. */
    .match-list-toolbar {
        flex-shrink: 0;
        display: flex;
        align-items: center;
        gap: var(--space-2);
        min-height: 24px;
        padding: var(--space-1) var(--space-2);
        border-bottom: 1px solid var(--color-border);
        background: var(--color-surface-alt);
    }

    .match-filter {
        flex: 1;
        min-width: 0;
        padding: 2px 6px;
        font-size: var(--font-size-small);
        background: var(--color-surface);
        color: var(--color-text);
        border: 1px solid var(--color-border);
        border-radius: 3px;
    }

    .match-count {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        white-space: nowrap;
    }

    .toolbar-btn {
        background: none;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        cursor: pointer;
        padding: 2px 8px;
        line-height: 1.6;
    }

    .toolbar-btn:hover:not(:disabled) {
        background: color-mix(in srgb, var(--color-primary) 12%, var(--color-surface));
        border-color: var(--color-primary);
        color: color-mix(in srgb, var(--color-primary) 80%, var(--color-text));
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

    .sheet-slot {
        flex: 0 0 55%;
        max-width: 55%;
        min-width: 0;
        height: 100%;
    }
</style>
