<script>
    // Le plan d'étude (ADR-0077) : « que dois-je travailler maintenant ? ». Les familles
    // d'erreurs récurrentes classées par MWC récupérable — la perte au-delà de celle d'un
    // joueur de référence dans les mêmes positions — avec leur intervalle. Une famille qui
    // manque de preuve n'est pas classée : elle est seulement nommée, à confirmer.
    import { get } from 'svelte/store';
    import { statsFilterStore, studyPlanStore, studyPlanLoadingStore, studyPlanErrorStore } from '../../stores/statsStore.js';
    import { startStudyPlanQueue } from '../../services/studyQueueService.js';
    import { quizOnPlan, deckFromIds } from '../../services/recurringStudy.js';
    import { t } from '../../i18n/index.js';

    let plan = $derived($studyPlanStore);
    let families = $derived(plan?.Families ?? []);
    let tentative = $derived(plan?.Tentative ?? []);
    let anyPriced = $derived(families.length + tentative.length + (plan?.Unthemed ?? 0) > 0);

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

    function familyName(f) {
        const theme = f.Kind === 'cube' && CUBE_LABELS[f.Theme] ? CUBE_LABELS[f.Theme] : `stats.recurringTheme_${f.Theme}`;
        return `${label(`stats.gameType_${f.GameType}`, f.GameType)} · ${$t(`stats.recurringKind_${f.Kind}`)} · ${label(theme, f.Theme)}`;
    }

    /** MWC fraction → percentage points, the unit the Match panel shows a loss in. */
    function pct(x) {
        return `${(100 * x).toFixed(2)} %`;
    }

    function ids(f) {
        return (f.Positions ?? []).map((p) => p.PositionID);
    }
</script>

<section class="chart-section study-plan" data-testid="study-plan">
    <h3 class="section-title">{$t('stats.planTitle')}</h3>
    <p class="hint">{$t('stats.planHint', { n: plan?.MinErrors ?? 5 })}</p>
    {#if $studyPlanLoadingStore}
        <p class="empty-subsection">{$t('stats.loading')}</p>
    {:else if $studyPlanErrorStore}
        <p class="empty-subsection">{$studyPlanErrorStore}</p>
    {:else if !anyPriced}
        <p class="empty-subsection">{$t('stats.planEmpty')}</p>
        {#if (plan?.Unpriced ?? 0) > 0}
            <p class="aside" data-testid="plan-unpriced-only">{$t('stats.planOutside', { unthemed: 0, unpriced: plan.Unpriced })}</p>
        {/if}
    {:else}
        {#if families.length === 0}
            <p class="empty-subsection">{$t('stats.planNoEvidence')}</p>
        {:else}
            <button type="button" class="study-btn worst" data-testid="plan-quiz-first" onclick={() => quizOnPlan(0)}>{$t('stats.planQuizFirst')}</button>
            <table>
                <thead>
                    <tr>
                        <th class="num">#</th>
                        <th>{$t('stats.planFamily')}</th>
                        <th class="num">{$t('stats.planErrors')}</th>
                        <th class="num">{$t('stats.planRecoverable')}</th>
                        <th class="num">{$t('stats.planInterval')}</th>
                        <th>{$t('stats.recurringActions')}</th>
                    </tr>
                </thead>
                <tbody>
                    {#each families as f, i (f.GameType + '/' + f.Kind + '/' + f.Theme)}
                        <tr class="family-row" data-testid="plan-family">
                            <td class="num">{i + 1}</td>
                            <td>{familyName(f)}</td>
                            <td class="num" title={$t('stats.planAvoidable', { n: f.Avoidable })}>{f.Errors}</td>
                            <td class="num"><b>{pct(f.Recoverable)}</b></td>
                            <td class="num interval">[{pct(f.Low)}, {pct(f.High)}]</td>
                            <td class="actions">
                                <button type="button" class="study-btn" onclick={() => startStudyPlanQueue(get(statsFilterStore), i + 1)}>{$t('stats.planStudy')}</button>
                                <button type="button" class="study-btn" onclick={() => quizOnPlan(i + 1)}>{$t('stats.planQuiz')}</button>
                                <button type="button" class="study-btn" onclick={() => deckFromIds($t('stats.planStudyName', { name: familyName(f) }), ids(f))}>{$t('stats.planDeck')}</button>
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        {/if}
        {#if tentative.length > 0}
            <p class="aside">{$t('stats.planTentative', { n: tentative.length, list: tentative.map((f) => `${familyName(f)} (${f.Errors})`).join(', ') })}</p>
        {/if}
        {#if plan.Unthemed > 0 || plan.Unpriced > 0}
            <p class="aside">{$t('stats.planOutside', { unthemed: plan.Unthemed, unpriced: plan.Unpriced })}</p>
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

    .actions {
        white-space: nowrap;
    }

    .study-btn {
        font-size: var(--font-size-small);
        padding: 1px 8px;
        margin-right: 4px;
    }

    .study-btn.worst {
        margin-bottom: 8px;
    }
</style>
