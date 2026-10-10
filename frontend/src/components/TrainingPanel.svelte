<script>
    // L'onglet Entraînement (ADR-0040). Tous les gestes de session sont ici ; le
    // plateau n'a aucun bouton (Décision s'y joue, mais annuler/valider sont ici).
    // Au repos : lanceur et bilan par exercice, sans graphique (règle 6).
    import { onMount } from 'svelte';
    import { t } from '../i18n';
    import { GetTrainingSeedSources, SaveTrainingSeedSource } from '../../wailsjs/go/main/Config.js';
    import { trainingSessionStore, trainingElapsedStore, trainingJournalStore, trainingRefusalStore } from '../stores/trainingTabStore.js';
    import { databasePathStore } from '../stores/databaseStore.js';
    import { TRAINING_EXERCISES, TIME_LIMITS, summarizeExercise, canAskAnother, isEnteredExercise, isChosenExercise, isChosenNumber } from '../services/trainingTab.js';
    import { quizPlayStore, quizPlayCompleteStore } from '../stores/quizPlayStore.js';
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
        refusalMessageKey,
        answerDecisionBoard,
        answerDecisionCube,
        undoDecisionStep,
        resetDecisionPlay,
        playDecisionNotation
    } from '../services/trainingTabService.js';
    import { logger } from '../utils/logger.js';
    import { numberTypeLabelKey } from '../services/trainingLabels.js';
    import { retakeMissed, missedToDeck, missedToCollection } from '../services/trainingMissed.js';
    import ScoreCard from './ScoreCard.svelte';
    import ExplanationLine from './ExplanationLine.svelte';
    import TrainingNumberCell from './TrainingNumberCell.svelte';
    import PanelHeader from './panels/PanelHeader.svelte';
    import FormGrid from './panels/FormGrid.svelte';
    import FormRow from './panels/FormRow.svelte';

    let exercise = $state(TRAINING_EXERCISES[0].id);
    let limitSeconds = $state(0);
    let unfolded = $state('');
    // Source mémorisée par exercice, qui n'offrent pas les mêmes (ADR-0041 règle 2).
    let rememberedSources = $state(/** @type {Record<string, string>} */ ({}));

    let session = $derived($trainingSessionStore);
    let question = $derived(session?.question ?? null);
    let chosen = $derived(TRAINING_EXERCISES.find((e) => e.id === exercise) ?? TRAINING_EXERCISES[0]);
    // « base » sans bibliothèque ouverte : désactivée d'emblée (ADR-0041 règle 2).
    let hasLibrary = $derived(!!$databasePathStore);
    // An exercise drawn only from the library (Décision) cannot start without one.
    let libraryMissing = $derived(!hasLibrary && chosen.sources.length > 0 && chosen.sources.every((source) => source === 'library'));
    let remembered = $derived(rememberedSources[exercise]);
    let seedSource = $derived(chosen.sources.includes(remembered) && !(remembered === 'library' && !hasLibrary) ? remembered : usableDefault(chosen, hasLibrary));

    /** @param {{sources: string[], defaultSource: string}} declared @param {boolean} library */
    function usableDefault(declared, library) {
        if (declared.defaultSource !== 'library' || library) return declared.defaultSource;
        return declared.sources.find((source) => source !== 'library') ?? declared.defaultSource;
    }

    let entered = $derived(!!session && isEnteredExercise(session.exercise));
    // Mode CHOISI (Décision) : c'est le juge du moteur qui répond, il n'y a ni
    // « Révéler » ni case à cocher.
    let judgedByEngine = $derived(!!session && isChosenExercise(session.exercise));
    let hasSteps = $derived(($quizPlayStore?.steps.length ?? 0) > 0);
    let verdict = $derived(session?.verdict ?? null);

    // Le coup tapé en notation : posé sur le damier à chaque frappe, jugé par ENTRÉE.
    let notation = $state('');
    let notationRefused = $state(false);
    $effect(() => {
        // Une autre question repart d'un champ vide.
        void question;
        notation = '';
        notationRefused = false;
    });

    /** @param {string} text */
    function typeNotation(text) {
        notation = text;
        notationRefused = text.trim() !== '' && playDecisionNotation(text) !== 'ok';
        if (text.trim() === '') resetDecisionPlay();
    }

    // Les gestes de la transcription, quand le panneau a le focus : RETOUR ARRIÈRE défait un pas,
    // ENTRÉE valide le coup complet, ÉCHAP le recommence. Dans un champ de texte, ils restent au champ.
    /** @param {KeyboardEvent} event */
    function onPanelKey(event) {
        if (!session || session.revealed || question?.kind !== 'decision' || question.prompt !== 'checker') return;
        const target = /** @type {HTMLElement} */ (event.target);
        if (event.ctrlKey || event.metaKey || event.altKey) return;
        if (target.matches?.('input, textarea, select')) return;
        if (event.key === 'Backspace') {
            event.preventDefault();
            undoDecisionStep();
            notation = '';
        } else if (event.key === 'Escape' && hasSteps) {
            event.preventDefault();
            resetDecisionPlay();
            notation = '';
        } else if (event.key === 'Enter' && !target.matches?.('button') && $quizPlayCompleteStore) {
            event.preventDefault();
            answerDecisionBoard();
        }
    }

    /** Les trois actions de videau, dans l'ordre où la décision se lit. */
    const CUBE_ACTIONS = [
        { id: 'nd', labelKey: 'training.noDouble' },
        { id: 'dt', labelKey: 'training.doubleTake' },
        { id: 'dp', labelKey: 'training.doublePass' }
    ];

    // Le bouton actionné disparaît : le focus va à « Suivante » (ou « Terminer »)
    // pour que ENTRÉE ait une cible.
    let nextButton = $state(/** @type {HTMLButtonElement | undefined} */ (undefined));
    let finishButton = $state(/** @type {HTMLButtonElement | undefined} */ (undefined));
    let revealedAt = $derived(session?.question && session.revealed ? session.askedQuestions + 1 : 0);
    $effect(() => {
        if (!revealedAt) return;
        (nextButton ?? finishButton)?.focus();
    });
    let another = $derived(!!session && canAskAnother(session.exercise, session.seedSource));

    let elapsedSeconds = $derived(Math.floor($trainingElapsedStore / 1000));

    // Cible du focus après le choix de l'action de videau.
    let revealButton = $state(/** @type {HTMLButtonElement | undefined} */ (undefined));
    let questionBox = $state(/** @type {HTMLDivElement | undefined} */ (undefined));

    /** Choisit une option d'un nombre choisi, sans rien juger.
     *  @param {number} index @param {string} option */
    function choose(index, option) {
        setTrainingAnswer(index, option);
        revealButton?.focus();
    }

    /**
     * ENTRÉE dans un champ : valide, ou mène le focus à l'option encore à choisir.
     */
    function validateFromField() {
        const numbers = session?.question?.numbers ?? [];
        const pending = numbers.findIndex((number, i) => isChosenNumber(number) && !session?.answers[i]);
        if (pending >= 0) {
            /** @type {HTMLButtonElement | null | undefined} */ (questionBox?.querySelector(`[data-testid="training-choice-${pending}-nd"]`))?.focus();
            return;
        }
        revealQuestion();
    }

    /** L'EPC d'un camp, lu dans la forme que le moteur rend. @param {any} side */
    function epcOf(side) {
        return (side?.epc?.epc ?? 0).toFixed(1);
    }

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
        // Mémorisée au lancement, pas au clic.
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

<div class="training-panel" data-testid="training-panel" role="presentation" onkeydown={onPanelKey}>
    {#if session}
        <div class="session" role="group" aria-label={$t('training.title')}>
            <!-- La bande porte la séance et ses gestes ; la primaire reste en bout, au même endroit
                 d'une question à l'autre, pour qui garde les mains sur le clavier. -->
            <PanelHeader title={$t(`training.exercise.${session.exercise}`)}>
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
                {#snippet actions()}
                    <span class="actions">
                        <button type="button" data-testid="training-quit" onclick={() => quitTrainingSession()}>{$t('training.leave')}</button>
                        <button type="button" data-testid="training-finish" bind:this={finishButton} onclick={() => finishTrainingSession()}>{$t('training.finish')}</button>
                        {#if !question}
                            <button type="button" class="launch" data-testid="training-retry" onclick={() => retryTrainingQuestion()}>{$t('training.retry')}</button>
                        {:else if !session.revealed}
                            {#if !judgedByEngine}
                                <button
                                    type="button"
                                    class="launch"
                                    data-testid="training-reveal"
                                    title={question.kind === 'scores' ? $t('training.scoresInstruction') : undefined}
                                    bind:this={revealButton}
                                    onclick={() => revealQuestion()}>{entered ? $t('training.validate') : $t('training.reveal')}</button
                                >
                            {/if}
                        {:else if another}
                            <button type="button" class="launch" data-testid="training-next" bind:this={nextButton} onclick={() => nextTrainingQuestion()}>{$t('training.next')}</button>
                        {/if}
                    </span>
                {/snippet}
            </PanelHeader>

            <div class="question" bind:this={questionBox}>
                {#if !question}
                    <!-- Question suivante impossible : la session reste ouverte, « Terminer » l'enregistre. -->
                    <p class="refusal" data-testid="training-question-failed">{$t(refusalMessageKey(session.questionError))}</p>
                {:else if question.kind === 'decision'}
                    <div class="decision">
                        {#if question.prompt === 'cube'}
                            <p class="prompt">{$t('training.cubePrompt')}</p>
                            <div class="choices" role="group" aria-label={$t('training.numbers.decisionCube')}>
                                {#each CUBE_ACTIONS as action (action.id)}
                                    <button
                                        type="button"
                                        class:selected={session.revealed && verdict?.notation === action.id}
                                        disabled={session.revealed}
                                        data-testid="training-cube-{action.id}"
                                        onclick={() => answerDecisionCube(action.id)}
                                    >
                                        {$t(action.labelKey)}
                                    </button>
                                {/each}
                            </div>
                        {:else}
                            <p class="prompt">{$t('training.playOnBoard')}</p>
                            {#if !session.revealed}
                                <div class="choices">
                                    <button type="button" disabled={!hasSteps} data-testid="training-undo-step" onclick={() => undoDecisionStep()}>{$t('training.undoHop')}</button>
                                    <button type="button" disabled={!hasSteps} data-testid="training-reset-play" onclick={() => resetDecisionPlay()}>{$t('training.resetPlay')}</button>
                                </div>
                                <input
                                    type="text"
                                    class="notation"
                                    class:fault={notationRefused}
                                    data-testid="training-notation"
                                    autocomplete="off"
                                    spellcheck="false"
                                    placeholder={$t('training.notationPlaceholder')}
                                    aria-label={$t('training.notationLabel')}
                                    value={notation}
                                    oninput={(event) => typeNotation(event.currentTarget.value)}
                                    onkeydown={(event) => {
                                        if (event.key === 'Enter') {
                                            event.preventDefault();
                                            if ($quizPlayCompleteStore) answerDecisionBoard();
                                        }
                                    }}
                                />
                            {/if}
                        {/if}
                        {#if session.revealed && verdict}
                            <!-- Illégal, non classé (sans prix) et noté sont trois issues ;
                                 hors délai, seulement la correction. -->
                            <p class="verdict" class:correct={!session.outOfTime && verdict.matched && verdict.errorMp === 0} data-testid="training-verdict">
                                {#if !session.outOfTime}
                                    <span class="outcome">
                                        {#if !verdict.legal}
                                            {$t('training.illegal')}
                                        {:else if !verdict.matched}
                                            {$t('training.unranked')}
                                        {:else if verdict.errorMp === 0}
                                            {$t('training.right')}
                                        {:else}
                                            {$t('training.cost', { mp: verdict.errorMp })}
                                        {/if}
                                    </span>
                                {/if}
                                {#if verdict.best}
                                    <span class="best">{$t('training.best', { move: verdict.best })}</span>
                                {/if}
                            </p>
                            {#if !session.outOfTime && verdict.matched && verdict.errorMp > 0 && question.positionId}
                                <ExplanationLine positionId={question.positionId} played={verdict.notation} neighbours={false} />
                            {/if}
                        {/if}
                    </div>
                {:else if question.kind === 'scores'}
                    <ScoreCard card={question.card} numbers={question.numbers} revealed={session.revealed} faults={session.faults} locked={session.outOfTime} onToggle={markFault} />
                {:else if entered}
                    <!-- Mode saisi : la vérité à côté de la saisie, sans écart imprimé
                         (ADR-0031) ; l'écart est enregistré signé. -->
                    <table class="entered">
                        <tbody>
                            {#each question.numbers as number, i (number.type)}
                                <tr>
                                    {#if isChosenNumber(number)}
                                        <!-- Choix retenu sans jugement ; « Valider » juge tout. -->
                                        <th scope="row"><span id="training-answer-label-{i}">{$t(numberTypeLabelKey(number.type))}</span></th>
                                        <td colspan="2">
                                            <div class="choices" role="group" aria-labelledby="training-answer-label-{i}">
                                                {#each CUBE_ACTIONS as action (action.id)}
                                                    <button
                                                        type="button"
                                                        class:selected={session.answers[i] === action.id}
                                                        class:fault={session.revealed && session.faults[i] && session.answers[i] === action.id}
                                                        aria-pressed={session.answers[i] === action.id}
                                                        disabled={session.revealed}
                                                        data-testid="training-choice-{i}-{action.id}"
                                                        onclick={() => choose(i, action.id)}
                                                    >
                                                        {$t(action.labelKey)}
                                                    </button>
                                                {/each}
                                            </div>
                                            {#if session.revealed}
                                                <!-- Verdict du moteur à quatre issues, « trop bon » compris. -->
                                                <p class="truth" data-testid="training-truth-{i}">
                                                    <span class="mark" aria-hidden="true">{session.faults[i] ? '×' : ''}</span><span class:fault={session.faults[i]}
                                                        >{$t(`cube.verdicts.${question.cubeVerdict}`)}</span
                                                    >
                                                </p>
                                            {/if}
                                        </td>
                                    {:else}
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
                                                        validateFromField();
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
                                    {/if}
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                    {#if session.revealed && question.kind === 'evaluation'}
                        <!-- Montré, jamais demandé, et seulement s'il est exact (ADR-0027). -->
                        {#if question.epc}
                            <p class="hint" data-testid="training-epc">
                                {$t('training.epcShown', { p1: $t('board.player1'), a: epcOf(question.epc.bottom), p2: $t('board.player2'), b: epcOf(question.epc.top) })}
                            </p>
                        {/if}
                        <p class="hint" data-testid="training-truth-source">
                            {question.regime === 'exact' ? $t('training.truthExact') : $t('training.truthEvaluated', { depth: question.depth })}
                        </p>
                    {/if}
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

            {#if question && session.revealed && !session.outOfTime && !entered && !judgedByEngine}
                <!-- Consigne à l'écran, pas en infobulle. -->
                <p class="hint" data-testid="training-fault-hint" title={question.kind === 'scores' ? $t('training.scoresInstruction') : undefined}>{$t('training.faultHint')}</p>
            {/if}
            {#if question && !session.revealed && entered}
                <!-- Tolérance écrite en toutes lettres par langue ; `trainingTab.test.js`
                     la lie à `EPC_TOLERANCE`. -->
                <p class="hint">{$t(session.exercise === 'evaluation' ? 'training.toleranceEvaluation' : 'training.tolerance')}</p>
            {/if}
        </div>
    {:else}
        <PanelHeader title={$t('training.title')}>
            {#snippet actions()}
                <button type="button" class="launch" data-testid="training-start" disabled={libraryMissing} onclick={start}>{$t('training.start')}</button>
            {/snippet}
        </PanelHeader>

        <div class="launcher">
            <FormGrid>
                <FormRow label={$t('training.exerciseLabel')}>
                    <div class="choices" role="group" aria-label={$t('training.exerciseLabel')}>
                        {#each TRAINING_EXERCISES as item (item.id)}
                            <button type="button" class:selected={exercise === item.id} data-testid="training-exercise-{item.id}" onclick={() => (exercise = item.id)}>
                                {$t(`training.exercise.${item.id}`)}
                            </button>
                        {/each}
                    </div>
                </FormRow>

                <!-- Décision ne connaît que la bibliothèque. -->
                {#if chosen.sources.length > 1}
                    <FormRow label={$t('training.source')}>
                        <div class="choices" role="group" aria-label={$t('training.source')}>
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
                    </FormRow>
                {/if}

                <FormRow label={$t('training.limit')}>
                    <div class="choices" role="group" aria-label={$t('training.limit')}>
                        {#each TIME_LIMITS as limit (limit)}
                            <button type="button" class:selected={limitSeconds === limit} data-testid="training-limit-{limit}" onclick={() => (limitSeconds = limit)}>
                                {limit === 0 ? $t('training.limitNone') : $t('training.limitSeconds', { n: limit })}
                            </button>
                        {/each}
                    </div>
                </FormRow>
            </FormGrid>

            {#if libraryMissing}
                <!-- Décision ne tire que de la bibliothèque : la raison est dite avant le clic (ADR-0041 règle 2). -->
                <p class="refusal" role="status" data-testid="training-refusal">{$t(refusalMessageKey('noLibrary'))}</p>
            {:else if $trainingRefusalStore}
                <!-- Refus affiché au lieu du clic, nommant le domaine (ADR-0041 règle 3). -->
                <p class="refusal" role="status" data-testid="training-refusal">{$t(refusalMessageKey($trainingRefusalStore))}</p>
            {/if}
        </div>

        <div class="journal">
            <h3>{$t('training.summary')}</h3>
            {#if !hasLibrary}
                <p class="muted" data-testid="training-journal-no-database">{$t('training.journalNoDatabase')}</p>
            {/if}
            {#each hasLibrary ? TRAINING_EXERCISES : [] as item (item.id)}
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
                        {#if item.id === 'decision' && summary.lastPr !== null}
                            <span class="figure">{$t('training.lastPr', { value: summary.lastPr.toFixed(2) })}</span>
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
                    {#if item.id === 'decision'}
                        <!-- Les ratés gardent leur position au journal : on y revient en un clic. -->
                        <div class="actions missed" data-testid="training-missed">
                            <button type="button" data-testid="training-missed-retake" onclick={retakeMissed}>{$t('training.missedRetake')}</button>
                            <button type="button" data-testid="training-missed-deck" onclick={missedToDeck}>{$t('training.missedDeck')}</button>
                            <button type="button" data-testid="training-missed-collection" onclick={missedToCollection}>{$t('training.missedCollection')}</button>
                        </div>
                    {/if}
                {/if}
            {/each}
        </div>
    {/if}
</div>

<style>
    .training-panel {
        /* Interface chrome is not text to copy; fields below opt back in. */
        user-select: none;
        -webkit-user-select: none;
        display: flex;
        flex-direction: column;
        text-align: start;
    }

    .session {
        display: flex;
        flex-direction: column;
        gap: var(--space-2);
        padding: 0 var(--space-2) var(--space-2);
    }

    /* The band runs edge to edge over the padded session. */
    .session > :global(.panel-header) {
        margin: 0 calc(-1 * var(--space-2));
    }

    .launcher > .refusal {
        padding: 0 var(--space-2);
    }

    .journal {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        padding: var(--space-2);
    }

    .actions,
    .summary-line {
        display: flex;
        align-items: center;
        gap: var(--space-1) var(--space-2);
        flex-wrap: wrap;
    }

    .detail-label {
        color: var(--color-text-muted);
    }

    .choices button,
    .actions button,
    .launch {
        cursor: pointer;
        padding: 2px var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
    }

    /* The gesture that moves the session on: filled, at the end of the band (ADR-0085, G3). */
    .actions button.launch,
    button.launch {
        font-size: var(--font-size-base);
        font-weight: 600;
        padding: 2px var(--space-3);
        background: var(--color-primary);
        border-color: var(--color-primary);
        color: white;
    }

    button.launch:disabled {
        cursor: default;
        opacity: 0.5;
    }

    .notation {
        width: 100%;
        box-sizing: border-box;
        margin-top: 0.4em;
        font-family: monospace;
    }

    .notation.fault {
        border-color: var(--color-danger, #c00);
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
    .hint,
    .prompt,
    .verdict {
        margin: 0;
    }

    .decision {
        display: flex;
        flex-direction: column;
        gap: 0.4em;
    }

    .verdict {
        display: flex;
        flex-wrap: wrap;
        gap: 0.5em;
        color: var(--color-danger);
    }

    /* Pas de jeton « succès » (ADR-0031) : l'accent suffit. */
    .verdict.correct {
        color: var(--color-primary);
    }

    .verdict .best {
        color: var(--color-text-muted);
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

    .entered p.truth {
        margin: 0.25em 0 0;
        padding-left: 0;
    }

    .choices button.fault {
        border-color: var(--color-danger);
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
        font-size: var(--font-size-small);
        font-weight: 600;
        color: var(--color-text-muted);
    }

    .disclose {
        border: none;
        background: transparent;
        padding: 0;
        color: var(--color-text);
        cursor: pointer;
        text-align: left;
        min-width: 10em;
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

    .training-panel input {
        user-select: text;
        -webkit-user-select: text;
    }
</style>
