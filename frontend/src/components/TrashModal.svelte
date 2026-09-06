<script>
    // La corbeille (#285, ADR-0036). Une suppression reste une suppression :
    // un instantané JSON de ce qui disparaît est écrit avant, et rien d'autre
    // dans la base ne sait que cette table existe. Ce panneau est la seule
    // fenêtre par laquelle on la regarde.
    import Modal from './Modal.svelte';
    import { ListTrash, RestoreFromTrash, DiscardFromTrash, EmptyTrash } from '../../wailsjs/go/database/Database.js';
    import { setStatusBarMessage } from '../services/databaseService.js';
    import { reloadPositions } from '../services/positionService.js';
    import { dbMutationCounterStore } from '../stores/uiStore';
    import { logger } from '../utils/logger.js';
    import { t, tMsg } from '../i18n';

    let { visible = false, onClose } = $props();

    let entries = $state([]);
    let busy = $state(false);

    $effect(() => {
        if (visible) {
            load();
        }
    });

    async function load() {
        try {
            entries = (await ListTrash('', 200, 0)) || [];
        } catch (error) {
            logger.error('could not read the trash:', error);
            entries = [];
        }
    }

    function kindLabel(kind) {
        switch (kind) {
            case 'position':
                return $t('trash.kindPosition');
            case 'collection':
                return $t('trash.kindCollection');
            case 'comment':
                return $t('trash.kindComment');
            case 'anki_card':
                return $t('trash.kindAnkiCard');
            default:
                return kind;
        }
    }

    async function restore(entry) {
        busy = true;
        try {
            await RestoreFromTrash(entry.id);
            setStatusBarMessage(tMsg('trash.restored', { what: entry.label }));
            dbMutationCounterStore.update((n) => n + 1);
            await reloadPositions();
            await load();
        } catch (error) {
            logger.error('restore failed:', error);
            setStatusBarMessage(tMsg('trash.restoreFailed', { error }));
        } finally {
            busy = false;
        }
    }

    async function discard(entry) {
        busy = true;
        try {
            await DiscardFromTrash(entry.id);
            await load();
        } catch (error) {
            logger.error('discard failed:', error);
        } finally {
            busy = false;
        }
    }

    async function empty() {
        busy = true;
        try {
            await EmptyTrash(0);
            await load();
        } catch (error) {
            logger.error('emptying the trash failed:', error);
        } finally {
            busy = false;
        }
    }
</script>

<Modal open={visible} onclose={onClose} size="large" label={$t('trash.title')}>
    <h2 class="modal-title">{$t('trash.title')}</h2>

    {#if entries.length === 0}
        <p class="empty">{$t('trash.empty')}</p>
    {:else}
        <p class="retention">{$t('trash.retention')}</p>
        <div class="list">
            {#each entries as entry (entry.id)}
                <div class="row">
                    <span class="kind">{kindLabel(entry.kind)}</span>
                    <span class="label" title={entry.label}>{entry.label}</span>
                    <span class="date">{entry.deletedAt}</span>
                    <button disabled={busy} onclick={() => restore(entry)}>{$t('trash.restore')}</button>
                    <button disabled={busy} onclick={() => discard(entry)}>{$t('trash.discard')}</button>
                </div>
            {/each}
        </div>
    {/if}

    {#snippet footer()}
        {#if entries.length > 0}
            <button disabled={busy} onclick={empty}>{$t('trash.emptyAction')}</button>
        {/if}
        <button onclick={onClose}>{$t('common.close')}</button>
    {/snippet}
</Modal>

<style>
    .empty,
    .retention {
        color: var(--color-text-muted);
    }
    .list {
        max-height: 60vh;
        overflow-y: auto;
    }
    .row {
        display: grid;
        grid-template-columns: 7em minmax(0, 1fr) 11em auto auto;
        align-items: baseline;
        gap: var(--space-2);
        padding: var(--space-1) 0;
        border-bottom: 1px solid var(--color-border);
    }
    .kind,
    .date {
        color: var(--color-text-muted);
    }
    .label {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
</style>
