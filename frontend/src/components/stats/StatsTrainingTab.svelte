<script>
    // L'entraînement dans le temps : le PR du quiz Décision, le PR des matchs
    // réels et la rétention Anki, sur les mêmes fenêtres calendaires. Le PR du
    // quiz est sur l'échelle du PR réel ; la rétention est sur sa propre échelle
    // (axe de droite, en %). Une fenêtre sans échantillon n'a pas de point : un
    // zéro dessiné se lirait comme un résultat parfait.
    import { t } from '../../i18n/index.js';
    import LineChart from './charts/LineChart.svelte';
    import { PRIMARY } from './charts/palette.js';

    /** @type {{ data: { Window: string, Sessions: Array<object>, Periods: Array<object> }|null, loading?: boolean, error?: string|null, period?: string }} */
    let { data = null, loading = false, error = null, period = $bindable('week') } = $props();

    // Les deux autres séries ont leur propre teinte, hors thème : un canvas ne lit pas les jetons CSS.
    const QUIZ = '#e65100';
    const RETENTION = '#2e7d32';

    let periods = $derived(data?.Periods ?? []);
    let sessions = $derived(data?.Sessions ?? []);

    const orNull = (count, value) => (count > 0 ? value : null);

    let labels = $derived(periods.map((p) => p.Start));
    let datasets = $derived([
        {
            label: $t('stats.trainingMatchPR'),
            data: periods.map((p) => orNull(p.MatchDecisions, p.MatchPR)),
            borderColor: PRIMARY,
            backgroundColor: PRIMARY,
            tension: 0.3,
            spanGaps: true,
            yAxisID: 'y'
        },
        {
            label: $t('stats.trainingQuizPR'),
            data: periods.map((p) => orNull(p.QuizDecisions, p.QuizPR)),
            borderColor: QUIZ,
            backgroundColor: QUIZ,
            borderDash: [6, 4],
            tension: 0.3,
            spanGaps: true,
            yAxisID: 'y'
        },
        {
            label: $t('stats.trainingRetention'),
            data: periods.map((p) => orNull(p.AnkiReviews, Math.round(p.AnkiRetention * 1000) / 10)),
            borderColor: RETENTION,
            backgroundColor: RETENTION,
            borderDash: [2, 3],
            tension: 0.3,
            spanGaps: true,
            yAxisID: 'y1'
        }
    ]);

    const options = {
        scales: {
            y: { beginAtZero: true, position: 'left' },
            y1: { min: 0, max: 100, position: 'right', grid: { drawOnChartArea: false } }
        }
    };

    const fmt = (count, value, digits = 2) => (count > 0 ? value.toFixed(digits) : '–');
</script>

<section class="chart-section training">
    <h3 class="section-title">{$t('stats.trainingTitle')}</h3>
    <p class="hint">{$t('stats.trainingHint')}</p>
    <div class="window" role="group" aria-label={$t('stats.trainingWindow')}>
        <button type="button" class="window-btn" class:active={period === 'week'} aria-pressed={period === 'week'} onclick={() => (period = 'week')}>{$t('stats.trainingWeek')}</button>
        <button type="button" class="window-btn" class:active={period === 'month'} aria-pressed={period === 'month'} onclick={() => (period = 'month')}>{$t('stats.trainingMonth')}</button>
    </div>
    {#if loading}
        <p class="empty-subsection">{$t('stats.loading')}</p>
    {:else if error}
        <p class="empty-subsection">{error}</p>
    {:else if periods.length === 0}
        <p class="empty-subsection">{$t('stats.trainingEmpty')}</p>
    {:else}
        <div class="chart-wrap">
            <LineChart {labels} {datasets} {options} />
        </div>
        <table>
            <thead>
                <tr>
                    <th>{$t('stats.trainingPeriod')}</th>
                    <th class="num">{$t('stats.trainingQuizPR')}</th>
                    <th class="num">{$t('stats.trainingMatchPR')}</th>
                    <th class="num">{$t('stats.trainingRetention')}</th>
                </tr>
            </thead>
            <tbody>
                {#each periods as p (p.Start)}
                    <tr>
                        <td>{p.Start}</td>
                        <td class="num">{fmt(p.QuizDecisions, p.QuizPR)} <span class="count">({p.QuizDecisions})</span></td>
                        <td class="num">{fmt(p.MatchDecisions, p.MatchPR)} <span class="count">({p.MatchDecisions})</span></td>
                        <td class="num">{p.AnkiReviews > 0 ? `${(p.AnkiRetention * 100).toFixed(1)} %` : '–'} <span class="count">({p.AnkiReviews})</span></td>
                    </tr>
                {/each}
            </tbody>
        </table>
        <p class="hint">{$t('stats.trainingSessions', { n: sessions.length })}</p>
    {/if}
</section>

<style>
    .chart-section {
        padding: 12px 16px 0;
    }

    .section-title {
        font-size: var(--font-size-base);
        font-weight: 600;
        color: var(--color-text-muted);
        text-transform: uppercase;
        letter-spacing: 0.05em;
        margin: 0 0 8px;
    }

    .hint {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        margin: 0 0 8px;
    }

    .window {
        display: flex;
        gap: 4px;
        margin-bottom: 8px;
    }

    .window-btn {
        font-size: var(--font-size-small);
        padding: 2px 10px;
        cursor: pointer;
    }

    .window-btn.active {
        font-weight: 600;
    }

    .chart-wrap {
        position: relative;
        height: 260px;
        margin-bottom: 12px;
    }

    .empty-subsection {
        color: var(--color-text-muted);
        font-size: var(--font-size-base);
        text-align: center;
        padding: 12px 16px;
        margin: 0;
        font-style: italic;
    }

    table {
        width: 100%;
        border-collapse: collapse;
    }

    th,
    td {
        text-align: left;
        padding: 3px 6px;
        border-bottom: 1px solid var(--color-border);
    }

    .num {
        text-align: right;
        font-variant-numeric: tabular-nums;
    }

    .count {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }
</style>
