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

    /** MWC fraction → percentage points, the unit the Match panel shows a loss in. */
    /** @param {number} x */
    function pct(x) {
        return `${(100 * x).toFixed(2)} %`;
    }

    // The bars measure each family against the first: the plan is a ranking,
    // and the length says by how much the first outweighs the rest.
    let top = $derived(Math.max(...families.map((/** @type {any} */ f) => f.Recoverable), 0));

    /** @param {any} f */
    function barWidth(f) {
        return top > 0 ? `${Math.max(2, (100 * f.Recoverable) / top).toFixed(1)}%` : '0%';
    }

    /** @param {any} f */
    function ids(f) {
        return (f.Positions ?? []).map((/** @type {any} */ p) => p.PositionID);
    }
</script>

<section class="study-plan" data-testid="study-plan">
    <header class="plan-head">
        <h3 class="section-title" title={$t('stats.planHint', { n: plan?.MinErrors ?? 5 })}>{$t('stats.planTitle')}</h3>
        {#if families.length > 0 && !$studyPlanLoadingStore && !$studyPlanErrorStore}
            <button type="button" class="study-btn primary" data-testid="plan-quiz-first" onclick={() => quizOnPlan(0)}>{$t('stats.planQuizFirst')}</button>
        {/if}
    </header>
    {#if $studyPlanLoadingStore}
        <p class="empty-subsection">{$t('stats.loading')}</p>
    {:else if $studyPlanErrorStore}
        <p class="empty-subsection">{$studyPlanErrorStore}</p>
    {:else if !anyPriced}
        <p class="empty-subsection">{$t('stats.planEmpty')}</p>
        {#if (plan?.Unpriced ?? 0) > 0}
            <p class="aside" data-testid="plan-unpriced-only" title={$t('stats.planOutside', { unthemed: 0, unpriced: plan.Unpriced })}>
                {$t('stats.planOutsideShort', { unthemed: 0, unpriced: plan.Unpriced })}
            </p>
        {/if}
    {:else}
        {#if families.length === 0}
            <p class="empty-subsection">{$t('stats.planNoEvidence')}</p>
        {:else}
            <ol class="families">
                {#each families as f, i (f.GameType + '/' + f.Kind + '/' + f.Theme)}
                    <li class="family" class:first={i === 0} data-testid="plan-family">
                        <span class="rank">{i + 1}</span>
                        <span class="what">
                            <span class="name">{familyName(f)}</span>
                            <span class="bar" aria-hidden="true"><span style:width={barWidth(f)}></span></span>
                        </span>
                        <span class="figure" title={$t('stats.planRecoverable')}>
                            <b>{pct(f.Recoverable)}</b>
                            <small class="interval" title={$t('stats.planInterval')}>[{pct(f.Low)}, {pct(f.High)}]</small>
                        </span>
                        <span class="errors" title={$t('stats.planAvoidable', { n: f.Avoidable })}>
                            <b>{f.Errors}</b>
                            <small>{$t('stats.planErrors')}</small>
                        </span>
                        <span class="actions">
                            <button type="button" class="study-btn primary" onclick={() => startStudyPlanQueue(get(statsFilterStore), i + 1)}>{$t('stats.planStudy')}</button>
                            <button type="button" class="study-btn" onclick={() => quizOnPlan(i + 1)}>{$t('stats.planQuiz')}</button>
                            <button type="button" class="study-btn" onclick={() => deckFromIds($t('stats.planStudyName', { name: familyName(f) }), ids(f))}>{$t('stats.planDeck')}</button>
                        </span>
                    </li>
                {/each}
            </ol>
        {/if}
        <p class="meta">
            {#if tentative.length > 0}
                <span>{$t('stats.planTentative', { n: tentative.length, list: tentative.map((/** @type {any} */ f) => `${familyName(f)} (${f.Errors})`).join(', ') })}</span>
            {/if}
            {#if plan.Unthemed > 0 || plan.Unpriced > 0}
                <span title={$t('stats.planOutside', { unthemed: plan.Unthemed, unpriced: plan.Unpriced })}>{$t('stats.planOutsideShort', { unthemed: plan.Unthemed, unpriced: plan.Unpriced })}</span>
            {/if}
        </p>
    {/if}
</section>

<style>
    .study-plan {
        margin: 16px 16px 0;
        padding: 10px 12px 8px;
        border: 1px solid var(--color-border);
        border-radius: 6px;
        text-align: left;
    }

    .plan-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        margin-bottom: 6px;
    }

    .section-title {
        font-size: var(--font-size-base);
        font-weight: 600;
        color: var(--color-text);
        text-transform: uppercase;
        letter-spacing: 0.05em;
        margin: 0;
        cursor: help;
    }

    .empty-subsection {
        color: var(--color-text-muted);
        font-size: var(--font-size-base);
        text-align: center;
        padding: 12px 16px;
        margin: 0;
        font-style: italic;
    }

    .families {
        list-style: none;
        margin: 0;
        padding: 0;
    }

    .family {
        display: grid;
        grid-template-columns: 24px minmax(0, 1fr) 150px 64px auto;
        align-items: center;
        gap: 12px;
        padding: 6px 4px;
        border-bottom: 1px solid var(--color-border);
    }

    .family:last-child {
        border-bottom: none;
    }

    .family:hover {
        background: var(--color-surface-alt);
    }

    .rank {
        font-variant-numeric: tabular-nums;
        color: var(--color-text-muted);
        text-align: right;
    }

    .first .rank,
    .first .name {
        color: var(--color-text);
        font-weight: 600;
    }

    .what {
        display: flex;
        flex-direction: column;
        gap: 4px;
        min-width: 0;
    }

    .name {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .bar {
        display: block;
        height: 4px;
        border-radius: 2px;
        background: var(--color-surface-alt);
    }

    .bar > span {
        display: block;
        height: 100%;
        border-radius: 2px;
        background: color-mix(in srgb, var(--color-primary) 55%, transparent);
    }

    .first .bar > span {
        background: var(--color-primary);
    }

    .figure,
    .errors {
        display: flex;
        flex-direction: column;
        align-items: flex-end;
        font-variant-numeric: tabular-nums;
        line-height: 1.2;
    }

    .figure b {
        font-size: var(--font-size-base);
    }

    .first .figure b {
        font-size: var(--font-size-title);
    }

    .figure small,
    .errors small {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        white-space: nowrap;
    }

    .actions {
        display: flex;
        gap: 4px;
        white-space: nowrap;
    }

    .study-btn {
        font-size: var(--font-size-small);
        padding: 2px 8px;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }

    .study-btn:hover {
        background: var(--color-surface-alt);
    }

    .study-btn.primary {
        border-color: var(--color-primary);
        font-weight: 600;
    }

    .meta,
    .aside {
        display: flex;
        flex-wrap: wrap;
        gap: 4px 16px;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        margin: 6px 0 0;
    }

    .meta:empty {
        display: none;
    }

    .meta span[title] {
        cursor: help;
    }

    @media (max-width: 600px) {
        .family {
            grid-template-columns: 20px minmax(0, 1fr) auto;
        }
        .errors {
            display: none;
        }
        .actions {
            grid-column: 2 / -1;
        }
    }
</style>
