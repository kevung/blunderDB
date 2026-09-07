<script>
    // L'onglet Entraînement (#320, ADR-0040).
    //
    // Tout le geste d'une session est ici : démarrer, révéler, cocher ses
    // fautes, passer à la suivante, terminer, quitter. Le plateau montre la
    // question de Pions et sa réponse une fois révélée — il ne porte aucun
    // bouton. C'est l'objection qui a supprimé la barre d'entraînement : les
    // saisies de l'application vivent dans ses panneaux.
    //
    // Au repos, le panneau montre le lanceur ET le bilan par exercice, que l'on
    // déplie en détail par type de nombre. Pas de graphique : une tendance en
    // chiffres suffit (règle 6).
    import { onMount } from 'svelte';
    import { t } from '../i18n';
    import { trainingSessionStore, trainingElapsedStore, trainingJournalStore } from '../stores/trainingTabStore.js';
    import { TRAINING_EXERCISES, TIME_LIMITS, summarizeExercise } from '../services/trainingTab.js';
    import {
        startTrainingSession,
        revealQuestion,
        markFault,
        nextTrainingQuestion,
        retryTrainingQuestion,
        finishTrainingSession,
        quitTrainingSession,
        refreshTrainingJournal
    } from '../services/trainingTabService.js';
    import { numberTypeLabelKey } from '../services/trainingLabels.js';
    import ScoreCard from './ScoreCard.svelte';
    import TrainingNumberCell from './TrainingNumberCell.svelte';

    let exercise = $state(TRAINING_EXERCISES[0].id);
    let limitSeconds = $state(0);
    let unfolded = $state('');

    let session = $derived($trainingSessionStore);
    let question = $derived(session?.question ?? null);
    let chosen = $derived(TRAINING_EXERCISES.find((e) => e.id === exercise) ?? TRAINING_EXERCISES[0]);
    // `$derived` inscriptible : on peut choisir une autre source, et changer
    // d'exercice la ramène à celle de l'exercice — c'est l'effet voulu, un
    // exercice n'ayant pas les mêmes sources qu'un autre.
    let seedSource = $derived(chosen.defaultSource);

    // La source « plateau » prend la position TELLE QUELLE : il n'y a donc
    // qu'une question à poser, et proposer « Suivante » reposerait la même.
    // Le voisinage d'une position — la graine jouée sur quelques coups — est
    // le générateur de l'ADR-0041, qui n'est pas de cette tranche.
    let canAskAnother = $derived(!!session && session.seedSource !== 'board');

    let elapsedSeconds = $derived(Math.floor($trainingElapsedStore / 1000));

    onMount(() => {
        refreshTrainingJournal();
    });

    function start() {
        startTrainingSession({ exercise, seedSource, limitSeconds });
    }

    /** @param {string} id */
    function summaryOf(id) {
        return summarizeExercise($trainingJournalStore[id]?.sessions ?? []);
    }

    /** @param {string} id */
    function numbersOf(id) {
        return $trainingJournalStore[id]?.numbers ?? [];
    }

    /** @param {number|null} rate */
    function percent(rate) {
        return rate === null ? '—' : `${Math.round(rate * 100)}`;
    }

    /** @param {number|null} ms */
    function seconds(ms) {
        return ms === null ? '—' : (ms / 1000).toFixed(1);
    }

    /** @param {number|null} trend */
    function signedPoints(trend) {
        if (trend === null) return null;
        const points = Math.round(trend * 100);
        return points > 0 ? `+${points}` : `${points}`;
    }
</script>

<div class="training-panel" data-testid="training-panel">
    {#if session}
        <div class="session" role="group" aria-label={$t('training.title')}>
            <div class="session-head">
                <span class="exercise">{$t(`training.exercise.${session.exercise}`)}</span>
                <span class="counter">{$t('training.questionCount', { n: session.askedQuestions + 1 })}</span>
                {#if question}
                    <span class="clock" data-testid="training-clock">
                        {#if session.limitSeconds > 0}
                            {$t('training.elapsedOfLimit', { n: elapsedSeconds, limit: session.limitSeconds })}
                        {:else}
                            {$t('training.elapsed', { n: elapsedSeconds })}
                        {/if}
                    </span>
                {/if}
                {#if session.outOfTime}
                    <span class="out-of-time" data-testid="training-out-of-time">{$t('training.outOfTime')}</span>
                {/if}
            </div>

            <div class="question">
                {#if !question}
                    <!-- La question suivante n'a pas pu être bâtie. La session
                         reste ouverte : ce qui a été répondu est encore là, et
                         « Terminer » l'enregistre. -->
                    <p class="refusal" data-testid="training-question-failed">{$t('training.noQuestion')}</p>
                {:else if question.kind === 'scores'}
                    <ScoreCard card={question.card} numbers={question.numbers} revealed={session.revealed} faults={session.faults} locked={session.outOfTime} onToggle={markFault} />
                {:else}
                    <table class="pips">
                        <tbody>
                            {#each question.numbers as number, i (number.type)}
                                <tr>
                                    <th scope="row">{$t(numberTypeLabelKey(number.type))}</th>
                                    <td>
                                        <TrainingNumberCell
                                            value={number.value}
                                            precision={0}
                                            revealed={session.revealed}
                                            locked={session.outOfTime}
                                            fault={session.faults[i] === true}
                                            label={$t(numberTypeLabelKey(number.type))}
                                            onToggle={() => markFault(i)}
                                        />
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                {/if}
            </div>

            {#if question && session.revealed && !session.outOfTime}
                <!-- Le geste ne se devine pas, et une infobulle ne se lit ni au
                     clavier ni au doigt : la consigne est à l'écran. -->
                <p class="hint">{$t('training.faultHint')}</p>
            {/if}

            <div class="actions">
                {#if !question}
                    <button type="button" data-testid="training-retry" onclick={() => retryTrainingQuestion()}>{$t('training.retry')}</button>
                {:else if !session.revealed}
                    <button type="button" data-testid="training-reveal" onclick={() => revealQuestion()}>{$t('training.reveal')}</button>
                {:else if canAskAnother}
                    <button type="button" data-testid="training-next" onclick={() => nextTrainingQuestion()}>{$t('training.next')}</button>
                {/if}
                <button type="button" data-testid="training-finish" onclick={() => finishTrainingSession()}>{$t('training.finish')}</button>
                <button type="button" data-testid="training-quit" onclick={() => quitTrainingSession()}>{$t('training.leave')}</button>
            </div>
        </div>
    {:else}
        <div class="launcher">
            <div class="row">
                <span class="field-label" id="training-exercise-label">{$t('training.exerciseLabel')}</span>
                <div class="choices" role="group" aria-labelledby="training-exercise-label">
                    {#each TRAINING_EXERCISES as item (item.id)}
                        <button type="button" class:selected={exercise === item.id} data-testid="training-exercise-{item.id}" onclick={() => (exercise = item.id)}>
                            {$t(`training.exercise.${item.id}`)}
                        </button>
                    {/each}
                </div>
            </div>

            {#if chosen.sources.length > 0}
                <div class="row">
                    <span class="field-label" id="training-source-label">{$t('training.source')}</span>
                    <div class="choices" role="group" aria-labelledby="training-source-label">
                        {#each chosen.sources as source (source)}
                            <button type="button" class:selected={seedSource === source} data-testid="training-source-{source}" onclick={() => (seedSource = source)}>
                                {$t(`training.sources.${source}`)}
                            </button>
                        {/each}
                    </div>
                </div>
            {/if}

            <div class="row">
                <span class="field-label" id="training-limit-label">{$t('training.limit')}</span>
                <div class="choices" role="group" aria-labelledby="training-limit-label">
                    {#each TIME_LIMITS as limit (limit)}
                        <button type="button" class:selected={limitSeconds === limit} data-testid="training-limit-{limit}" onclick={() => (limitSeconds = limit)}>
                            {limit === 0 ? $t('training.limitNone') : $t('training.limitSeconds', { n: limit })}
                        </button>
                    {/each}
                </div>
            </div>

            <div class="row">
                <button type="button" class="start" data-testid="training-start" onclick={start}>{$t('training.start')}</button>
            </div>
        </div>

        <div class="journal">
            <h3>{$t('training.summary')}</h3>
            {#each TRAINING_EXERCISES as item (item.id)}
                {@const summary = summaryOf(item.id)}
                <div class="summary-line" data-testid="training-summary-{item.id}">
                    <button type="button" class="disclose" aria-expanded={unfolded === item.id} disabled={summary.sessions === 0} onclick={() => (unfolded = unfolded === item.id ? '' : item.id)}>
                        {$t(`training.exercise.${item.id}`)}
                    </button>
                    {#if summary.sessions === 0}
                        <span class="muted">{$t('training.noSession')}</span>
                    {:else}
                        <span class="figure">{$t('training.sessionsCount', { n: summary.sessions })}</span>
                        <span class="figure">{$t('training.faultRate', { rate: percent(summary.faultRate), faults: summary.faults, asked: summary.numbersAsked })}</span>
                        {#if summary.meanDeviation !== null}
                            <span class="figure">{$t('training.meanDeviation', { value: summary.meanDeviation.toFixed(1) })}</span>
                        {/if}
                        <span class="figure">{$t('training.medianTime', { n: seconds(summary.medianMs) })}</span>
                        {#if summary.trend !== null}
                            <span class="figure">{$t('training.trend', { value: signedPoints(summary.trend) })}</span>
                        {/if}
                    {/if}
                </div>
                {#if unfolded === item.id}
                    <ul class="detail" data-testid="training-detail-{item.id}">
                        {#each numbersOf(item.id) as number (number.numberType)}
                            <li>
                                <span class="detail-label">{$t(numberTypeLabelKey(number.numberType))}</span>
                                <span class="figure">{$t('training.detailFaults', { faults: number.faults, asked: number.asked })}</span>
                            </li>
                        {/each}
                    </ul>
                {/if}
            {/each}
        </div>
    {/if}
</div>

<style>
    .training-panel {
        display: flex;
        flex-direction: column;
        gap: 0.8em;
        padding: 0.6em 0.8em;
    }

    .row,
    .session-head,
    .actions,
    .summary-line {
        display: flex;
        align-items: center;
        gap: 0.5em;
        flex-wrap: wrap;
    }

    .field-label,
    .detail-label {
        color: var(--color-text-muted);
    }

    .choices button,
    .actions button,
    .start {
        cursor: pointer;
        padding: 0.15em 0.6em;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        background: var(--color-surface);
        color: var(--color-text);
    }

    .choices {
        display: flex;
        gap: 0.25em;
    }

    .choices button.selected {
        border-color: var(--color-primary);
        color: var(--color-primary);
    }

    .exercise {
        font-weight: 600;
    }

    .counter,
    .clock {
        color: var(--color-text-muted);
        font-variant-numeric: tabular-nums;
    }

    .out-of-time,
    .refusal {
        color: var(--color-danger);
    }

    .refusal,
    .hint {
        margin: 0;
    }

    .hint {
        color: var(--color-text-muted);
    }

    .pips {
        border-collapse: collapse;
    }

    .pips th {
        text-align: left;
        font-weight: normal;
        color: var(--color-text-muted);
        padding-right: 0.5em;
    }

    h3 {
        margin: 0;
        font-size: var(--font-size-base);
        font-weight: 600;
    }

    .disclose {
        border: none;
        background: transparent;
        padding: 0;
        color: var(--color-text);
        cursor: pointer;
        text-align: left;
        min-width: 6em;
    }

    .disclose:disabled {
        cursor: default;
        color: var(--color-text-muted);
    }

    .figure {
        color: var(--color-text-muted);
        font-variant-numeric: tabular-nums;
    }

    .muted {
        color: var(--color-text-muted);
    }

    .detail {
        list-style: none;
        margin: 0.1em 0 0.4em 1.2em;
        padding: 0;
    }

    .detail li {
        display: flex;
        gap: 0.5em;
    }
</style>
