<script>
    import { get } from 'svelte/store';
    import { statsFilterStore } from '../../stores/statsStore.js';
    import { loadPositionsFromStatsSelection } from '../../services/positionLoader.js';
    import { t, translate } from '../../i18n/index.js';
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
                        return [translate('stats.tooltipDecisions', { n: c.NumDecisions }), translate('stats.tooltipBlunders', { n: c.BlunderCount, rate })];
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

    // ── 1b. Cube error directions ─────────────────────────────────────────────
    // Section 1 says how much the cube costs; this one says in which direction
    // it goes wrong. Offering and answering are two decisions taken by two
    // different players, hence two rows and never one score: a player can be
    // late to double while taking too wide, and an average calls that
    // "balanced".
    let directions = $derived(result?.CubeDirections ?? null);
    let directionTotal = $derived(
        !directions ? 0 : directions.Offer.Right + directions.Offer.Missed + directions.Offer.Premature + directions.Answer.Right + directions.Answer.WrongPass + directions.Answer.WrongTake
    );

    let directionRows = $derived(
        !directions
            ? []
            : [
                  {
                      axis: $t('stats.cubeAxisOffer'),
                      cells: [
                          { cell: 'offer_right', label: $t('stats.cubeRight'), n: directions.Offer.Right, mp: 0 },
                          {
                              cell: 'offer_missed',
                              label: $t('stats.cubeOfferMissed'),
                              n: directions.Offer.Missed,
                              mp: directions.Offer.MissedMP
                          },
                          {
                              cell: 'offer_premature',
                              label: $t('stats.cubeOfferPremature'),
                              n: directions.Offer.Premature,
                              mp: directions.Offer.PrematureMP
                          }
                      ]
                  },
                  {
                      axis: $t('stats.cubeAxisAnswer'),
                      cells: [
                          { cell: 'answer_right', label: $t('stats.cubeRight'), n: directions.Answer.Right, mp: 0 },
                          {
                              cell: 'answer_wrong_pass',
                              label: $t('stats.cubeAnswerWrongPass'),
                              n: directions.Answer.WrongPass,
                              mp: directions.Answer.WrongPassMP
                          },
                          {
                              cell: 'answer_wrong_take',
                              label: $t('stats.cubeAnswerWrongTake'),
                              n: directions.Answer.WrongTake,
                              mp: directions.Answer.WrongTakeMP
                          }
                      ]
                  }
              ]
    );

    function handleDirectionClick(cell, n) {
        if (n === 0) return; // nothing behind an empty cell
        const filter = get(statsFilterStore);
        loadPositionsFromStatsSelection(filter, { Kind: 'cube_direction', CubeCell: cell });
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
    <p class="empty-state">{$t('stats.noDecisionsEmpty')}</p>
{:else}
    <!-- ── 1. Cube action breakdown ────────────────────────────────────────── -->
    <section class="chart-section">
        <h3 class="section-title">{$t('stats.breakdownCubeAction', { metric: yAxisLabel() })}</h3>
        {#if !hasCubeData}
            <p class="empty-subsection">{$t('stats.noCubeDecisions')}</p>
        {:else}
            <div class="chart-wrapper">
                <BarChart labels={cubeLabels} datasets={cubeDatasets} options={cubeChartOptions} onBarClick={handleCubeBarClick} />
            </div>
        {/if}
    </section>

    <!-- ── 1b. Cube error directions ──────────────────────────────────────── -->
    <section class="chart-section">
        <h3 class="section-title">{$t('stats.cubeDirections')}</h3>
        {#if directionTotal === 0}
            <p class="empty-subsection">{$t('stats.noCubeDecisions')}</p>
        {:else}
            <table class="direction-table">
                <tbody>
                    {#each directionRows as row (row.axis)}
                        <tr>
                            <th scope="row">{row.axis}</th>
                            {#each row.cells as c (c.cell)}
                                <td>
                                    <button
                                        type="button"
                                        class="direction-cell"
                                        class:is-empty={c.n === 0}
                                        disabled={c.n === 0}
                                        title={c.mp > 0 ? `${(c.mp / 1000).toFixed(3)}` : ''}
                                        onclick={() => handleDirectionClick(c.cell, c.n)}
                                    >
                                        <span class="direction-count">{c.n}</span>
                                        <span class="direction-label">{c.label}</span>
                                    </button>
                                </td>
                            {/each}
                        </tr>
                    {/each}
                </tbody>
            </table>
        {/if}
    </section>

    <!-- ── 2. Checker vs Cube ─────────────────────────────────────────────── -->
    <section class="chart-section">
        <h3 class="section-title">{$t('stats.checkerVsCube', { metric: yAxisLabel() })}</h3>
        <div class="chart-wrapper chart-wrapper--small">
            <BarChart labels={['Checker', 'Cube']} datasets={compDatasets} options={compChartOptions} onBarClick={handleCompBarClick} />
        </div>
    </section>

    <!-- ── 3. Error magnitude histogram ──────────────────────────────────── -->
    <section class="chart-section">
        <h3 class="section-title">{$t('stats.errorMagnitudeHistogram')}</h3>
        {#if !hasHistData}
            <p class="empty-subsection">{$t('stats.noErrors')}</p>
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
        font-size: var(--font-size-base);
        text-align: center;
        padding: 32px 16px;
    }

    .empty-subsection {
        color: #aaa;
        font-size: var(--font-size-base);
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
        font-size: var(--font-size-base);
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

    /* ── Cube error directions ── */
    .direction-table {
        width: 100%;
        border-collapse: collapse;
        table-layout: fixed;
    }

    .direction-table th {
        text-align: left;
        font-weight: 500;
        color: #555;
        width: 22%;
        padding-right: 8px;
    }

    .direction-table td {
        padding: 2px;
    }

    .direction-cell {
        width: 100%;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 2px;
        padding: 6px 4px;
        border: 1px solid #ddd;
        border-radius: 3px;
        background: transparent;
        cursor: pointer;
    }

    .direction-cell:hover:not(:disabled) {
        background: #f0f0f0;
    }

    .direction-cell.is-empty {
        cursor: default;
        opacity: 0.45;
    }

    .direction-count {
        font-weight: 600;
    }

    .direction-label {
        font-size: var(--font-size-small);
        color: #777;
        text-align: center;
        line-height: 1.2;
    }
</style>
