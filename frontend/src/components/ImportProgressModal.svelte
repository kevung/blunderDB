<script>
    import Modal from './Modal.svelte';
    import { t } from '../i18n';

    let {
        visible = false,
        mode = 'analyzing',
        analysis = {
            toAdd: 0,
            toMerge: 0,
            toSkip: 0,
            total: 0,
            matches: 0,
            matchesKnown: 0,
            importPath: ''
        },
        result = {
            added: 0,
            merged: 0,
            skipped: 0,
            total: 0,
            matches: 0,
            matchesKnown: 0,
            livingCollectionsFrozen: [],
            livingCollectionsOnManual: [],
            unreadable: []
        },
        onCancel,
        onCommit,
        onClose
    } = $props();

    // Escape only closes once a terminal state with a visible "Fermer" button is
    // reached: mode 'completed', or the 'preview' branch that has nothing to
    // import (its own "Fermer" button, no commit/cancel choice to make). While
    // analyzing/committing, Escape must not silently abandon the import.
    let nothingToImport = $derived(analysis.toAdd === 0 && analysis.toMerge === 0 && !analysis.matches);
    const notes = $derived(
        [
            ['import.livingFrozen', result.livingCollectionsFrozen],
            ['import.livingOnManual', result.livingCollectionsOnManual],
            ['import.unreadable', result.unreadable]
        ].filter(([, names]) => names && names.length > 0)
    );
    let closable = $derived(mode === 'completed' || (mode === 'preview' && nothingToImport));
</script>

<Modal open={visible} onclose={onClose} size="large" layer="top" closeButton={false} closeOnEscape={closable} label={$t('import.progressTitle')}>
    {#if mode === 'analyzing'}
        <h2 class="modal-title">{$t('import.analyzing')} <span class="spinner"></span></h2>
        <p class="status-text">{$t('import.analyzingWait')}</p>
    {:else if mode === 'preview'}
        <h2 class="modal-title">{$t('import.previewTitle')}</h2>

        <div class="summary">
            <p><strong>{$t('import.databaseToImport')}</strong> {$t('import.positionCount', { count: analysis.total })}</p>
            <p>{$t('import.willMakeChanges')}</p>
        </div>

        <div class="stats">
            <div class="stat-item">
                <div class="stat-label">{$t('import.willAdd')}</div>
                <div class="stat-value">{analysis.toAdd}</div>
            </div>
            <div class="stat-item">
                <div class="stat-label">{$t('import.willMerge')}</div>
                <div class="stat-value">{analysis.toMerge}</div>
            </div>
            <div class="stat-item">
                <div class="stat-label">{$t('import.willSkip')}</div>
                <div class="stat-value">{analysis.toSkip}</div>
            </div>
        </div>

        {#if analysis.matches > 0 || analysis.matchesKnown > 0}
            <div class="stats">
                <div class="stat-item" data-testid="import-matches">
                    <div class="stat-label">{$t('import.matchesToAdd')}</div>
                    <div class="stat-value">{analysis.matches}</div>
                </div>
                <div class="stat-item" data-testid="import-matches-known">
                    <div class="stat-label">{$t('import.matchesToSkip')}</div>
                    <div class="stat-value">{analysis.matchesKnown}</div>
                </div>
            </div>
        {/if}

        {#if analysis.toMerge > 0}
            <div class="summary warning">
                <p><strong>{$t('import.note')}</strong> {$t('import.mergeNote', { count: analysis.toMerge })}</p>
            </div>
        {/if}

        {#if nothingToImport}
            <div class="summary warning">
                <p><strong>{$t('import.nothingToImport')}</strong> {$t('import.nothingToImportDetail')}</p>
            </div>
        {/if}
    {:else if mode === 'committing'}
        <h2 class="modal-title">{$t('import.committing')} <span class="spinner"></span></h2>
        <p class="status-text">{$t('import.committingWait')}</p>
        <p class="status-text">{$t('import.committingAtomic')}</p>
    {:else if mode === 'completed'}
        <h2 class="modal-title">{$t('import.completedTitle')}</h2>

        <div class="summary">
            <p><strong>{$t('import.successful')}</strong> {$t('import.databaseUpdated')}</p>
        </div>

        <div class="stats">
            <div class="stat-item">
                <div class="stat-label">{$t('import.added')}</div>
                <div class="stat-value">{result.added}</div>
            </div>
            <div class="stat-item">
                <div class="stat-label">{$t('import.merged')}</div>
                <div class="stat-value">{result.merged}</div>
            </div>
            <div class="stat-item">
                <div class="stat-label">{$t('import.skipped')}</div>
                <div class="stat-value">{result.skipped}</div>
            </div>
        </div>

        {#if result.matches > 0 || result.matchesKnown > 0}
            <div class="stats">
                <div class="stat-item" data-testid="result-matches">
                    <div class="stat-label">{$t('import.matchesAdded')}</div>
                    <div class="stat-value">{result.matches}</div>
                </div>
                <div class="stat-item" data-testid="result-matches-known">
                    <div class="stat-label">{$t('import.matchesKnown')}</div>
                    <div class="stat-value">{result.matchesKnown}</div>
                </div>
            </div>
        {/if}

        {#each notes as [key, names] (key)}
            <div class="summary warning" data-testid={key}>
                <p>{$t(key, { names: names.join(', ') })}</p>
            </div>
        {/each}
    {/if}
    {#snippet footer()}
        {#if mode === 'analyzing'}
            <button onclick={onCancel}>{$t('common.cancel')}</button>
        {:else if mode === 'preview' && nothingToImport}
            <button onclick={onClose}>{$t('common.close')}</button>
        {:else if mode === 'preview'}
            <button onclick={onCancel}>{$t('common.cancel')}</button>
            <button class="primary" onclick={onCommit}>{$t('import.commitImport')}</button>
        {:else if mode === 'committing'}
            <button onclick={onCancel}>{$t('import.abortImport')}</button>
        {:else if mode === 'completed'}
            <button onclick={onClose}>{$t('common.close')}</button>
        {/if}
    {/snippet}
</Modal>

<style>
    .stats {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 15px;
        margin-top: 10px;
    }

    .stat-item {
        text-align: center;
        padding: 15px;
        background-color: var(--color-surface-alt);
        border-radius: 4px;
        border: 1px solid var(--color-border);
    }

    .stat-label {
        font-size: var(--font-size-base);
        color: var(--color-text-muted);
        text-transform: uppercase;
        margin-bottom: 5px;
    }

    .stat-value {
        font-size: var(--font-size-stat-figure);
        font-weight: bold;
        color: var(--color-text);
    }
</style>
