<script>
    export let visible = false;
    export let mode = 'analyzing'; // 'analyzing', 'preview', 'committing', 'completed'
    export let analysis = {
        toAdd: 0,
        toMerge: 0,
        toSkip: 0,
        total: 0,
        importPath: ''
    };
    export let result = {
        added: 0,
        merged: 0,
        skipped: 0,
        total: 0
    };
    export let onCancel;
    export let onCommit;
    export let onClose;
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
        width: 500px;
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
        font-size: 14px;
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

{#if visible}
    <div class="modal-overlay">
        <div class="modal-content">
            {#if mode === 'analyzing'}
                <h2>Analyzing Import <span class="spinner"></span></h2>
                <p class="status-text">Please wait while we analyze the database to import...</p>
                
                <div class="button-group">
                    <button on:click={onCancel}>Cancel</button>
                </div>
            
            {:else if mode === 'preview'}
                <h2>Import Preview</h2>
                
                <div class="summary">
                    <p><strong>Database to import:</strong> {analysis.total} position(s)</p>
                    <p>The import operation will make the following changes:</p>
                </div>

                <div class="stats">
                    <div class="stat-item">
                        <div class="stat-label">Will Add</div>
                        <div class="stat-value">{analysis.toAdd}</div>
                    </div>
                    <div class="stat-item">
                        <div class="stat-label">Will Merge</div>
                        <div class="stat-value">{analysis.toMerge}</div>
                    </div>
                    <div class="stat-item">
                        <div class="stat-label">Will Skip</div>
                        <div class="stat-value">{analysis.toSkip}</div>
                    </div>
                </div>

                {#if analysis.toMerge > 0}
                    <div class="summary warning">
                        <p><strong>Note:</strong> {analysis.toMerge} position(s) already exist and will be merged with their analysis and comments.</p>
                    </div>
                {/if}

                {#if analysis.toAdd === 0 && analysis.toMerge === 0}
                    <div class="summary warning">
                        <p><strong>Nothing to import:</strong> All positions already exist in the database with identical data.</p>
                    </div>
                    <div class="button-group">
                        <button on:click={onClose}>Close</button>
                    </div>
                {:else}
                    <div class="button-group">
                        <button on:click={onCancel}>Cancel</button>
                        <button class="btn-commit" on:click={onCommit}>Commit Import</button>
                    </div>
                {/if}

            {:else if mode === 'committing'}
                <h2>Committing Import <span class="spinner"></span></h2>
                <p class="status-text">Please wait while the database is being imported...</p>
                <p class="status-text">This operation is atomic and will not modify your database until completion.</p>

                <div class="button-group">
                    <button on:click={onCancel}>Abort Import</button>
                </div>

            {:else if mode === 'completed'}
                <h2>Import Completed</h2>
                
                <div class="summary">
                    <p><strong>Import successful!</strong> The database has been updated.</p>
                </div>

                <div class="stats">
                    <div class="stat-item">
                        <div class="stat-label">Added</div>
                        <div class="stat-value">{result.added}</div>
                    </div>
                    <div class="stat-item">
                        <div class="stat-label">Merged</div>
                        <div class="stat-value">{result.merged}</div>
                    </div>
                    <div class="stat-item">
                        <div class="stat-label">Skipped</div>
                        <div class="stat-value">{result.skipped}</div>
                    </div>
                </div>

                <div class="button-group">
                    <button on:click={onClose}>Close</button>
                </div>
            {/if}
        </div>
    </div>
{/if}
