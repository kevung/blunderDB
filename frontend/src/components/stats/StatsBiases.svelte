<script>
    // Biais signés (ADR-0078) : dans quel sens les décisions se trompent, pas seulement combien.
    // Chaque biais est la part des décisions fautives dans un sens moins la part dans l'autre,
    // avec son intervalle ; un sens n'est nommé que si l'intervalle exclut zéro.
    import { biasesStore, studyLoopLoadingStore, studyLoopErrorStore } from '../../stores/statsStore.js';
    import { t } from '../../i18n/index.js';

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
    let scoreCells = $derived((biases?.DoublesByScore ?? []).filter((c) => c.Decisions >= (biases?.MinDecisions ?? 20)));

    /** Share of the decisions → signed percentage points. */
    function pp(x) {
        const v = 100 * x;
        return `${v > 0 ? '+' : ''}${v.toFixed(1)} %`;
    }

    function verdict(id, b) {
        return b.Verdict === 'too_much' || b.Verdict === 'too_little' ? $t(`stats.biases_${id}_${b.Verdict}`) : $t(`stats.biasesVerdict_${b.Verdict}`);
    }

    function score(c) {
        return c.MoverAway === 0 && c.OpponentAway === 0 ? $t('stats.biasesMoney') : `${c.MoverAway}-${c.OpponentAway}`;
    }
</script>

<section class="chart-section biases" data-testid="biases">
    <h3 class="section-title">{$t('stats.biasesTitle')}</h3>
    <p class="hint">{$t('stats.biasesHint', { n: biases?.MinDecisions ?? 20 })}</p>
    {#if $studyLoopLoadingStore}
        <p class="empty-subsection">{$t('stats.loading')}</p>
    {:else if $studyLoopErrorStore}
        <p class="empty-subsection">{$studyLoopErrorStore}</p>
    {:else if biases}
        <table>
            <thead>
                <tr>
                    <th>{$t('stats.biasesBias')}</th>
                    <th class="num">{$t('stats.biasesDecisions')}</th>
                    <th class="num">{$t('stats.biasesTooMuch')}</th>
                    <th class="num">{$t('stats.biasesTooLittle')}</th>
                    <th class="num">{$t('stats.biasesValue')}</th>
                    <th class="num">{$t('stats.planInterval')}</th>
                    <th>{$t('stats.effectVerdict')}</th>
                </tr>
            </thead>
            <tbody>
                {#each rows as r (r.id)}
                    <tr class="family-row" data-testid="bias-row">
                        <td>{$t(`stats.biases_${r.id}`)}</td>
                        <td class="num">{r.b.Decisions}</td>
                        <td class="num" title={$t(`stats.biases_${r.id}_plus`)}>{r.b.Plus} <span class="count">({r.b.PlusMP} mp)</span></td>
                        <td class="num" title={$t(`stats.biases_${r.id}_minus`)}>{r.b.Minus} <span class="count">({r.b.MinusMP} mp)</span></td>
                        <td class="num"><b>{pp(r.b.Bias)}</b></td>
                        <td class="num interval">[{pp(r.b.Low)}, {pp(r.b.High)}]</td>
                        <td class="verdict {r.b.Verdict}">{verdict(r.id, r.b)}</td>
                    </tr>
                {/each}
            </tbody>
        </table>
        {#if biases.BlotsUnread > 0}
            <p class="aside">{$t('stats.biasesUnread', { n: biases.BlotsUnread })}</p>
        {/if}
        {#if scoreCells.length > 0}
            <h4 class="sub-title">{$t('stats.biasesByScore')}</h4>
            <table>
                <thead>
                    <tr>
                        <th>{$t('stats.biasesScore')}</th>
                        <th class="num">{$t('stats.biasesDecisions')}</th>
                        <th class="num">{$t('stats.biases_doubles_plus')}</th>
                        <th class="num">{$t('stats.biases_doubles_minus')}</th>
                        <th class="num">{$t('stats.biasesValue')}</th>
                        <th class="num">{$t('stats.planInterval')}</th>
                        <th>{$t('stats.effectVerdict')}</th>
                    </tr>
                </thead>
                <tbody>
                    {#each scoreCells as c (c.MoverAway + '-' + c.OpponentAway)}
                        <tr class="family-row" data-testid="bias-score">
                            <td>{score(c)}</td>
                            <td class="num">{c.Decisions}</td>
                            <td class="num">{c.Plus}</td>
                            <td class="num">{c.Minus}</td>
                            <td class="num"><b>{pp(c.Bias)}</b></td>
                            <td class="num interval">[{pp(c.Low)}, {pp(c.High)}]</td>
                            <td class="verdict {c.Verdict}">{verdict('doubles', c)}</td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        {:else}
            <p class="aside">{$t('stats.biasesByScoreEmpty', { n: biases.MinDecisions })}</p>
        {/if}
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

    .hint,
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

    .interval {
        color: var(--color-text-muted);
        white-space: nowrap;
    }

    .family-row:hover {
        background: var(--color-surface-alt);
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
