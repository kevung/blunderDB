<script>
    // La barre de la file d'étude (#259, fiche I.3).
    //
    // Quatre gestes par position, et trois d'entre eux ouvrent simplement le
    // panneau où ce geste se prend déjà. C'est délibéré : commenter, ranger en
    // collection et faire une carte existent, avec leurs règles et leurs
    // messages ; les refaire ici en aurait fait des demi-copies qui dérivent.
    // La file apporte l'ordre et le parcours, pas de nouveaux gestes.
    import { studyQueueStore, studyQueueIndexStore, studyQueueActiveStore, studyQueueCurrentStore } from '../stores/studyQueueStore.js';
    import { nextInQueue, previousInQueue, stopStudyQueue, actOnCurrent } from '../services/studyQueueService.js';
    import { t } from '../i18n';

    let total = $derived($studyQueueStore.length);
    let position = $derived($studyQueueIndexStore + 1);
    let entry = $derived($studyQueueCurrentStore);

    /** @param {string} reason */
    function reasonLabel(reason) {
        switch (reason) {
            case 'blunder':
                return $t('studyQueue.reasonBlunder');
            case 'flagged':
                return $t('studyQueue.reasonFlagged');
            case 'close':
                return $t('studyQueue.reasonClose');
            default:
                return reason;
        }
    }
</script>

{#if $studyQueueActiveStore && entry}
    <div class="study-queue-bar" role="region" aria-label={$t('studyQueue.title')}>
        <span class="progress">{$t('studyQueue.progress', { i: position, n: total })}</span>
        <span class="reason">
            {reasonLabel(entry.reason)}
            {#if entry.errorMp > 0}
                <span class="cost">{(entry.errorMp / 1000).toFixed(3)}</span>
            {/if}
        </span>
        <span class="match">{entry.label}</span>
        <span class="actions">
            <button type="button" onclick={() => actOnCurrent('comments')}>{$t('studyQueue.comment')}</button>
            <button type="button" onclick={() => actOnCurrent('collections')}>{$t('studyQueue.collection')}</button>
            <button type="button" onclick={() => actOnCurrent('anki')}>{$t('studyQueue.anki')}</button>
            <button type="button" disabled={position <= 1} onclick={previousInQueue}>{$t('studyQueue.previous')}</button>
            <button type="button" onclick={nextInQueue}>{position >= total ? $t('studyQueue.finish') : $t('studyQueue.skip')}</button>
            <button type="button" onclick={() => stopStudyQueue()}>{$t('studyQueue.leave')}</button>
        </span>
    </div>
{/if}

<style>
    .study-queue-bar {
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

    .reason,
    .match {
        color: var(--color-text-muted);
    }

    .cost {
        color: var(--color-danger);
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
