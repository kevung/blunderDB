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
    import { GetTrainingSeedSources, SaveTrainingSeedSource } from '../../wailsjs/go/main/Config.js';
    import { trainingSessionStore, trainingElapsedStore, trainingJournalStore, trainingRefusalStore } from '../stores/trainingTabStore.js';
    import { databasePathStore } from '../stores/databaseStore.js';
    import { TRAINING_EXERCISES, TIME_LIMITS, summarizeExercise, canAskAnother, isEnteredExercise } from '../services/trainingTab.js';
    import {
        startTrainingSession,
        revealQuestion,
        markFault,
        setTrainingAnswer,
        nextTrainingQuestion,
        retryTrainingQuestion,
        finishTrainingSession,
        quitTrainingSession,
        refreshTrainingJournal,
        refusalMessageKey
    } from '../services/trainingTabService.js';
    import { logger } from '../utils/logger.js';
    import { numberTypeLabelKey } from '../services/trainingLabels.js';
    import ScoreCard from './ScoreCard.svelte';
    import TrainingNumberCell from './TrainingNumberCell.svelte';

    let exercise = $state(TRAINING_EXERCISES[0].id);
    let limitSeconds = $state(0);
    let unfolded = $state('');
    // La source choisie, PAR EXERCICE et mémorisée d'une session à l'autre
    // (ADR-0041 règle 2). Par exercice parce qu'ils n'offrent pas les mêmes
    // sources : une seule mémoire se serait réinitialisée à chaque passage de
    // Bearoff à Pions.
    let rememberedSources = $state(/** @type {Record<string, string>} */ ({}));

    let session = $derived($trainingSessionStore);
    let question = $derived(session?.question ?? null);
    let chosen = $derived(TRAINING_EXERCISES.find((e) => e.id === exercise) ?? TRAINING_EXERCISES[0]);
    // La source « base » demande une bibliothèque ouverte : sans elle il n'y a
    // rien à tirer, et un bouton qui accepte le clic pour refuser ensuite fait
    // faire le geste avant de dire qu'il ne mène nulle part (ADR-0041 règle 2).
    let hasLibrary = $derived(!!$databasePathStore);
    let remembered = $derived(rememberedSources[exercise]);
    let seedSource = $derived(chosen.sources.includes(remembered) && !(remembered === 'library' && !hasLibrary) ? remembered : usableDefault(chosen, hasLibrary));

    /** @param {{sources: string[], defaultSource: string}} declared @param {boolean} library */
    function usableDefault(declared, library) {
        if (declared.defaultSource !== 'library' || library) return declared.defaultSource;
        return declared.sources.find((source) => source !== 'library') ?? declared.defaultSource;
    }

    let entered = $derived(!!session && isEnteredExercise(session.exercise));
    let another = $derived(!!session && canAskAnother(session.exercise, session.seedSource));

    let elapsedSeconds = $derived(Math.floor($trainingElapsedStore / 1000));

    onMount(() => {
        refreshTrainingJournal();
        GetTrainingSeedSources()
            .then((sources) => {
                rememberedSources = sources || {};
            })
            .catch((error) => logger.error('could not read the remembered training sources:', error));
    });

    /** @param {string} source */
    function chooseSource(source) {
        rememberedSources = { ...rememberedSources, [exercise]: source };
    }

    function start() {
        // La mémoire est écrite au LANCEMENT et non au clic : c'est démarrer
        // qui dit qu'on a choisi, cliquer pour regarder ne le dit pas.
        SaveTrainingSeedSource(exercise, seedSource).catch((error) => logger.error('could not remember the training source:', error));
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
                    <p class="refusal" data-testid="training-question-failed">{$t(refusalMessageKey(session.questionError))}</p>
                {:else if question.kind === 'scores'}
                    <ScoreCard card={question.card} numbers={question.numbers} revealed={session.revealed} faults={session.faults} locked={session.outOfTime} onToggle={markFault} />
                {:else if entered}
                    <!-- Mode SAISI : on tape, l'application juge. La vérité
                         apparaît à côté de ce qu'on a écrit — l'écart se lit
                         entre les deux, et l'imprimer serait la même
                         information une troisième fois (ADR-0031). Il est
                         enregistré, SIGNÉ, et c'est le bilan qui en fait une
                         moyenne. -->
                    <table class="entered">
                        <tbody>
                            {#each question.numbers as number, i (number.type)}
                                <tr>
                                    <th scope="row"><label for="training-answer-{i}">{$t(numberTypeLabelKey(number.type))}</label></th>
                                    <td>
                                        <input
                                            id="training-answer-{i}"
                                            data-testid="training-answer-{i}"
                                            type="text"
                                            inputmode="decimal"
                                            autocomplete="off"
                                            class:fault={session.revealed && session.faults[i]}
                                            disabled={session.revealed}
                                            value={session.answers[i] ?? ''}
                                            oninput={(event) => setTrainingAnswer(i, event.currentTarget.value)}
                                            onkeydown={(event) => {
                                                if (event.key === 'Enter') {
                                                    event.preventDefault();
                                                    revealQuestion();
                                                }
                                            }}
                                        />
                                    </td>
                                    <td class="truth" data-testid="training-truth-{i}">
                                        {#if session.revealed}
                                            <span class="mark" aria-hidden="true">{session.faults[i] ? '×' : ''}</span><span class:fault={session.faults[i]}
                                                >{number.value.toFixed(number.precision ?? 0)}</span
                                            >
                                        {/if}
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
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

            {#if question && session.revealed && !session.outOfTime && !entered}
                <!-- Le geste ne se devine pas, et une infobulle ne se lit ni au
                     clavier ni au doigt : la consigne est à l'écran. En mode
                     SAISI il n'y a rien à cocher : c'est l'application qui
                     juge, contre la tolérance. -->
                <p class="hint">{$t('training.faultHint')}</p>
            {/if}
            {#if question && !session.revealed && entered}
                <p class="hint">{$t('training.tolerance', { value: (question.numbers[0]?.tolerance ?? 0).toFixed(1) })}</p>
            {/if}

            <div class="actions">
                {#if !question}
                    <button type="button" data-testid="training-retry" onclick={() => retryTrainingQuestion()}>{$t('training.retry')}</button>
                {:else if !session.revealed}
                    <button type="button" data-testid="training-reveal" onclick={() => revealQuestion()}>{entered ? $t('training.validate') : $t('training.reveal')}</button>
                {:else if another}
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
                            <button
                                type="button"
                                class:selected={seedSource === source}
                                disabled={source === 'library' && !hasLibrary}
                                data-testid="training-source-{source}"
                                onclick={() => chooseSource(source)}
                            >
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

            {#if $trainingRefusalStore}
                <!-- Le refus se lit LÀ OÙ l'on vient de cliquer, et il nomme le
                     domaine (ADR-0041 règle 3). Un message de barre d'état
                     s'efface au geste suivant ; celui-ci reste tant qu'on n'a
                     pas démarré autre chose. -->
                <p class="refusal" role="status" data-testid="training-refusal">{$t(refusalMessageKey($trainingRefusalStore))}</p>
            {/if}
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

    .choices button:disabled {
        cursor: default;
        color: var(--color-text-muted);
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

    .pips,
    .entered {
        border-collapse: collapse;
    }

    .pips th,
    .entered th {
        text-align: left;
        font-weight: normal;
        color: var(--color-text-muted);
        padding-right: 0.5em;
    }

    .entered input {
        width: 5em;
        padding: 0.1em 0.35em;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        background: var(--color-surface);
        color: var(--color-text);
        text-align: right;
        font-variant-numeric: tabular-nums;
    }

    .entered input.fault {
        border-color: var(--color-danger);
    }

    .entered .truth {
        padding-left: 0.6em;
        font-variant-numeric: tabular-nums;
    }

    /* La faute se dit par un glyphe que la couleur redouble, jamais par la
       couleur seule (ADR-0031). */
    .entered .truth .fault,
    .entered .mark {
        color: var(--color-danger);
    }

    .entered .mark {
        font-weight: 600;
        padding-right: 0.15em;
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
