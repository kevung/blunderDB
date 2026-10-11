<script>
    // Biais signés (ADR-0079) : dans quel sens les décisions se trompent, pas seulement combien.
    // Chaque biais est la part des décisions fautives dans un sens moins la part dans l'autre,
    // avec son intervalle ; un sens n'est nommé que si l'intervalle exclut zéro.
    import { biasesStore, studyLoopLoadingStore, studyLoopErrorStore } from '../../stores/statsStore.js';
    import { t } from '../../i18n/index.js';
    import PanelTable from '../panels/PanelTable.svelte';

    let biases = $derived($biasesStore);
    let rows = $derived(
        biases
            ? [
                  { id: 'takePass', b: biases.TakePass },
                  { id: 'doubles', b: biases.Doubles },
                  { id: 'blots', b: biases.Blots }
              ]
            : []
    );
    let scoreCells = $derived((biases?.DoublesByScore ?? []).filter((/** @type {any} */ c) => c.Decisions >= (biases?.MinDecisions ?? 20)));

    /** Share of the decisions → signed percentage points. */
    /** @param {number} x */
    function pp(x) {
        const v = 100 * x;
        return `${v > 0 ? '+' : ''}${v.toFixed(1)} %`;
    }

    /** @param {string} id @param {any} b */
    function verdict(id, b) {
        return b.Verdict === 'too_much' || b.Verdict === 'too_little' ? $t(`stats.biases_${id}_${b.Verdict}`) : $t(`stats.biasesVerdict_${b.Verdict}`);
    }

    /** @param {any} c */
    function score(c) {
        return c.Money ? $t('stats.biasesMoney') : `${c.MoverAway}-${c.OpponentAway}`;
    }

    /**
     * The figures every bias table shares, read from a row's bias by `of`.
     * @param {(row: any) => any} of
     * @param {(row: any) => string} verdictOf
     */
    function figureColumns(of, verdictOf) {
        const num = /** @type {const} */ ('number');
        return [
            { key: 'decisions', label: $t('stats.biasesDecisions'), sortable: true, defaultDir: 'desc', align: 'right', filter: num, value: (/** @type {any} */ r) => of(r).Decisions },
            { key: 'plus', sortable: true, defaultDir: 'desc', align: 'right', filter: num, value: (/** @type {any} */ r) => of(r).Plus },
            { key: 'minus', sortable: true, defaultDir: 'desc', align: 'right', filter: num, value: (/** @type {any} */ r) => of(r).Minus },
            { key: 'bias', label: $t('stats.biasesValue'), sortable: true, align: 'right', filter: num, value: (/** @type {any} */ r) => 100 * of(r).Bias },
            { key: 'interval', label: $t('stats.planInterval'), align: 'right' },
            { key: 'verdict', label: $t('stats.effectVerdict'), sortable: true, filter: /** @type {const} */ ('text'), value: verdictOf }
        ];
    }

    const axisColumns = $derived.by(() => {
        const [decisions, plus, minus, ...rest] = figureColumns(
            (r) => r.b,
            (r) => verdict(r.id, r.b)
        );
        return [
            { key: 'bias', label: $t('stats.biasesBias'), sortable: true, filter: /** @type {const} */ ('text'), value: (/** @type {any} */ r) => $t(`stats.biases_${r.id}`) },
            decisions,
            { ...plus, label: $t('stats.biasesTooMuch') },
            { ...minus, label: $t('stats.biasesTooLittle') },
            ...rest.map((c) => (c.key === 'bias' ? { ...c, key: 'value' } : c))
        ];
    });

    const scoreColumns = $derived.by(() => {
        const [decisions, plus, minus, ...rest] = figureColumns(
            (c) => c,
            (c) => verdict('doubles', c)
        );
        return [
            { key: 'score', label: $t('stats.biasesScore'), sortable: true, filter: /** @type {const} */ ('text'), value: score },
            decisions,
            { ...plus, label: $t('stats.biases_doubles_plus') },
            { ...minus, label: $t('stats.biases_doubles_minus') },
            ...rest
        ];
    });
</script>

<section class="chart-section biases" data-testid="biases">
    <h3 class="section-title" title={$t('stats.biasesHint', { n: biases?.MinDecisions ?? 20 })}>{$t('stats.biasesTitle')}</h3>
    {#if $studyLoopLoadingStore}
        <p class="empty-subsection">{$t('stats.loading')}</p>
    {:else if $studyLoopErrorStore}
        <p class="empty-subsection">{$studyLoopErrorStore}</p>
    {:else if biases}
        <PanelTable viewId="stats.biases.axes" {rows} columns={axisColumns} rowAttrs={() => ({ 'data-testid': 'bias-row' })}>
            {#snippet cells(/** @type {any} */ r)}
                <td>{$t(`stats.biases_${r.id}`)}</td>
                <td class="align-right">{r.b.Decisions}</td>
                <td class="align-right" title={$t(`stats.biases_${r.id}_plus`)}>{r.b.Plus} <span class="count">({r.b.PlusMP} mp)</span></td>
                <td class="align-right" title={$t(`stats.biases_${r.id}_minus`)}>{r.b.Minus} <span class="count">({r.b.MinusMP} mp)</span></td>
                <td class="align-right"><b>{pp(r.b.Bias)}</b></td>
                <td class="align-right interval">[{pp(r.b.Low)}, {pp(r.b.High)}]</td>
                <td class="verdict {r.b.Verdict}">{verdict(r.id, r.b)}</td>
            {/snippet}
        </PanelTable>
        {#if biases.BlotsUnread > 0}
            <p class="aside" data-testid="bias-unread">
                {$t('stats.biasesUnread', {
                    n: biases.BlotsUnread,
                    total: biases.BlotsUnread + biases.Blots.Decisions,
                    share: `${((100 * biases.BlotsUnread) / (biases.BlotsUnread + biases.Blots.Decisions)).toFixed(1)} %`
                })}
            </p>
        {/if}
        {#if scoreCells.length > 0}
            <h4 class="sub-title">{$t('stats.biasesByScore')}</h4>
            <PanelTable
                viewId="stats.biases.scores"
                rows={scoreCells}
                rowKey={(c) => (c.Money ? 'money' : '') + c.MoverAway + '-' + c.OpponentAway}
                columns={scoreColumns}
                rowAttrs={() => ({ 'data-testid': 'bias-score' })}
            >
                {#snippet cells(/** @type {any} */ c)}
                    <td>{score(c)}</td>
                    <td class="align-right">{c.Decisions}</td>
                    <td class="align-right">{c.Plus}</td>
                    <td class="align-right">{c.Minus}</td>
                    <td class="align-right"><b>{pp(c.Bias)}</b></td>
                    <td class="align-right interval">[{pp(c.Low)}, {pp(c.High)}]</td>
                    <td class="verdict {c.Verdict}">{verdict('doubles', c)}</td>
                {/snippet}
            </PanelTable>
        {:else}
            <p class="aside">{$t('stats.biasesByScoreEmpty', { n: biases.MinDecisions })}</p>
        {/if}
    {:else}
        <p class="empty-subsection">{$t('stats.noData')}</p>
    {/if}
</section>

<style>
    .chart-section {
        padding: 12px 16px 0;
    }

    .section-title {
        font-size: var(--font-size-small);
        font-weight: 600;
        color: var(--color-text-muted);
        text-transform: uppercase;
        letter-spacing: 0.05em;
        margin: 0 0 8px;
        cursor: help;
    }

    .aside {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        margin: 0 0 8px;
    }

    .aside {
        margin-top: 8px;
    }

    .empty-subsection {
        color: var(--color-text-muted);
        font-size: var(--font-size-base);
        text-align: center;
        padding: 12px 16px;
        margin: 0;
        font-style: italic;
    }

    .interval {
        color: var(--color-text-muted);
        white-space: nowrap;
    }

    .count {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .verdict.improved,
    .verdict.too_much,
    .verdict.too_little,
    .verdict.worse {
        font-weight: 600;
    }

    .verdict.insufficient,
    .verdict.undetermined,
    .verdict.balanced {
        color: var(--color-text-muted);
    }

    .sub-title {
        font-size: var(--font-size-small);
        font-weight: 600;
        color: var(--color-text-muted);
        margin: 12px 0 4px;
    }
</style>
