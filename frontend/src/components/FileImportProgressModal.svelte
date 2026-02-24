<script>
    export let visible = false;
    export let mode = 'idle'; // 'idle', 'picking', 'importing', 'completed'
    export let totalFiles = 0;
    export let currentIndex = 0;
    export let currentFile = '';
    export let results = { succeeded: 0, failed: 0, skipped: 0, errors: [] };
    export let onClose;
    export let onCancel;

    $: progressPercent = totalFiles > 0 ? Math.round((currentIndex / totalFiles) * 100) : 0;

    function basename(path) {
        if (!path) return '';
        return path.split('/').pop().split('\\').pop();
    }
</script>

<style>
    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background-color: rgba(0, 0, 0, 0.7);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 2000;
    }

    .modal-content {
        background-color: white;
        padding: 30px;
        border-radius: 8px;
        width: 520px;
        max-height: 80vh;
        box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
        display: flex;
        flex-direction: column;
        gap: 20px;
    }

    h2 {
        margin: 0;
        font-size: 20px;
        color: #333;
    }

    .status-text {
        color: #666;
        font-size: 14px;
        margin: 0;
    }

    .current-file {
        color: #999;
        font-size: 12px;
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
        font-size: 13px;
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
        font-size: 12px;
        color: #666;
        text-transform: uppercase;
        margin-bottom: 5px;
    }

    .stat-value {
        font-size: 28px;
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
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
    }

    .button-group {
        display: flex;
        gap: 10px;
        justify-content: flex-end;
        margin-top: 10px;
    }

    button {
        padding: 10px 20px;
        border: 1px solid #ccc;
        border-radius: 4px;
        font-size: 14px;
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

    .summary {
        background-color: #f9f9f9;
        padding: 15px;
        border-radius: 4px;
        border-left: 4px solid #666;
    }

    .summary p {
        margin: 5px 0;
        font-size: 14px;
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
        font-size: 12px;
        color: #833;
        margin: 4px 0;
        word-break: break-all;
    }

    .error-item .error-file {
        font-weight: 600;
    }
</style>

{#if visible}
    <div class="modal-overlay">
        <div class="modal-content">
            {#if mode === 'importing'}
                <h2>Importing Files <span class="spinner"></span></h2>
                <p class="status-text">Importing file {currentIndex} of {totalFiles}...</p>
                <p class="current-file" title={currentFile}>{basename(currentFile)}</p>

                <div class="progress-bar-container">
                    <div class="progress-bar" style="width: {progressPercent}%"></div>
                </div>
                <p class="progress-text">{progressPercent}%</p>

                <div class="button-group">
                    <button on:click={onCancel}>Cancel</button>
                </div>

            {:else if mode === 'completed'}
                <h2>Import Completed</h2>

                <div class="summary">
                    <p><strong>Import finished.</strong> Processed {results.succeeded + results.failed + results.skipped} of {totalFiles} file(s).</p>
                </div>

                <div class="stats">
                    <div class="stat-item">
                        <div class="stat-label">Imported</div>
                        <div class="stat-value">{results.succeeded}</div>
                    </div>
                    <div class="stat-item">
                        <div class="stat-label">Skipped</div>
                        <div class="stat-value">{results.skipped}</div>
                    </div>
                    <div class="stat-item">
                        <div class="stat-label">Failed</div>
                        <div class="stat-value" class:errors={results.failed > 0}>{results.failed}</div>
                    </div>
                </div>

                {#if results.errors.length > 0}
                    <div class="error-list">
                        {#each results.errors as err}
                            <div class="error-item">
                                <span class="error-file">{basename(err.file)}</span>: {err.message}
                            </div>
                        {/each}
                    </div>
                {/if}

                <div class="button-group">
                    <button on:click={onClose}>Close</button>
                </div>
            {/if}
        </div>
    </div>
{/if}
