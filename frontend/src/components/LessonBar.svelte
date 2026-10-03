<script>
    // La barre de lecture d'une Leçon (ADR-0066) : le texte de l'étape au-dessus du plateau,
    // l'étape précédente et la suivante. Ce que l'étape montre s'ouvre par les gestes existants.
    import { t } from '../i18n';
    import { lessonStore, lessonStepIndexStore, lessonStepStore } from '../stores/lessonStore.js';
    import { nextStep, previousStep, closeLesson } from '../services/lessonService.js';

    let total = $derived($lessonStore?.steps?.length ?? 0);
    let index = $derived($lessonStepIndexStore);
    let step = $derived($lessonStepStore);
</script>

{#if $lessonStore && step}
    <div class="lesson-bar" role="region" aria-label={$t('lesson.title')}>
        <div class="head">
            <span class="name">{$lessonStore.name}</span>
            <span class="progress">{$t('lesson.progress', { i: index + 1, n: total })}</span>
            {#if step.title}<span class="step-title">{step.title}</span>{/if}
            <span class="actions">
                <button type="button" onclick={previousStep} disabled={index === 0}>{$t('lesson.previous')}</button>
                <button type="button" onclick={nextStep} disabled={index + 1 >= total}>{$t('lesson.next')}</button>
                <button type="button" onclick={closeLesson}>{$t('lesson.close')}</button>
            </span>
        </div>
        {#if step.text}<p class="text">{step.text}</p>{/if}
    </div>
{/if}

<style>
    .lesson-bar {
        padding: 0.25em 0.6em;
        border-bottom: 1px solid var(--color-border);
        background: var(--color-surface-alt);
    }

    .head {
        display: flex;
        align-items: center;
        gap: 0.8em;
        flex-wrap: wrap;
    }

    .name,
    .progress {
        font-weight: 600;
    }

    .step-title {
        color: var(--color-text-muted);
    }

    .text {
        margin: 0.3em 0 0.1em;
        white-space: pre-wrap;
        max-height: 8em;
        overflow-y: auto;
    }

    .actions {
        margin-left: auto;
        display: flex;
        gap: 0.3em;
    }

    .actions button {
        cursor: pointer;
    }
</style>
