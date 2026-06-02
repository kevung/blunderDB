<script>
    import { loadPositionsFromTournament, loadPositionsFromMatch, openTournamentInPanel, openMatchInPanel } from '../../services/positionLoader.js';
    import { t, translate } from '../../i18n/index.js';
    import LineChart from './charts/LineChart.svelte';
    import ScatterChart from './charts/ScatterChart.svelte';
    import ContextMenu from './ContextMenu.svelte';
    import { GRADE_BANDS, gradeForPR, makeGradeBandPlugin } from './gradeBands.js';
    import { PRIMARY, PRIMARY_ALPHA } from './charts/palette.js';

    /** @type {{ result: import('../../stores/statsStore.js').StatsResult|null, metric: string }} */
    let { result = null, metric = 'pr' } = $props();

    // ── Context menu ──────────────────────────────────────────────────────────
    /** @type {{ x: number, y: number, items: Array<{ label: string, onClick: () => void }> } | null} */
    let contextMenu = $state(null);

    function showMenu(nativeEvent, items) {
        const x = nativeEvent?.clientX ?? 0;
        const y = nativeEvent?.clientY ?? 0;
        contextMenu = { x, y, items };
    }

    // ── Grade-band plugin (shared instance for both charts) ───────────────────
    const gradeBandPlugin = makeGradeBandPlugin(GRADE_BANDS);

    // ── Derived data ──────────────────────────────────────────────────────────
    let tournaments = $derived(result?.PerTournament ?? []);
    let matches = $derived(result?.PerMatch ?? []);

    // ── Tournament line chart ─────────────────────────────────────────────────
    let tourLabels = $derived(tournaments.map((tour) => truncateLabel(tour.Name)));

    let tourDatasets = $derived([
        {
            label: metric === 'pr' ? 'PR' : 'MWC loss',
            data: tournaments.map((tour) => (metric === 'pr' ? tour.PR : tour.MWC)),
            borderColor: PRIMARY,
            backgroundColor: PRIMARY_ALPHA,
            tension: 0.3,
            pointRadius: 5,
            pointHoverRadius: 7,
            fill: false
        }
    ]);

    const tourChartOptions = {
        plugins: { legend: { display: false } },
        scales: {
            y: { beginAtZero: true, title: { display: false } }
        }
    };

    function handleTourClick(dataIndex, _dsIdx, nativeEvent) {
        const tour = tournaments[dataIndex];
        if (!tour) return;
        showMenu(nativeEvent, [
            { label: translate('stats.openTournament'), onClick: () => openTournamentInPanel(tour.ID) },
            { label: translate('stats.openPositions'), onClick: () => loadPositionsFromTournament(tour.ID) }
        ]);
    }

    // ── Match scatter chart ───────────────────────────────────────────────────
    let matchDatasets = $derived([
        {
            label: metric === 'pr' ? 'PR per match' : 'MWC loss per match',
            data: matches.map((m) => ({
                x: parseDateMs(m.Date),
                y: metric === 'pr' ? m.PR : m.MWC
            })),
            backgroundColor: PRIMARY_ALPHA,
            borderColor: PRIMARY,
            pointRadius: matches.map((m) => clampRadius(m.NumDecisions)),
            pointHoverRadius: matches.map((m) => clampRadius(m.NumDecisions) + 2)
        }
    ]);

    const scatterChartOptions = {
        plugins: { legend: { display: false } },
        scales: {
            x: {
                type: 'linear',
                ticks: {
                    callback: (v) => fmtTimestamp(v)
                }
            },
            y: { beginAtZero: true }
        }
    };

    function handleMatchClick(dataIndex, _dsIdx, nativeEvent) {
        const m = matches[dataIndex];
        if (!m) return;
        showMenu(nativeEvent, [
            { label: translate('stats.openMatch'), onClick: () => openMatchInPanel(m.ID) },
            { label: translate('stats.openPositions'), onClick: () => loadPositionsFromMatch(m.ID) }
        ]);
    }

    // ── Helpers ───────────────────────────────────────────────────────────────
    function truncateLabel(name, max = 22) {
        if (!name) return '';
        return name.length > max ? name.slice(0, max - 1) + '…' : name;
    }

    function clampRadius(n) {
        return Math.max(4, Math.min(12, 4 + (n / 500) * 8));
    }

    function parseDateMs(dateStr) {
        if (!dateStr) return 0;
        return new Date(dateStr).getTime();
    }

    function fmtTimestamp(ms) {
        if (!ms) return '';
        return new Date(ms).toISOString().slice(0, 10);
    }

    function fmtDate(dateStr) {
        if (!dateStr) return '';
        return dateStr.slice(0, 10);
    }

    function fmtVal(v) {
        if (v == null || isNaN(v)) return '—';
        return metric === 'pr' ? v.toFixed(2) : (v * 100).toFixed(2) + '%';
    }

    function yAxisLabel() {
        return metric === 'pr' ? 'PR' : 'MWC loss';
    }
</script>

{#if !result || (tournaments.length === 0 && matches.length === 0)}
    <!-- ── Empty state ──────────────────────────────────────────────────────── -->
    <p class="empty-state">{$t('stats.noProgressionData')}</p>
{:else}
    <!-- ── Tournament line chart ────────────────────────────────────────────── -->
    {#if tournaments.length > 0}
        <section class="chart-section">
            <h3 class="section-title">{$t('stats.prPerTournament', { metric: yAxisLabel() })}</h3>

            {#if tournaments.length === 1}
                <!-- Single-tournament: show a card instead of a 1-point curve -->
                {@const tourn = tournaments[0]}
                <div class="single-card">
                    <span class="single-value">{fmtVal(metric === 'pr' ? tourn.PR : tourn.MWC)}</span>
                    <span class="single-label">{tourn.Name || $t('stats.tournament')}</span>
                    <span class="single-meta">{fmtDate(tourn.Date)} · {tourn.NumDecisions} {$t('stats.decisions')}</span>
                    {#if metric === 'pr'}
                        <span class="single-grade">{gradeForPR(tourn.PR)}</span>
                    {/if}
                    <div class="single-actions">
                        <button onclick={() => openTournamentInPanel(tourn.ID)}>{$t('stats.openTournament')}</button>
                        <button onclick={() => loadPositionsFromTournament(tourn.ID)}>{$t('stats.openPositions')}</button>
                    </div>
                </div>
            {:else}
                <div class="chart-wrap">
                    <LineChart labels={tourLabels} datasets={tourDatasets} options={tourChartOptions} plugins={[gradeBandPlugin]} onPointClick={handleTourClick} />
                </div>
            {/if}
        </section>
    {/if}

    <!-- ── Match scatter chart ───────────────────────────────────────────────── -->
    {#if matches.length > 0}
        <section class="chart-section">
            <h3 class="section-title">{$t('stats.prPerMatch', { metric: yAxisLabel() })}</h3>
            <div class="chart-wrap">
                <ScatterChart datasets={matchDatasets} options={scatterChartOptions} plugins={[gradeBandPlugin]} onPointClick={handleMatchClick} />
            </div>
            <p class="chart-hint">{$t('stats.pointSizeHint')}</p>
        </section>
    {/if}

    <!-- ── Grade legend ──────────────────────────────────────────────────────── -->
    <section class="grade-legend">
        {#each GRADE_BANDS as band (band.label)}
            <span class="grade-pill" style="background:{band.color.replace('0.09', '0.25').replace('0.10', '0.25')}">
                {band.label}
                {#if band.max === Infinity}≥{band.min}{:else}{band.min}–{band.max}{/if}
            </span>
        {/each}
    </section>
{/if}

<!-- ── Context menu ──────────────────────────────────────────────────────────── -->
{#if contextMenu}
    <ContextMenu x={contextMenu.x} y={contextMenu.y} items={contextMenu.items} onClose={() => (contextMenu = null)} />
{/if}

<style>
    /* ── Empty state ── */
    .empty-state {
        color: #888;
        font-size: 13px;
        text-align: center;
        padding: 32px 16px;
    }

    /* ── Chart sections ── */
    .chart-section {
        padding: 8px 12px 4px;
    }

    .section-title {
        margin: 0 0 6px;
        font-size: 12px;
        font-weight: 600;
        color: #555;
        text-transform: uppercase;
        letter-spacing: 0.04em;
    }

    .chart-wrap {
        height: 260px;
        position: relative;
    }

    .chart-hint {
        margin: 3px 0 0;
        font-size: 11px;
        color: #aaa;
        text-align: right;
    }

    /* ── Single tournament card ── */
    .single-card {
        display: flex;
        flex-direction: column;
        gap: 4px;
        padding: 12px 16px;
        background: #fafafa;
        border: 1px solid #e8e8e8;
        border-radius: 4px;
    }

    .single-value {
        font-size: 28px;
        font-weight: 700;
        font-variant-numeric: tabular-nums;
        color: #1976d2;
    }

    .single-label {
        font-size: 13px;
        color: #333;
    }

    .single-meta {
        font-size: 11px;
        color: #888;
    }

    .single-grade {
        font-size: 11px;
        color: #555;
        font-style: italic;
    }

    .single-actions {
        display: flex;
        gap: 6px;
        margin-top: 4px;
    }

    .single-actions button {
        font-size: 11px;
        padding: 3px 8px;
        border: 1px solid #ccc;
        border-radius: 3px;
        background: none;
        cursor: pointer;
        color: #333;
    }

    .single-actions button:hover {
        background: #f0f4ff;
    }

    /* ── Grade legend ── */
    .grade-legend {
        display: flex;
        flex-wrap: wrap;
        gap: 4px;
        padding: 6px 12px 10px;
    }

    .grade-pill {
        font-size: 10px;
        padding: 2px 7px;
        border-radius: 10px;
        color: #333;
        white-space: nowrap;
    }
</style>
