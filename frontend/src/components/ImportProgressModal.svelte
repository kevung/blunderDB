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
            importPath: ''
        },
        result = {
            added: 0,
            merged: 0,
            skipped: 0,
            total: 0
        },
        onCancel,
        onCommit,
        onClose
    } = $props();

    // Escape only closes once a terminal state with a visible "Fermer" button is
    // reached: mode 'completed', or the 'preview' branch that has nothing to
    // import (its own "Fermer" button, no commit/cancel choice to make). While
    // analyzing/committing, Escape must not silently abandon the import.
    let nothingToImport = $derived(analysis.toAdd === 0 && analysis.toMerge === 0);
    let closable = $derived(mode === 'completed' || (mode === 'preview' && nothingToImport));
</script>

<Modal open={visible} onclose={onClose} size="large" layer="top" closeButton={false} closeOnEscape={closable} label={$t('import.progressTitle')}>
    {#if mode === 'analyzing'}
        <h2>{$t('import.analyzing')} <span class="spinner"></span></h2>
        <p class="status-text">{$t('import.analyzingWait')}</p>
    {:else if mode === 'preview'}
        <h2>{$t('import.previewTitle')}</h2>

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
        <h2>{$t('import.committing')} <span class="spinner"></span></h2>
        <p class="status-text">{$t('import.committingWait')}</p>
        <p class="status-text">{$t('import.committingAtomic')}</p>
    {:else if mode === 'completed'}
        <h2>{$t('import.completedTitle')}</h2>

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
    {/if}
    {#snippet footer()}
        {#if mode === 'analyzing'}
            <button onclick={onCancel}>{$t('common.cancel')}</button>
        {:else if mode === 'preview' && nothingToImport}
            <button onclick={onClose}>{$t('common.close')}</button>
        {:else if mode === 'preview'}
            <button onclick={onCancel}>{$t('common.cancel')}</button>
            <button class="btn-commit" onclick={onCommit}>{$t('import.commitImport')}</button>
        {:else if mode === 'committing'}
            <button onclick={onCancel}>{$t('import.abortImport')}</button>
        {:else if mode === 'completed'}
            <button onclick={onClose}>{$t('common.close')}</button>
        {/if}
    {/snippet}
</Modal>

<style>
    h2 {
        margin: 0;
        font-size: var(--font-size-dialog-title);
        color: #333;
    }

    .status-text {
        color: #666;
        font-size: var(--font-size-base);
        margin: 0;
    }

    .stats {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 15px;
        margin-top: 10px;
    }

    .stat-item {
        text-align: center;
        padding: 15px;
        background-color: #f5f5f5;
        border-radius: 4px;
        border: 1px solid #ddd;
    }

    .stat-label {
        font-size: var(--font-size-base);
        color: #666;
        text-transform: uppercase;
        margin-bottom: 5px;
    }

    .stat-value {
        font-size: var(--font-size-stat-figure);
        font-weight: bold;
        color: #333;
    }

    .spinner {
        display: inline-block;
        width: 16px;
        height: 16px;
        border: 3px solid #e0e0e0;
        border-top: 3px solid #666;
        border-radius: 50%;
        animation: spin 1s linear infinite;
        margin-left: 10px;
        vertical-align: middle;
    }

    @keyframes spin {
        0% {
            transform: rotate(0deg);
        }
        100% {
            transform: rotate(360deg);
        }
    }

    button {
        padding: 10px 20px;
        border: 1px solid #ccc;
        border-radius: 4px;
        font-size: var(--font-size-base);
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s ease;
        background-color: white;
        color: #333;
    }

    button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    button:hover:not(:disabled) {
        background-color: #f5f5f5;
        border-color: #999;
    }

    .btn-commit {
        background-color: #333;
        color: white;
        border-color: #333;
    }

    .btn-commit:hover:not(:disabled) {
        background-color: #555;
        border-color: #555;
    }

    .summary {
        background-color: #f9f9f9;
        padding: 15px;
        border-radius: 4px;
        border-left: 4px solid #666;
    }

    .summary p {
        margin: 5px 0;
        font-size: var(--font-size-base);
        color: #555;
    }

    .summary strong {
        color: #333;
    }

    .warning {
        background-color: #f5f5f5;
        border-left-color: #999;
    }
</style>
