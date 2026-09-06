<script>
    // La barre d'entraînement (#273, fiche I.17).
    //
    // La question EST la position affichée : compter les pions d'un plateau
    // qu'on ne voit pas n'a pas de sens, et recopier le plateau dans une
    // fenêtre à part aurait fait un second rendu à tenir en phase avec le
    // premier. La barre ne porte donc que la question, la saisie et la
    // correction — le plateau, lui, est celui de l'application.
    //
    // Le point de prise est l'exception qui confirme la règle : sa question
    // est un score, pas un damier, et elle s'écrit dans la barre.
    import { trainingActiveStore, trainingCurrentStore, trainingIndexStore, trainingQuestionsStore, trainingVerdictStore } from '../stores/trainingStore.js';
    import { answerCurrent, nextQuestion, stopTraining } from '../services/trainingSessionService.js';
    import { TOLERANCE } from '../services/trainingService.js';
    import { t } from '../i18n';

    let input = $state('');
    let field;

    let total = $derived($trainingQuestionsStore.length);
    let index = $derived($trainingIndexStore + 1);
    let question = $derived($trainingCurrentStore);
    let verdict = $derived($trainingVerdictStore);

    // La saisie se vide et reprend le focus à chaque nouvelle question :
    // l'exercice se fait au clavier, et redemander la souris entre deux
    // questions casserait le rythme qu'on chronomètre.
    $effect(() => {
        void $trainingIndexStore;
        void $trainingActiveStore;
        input = '';
        field?.focus();
    });

    function submit() {
        if (verdict) {
            nextQuestion();
            return;
        }
        const value = parseFloat(String(input).replace(',', '.'));
        answerCurrent(value);
    }

    /** @param {KeyboardEvent} event */
    function onKeydown(event) {
        if (event.key === 'Enter') {
            event.preventDefault();
            submit();
        } else if (event.key === 'Escape') {
            event.preventDefault();
            stopTraining();
        }
        // Les autres touches restent dans le champ : sans cela le raccourci
        // global d'une lettre partirait pendant qu'on tape une réponse.
        event.stopPropagation();
    }

    function label(drill) {
        switch (drill) {
            case 'pips':
                return $t('training.drillPips');
            case 'epc':
                return $t('training.drillEpc');
            case 'takepoint':
                return $t('training.drillTakePoint');
            default:
                return drill;
        }
    }

    function decimals(drill) {
        return TOLERANCE[drill] === 0 ? 0 : 1;
    }
</script>

{#if $trainingActiveStore && question}
    <div class="training-bar" role="region" aria-label={$t('training.title')}>
        <span class="progress">{$t('training.progress', { i: index, n: total })}</span>
        <span class="drill">{label(question.drill)}</span>
        {#if question.drill === 'takepoint'}
            <span class="prompt">{$t('training.takePointPrompt', { away: question.prompt.replace(':', '/') })}</span>
        {/if}
        {#if verdict}
            <span class="verdict" class:correct={verdict.correct}>
                {verdict.correct ? $t('training.right') : $t('training.wrong')}
                — {$t('training.truth', { value: question.truth.toFixed(decimals(question.drill)) })}
            </span>
        {/if}
        <span class="actions">
            <input
                bind:this={field}
                bind:value={input}
                type="text"
                inputmode="decimal"
                disabled={!!verdict}
                aria-label={$t('training.answer')}
                placeholder={$t('training.answer')}
                onkeydown={onKeydown}
            />
            <button type="button" onclick={submit}>
                {verdict ? (index >= total ? $t('training.finish') : $t('training.next')) : $t('training.check')}
            </button>
            <button type="button" onclick={() => stopTraining()}>{$t('training.leave')}</button>
        </span>
    </div>
{/if}

<style>
    .training-bar {
        display: flex;
        align-items: center;
        gap: 0.8em;
        flex-wrap: wrap;
        padding: 0.25em 0.6em;
        border-bottom: 1px solid var(--color-border);
        background: var(--color-surface-alt);
    }

    .progress {
        font-weight: 600;
    }

    .drill,
    .prompt {
        color: var(--color-text-muted);
    }

    .verdict {
        color: var(--color-danger);
    }

    /* Pas de jeton « succès » dans la palette (ADR-0031) et ce n'est pas ici
       qu'on en ajoute un : l'accent unique dit « juste » aussi bien qu'un vert
       de plus, et une bonne réponse n'a pas besoin d'être fêtée. */
    .verdict.correct {
        color: var(--color-primary);
    }

    .actions {
        margin-left: auto;
        display: flex;
        gap: 0.3em;
    }

    .actions input {
        width: 7em;
        text-align: right;
    }

    .actions button {
        cursor: pointer;
    }
</style>
