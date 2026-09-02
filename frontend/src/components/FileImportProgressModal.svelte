<script>
    import Modal from './Modal.svelte';
    import { t } from '../i18n';

    let { visible = false, mode = 'idle', totalFiles = 0, currentIndex = 0, currentFile = '', results = { succeeded: 0, failed: 0, skipped: 0, errors: [] }, onClose, onCancel } = $props();

    let progressPercent = $derived(totalFiles > 0 ? Math.round((currentIndex / totalFiles) * 100) : 0);
    // Escape only closes once the terminal state is reached (the "Fermer" button
    // is visible) — while importing is in progress, Escape must not silently
    // abandon it: the user has Annuler for that, deliberately.
    let closable = $derived(mode === 'completed');

    function basename(path) {
        if (!path) return '';
        return path.split('/').pop().split('\\').pop();
    }
</script>

<Modal open={visible} onclose={onClose} size="large" layer="top" closeButton={false} closeOnEscape={closable} label={$t('import.fileProgressTitle')}>
    {#if mode === 'importing'}
        <h2>{$t('import.importingFiles')} <span class="spinner"></span></h2>
        <p class="status-text">{$t('import.importingFileN', { current: currentIndex, total: totalFiles })}</p>
        <p class="current-file" title={currentFile}>{basename(currentFile)}</p>

        <div class="progress-bar-container">
            <div class="progress-bar" style="width: {progressPercent}%"></div>
        </div>
        <p class="progress-text">{progressPercent}%</p>
    {:else if mode === 'completed'}
        <h2>{$t('import.completedTitle')}</h2>

        <div class="summary">
            <p><strong>{$t('import.finished')}</strong> {$t('import.processedN', { processed: results.succeeded + results.failed + results.skipped, total: totalFiles })}</p>
        </div>

        <div class="stats">
            <div class="stat-item">
                <div class="stat-label">{$t('import.imported')}</div>
                <div class="stat-value">{results.succeeded}</div>
            </div>
            <div class="stat-item">
                <div class="stat-label">{$t('import.skipped')}</div>
                <div class="stat-value">{results.skipped}</div>
            </div>
            <div class="stat-item">
                <div class="stat-label">{$t('import.failed')}</div>
                <div class="stat-value" class:errors={results.failed > 0}>{results.failed}</div>
            </div>
        </div>

        {#if results.errors.length > 0}
            <div class="error-list">
                {#each results.errors as err, i (i)}
                    <div class="error-item">
                        <span class="error-file">{basename(err.file)}</span>: {err.message}
                    </div>
                {/each}
            </div>
        {/if}
    {/if}
    {#snippet footer()}
        {#if mode === 'importing'}
            <button onclick={onCancel}>{$t('common.cancel')}</button>
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

    .current-file {
        color: #999;
        font-size: var(--font-size-base);
        margin: 4px 0 0 0;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .progress-bar-container {
        width: 100%;
        background-color: #e0e0e0;
        border-radius: 4px;
        overflow: hidden;
        height: 8px;
    }

    .progress-bar {
        height: 100%;
        background-color: #333;
        transition: width 0.2s ease;
        border-radius: 4px;
    }

    .progress-text {
        font-size: var(--font-size-base);
        color: #666;
        text-align: right;
        margin-top: 4px;
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

    .stat-value.errors {
        color: #c33;
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

    .error-list {
        max-height: 150px;
        overflow-y: auto;
        background-color: #fff5f5;
        border: 1px solid #e0c0c0;
        border-radius: 4px;
        padding: 10px;
    }

    .error-item {
        font-size: var(--font-size-base);
        color: #833;
        margin: 4px 0;
        word-break: break-all;
    }

    .error-item .error-file {
        font-weight: 600;
    }
</style>
