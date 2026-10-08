<script>
    // Les positions de référence proposées (ADR-0079) : le moteur choisit et
    // donne les composantes de chaque raison ; ce panneau les met en phrases,
    // laisse cocher, et fait du choix une collection, un paquet ou un quiz.
    import { onMount } from 'svelte';
    import { get } from 'svelte/store';
    import { SvelteSet } from 'svelte/reactivity';
    import { SuggestReferencePositions, GetAllTournaments } from '../../wailsjs/go/database/Database.js';
    import { statsFilterStore } from '../stores/statsStore.js';
    import { matchContextStore, lastVisitedMatchStore } from '../stores/positionStore.js';
    import { tournamentsStore } from '../stores/tournamentStore.js';
    import { collectionFromIds, deckFromIds, quizOnIds } from '../services/recurringStudy.js';
    import { loadPositionsFromSelection } from '../services/positionLoader.js';
    import { t } from '../i18n';
    import { logger } from '../utils/logger.js';

    let { onClose = () => {}, onCollectionCreated = () => {} } = $props();

    const SIZES = [10, 20, 50];

    let scope = $state('library');
    let size = $state(20);
    let tournamentId = $state(0);
    let loading = $state(false);
    let error = $state('');
    let result = $state.raw(null);
    let checked = new SvelteSet();
    let name = $state('');

    let player = $derived($statsFilterStore?.playerName || '');
    let matchId = $derived($matchContextStore?.matchID || $lastVisitedMatchStore?.matchID || 0);
    let tournaments = $derived($tournamentsStore ?? []);
    let references = $derived(result?.References ?? []);
    let keptIds = $derived(references.filter((r) => checked.has(r.PositionID)).map((r) => r.PositionID));

    onMount(async () => {
        if (get(tournamentsStore).length === 0) {
            try {
                tournamentsStore.set((await GetAllTournaments()) ?? []);
            } catch (e) {
                logger.error('could not list the tournaments:', e);
            }
        }
    });

    /** The filter of the chosen scope: the Statistics player, narrowed or not. */
    function filterFor() {
        const f = get(statsFilterStore);
        if (scope === 'filter') return { filter: f, matchIDs: [] };
        const base = { playerName: f.playerName, playerAliases: f.playerAliases ?? [], tournamentIDs: [], dateFrom: '', dateTo: '', decisionType: -1, matchLength: [] };
        if (scope === 'tournament') return { filter: { ...base, tournamentIDs: tournamentId ? [tournamentId] : [] }, matchIDs: [] };
        if (scope === 'match') return { filter: base, matchIDs: matchId ? [matchId] : [] };
        return { filter: base, matchIDs: [] };
    }

    async function suggest() {
        loading = true;
        error = '';
        try {
            const { filter, matchIDs } = filterFor();
            result = await SuggestReferencePositions(filter, matchIDs, size);
            checked.clear();
            for (const r of result?.References ?? []) checked.add(r.PositionID);
            name = $t('collection.suggestDefaultName', { date: new Date().toISOString().slice(0, 10) });
        } catch (e) {
            logger.error('could not suggest reference positions:', e);
            error = String(e);
            result = null;
        } finally {
            loading = false;
        }
    }

    function toggle(id) {
        if (checked.has(id)) checked.delete(id);
        else checked.add(id);
    }

    const CUBE_LABELS = {
        offer_missed: 'stats.cubeOfferMissed',
        offer_premature: 'stats.cubeOfferPremature',
        answer_wrong_pass: 'stats.cubeAnswerWrongPass',
        answer_wrong_take: 'stats.cubeAnswerWrongTake'
    };

    function label(key, fallback) {
        const s = $t(key);
        return s === key ? fallback : s;
    }

    function familyName(r) {
        const theme = r.Kind === 'cube' && CUBE_LABELS[r.Theme] ? CUBE_LABELS[r.Theme] : `stats.recurringTheme_${r.Theme}`;
        const parts = [label(`stats.gameType_${r.GameType}`, r.GameType), $t(`stats.recurringKind_${r.Kind}`), label(theme, r.Theme)];
        if (r.Kind === 'cube') parts.push($t('collection.suggestScore', { me: r.AwayOnRoll, opp: r.AwayOpponent }));
        return parts.join(' · ');
    }

    /** MWC fraction → percentage points, the unit the Match panel shows a loss in. */
    function pct(x) {
        return `${(100 * x).toFixed(2)} %`;
    }

    /** The reason of a proposal, from the components the engine returned. */
    function reason(r) {
        const parts = [
            $t('collection.suggestCovers', { errors: r.Errors, matches: r.Matches }),
            $t('collection.suggestFamily', { n: r.FamilyErrors, decisions: result?.NumDecisions ?? 0 }),
            $t('collection.suggestOwn', { pct: pct(r.Excess) })
        ];
        if (r.GapMP >= 0) parts.push($t('collection.suggestGap', { mp: r.GapMP }));
        if (r.Close) parts.push($t('collection.suggestClose'));
        if (r.Unstable) parts.push($t('collection.suggestUnstable'));
        if (r.Gain < r.Covered) parts.push($t('collection.suggestCounted', { pct: pct(r.Gain) }));
        if (r.RolledOut) parts.push($t('collection.suggestRolledOut'));
        return parts.join(' ; ');
    }

    /** Browse the kept positions, opening on the one clicked; the list runs in id order. */
    function open(id) {
        const ids = (keptIds.includes(id) ? [...keptIds] : [...keptIds, id]).sort((a, b) => a - b);
        loadPositionsFromSelection(ids, { focusIndex: ids.indexOf(id) });
    }

    async function keepAsCollection() {
        if (await collectionFromIds(name.trim() || $t('collection.suggestTitle'), keptIds)) {
            onCollectionCreated();
            result = null;
        }
    }
</script>

<div class="suggest" data-testid="reference-suggest">
    <div class="suggest-head">
        <span class="detail-title">{$t('collection.suggestTitle')}</span>
        <button class="icon-btn" type="button" onclick={onClose} title={$t('collection.backToCollections')}>←</button>
    </div>
    <p class="hint">{$t('collection.suggestHint')}</p>
    <p class="hint">{player ? $t('collection.suggestPlayer', { name: player }) : $t('collection.suggestAllPlayers')}</p>
    <div class="controls">
        <label>
            {$t('collection.suggestScope')}
            <select bind:value={scope} data-testid="suggest-scope">
                <option value="library">{$t('collection.suggestScopeLibrary')}</option>
                <option value="match" disabled={!matchId}>{$t('collection.suggestScopeMatch')}</option>
                <option value="tournament" disabled={tournaments.length === 0}>{$t('collection.suggestScopeTournament')}</option>
                <option value="filter">{$t('collection.suggestScopeFilter')}</option>
            </select>
        </label>
        {#if scope === 'tournament'}
            <select bind:value={tournamentId} data-testid="suggest-tournament">
                {#each tournaments as tour (tour.id)}
                    <option value={tour.id}>{tour.name}</option>
                {/each}
            </select>
        {/if}
        <label>
            {$t('collection.suggestSize')}
            <select bind:value={size} data-testid="suggest-size">
                {#each SIZES as n (n)}
                    <option value={n}>{n}</option>
                {/each}
            </select>
        </label>
        <button type="button" class="study-btn" data-testid="suggest-run" onclick={suggest} disabled={loading || (scope === 'tournament' && !tournamentId)}>{$t('collection.suggestRun')}</button>
    </div>

    {#if loading}
        <p class="empty">{$t('stats.loading')}</p>
    {:else if error}
        <p class="empty">{error}</p>
    {:else if result}
        <p class="hint">{$t('collection.suggestSummary', { candidates: result.Candidates, handled: result.Handled, radius: result.Radius })}</p>
        {#if references.length === 0}
            <p class="empty">{$t('collection.suggestNone')}</p>
        {:else}
            <ol class="refs">
                {#each references as r (r.PositionID)}
                    <li class="ref" data-testid="suggest-row">
                        <input type="checkbox" checked={checked.has(r.PositionID)} onchange={() => toggle(r.PositionID)} aria-label={String(r.PositionID)} />
                        <button type="button" class="ref-body" onclick={() => open(r.PositionID)}>
                            <span class="ref-title">{familyName(r)} <span class="gain">{pct(r.Covered)}</span></span>
                            <span class="ref-reason">{reason(r)}</span>
                        </button>
                    </li>
                {/each}
            </ol>
            <div class="keep">
                <input class="name-input" type="text" bind:value={name} placeholder={$t('collection.newCollectionPlaceholder')} />
                <button type="button" class="study-btn" data-testid="suggest-collection" disabled={keptIds.length === 0} onclick={keepAsCollection}
                    >{$t('collection.suggestKeepCollection', { n: keptIds.length })}</button
                >
                <button type="button" class="study-btn" data-testid="suggest-deck" disabled={keptIds.length === 0} onclick={() => deckFromIds(name.trim() || $t('collection.suggestTitle'), keptIds)}
                    >{$t('collection.suggestKeepDeck')}</button
                >
                <button type="button" class="study-btn" data-testid="suggest-quiz" disabled={keptIds.length === 0} onclick={() => quizOnIds(keptIds)}>{$t('collection.suggestQuiz')}</button>
            </div>
        {/if}
    {/if}
</div>

<style>
    .suggest {
        padding: 6px 10px;
        overflow-y: auto;
        height: 100%;
        box-sizing: border-box;
    }
    .suggest-head {
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    .detail-title {
        font-weight: 600;
    }
    .hint {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        margin: 4px 0;
    }
    .controls,
    .keep {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        align-items: center;
        margin: 6px 0;
        font-size: var(--font-size-small);
    }
    .empty {
        color: var(--color-text-muted);
        font-style: italic;
        text-align: center;
        padding: 8px;
        margin: 0;
    }
    .refs {
        margin: 4px 0;
        padding-left: 1.6em;
    }
    .ref {
        display: flex;
        gap: 6px;
        align-items: flex-start;
        border-bottom: 1px solid var(--color-border);
        padding: 3px 0;
    }
    .ref-body {
        display: flex;
        flex-direction: column;
        text-align: left;
        background: none;
        border: none;
        padding: 0;
        cursor: pointer;
        color: inherit;
    }
    .ref-body:hover {
        background: var(--color-surface-alt);
    }
    .ref-title {
        font-weight: 500;
    }
    .gain {
        font-variant-numeric: tabular-nums;
        color: var(--color-text-muted);
        margin-left: 6px;
    }
    .ref-reason {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }
    .study-btn {
        font-size: var(--font-size-small);
        padding: 1px 8px;
    }
    .icon-btn {
        background: none;
        border: none;
        cursor: pointer;
    }
    .name-input {
        flex: 1;
        min-width: 10em;
    }
</style>
