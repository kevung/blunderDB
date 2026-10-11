<script>
    import { onMount, untrack } from 'svelte';
    import { get } from 'svelte/store';
    import { statsFilterStore, statsInvalidationKeyStore, initStatsFilter, patchStatsFilter, resetStatsFilter, dropMissingPlayer } from '../../stores/statsStore.js';
    import { databaseLoadedStore } from '../../stores/databaseStore.js';
    import { t } from '../../i18n/index.js';
    import { GetAllPlayerNames, GetAllTournaments, GetStatsAnalysisEngines, GetStatsDateRange } from '../../../wailsjs/go/database/Database.js';
    import EntityAutocomplete from '../EntityAutocomplete.svelte';

    /**
     * Players tab on screen: the decision type is disabled, since that table splits checker
     * from cube itself. The player stays active: it marks the player's row.
     * @type {{ playersTab?: boolean, dbEmpty?: boolean }}
     */
    // `dbEmpty`: no match imported yet. The panel then shows its empty state in place of the
    // filters, which have nothing to filter.
    let { playersTab = false, dbEmpty = $bindable(false) } = $props();

    /** @type {Array<{Name: string, Count: number}>} */
    let playerList = $state([]);
    /** @type {Array<{id: number, name: string}>} */
    let tournamentList = $state([]);
    /** Engine labels stored in the analyses: the options of the engine menu. */
    /** @type {string[]} */
    let engineList = $state([]);
    /** Product names for the labels the importers write; any other label shows as stored. */
    /** @type {Record<string, string>} */
    const ENGINE_NAMES = { XG: 'eXtreme Gammon', GNUbg: 'GNU Backgammon' };
    /** @type {string} earliest match date for input min/placeholder */
    let dateRangeMin = $state('');
    /** @type {string} latest match date for input max/placeholder */
    let dateRangeMax = $state('');
    /** whether the tournament dropdown is open */
    let tourOpen = $state(false);

    const filter = $derived($statsFilterStore);

    /** The text of the player field: typing overrides it until the filter changes. */
    let playerDraft = $derived(filter.playerName);

    /** The stored labels, plus the retained one when the database no longer holds it. */
    let engineOptions = $derived(filter.analysisEngine && !engineList.includes(filter.analysisEngine) ? [...engineList, filter.analysisEngine] : engineList);
    let dateError = $state(false);
    let mounted = $state(false);

    const MATCH_LENGTHS = [1, 3, 5, 7, 9, 11, 13, 15, 21];

    /** @param {Partial<import('../../stores/statsStore.js').StatsFilterState>} patch */
    function apply(patch) {
        dateError = !patchStatsFilter(patch);
        return !dateError;
    }

    /**
     * A refused date leaves the filter as it was; the field goes back to the applied date, so it
     * never shows a range the figures do not cover.
     */
    function applyDate(/** @type {Event} */ e, /** @type {'dateFrom' | 'dateTo'} */ field) {
        const input = /** @type {HTMLInputElement} */ (e.currentTarget);
        if (!apply({ [field]: input.value })) input.value = filter[field];
    }

    /** @param {string} name */
    function choosePlayer(name) {
        playerDraft = name;
        apply({ playerName: name });
    }

    /** Enter on the typed text: an exact name picks it, an empty field means every player. */
    function submitPlayer(/** @type {string} */ text) {
        const typed = text.trim();
        if (typed === '') return choosePlayer('');
        const match = playerList.find((p) => p.Name.toLowerCase() === typed.toLowerCase());
        if (match) choosePlayer(match.Name);
        else playerDraft = filter.playerName;
    }

    /** Opening the list on the chosen name shows every player, not just that one. */
    function playerMatches(/** @type {{Name: string}} */ p, /** @type {string} */ query) {
        return query === filter.playerName || p.Name.toLowerCase().includes(query.toLowerCase());
    }

    /**
     * Toggle a match length.
     * - When all are currently selected (matchLength = []) and the user clicks one,
     *   enter exclusive mode with only that length selected.
     * - Deselecting the last selected length goes back to "all" (empty array).
     */
    function toggleMatchLength(/** @type {number} */ ml) {
        const current = filter.matchLength;
        if (current.length === 0) apply({ matchLength: [ml] });
        else if (!current.includes(ml)) apply({ matchLength: [...current, ml] });
        else apply({ matchLength: current.filter((v) => v !== ml) });
    }

    /** Whether a match-length button appears active (selected). */
    function mlActive(/** @type {number} */ ml) {
        return filter.matchLength.length === 0 || filter.matchLength.includes(ml);
    }

    function toggleTournament(/** @type {number} */ id) {
        const ids = filter.tournamentIDs;
        apply({ tournamentIDs: ids.includes(id) ? ids.filter((v) => v !== id) : [...ids, id] });
    }

    /** Label for the tournament dropdown button. */
    let tourLabel = $derived(
        filter.tournamentIDs.length === 0
            ? $t('stats.allTournaments')
            : filter.tournamentIDs.length === 1
              ? (tournamentList.find((tour) => tour.id === filter.tournamentIDs[0])?.name ?? $t('stats.tournament'))
              : $t('stats.nTournaments', { n: filter.tournamentIDs.length })
    );

    /**
     * Load players, tournaments and date bounds; re-read on every database
     * change so the dropdowns never show what is gone.
     */
    async function loadLists() {
        const dbLoaded = get(databaseLoadedStore);
        const [players, tournaments, dateRange, engines] = dbLoaded
            ? await Promise.all([GetAllPlayerNames(), GetAllTournaments(), GetStatsDateRange(), GetStatsAnalysisEngines()])
            : [[], [], null, []];

        // Derive from the local value, never playerList: without a database this
        // runs synchronously, and reading back what it wrote loops the caller
        // effect (effect_update_depth_exceeded).
        const loadedPlayers = players ?? [];
        playerList = loadedPlayers;
        engineList = engines ?? [];
        tournamentList = (tournaments ?? []).map((t) => ({ id: t.id, name: t.name }));
        dbEmpty = loadedPlayers.length === 0;
        dateRangeMin = dateRange?.DateFrom ?? '';
        dateRangeMax = dateRange?.DateTo ?? '';
        return loadedPlayers;
    }

    /** Without a database the list is empty, not authoritative: the saved player stays. */
    function dropMissing(/** @type {Array<{Name: string}>} */ players) {
        if (get(databaseLoadedStore)) dropMissingPlayer(players.map((p) => p.Name));
    }

    // Re-read on every database mutation: the key is the effect's only
    // dependency; everything else is untracked, as it writes the $state it reads.
    $effect(() => {
        void $statsInvalidationKeyStore;
        untrack(() => {
            if (!mounted) return; // onMount does the first load
            loadLists()
                .then(dropMissing)
                // eslint-disable-next-line no-console
                .catch((err) => console.error('StatsFilterBar reload error:', err));
        });
    });

    onMount(async () => {
        try {
            const [players] = await Promise.all([loadLists(), initStatsFilter()]);
            // A saved filter can name a player who has since been merged away or whose
            // matches were deleted.
            dropMissing(players);
            mounted = true;
        } catch (err) {
            // eslint-disable-next-line no-console
            console.error('StatsFilterBar init error:', err);
        }
    });
</script>

{#if !dbEmpty}
    <div class="filter-bar" aria-label={$t('stats.title')}>
        <!-- Player: empty means every perspective -->
        <label class="fb-label" for="fb-player">{$t('stats.playerLabel')}</label>
        <div class="fb-player" title={filter.playerName || undefined}>
            <EntityAutocomplete
                id="fb-player"
                variant="field"
                bind:value={playerDraft}
                items={playerList}
                label={(/** @type {{Name: string}} */ p) => p.Name}
                key={(/** @type {{Name: string}} */ p) => p.Name}
                filter={playerMatches}
                placeholder={$t('stats.allPerspectives')}
                maxHeight={240}
                onSelect={(/** @type {{Name: string}} */ p) => choosePlayer(p.Name)}
                onSubmit={submitPlayer}
                onCancel={() => (playerDraft = filter.playerName)}
                onDismiss={() => (playerDraft = filter.playerName)}
            >
                {#snippet item(/** @type {{Name: string, Count: number}} */ p)}<span title={p.Name}>{p.Name} ({p.Count})</span>{/snippet}
            </EntityAutocomplete>
        </div>
        {#if filter.playerName}
            <button class="fb-clear" onclick={() => choosePlayer('')} aria-label={$t('stats.allPerspectives')} title={$t('stats.allPerspectives')}>✕</button>
        {/if}

        <!-- Tournaments -->
        {#if tournamentList.length > 0}
            <span class="fb-label">{$t('stats.tournamentsLabel')}</span>
            <div class="fb-tour-wrap" class:open={tourOpen}>
                <button class="fb-tour-btn" class:filtered={filter.tournamentIDs.length > 0} onclick={() => (tourOpen = !tourOpen)} aria-expanded={tourOpen} aria-haspopup="listbox"
                    >{tourLabel} ▾</button
                >
                {#if tourOpen}
                    <div class="fb-tour-dropdown" role="listbox" aria-multiselectable="true">
                        <label class="fb-check-label fb-tour-all">
                            <input type="checkbox" checked={filter.tournamentIDs.length === 0} onchange={() => apply({ tournamentIDs: [] })} />
                            {$t('stats.allTournaments')}
                        </label>
                        <hr class="fb-tour-sep" />
                        {#each tournamentList as t (t.id)}
                            <label class="fb-check-label">
                                <input type="checkbox" checked={filter.tournamentIDs.includes(t.id)} onchange={() => toggleTournament(t.id)} />
                                {t.name}
                            </label>
                        {/each}
                    </div>
                {/if}
            </div>
        {/if}

        <!-- Date range -->
        <label class="fb-label" for="fb-date-from">{$t('stats.dateFromLabel')}</label>
        <input
            id="fb-date-from"
            type="date"
            class="fb-date"
            class:date-error={dateError}
            value={filter.dateFrom}
            min={dateRangeMin}
            max={dateRangeMax}
            placeholder={dateRangeMin}
            title={dateRangeMin ? $t('stats.firstMatchDate', { date: dateRangeMin }) : ''}
            onchange={(e) => applyDate(e, 'dateFrom')}
        />
        <label class="fb-label" for="fb-date-to">{$t('stats.dateToLabel')}</label>
        <input
            id="fb-date-to"
            type="date"
            class="fb-date"
            class:date-error={dateError}
            value={filter.dateTo}
            min={dateRangeMin}
            max={dateRangeMax}
            placeholder={dateRangeMax}
            title={dateRangeMax ? $t('stats.lastMatchDate', { date: dateRangeMax }) : ''}
            onchange={(e) => applyDate(e, 'dateTo')}
        />

        <!-- Decision type -->
        <fieldset class="fb-radio-group" aria-label={$t('stats.decisionTypeLabel')} disabled={playersTab} title={playersTab ? $t('stats.filterDisabledOnPlayersTab') : undefined}>
            <legend class="fb-label-inline">{$t('stats.decisionLegend')}</legend>
            {#each [[-1, $t('stats.decisionAll')], [0, $t('stats.decisionChecker')], [1, $t('stats.decisionCube')]] as [val, lbl] (val)}
                <label class="fb-radio-label">
                    <input type="radio" name="decision-type" value={val} checked={filter.decisionType === val} onchange={() => apply({ decisionType: Number(val) })} />
                    {lbl}
                </label>
            {/each}
        </fieldset>

        <!-- Match length: empty array = all; every button appears active -->
        <span class="fb-label">{$t('stats.lengthLabel')}</span>
        <div class="fb-ml-group" role="group" aria-label={$t('stats.matchLengthLabel')}>
            {#each MATCH_LENGTHS as ml (ml)}
                <button
                    class="fb-ml-btn"
                    class:active={mlActive(ml)}
                    onclick={() => toggleMatchLength(ml)}
                    aria-pressed={mlActive(ml)}
                    title={filter.matchLength.length === 0 ? $t('stats.filterLengthHint') : ''}>{ml}</button
                >
            {/each}
            {#if filter.matchLength.length > 0}
                <button class="fb-ml-all" onclick={() => apply({ matchLength: [] })} title={$t('stats.selectAllLengths')}>{$t('stats.all')}</button>
            {/if}
        </div>

        <!-- Provenance: which engine analysed the decisions, and how deep -->
        <label class="fb-label" for="fb-engine">{$t('stats.engineLabel')}</label>
        <select id="fb-engine" class="fb-select" title={$t('stats.engineHint')} value={filter.analysisEngine} onchange={(e) => apply({ analysisEngine: e.target.value })}>
            <option value="">{$t('stats.engineAny')}</option>
            {#each engineOptions as eng (eng)}
                <option value={eng} title={eng}>{ENGINE_NAMES[eng] ?? eng}</option>
            {/each}
        </select>
        <label class="fb-label" for="fb-min-depth">{$t('stats.minDepthLabel')}</label>
        <input
            id="fb-min-depth"
            class="fb-depth"
            type="number"
            min="0"
            value={filter.minAnalysisDepth}
            title={$t('stats.minDepthHint')}
            onchange={(e) => apply({ minAnalysisDepth: Math.max(0, Number(e.target.value) || 0) })}
        />

        <!-- Reset -->
        <button class="fb-reset" onclick={() => (dateError = !resetStatsFilter())} title={$t('stats.resetFiltersHint')}>{$t('stats.resetFilters')}</button>
    </div>
{/if}

<style>
    .fb-depth {
        width: 4em;
    }
    .filter-bar {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 4px 8px;
        padding: 4px 8px;
        border-bottom: 1px solid var(--color-border);
        font-size: var(--font-size-small);
        flex-shrink: 0;
        min-height: 28px;
        background: var(--panel-bg, var(--color-surface-alt));
    }

    .fb-label {
        color: var(--color-text-muted);
        white-space: nowrap;
        user-select: none;
    }

    .fb-label-inline {
        border: none;
        padding: 0;
        margin: 0;
        font-size: inherit;
        color: var(--color-text-muted);
    }

    /* A long name is cut with an ellipsis, never allowed to widen the bar; the whole select is
       the hit area. */
    .fb-player {
        flex: 0 1 18em;
        min-width: 8em;
        max-width: 100%;
    }

    .fb-player :global(input) {
        text-overflow: ellipsis;
        min-height: calc(1.4em + 12px);
    }

    .fb-clear {
        border: none;
        background: none;
        color: var(--color-text-muted);
        cursor: pointer;
        padding: 0 2px;
        margin-left: -6px;
    }

    .fb-select,
    .fb-date {
        font-size: var(--font-size-small);
        border: 1px solid var(--color-border);
        border-radius: 3px;
        padding: 1px 4px;
        background: var(--color-surface);
        color: inherit;
    }

    .fb-select {
        cursor: pointer;
    }

    .fb-select:disabled {
        cursor: default;
    }

    .fb-date.date-error {
        border-color: var(--color-danger);
    }

    /* ── Tournament dropdown ── */
    .fb-tour-wrap {
        position: relative;
    }

    .fb-tour-btn {
        font-size: var(--font-size-small);
        padding: 1px 6px;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        background: var(--color-surface);
        cursor: pointer;
        white-space: nowrap;
        color: var(--color-text-muted);
    }

    .fb-tour-btn.filtered {
        border-color: var(--color-primary);
        color: color-mix(in srgb, var(--color-primary) 80%, var(--color-text));
        font-weight: 600;
    }

    .fb-tour-dropdown {
        position: absolute;
        top: calc(100% + 2px);
        left: 0;
        z-index: 100;
        background: var(--color-surface);
        border: 1px solid var(--color-border);
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
        font-size: var(--font-size-small);
    }

    .fb-check-label:hover {
        background: color-mix(in srgb, var(--color-primary) 6%, var(--color-surface));
    }

    .fb-tour-all {
        font-weight: 600;
    }

    .fb-tour-sep {
        border: none;
        border-top: 1px solid var(--color-border);
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
        font-size: var(--font-size-small);
        padding: 1px 5px;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        background: var(--color-surface-alt);
        color: var(--color-text);
        cursor: pointer;
        line-height: 1.4;
    }

    .fb-ml-btn.active {
        background: var(--color-primary);
        color: var(--color-surface);
        border-color: var(--color-primary);
    }

    .fb-ml-all {
        font-size: var(--font-size-small);
        padding: 1px 6px;
        border: 1px solid var(--color-primary);
        border-radius: 3px;
        background: color-mix(in srgb, var(--color-primary) 8%, var(--color-surface));
        color: color-mix(in srgb, var(--color-primary) 80%, var(--color-text));
        cursor: pointer;
        line-height: 1.4;
    }

    .fb-ml-all:hover {
        background: color-mix(in srgb, var(--color-primary) 16%, var(--color-surface));
    }

    /* ── Reset ── */
    .fb-reset {
        margin-left: auto;
        font-size: var(--font-size-small);
        padding: 1px 6px;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        background: var(--color-surface-alt);
        color: var(--color-text);
        cursor: pointer;
        white-space: nowrap;
    }

    .fb-reset:hover {
        background: color-mix(in srgb, var(--color-text) 6%, var(--color-surface-alt));
    }

    /* Every control of the bar takes its height from its content, so none clips its text and
       they all line up. */
    .fb-select,
    .fb-date,
    .fb-depth,
    .fb-tour-btn,
    .fb-ml-btn,
    .fb-ml-all,
    .fb-reset {
        box-sizing: border-box;
        height: auto;
        min-height: calc(1.4em + 12px);
        font-size: var(--font-size-small);
        line-height: 1.4;
        padding: 2px 6px;
    }
</style>
