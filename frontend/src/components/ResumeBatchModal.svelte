<script>
    // Reprendre un lot d'import que rien n'a terminé : coupure de l'application.
    // Le journal du lot (déjà en base) dit ce qui est décidé ; on ne redemande à
    // l'utilisateur que l'endroit où sont les fichiers, comme `--dir` en CLI.
    import Modal from './Modal.svelte';
    import { ListImportBatches } from '../../wailsjs/go/database/Database.js';
    import { pickFilesToResumeFromFolder, pickFilesToResume, resumeImportBatch } from '../services/importService.js';
    import { fileImportModeStore } from '../stores/importModalStore.js';
    import { logger } from '../utils/logger.js';
    import { t } from '../i18n';

    let { visible = false, onClose } = $props();

    /** @typedef {import('../../wailsjs/go/models').domain.ImportBatch} ImportBatch */

    /** @type {ImportBatch[]} */
    let batches = $state([]);
    let busy = $state(false);
    // A running import has no end date either: resuming its batch would start a second one on it.
    let importing = $derived($fileImportModeStore === 'importing');

    // A batch the import never finished has no end date.
    const PAGE = 200;

    $effect(() => {
        if (visible) load();
    });

    async function load() {
        try {
            batches = ((await ListImportBatches(PAGE, 0)) || []).filter((b) => !b.finishedAt);
        } catch (error) {
            logger.error('could not list the import batches:', error);
            batches = [];
        }
    }

    /**
     * @param {ImportBatch} batch
     * @param {'folder' | 'files'} how
     */
    async function resume(batch, how) {
        if (importing) return;
        busy = true;
        try {
            const files = how === 'folder' ? await pickFilesToResumeFromFolder() : await pickFilesToResume();
            if (!files) return;
            // The import has its own progress window: this one steps aside first.
            onClose?.();
            await resumeImportBatch(batch.id, files);
        } finally {
            busy = false;
        }
    }
</script>

<Modal open={visible} onclose={onClose} size="large" label={$t('resumeBatch.title')}>
    <h2 class="modal-title">{$t('resumeBatch.title')}</h2>

    {#if importing}
        <p class="empty" data-testid="resume-batch-running">{$t('resumeBatch.running')}</p>
    {:else if batches.length === 0}
        <p class="empty" data-testid="resume-batch-empty">{$t('resumeBatch.empty')}</p>
    {:else}
        <p class="hint">{$t('resumeBatch.hint')}</p>
        <div class="list" data-testid="resume-batch-list">
            {#each batches as batch (batch.id)}
                <div class="row" data-testid="resume-batch-row">
                    <span class="label" title={batch.source}>{batch.source}</span>
                    <span class="date">{batch.startedAt}</span>
                    <button disabled={busy} onclick={() => resume(batch, 'folder')}>{$t('resumeBatch.chooseFolder')}</button>
                    <button disabled={busy} onclick={() => resume(batch, 'files')}>{$t('resumeBatch.chooseFiles')}</button>
                </div>
            {/each}
        </div>
    {/if}

    {#snippet footer()}
        <button onclick={onClose}>{$t('common.close')}</button>
    {/snippet}
</Modal>

<style>
    .empty,
    .hint {
        color: var(--color-text-muted);
    }
    .list {
        max-height: 60vh;
        overflow-y: auto;
    }
    .row {
        display: grid;
        grid-template-columns: minmax(0, 1fr) 11em auto auto;
        align-items: baseline;
        gap: var(--space-2);
        padding: var(--space-1) 0;
        border-bottom: 1px solid var(--color-border);
    }
    .date {
        color: var(--color-text-muted);
    }
    .label {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
</style>
