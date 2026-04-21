<script>
    import { get } from 'svelte/store';
    import { statsFilterStore } from '../../stores/statsStore.js';
    import { loadPositionsFromStatsSelection } from '../../services/positionLoader.js';
    import BarChart from './charts/BarChart.svelte';
    import Histogram from './charts/Histogram.svelte';
    import { PRIMARY } from './charts/palette.js';

    /** @type {{ result: import('../../stores/statsStore.js').StatsResult|null, metric: string }} */
    let { result = null, metric = 'pr' } = $props();

    // ── Derived: summary ──────────────────────────────────────────────────────
    let numDecisions = $derived(result?.Totals?.NumDecisions ?? 0);

    // ── 1. Cube action breakdown ──────────────────────────────────────────────
    let cubeBreakdown = $derived(result?.CubeActionBreakdown ?? []);
    let hasCubeData = $derived(cubeBreakdown.length > 0);

    let cubeLabels = $derived(cubeBreakdown.map((c) => c.Action));

    let cubeDatasets = $derived([
        {
            label: metric === 'pr' ? 'PR' : 'MWC loss',
            data: cubeBreakdown.map((c) => (metric === 'pr' ? c.PR : c.MWC)),
            backgroundColor: PRIMARY,
            borderColor: PRIMARY,
            borderWidth: 1
        }
    ]);

    const cubeChartOptions = {
        plugins: {
            legend: { display: false },
            tooltip: {
                callbacks: {
                    afterBody: (items) => {
                        const idx = items[0]?.dataIndex;
                        if (idx == null) return [];
                        const c = cubeBreakdown[idx];
                        if (!c) return [];
                        const rate = blunderRate(c);
                        return [`Décisions : ${c.NumDecisions}`, `Blunders  : ${c.BlunderCount} (${rate}%)`];
                    }
                }
            }
        },
        scales: { y: { beginAtZero: true } }
    };

    function handleCubeBarClick(dataIndex) {
        const c = cubeBreakdown[dataIndex];
        if (!c) return;
        const filter = get(statsFilterStore);
        loadPositionsFromStatsSelection(filter, {
            Kind: 'cube_action',
            CubeAction: c.Action,
            OnlyWithError: true
        });
    }

    // ── 2. Checker vs Cube comparison ─────────────────────────────────────────
    let compDatasets = $derived([
        {
            label: metric === 'pr' ? 'PR' : 'MWC loss',
            data: [metric === 'pr' ? (result?.PRChecker ?? 0) : (result?.MWCChecker ?? 0), metric === 'pr' ? (result?.PRCube ?? 0) : (result?.MWCCube ?? 0)],
            backgroundColor: PRIMARY,
            borderColor: PRIMARY,
            borderWidth: 1
        }
    ]);

    const compChartOptions = {
        plugins: { legend: { display: false } },
        scales: { y: { beginAtZero: true } }
    };

    function handleCompBarClick(dataIndex) {
        const kind = dataIndex === 0 ? 'checker' : 'cube';
        const filter = get(statsFilterStore);
        loadPositionsFromStatsSelection(filter, { Kind: kind, OnlyWithError: true });
    }

    // ── 3. Error magnitude histogram ──────────────────────────────────────────
    let histogram = $derived(result?.ErrorHistogram ?? []);
    let hasHistData = $derived(histogram.some((b) => b.Count > 0));

    let histLabels = $derived(histogram.map(bucketLabel));
    let histDatasets = $derived([
        {
            label: 'Positions',
            data: histogram.map((b) => b.Count),
            backgroundColor: PRIMARY,
            borderColor: PRIMARY,
            borderWidth: 1
        }
    ]);

    const histChartOptions = {
        plugins: { legend: { display: false } },
        scales: { y: { beginAtZero: true } }
    };

    function handleHistBarClick(dataIndex) {
        const b = histogram[dataIndex];
        if (!b) return;
        const filter = get(statsFilterStore);
        loadPositionsFromStatsSelection(filter, {
            Kind: 'error_bucket',
            BucketMinMP: b.MinMP,
            BucketMaxMP: b.MaxMP
        });
    }

    // ── Helpers ───────────────────────────────────────────────────────────────

    /** Build a human-readable bucket label in EMG units. */
    function bucketLabel(bucket) {
        const lo = (bucket.MinMP / 1000).toFixed(3);
        if (bucket.MaxMP < 0) return `\u2265${lo}`;
        const hi = (bucket.MaxMP / 1000).toFixed(3);
        return `${lo}\u2013${hi}`;
    }

    /** Blunder rate as a percentage string. */
    function blunderRate(c) {
        if (!c || c.NumDecisions === 0) return '0.0';
        return ((c.BlunderCount / c.NumDecisions) * 100).toFixed(1);
    }

    /** Y-axis label based on metric. */
    function yAxisLabel() {
        return metric === 'pr' ? 'PR' : 'MWC loss';
    }
</script>

{#if !result || numDecisions === 0}
    <p class="empty-state">Aucune décision sur la période filtrée. Élargissez les filtres.</p>
{:else}
    <!-- ── 1. Cube action breakdown ────────────────────────────────────────── -->
    <section class="chart-section">
        <h3 class="section-title">Breakdown par action cube ({yAxisLabel()})</h3>
        {#if !hasCubeData}
            <p class="empty-subsection">Aucune décision cube sur la période.</p>
        {:else}
            <div class="chart-wrapper">
                <BarChart labels={cubeLabels} datasets={cubeDatasets} options={cubeChartOptions} onBarClick={handleCubeBarClick} />
            </div>
        {/if}
    </section>

    <!-- ── 2. Checker vs Cube ─────────────────────────────────────────────── -->
    <section class="chart-section">
        <h3 class="section-title">Checker vs Cube ({yAxisLabel()})</h3>
        <div class="chart-wrapper chart-wrapper--small">
            <BarChart labels={['Checker', 'Cube']} datasets={compDatasets} options={compChartOptions} onBarClick={handleCompBarClick} />
        </div>
    </section>

    <!-- ── 3. Error magnitude histogram ──────────────────────────────────── -->
    <section class="chart-section">
        <h3 class="section-title">Répartition des magnitudes d'erreur (nb positions)</h3>
        {#if !hasHistData}
            <p class="empty-subsection">Aucune erreur dans cette période.</p>
        {:else}
            <div class="chart-wrapper">
                <Histogram labels={histLabels} datasets={histDatasets} options={histChartOptions} onBarClick={handleHistBarClick} />
            </div>
        {/if}
    </section>
{/if}

<style>
    /* ── Shared ── */
    .empty-state {
        color: #888;
        font-size: 13px;
        text-align: center;
        padding: 32px 16px;
    }

    .empty-subsection {
        color: #aaa;
        font-size: 12px;
        text-align: center;
        padding: 12px 16px;
        margin: 0;
        font-style: italic;
    }

    /* ── Section layout ── */
    .chart-section {
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

    /* ── Chart wrappers ── */
    .chart-wrapper {
        position: relative;
        height: 160px;
        width: 100%;
    }

    .chart-wrapper--small {
        height: 120px;
    }
</style>
