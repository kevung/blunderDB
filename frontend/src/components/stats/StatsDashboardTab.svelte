<script>
    import { get } from 'svelte/store';
    import { statsFilterStore } from '../../stores/statsStore.js';
    import { loadPositionsFromStatsSelection, openMatchInPanel } from '../../services/positionLoader.js';
    import { t } from '../../i18n/index.js';

    /** @type {{ result: import('../../stores/statsStore.js').StatsResult|null, metric: string }} */
    let { result = null, metric = 'pr' } = $props();

    const ROLLING_NS = [5, 10, 50, 100, 250, 500, 1000];

    /** Format a PR value (millipoint → display). Returns '—' for unavailable. */
    function fmtPR(val) {
        if (val == null || isNaN(val)) return '—';
        return val.toFixed(2);
    }

    /** Format a MWC cumulative loss. Returns '—' for unavailable. */
    function fmtMWC(val) {
        if (val == null || isNaN(val)) return '—';
        return (val * 100).toFixed(2) + '%';
    }

    /** Return the display value for a main card based on current metric. */
    function cardValue(kind) {
        if (!result) return '—';
        if (metric === 'pr') {
            if (kind === 'all') return fmtPR(result.PRGlobal);
            if (kind === 'checker') return fmtPR(result.PRChecker);
            if (kind === 'cube') return fmtPR(result.PRCube);
        } else {
            if (kind === 'all') return fmtMWC(result.MWCGlobal);
            if (kind === 'checker') return fmtMWC(result.MWCChecker);
            if (kind === 'cube') return fmtMWC(result.MWCCube);
        }
        return '—';
    }

    /** Return the rolling value for N decisions. */
    function rollingValue(n) {
        if (!result) return null;
        const map = metric === 'pr' ? result.PRRolling : result.MWCRolling;
        if (!map) return null;
        const v = map[n];
        return v != null && !isNaN(v) ? v : null;
    }

    /** How many decisions are actually available for rolling N. */
    function rollingDecisions(n) {
        if (!result || !result.Totals) return 0;
        return Math.min(n, result.Totals.NumDecisions);
    }

    /** Format rolling value for display. */
    function fmtRolling(n) {
        const v = rollingValue(n);
        if (v == null) return '—';
        return metric === 'pr' ? v.toFixed(2) : (v * 100).toFixed(2) + '%';
    }

    /** Format an error from a blunder entry. */
    function fmtBlunderError(entry) {
        if (metric === 'mwc' && entry.MWCLoss > 0) return (entry.MWCLoss * 100).toFixed(2) + '%';
        if (entry.ErrorMP == null) return '—';
        return (entry.ErrorMP / 1000).toFixed(3);
    }

    /** Short date display (YYYY-MM-DD → YYYY-MM-DD or first 10 chars). */
    function shortDate(dateStr) {
        if (!dateStr) return '';
        return dateStr.slice(0, 10);
    }

    /** Number of decisions for tooltip. */
    function numDecisions() {
        return result?.Totals?.NumDecisions ?? 0;
    }

    async function openCard(kind) {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: kind, OnlyWithError: false });
    }

    async function openRollingN(n) {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'last_n', LastN: n });
    }

    async function openBlunder(positionID) {
        const filter = get(statsFilterStore);
        await loadPositionsFromStatsSelection(filter, { Kind: 'position', PositionID: positionID });
    }

    function openMatch(matchID) {
        openMatchInPanel(matchID);
    }
</script>

{#if !result || result.Totals.NumDecisions === 0}
    <p class="empty-state">{$t('stats.noDecisionsEmpty')}</p>
{:else}
    <!-- ── Main cards ────────────────────────────────────────────── -->
    <div class="cards-grid">
        {#each [{ kind: 'all', label: $t('stats.cardLabelAll'), unit: metric === 'pr' ? 'PR' : 'MWC loss' }, { kind: 'checker', label: $t('stats.cardLabelChecker'), unit: metric === 'pr' ? 'PR' : 'MWC loss' }, { kind: 'cube', label: $t('stats.cardLabelCube'), unit: metric === 'pr' ? 'PR' : 'MWC loss' }] as card (card.kind)}
            <button class="stat-card" onclick={() => openCard(card.kind)} aria-label="Open {numDecisions()} positions — {card.label}" title="{numDecisions()} decisions · click to open positions">
                <span class="card-value">{cardValue(card.kind)}</span>
                <span class="card-label">{card.label}</span>
                <span class="card-unit">{card.unit}</span>
            </button>
        {/each}
    </div>

    <!-- ── Totals line ──────────────────────────────────────────── -->
    <p class="stats-totals">
        {result.Totals.NumTournaments}
        {$t('stats.tournaments')} ·
        {result.Totals.NumMatches}
        {$t('stats.matches')} ·
        {result.Totals.NumDecisions}
        {$t('stats.decisions')}
    </p>

    <!-- ── Rolling N ────────────────────────────────────────────── -->
    <section class="rolling-section">
        <h3 class="section-title">{$t('stats.rolling', { metric: metric === 'pr' ? 'PR' : 'MWC loss' })}</h3>
        <div class="rolling-row">
            {#each ROLLING_NS as n (n)}
                {@const avail = rollingDecisions(n) >= n}
                <button class="rolling-cell" class:unavailable={!avail} onclick={() => avail && openRollingN(n)} disabled={!avail} title="{rollingDecisions(n)} decisions used">
                    <span class="rolling-n">N={n}</span>
                    <span class="rolling-val">{avail ? fmtRolling(n) : '—'}</span>
                </button>
            {/each}
        </div>
    </section>

    <!-- ── Top blunders ─────────────────────────────────────────── -->
    {#if result.TopBlunders && result.TopBlunders.length > 0}
        <section class="blunders-section">
            <h3 class="section-title">{$t('stats.topBlunders')}</h3>
            <ol class="blunders-list">
                {#each result.TopBlunders as entry, i (entry.PositionID)}
                    <li class="blunder-item">
                        <button
                            class="blunder-main"
                            onclick={() => openBlunder(entry.PositionID)}
                            title="{entry.DecisionType === 1 ? $t('stats.decisionTypeCube') : $t('stats.decisionTypeChecker')} — {$t('stats.clickToOpenPosition')}"
                        >
                            <span class="blunder-rank">#{i + 1}</span>
                            <span class="blunder-type">{entry.DecisionType === 1 ? $t('stats.decisionTypeCube') : $t('stats.decisionTypeChecker')}</span>
                            <span class="blunder-error">{fmtBlunderError(entry)}</span>
                            <span class="blunder-match">
                                {entry.PlayerNames || 'Match #' + entry.MatchID}
                                {#if entry.MatchDate}
                                    <span class="blunder-date">{shortDate(entry.MatchDate)}</span>
                                {/if}
                            </span>
                        </button>
                        <button class="blunder-open-match" onclick={() => openMatch(entry.MatchID)} title={$t('stats.openMatch')} aria-label="{$t('stats.openMatch')} #{entry.MatchID}">↗</button>
                    </li>
                {/each}
            </ol>
        </section>
    {/if}
{/if}

<style>
    /* ── Empty state ── */
    .empty-state {
        color: #888;
        font-size: 13px;
        text-align: center;
        padding: 32px 16px;
    }

    /* ── Cards ── */
    .cards-grid {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 12px;
        padding: 16px 16px 0;
    }

    @media (max-width: 600px) {
        .cards-grid {
            grid-template-columns: 1fr;
        }
    }

    .stat-card {
        background: #f5f5f5;
        border: 1px solid #e0e0e0;
        border-radius: 6px;
        padding: 14px 12px;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 4px;
        cursor: pointer;
        text-align: center;
        transition: background 0.15s;
    }

    .stat-card:hover {
        background: #ececec;
    }

    .card-value {
        font-size: 28px;
        font-variant-numeric: tabular-nums;
        font-weight: 600;
        color: #1a1a1a;
        line-height: 1.1;
    }

    .card-label {
        font-size: 11px;
        color: #555;
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }

    .card-unit {
        font-size: 10px;
        color: #999;
    }

    /* ── Totals line ── */
    .stats-totals {
        font-size: 12px;
        color: #777;
        padding: 8px 16px 0;
        margin: 0;
    }

    /* ── Rolling N ── */
    .rolling-section {
        padding: 12px 16px 0;
    }

    .section-title {
        font-size: 12px;
        font-weight: 600;
        color: #555;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        margin: 0 0 8px;
    }

    .rolling-row {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
    }

    .rolling-cell {
        background: #f5f5f5;
        border: 1px solid #e0e0e0;
        border-radius: 4px;
        padding: 6px 10px;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 2px;
        cursor: pointer;
        min-width: 56px;
    }

    .rolling-cell:not(.unavailable):hover {
        background: #ececec;
    }

    .rolling-cell.unavailable {
        cursor: default;
        opacity: 0.45;
    }

    .rolling-n {
        font-size: 10px;
        color: #888;
    }

    .rolling-val {
        font-size: 13px;
        font-variant-numeric: tabular-nums;
        font-weight: 600;
        color: #1a1a1a;
    }

    /* ── Blunders ── */
    .blunders-section {
        padding: 12px 16px 16px;
    }

    .blunders-list {
        list-style: none;
        margin: 0;
        padding: 0;
    }

    .blunder-item {
        display: flex;
        align-items: center;
        gap: 4px;
        border-bottom: 1px solid #f0f0f0;
    }

    .blunder-item:last-child {
        border-bottom: none;
    }

    .blunder-main {
        flex: 1;
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 6px 4px;
        background: transparent;
        border: none;
        text-align: left;
        cursor: pointer;
        border-radius: 4px;
        font-size: 12px;
        color: #1a1a1a;
    }

    .blunder-main:hover {
        background: #f5f5f5;
    }

    .blunder-rank {
        color: #999;
        min-width: 24px;
        font-variant-numeric: tabular-nums;
    }

    .blunder-type {
        color: #555;
        min-width: 48px;
        font-size: 11px;
    }

    .blunder-error {
        font-variant-numeric: tabular-nums;
        font-weight: 600;
        min-width: 52px;
        color: #b71c1c;
    }

    .blunder-match {
        flex: 1;
        color: #444;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .blunder-date {
        color: #888;
        margin-left: 4px;
        font-size: 11px;
    }

    .blunder-open-match {
        background: transparent;
        border: none;
        color: #999;
        cursor: pointer;
        padding: 4px 6px;
        font-size: 13px;
        border-radius: 3px;
        flex-shrink: 0;
    }

    .blunder-open-match:hover {
        color: #1976d2;
        background: #e3f2fd;
    }

    @media (max-width: 600px) {
        .blunder-match {
            display: none;
        }
    }
</style>
