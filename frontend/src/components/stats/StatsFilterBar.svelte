<script>
    import { onMount } from 'svelte';
    import { statsFilterStore, statsMetricStore } from '../../stores/statsStore.js';
    import { GetAllPlayerNames } from '../../../wailsjs/go/main/Database.js';
    import { GetAllTournaments } from '../../../wailsjs/go/main/Database.js';
    import { GetStatsFilter, SaveStatsFilter } from '../../../wailsjs/go/main/Config.js';

    /** @type {Array<{Name: string, Count: number}>} */
    let playerList = $state([]);
    /** @type {Array<{id: number, name: string}>} */
    let tournamentList = $state([]);
    let dbEmpty = $state(false);

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
        const metric = $statsMetricStore;
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
                decision_type: localFilter.decisionType,
                match_length: localFilter.matchLength,
                metric: $statsMetricStore
            };
            SaveStatsFilter(persisted).catch(console.error);
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
        const autoPlayer = localFilter.playerName; // keep auto-detected player
        localFilter = {
            playerName: autoPlayer,
            tournamentIDs: [],
            dateFrom: '',
            dateTo: '',
            decisionType: -1,
            matchLength: []
        };
        dateError = false;
        applyFilter();
    }

    function toggleMatchLength(ml) {
        const idx = localFilter.matchLength.indexOf(ml);
        if (idx === -1) {
            localFilter = { ...localFilter, matchLength: [...localFilter.matchLength, ml] };
        } else {
            localFilter = { ...localFilter, matchLength: localFilter.matchLength.filter(v => v !== ml) };
        }
        applyFilter();
    }

    function toggleTournament(id) {
        const idx = localFilter.tournamentIDs.indexOf(id);
        if (idx === -1) {
            localFilter = { ...localFilter, tournamentIDs: [...localFilter.tournamentIDs, id] };
        } else {
            localFilter = { ...localFilter, tournamentIDs: localFilter.tournamentIDs.filter(v => v !== id) };
        }
        applyFilter();
    }

    onMount(async () => {
        try {
            const [players, tournaments, persisted] = await Promise.all([
                GetAllPlayerNames(),
                GetAllTournaments(),
                GetStatsFilter()
            ]);

            playerList = players ?? [];
            tournamentList = (tournaments ?? []).map(t => ({ id: t.id, name: t.name }));
            dbEmpty = playerList.length === 0;

            // Restore persisted filter
            if (persisted) {
                const savedPlayer = persisted.player_name ?? '';
                // Auto-detect player only if no saved player name
                const detectedPlayer = (!savedPlayer && playerList.length > 0)
                    ? playerList[0].Name
                    : savedPlayer;

                localFilter = {
                    playerName: detectedPlayer,
                    tournamentIDs: persisted.tournament_ids ?? [],
                    dateFrom: persisted.date_from ?? '',
                    dateTo: persisted.date_to ?? '',
                    decisionType: persisted.decision_type ?? -1,
                    matchLength: persisted.match_length ?? []
                };

                if (persisted.metric && (persisted.metric === 'pr' || persisted.metric === 'mwc')) {
                    statsMetricStore.set(persisted.metric);
                }

                // If auto-detected, save immediately
                if (!savedPlayer && detectedPlayer) {
                    SaveStatsFilter({
                        player_name: detectedPlayer,
                        tournament_ids: [],
                        date_from: '',
                        date_to: '',
                        decision_type: -1,
                        match_length: [],
                        metric: $statsMetricStore
                    }).catch(console.error);
                }
            }

            // Sync to store
            statsFilterStore.set({ ...localFilter });
            mounted = true;
        } catch (err) {
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
            onchange={(e) => { localFilter = { ...localFilter, playerName: e.target.value }; applyFilter(); }}
        >
            <option value="">Toutes perspectives</option>
            {#each playerList as p}
                <option value={p.Name}>{p.Name} ({p.Count})</option>
            {/each}
        </select>

        <!-- Tournaments -->
        {#if tournamentList.length > 0}
            <label class="fb-label">Tournois</label>
            <div class="fb-multi">
                {#each tournamentList as t}
                    <label class="fb-check-label">
                        <input
                            type="checkbox"
                            checked={localFilter.tournamentIDs.includes(t.id)}
                            onchange={() => toggleTournament(t.id)}
                        />
                        {t.name}
                    </label>
                {/each}
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
            onchange={(e) => { localFilter = { ...localFilter, dateFrom: e.target.value }; applyFilter(); }}
        />
        <label class="fb-label" for="fb-date-to">À</label>
        <input
            id="fb-date-to"
            type="date"
            class="fb-date"
            class:date-error={dateError}
            value={localFilter.dateTo}
            onchange={(e) => { localFilter = { ...localFilter, dateTo: e.target.value }; applyFilter(); }}
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
                        onchange={() => { localFilter = { ...localFilter, decisionType: val }; applyFilter(); }}
                    />
                    {lbl}
                </label>
            {/each}
        </fieldset>

        <!-- Match length -->
        <label class="fb-label">Longueur</label>
        <div class="fb-ml-group">
            {#each MATCH_LENGTHS as ml}
                <button
                    class="fb-ml-btn"
                    class:active={localFilter.matchLength.includes(ml)}
                    onclick={() => toggleMatchLength(ml)}
                    aria-pressed={localFilter.matchLength.includes(ml)}
                >{ml}</button>
            {/each}
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

    .fb-multi {
        display: flex;
        flex-wrap: wrap;
        gap: 2px 6px;
    }

    .fb-check-label,
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

    .fb-ml-group {
        display: flex;
        gap: 2px;
        flex-wrap: wrap;
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
