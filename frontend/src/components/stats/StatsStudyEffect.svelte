<script>
    // Avant/après l'étude (ADR-0079) : pour chaque famille dont une position a été étudiée, la perte
    // de MWC par décision de son plan et de sa nature, dans les matchs joués avant le jour de la
    // première action d'étude et dans ceux joués après. Un sens n'est affirmé que si l'intervalle
    // l'exclut et que chaque fenêtre a assez de décisions.
    import { studyEffectStore, studyLoopLoadingStore, studyLoopErrorStore } from '../../stores/statsStore.js';
    import { t } from '../../i18n/index.js';
    import PanelTable from '../panels/PanelTable.svelte';

    let effect = $derived($studyEffectStore);
    let families = $derived(effect?.Families ?? []);

    /** @type {Record<string, string>} */
    const CUBE_LABELS = {
        offer_missed: 'stats.cubeOfferMissed',
        offer_premature: 'stats.cubeOfferPremature',
        answer_wrong_pass: 'stats.cubeAnswerWrongPass',
        answer_wrong_take: 'stats.cubeAnswerWrongTake'
    };

    /** @param {string} key @param {string} fallback */
    function label(key, fallback) {
        const s = $t(key);
        return s === key ? fallback : s;
    }

    /** @param {any} f */
    function familyName(f) {
        const theme = f.Kind === 'cube' && CUBE_LABELS[f.Theme] ? CUBE_LABELS[f.Theme] : `stats.recurringTheme_${f.Theme}`;
        return `${label(`stats.gameType_${f.GameType}`, f.GameType)} · ${$t(`stats.recurringKind_${f.Kind}`)} · ${label(theme, f.Theme)}`;
    }

    /** MWC fraction per decision → MWC percentage points per 100 decisions. */
    /** @param {number} x @param {boolean} [signed] */
    function per100(x, signed = false) {
        const v = 10000 * x;
        return `${signed && v > 0 ? '+' : ''}${v.toFixed(2)}`;
    }

    const num = /** @type {const} */ ('number');
    const text = /** @type {const} */ ('text');
    const columns = $derived([
        { key: 'family', label: $t('stats.planFamily'), sortable: true, filter: text, value: familyName },
        { key: 'studiedOn', label: $t('stats.effectStudiedOn'), sortable: true, filter: text, value: (/** @type {any} */ f) => f.StudiedOn },
        { key: 'before', label: $t('stats.effectBefore'), sortable: true, align: 'right', filter: num, value: (/** @type {any} */ f) => 10000 * f.Before.Rate },
        { key: 'after', label: $t('stats.effectAfter'), sortable: true, align: 'right', filter: num, value: (/** @type {any} */ f) => 10000 * f.After.Rate },
        { key: 'gain', label: $t('stats.effectGain'), sortable: true, defaultDir: 'desc', align: 'right', filter: num, value: (/** @type {any} */ f) => 10000 * f.Gain },
        { key: 'interval', label: $t('stats.planInterval'), align: 'right' },
        { key: 'verdict', label: $t('stats.effectVerdict'), sortable: true, filter: text, value: (/** @type {any} */ f) => $t(`stats.effectVerdict_${f.Verdict}`) }
    ]);
</script>

<section class="chart-section study-effect" data-testid="study-effect">
    <h3
        class="section-title"
        title="{$t('stats.effectHint', { n: effect?.MinDecisions ?? 30 })}

{$t('stats.effectCaveat')}"
    >
        {$t('stats.effectTitle')}
    </h3>
    {#if $studyLoopLoadingStore}
        <p class="empty-subsection">{$t('stats.loading')}</p>
    {:else if $studyLoopErrorStore}
        <p class="empty-subsection">{$studyLoopErrorStore}</p>
    {:else if families.length === 0}
        <p class="empty-subsection">{$t('stats.effectEmpty')}</p>
    {:else}
        <PanelTable viewId="stats.studyEffect" rows={families} rowKey={(f) => f.GameType + '/' + f.Kind + '/' + f.Theme} {columns} rowAttrs={() => ({ 'data-testid': 'effect-family' })}>
            {#snippet cells(/** @type {any} */ f)}
                <td>{familyName(f)}</td>
                <td title={$t('stats.effectStudied', { n: f.Studied })}>{f.StudiedOn}</td>
                <td class="align-right">{per100(f.Before.Rate)} <span class="count">({f.Before.Decisions})</span></td>
                <td class="align-right">{per100(f.After.Rate)} <span class="count">({f.After.Decisions})</span></td>
                <td class="align-right"><b>{per100(f.Gain, true)}</b></td>
                <td class="align-right interval">[{per100(f.Low, true)}, {per100(f.High, true)}]</td>
                <td class="verdict {f.Verdict}">{$t(`stats.effectVerdict_${f.Verdict}`)}</td>
            {/snippet}
        </PanelTable>
    {/if}
    {#if effect?.Unstudied > 0}
        <p class="aside">{$t('stats.effectUnstudied', { n: effect.Unstudied })}</p>
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
</style>
