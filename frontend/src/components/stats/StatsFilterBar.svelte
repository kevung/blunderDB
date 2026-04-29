<script>
    import { onMount } from 'svelte';
    import { statsFilterStore, statsMetricStore } from '../../stores/statsStore.js';
    import { GetAllPlayerNames, GetAllTournaments, GetStatsDateRange } from '../../../wailsjs/go/main/Database.js';
    import { GetStatsFilter, SaveStatsFilter } from '../../../wailsjs/go/main/Config.js';

    /** @type {Array<{Name: string, Count: number}>} */
    let playerList = $state([]);
    /** @type {Array<{id: number, name: string}>} */
    let tournamentList = $state([]);
    let dbEmpty = $state(false);
    /** @type {string} earliest match date for input min/placeholder */
    let dateRangeMin = $state('');
    /** @type {string} latest match date for input max/placeholder */
    let dateRangeMax = $state('');
    /** whether the tournament dropdown is open */
    let tourOpen = $state(false);

    // Local mirror of the filter store (for controlled inputs)
    let localFilter = $state({
        playerName: '',
        tournamentIDs: [],
        dateFrom: '',
        dateTo: '',
        decisionType: -1,
        matchLength: []
    });
    let dateError = $state(false);
    let mounted = $state(false);

    // When the metric toggles after mount, persist it
    $effect(() => {
        const _metric = $statsMetricStore;
        if (!mounted) return;
        scheduleSave();
    });

    const MATCH_LENGTHS = [1, 3, 5, 7, 9, 11, 13, 15, 21];

    let saveTimer = null;

    /** Debounced save to Config.yaml. */
    function scheduleSave() {
        clearTimeout(saveTimer);
        saveTimer = setTimeout(() => {
            const persisted = {
                player_name: localFilter.playerName,
                tournament_ids: localFilter.tournamentIDs,
                date_from: localFilter.dateFrom,
                date_to: localFilter.dateTo,
                // send null when -1 so Go *int receives nil
                decision_type: localFilter.decisionType === -1 ? null : localFilter.decisionType,
                match_length: localFilter.matchLength,
                metric: $statsMetricStore
            };
            SaveStatsFilter(persisted).catch(console.error); // eslint-disable-line no-console
        }, 500);
    }

    /** Push localFilter → statsFilterStore (triggers refreshStats via StatsPanel). */
    function applyFilter() {
        if (localFilter.dateFrom && localFilter.dateTo && localFilter.dateFrom > localFilter.dateTo) {
            dateError = true;
            return;
        }
        dateError = false;
        statsFilterStore.set({ ...localFilter });
        scheduleSave();
    }

    function resetFilters() {
        localFilter = {
            playerName: localFilter.playerName,
            tournamentIDs: [],
            dateFrom: '',
            dateTo: '',
            decisionType: -1,
            matchLength: []
        };
        dateError = false;
        applyFilter();
    }

    /**
     * Toggle a match length.
     * - When all are currently selected (matchLength = []) and the user clicks one,
     *   enter exclusive mode with only that length selected.
     * - Deselecting the last selected length goes back to "all" (empty array).
     */
    function toggleMatchLength(ml) {
        if (localFilter.matchLength.length === 0) {
            // Currently "all" — select only this one
            localFilter = { ...localFilter, matchLength: [ml] };
        } else {
            const idx = localFilter.matchLength.indexOf(ml);
            if (idx === -1) {
                localFilter = { ...localFilter, matchLength: [...localFilter.matchLength, ml] };
            } else {
                const next = localFilter.matchLength.filter((v) => v !== ml);
                // If we just removed the last selected one, go back to "all"
                localFilter = { ...localFilter, matchLength: next };
            }
        }
        applyFilter();
    }

    /** Whether a match-length button appears active (selected). */
    function mlActive(ml) {
        // "All" mode: every button is highlighted
        if (localFilter.matchLength.length === 0) return true;
        return localFilter.matchLength.includes(ml);
    }

    function toggleTournament(id) {
        const idx = localFilter.tournamentIDs.indexOf(id);
        if (idx === -1) {
            localFilter = { ...localFilter, tournamentIDs: [...localFilter.tournamentIDs, id] };
        } else {
            localFilter = { ...localFilter, tournamentIDs: localFilter.tournamentIDs.filter((v) => v !== id) };
        }
        applyFilter();
    }

    /** Label for the tournament dropdown button. */
    let tourLabel = $derived(
        localFilter.tournamentIDs.length === 0
            ? 'Tous'
            : localFilter.tournamentIDs.length === 1
              ? (tournamentList.find((t) => t.id === localFilter.tournamentIDs[0])?.name ?? '1 tournoi')
              : `${localFilter.tournamentIDs.length} tournois`
    );

    onMount(async () => {
        try {
            const [players, tournaments, persisted, dateRange] = await Promise.all([GetAllPlayerNames(), GetAllTournaments(), GetStatsFilter(), GetStatsDateRange()]);

            playerList = players ?? [];
            tournamentList = (tournaments ?? []).map((t) => ({ id: t.id, name: t.name }));
            dbEmpty = playerList.length === 0;
            dateRangeMin = dateRange?.DateFrom ?? '';
            dateRangeMax = dateRange?.DateTo ?? '';

            // Restore persisted filter
            // decision_type is now *int in Go: null means all (-1), 0 = checker, 1 = cube
            if (persisted) {
                localFilter = {
                    playerName: persisted.player_name ?? '',
                    tournamentIDs: persisted.tournament_ids ?? [],
                    dateFrom: persisted.date_from ?? '',
                    dateTo: persisted.date_to ?? '',
                    decisionType: persisted.decision_type ?? -1,
                    matchLength: persisted.match_length ?? []
                };

                if (persisted.metric && (persisted.metric === 'pr' || persisted.metric === 'mwc')) {
                    statsMetricStore.set(persisted.metric);
                }
            }

            // Sync to store
            statsFilterStore.set({ ...localFilter });
            mounted = true;
        } catch (err) {
            // eslint-disable-next-line no-console
            console.error('StatsFilterBar init error:', err);
        }
    });
</script>

<div class="filter-bar" aria-label="Stats filter bar">
    {#if dbEmpty}
        <span class="empty-hint">Importez des matchs pour activer les filtres.</span>
    {:else}
        <!-- Player -->
        <label class="fb-label" for="fb-player">Joueur</label>
        <select
            id="fb-player"
            class="fb-select"
            disabled={dbEmpty}
            value={localFilter.playerName}
            onchange={(e) => {
                localFilter = { ...localFilter, playerName: e.target.value };
                applyFilter();
            }}
        >
            <option value="">Toutes perspectives</option>
            {#each playerList as p}
                <option value={p.Name}>{p.Name} ({p.Count})</option>
            {/each}
        </select>

        <!-- Tournaments -->
        {#if tournamentList.length > 0}
            <span class="fb-label">Tournois</span>
            <div class="fb-tour-wrap" class:open={tourOpen}>
                <button class="fb-tour-btn" class:filtered={localFilter.tournamentIDs.length > 0} onclick={() => (tourOpen = !tourOpen)} aria-expanded={tourOpen} aria-haspopup="listbox"
                    >{tourLabel} ▾</button
                >
                {#if tourOpen}
                    <div class="fb-tour-dropdown" role="listbox" aria-multiselectable="true">
                        <label class="fb-check-label fb-tour-all">
                            <input
                                type="checkbox"
                                checked={localFilter.tournamentIDs.length === 0}
                                onchange={() => {
                                    localFilter = { ...localFilter, tournamentIDs: [] };
                                    applyFilter();
                                }}
                            />
                            Tous
                        </label>
                        <hr class="fb-tour-sep" />
                        {#each tournamentList as t}
                            <label class="fb-check-label">
                                <input type="checkbox" checked={localFilter.tournamentIDs.includes(t.id)} onchange={() => toggleTournament(t.id)} />
                                {t.name}
                            </label>
                        {/each}
                    </div>
                {/if}
            </div>
        {/if}

        <!-- Date range -->
        <label class="fb-label" for="fb-date-from">De</label>
        <input
            id="fb-date-from"
            type="date"
            class="fb-date"
            class:date-error={dateError && localFilter.dateFrom > localFilter.dateTo}
            value={localFilter.dateFrom}
            min={dateRangeMin}
            max={dateRangeMax}
            placeholder={dateRangeMin}
            title={dateRangeMin ? `Premiers matchs le ${dateRangeMin}` : ''}
            onchange={(e) => {
                localFilter = { ...localFilter, dateFrom: e.target.value };
                applyFilter();
            }}
        />
        <label class="fb-label" for="fb-date-to">À</label>
        <input
            id="fb-date-to"
            type="date"
            class="fb-date"
            class:date-error={dateError}
            value={localFilter.dateTo}
            min={dateRangeMin}
            max={dateRangeMax}
            placeholder={dateRangeMax}
            title={dateRangeMax ? `Derniers matchs le ${dateRangeMax}` : ''}
            onchange={(e) => {
                localFilter = { ...localFilter, dateTo: e.target.value };
                applyFilter();
            }}
        />

        <!-- Decision type -->
        <fieldset class="fb-radio-group" aria-label="Type de décision">
            <legend class="fb-label-inline">Décision</legend>
            {#each [[-1, 'Tout'], [0, 'Coup'], [1, 'Cube']] as [val, lbl]}
                <label class="fb-radio-label">
                    <input
                        type="radio"
                        name="decision-type"
                        value={val}
                        checked={localFilter.decisionType === val}
                        onchange={() => {
                            localFilter = { ...localFilter, decisionType: Number(val) };
                            applyFilter();
                        }}
                    />
                    {lbl}
                </label>
            {/each}
        </fieldset>

        <!-- Match length: empty array = all; every button appears active -->
        <span class="fb-label">Longueur</span>
        <div class="fb-ml-group" role="group" aria-label="Longueur de match">
            {#each MATCH_LENGTHS as ml}
                <button
                    class="fb-ml-btn"
                    class:active={mlActive(ml)}
                    onclick={() => toggleMatchLength(ml)}
                    aria-pressed={mlActive(ml)}
                    title={localFilter.matchLength.length === 0 ? 'Cliquer pour filtrer sur cette longueur uniquement' : ''}>{ml}</button
                >
            {/each}
            {#if localFilter.matchLength.length > 0}
                <button
                    class="fb-ml-all"
                    onclick={() => {
                        localFilter = { ...localFilter, matchLength: [] };
                        applyFilter();
                    }}
                    title="Sélectionner toutes les longueurs">Tout</button
                >
            {/if}
        </div>

        <!-- Reset -->
        <button class="fb-reset" onclick={resetFilters} title="Réinitialiser les filtres">↺ Reset</button>
    {/if}
</div>

<style>
    .filter-bar {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 4px 8px;
        padding: 4px 8px;
        border-bottom: 1px solid #e0e0e0;
        font-size: 11px;
        flex-shrink: 0;
        min-height: 28px;
        background: var(--panel-bg, #fafafa);
    }

    .empty-hint {
        font-style: italic;
        color: #aaa;
    }

    .fb-label {
        color: #888;
        white-space: nowrap;
        user-select: none;
    }

    .fb-label-inline {
        border: none;
        padding: 0;
        margin: 0;
        font-size: inherit;
        color: #888;
    }

    .fb-select,
    .fb-date {
        font-size: 11px;
        border: 1px solid #d0d0d0;
        border-radius: 3px;
        padding: 1px 4px;
        background: #fff;
        color: inherit;
        height: 22px;
    }

    .fb-date.date-error {
        border-color: #e05050;
    }

    /* ── Tournament dropdown ── */
    .fb-tour-wrap {
        position: relative;
    }

    .fb-tour-btn {
        font-size: 11px;
        padding: 1px 6px;
        border: 1px solid #d0d0d0;
        border-radius: 3px;
        background: #fff;
        cursor: pointer;
        white-space: nowrap;
        height: 22px;
        color: #555;
    }

    .fb-tour-btn.filtered {
        border-color: #4a7ebb;
        color: #1a5ca8;
        font-weight: 600;
    }

    .fb-tour-dropdown {
        position: absolute;
        top: calc(100% + 2px);
        left: 0;
        z-index: 100;
        background: #fff;
        border: 1px solid #ccc;
        border-radius: 4px;
        box-shadow: 0 3px 8px rgba(0, 0, 0, 0.15);
        padding: 4px 0;
        min-width: 160px;
        max-height: 220px;
        overflow-y: auto;
    }

    .fb-check-label {
        display: flex;
        align-items: center;
        gap: 5px;
        padding: 3px 10px;
        cursor: pointer;
        white-space: nowrap;
        font-size: 11px;
    }

    .fb-check-label:hover {
        background: #f0f4ff;
    }

    .fb-tour-all {
        font-weight: 600;
    }

    .fb-tour-sep {
        border: none;
        border-top: 1px solid #eee;
        margin: 2px 0;
    }

    /* ── Radio group ── */
    .fb-radio-label {
        display: flex;
        align-items: center;
        gap: 2px;
        cursor: pointer;
        white-space: nowrap;
    }

    .fb-radio-group {
        border: none;
        padding: 0;
        margin: 0;
        display: flex;
        align-items: center;
        gap: 4px;
    }

    /* ── Match length ── */
    .fb-ml-group {
        display: flex;
        gap: 2px;
        flex-wrap: wrap;
        align-items: center;
    }

    .fb-ml-btn {
        font-size: 10px;
        padding: 1px 5px;
        border: 1px solid #ccc;
        border-radius: 3px;
        background: #f5f5f5;
        cursor: pointer;
        line-height: 1.4;
    }

    .fb-ml-btn.active {
        background: #4a7ebb;
        color: #fff;
        border-color: #3a6da0;
    }

    .fb-ml-all {
        font-size: 10px;
        padding: 1px 6px;
        border: 1px solid #4a7ebb;
        border-radius: 3px;
        background: #edf3ff;
        color: #1a5ca8;
        cursor: pointer;
        line-height: 1.4;
    }

    .fb-ml-all:hover {
        background: #d8e8ff;
    }

    /* ── Reset ── */
    .fb-reset {
        margin-left: auto;
        font-size: 10px;
        padding: 1px 6px;
        border: 1px solid #ccc;
        border-radius: 3px;
        background: #f5f5f5;
        cursor: pointer;
        white-space: nowrap;
    }

    .fb-reset:hover {
        background: #e8e8e8;
    }
</style>
