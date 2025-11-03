<script>
    import { onMount, onDestroy } from 'svelte';

    export let visible = false;
    export let mode = 'preparing'; // 'preparing', 'metadata', 'exporting', 'completed'
    export let positionCount = 0;
    export let onCancel;
    export let onExport;
    export let onClose;
    export let metadata = {
        user: '',
        description: '',
        dateOfCreation: ''
    };
    export let exportOptions = {
        includeAnalysis: true,
        includeComments: true,
        includeFilterLibrary: false
    };

    // Get current date in YYYY-MM-DD format
    function getCurrentDate() {
        const now = new Date();
        const year = now.getFullYear();
        const month = String(now.getMonth() + 1).padStart(2, '0');
        const day = String(now.getDate()).padStart(2, '0');
        return `${year}-${month}-${day}`;
    }

    // Initialize date when modal becomes visible in metadata mode
    $: if (visible && mode === 'metadata' && !metadata.dateOfCreation) {
        metadata.dateOfCreation = getCurrentDate();
    }

    // Computed description of what will be exported
    $: exportDescription = (() => {
        let parts = [];
        if (exportOptions.includeAnalysis) parts.push('analysis');
        if (exportOptions.includeComments) parts.push('comments');
        if (exportOptions.includeFilterLibrary) parts.push('filter library');
        
        if (parts.length === 0) {
            return 'positions only';
        } else if (parts.length === 1) {
            return `${parts[0]}`;
        } else if (parts.length === 2) {
            return `${parts[0]} and ${parts[1]}`;
        } else {
            return `${parts.slice(0, -1).join(', ')}, and ${parts[parts.length - 1]}`;
        }
    })();

    function handleKeyDown(event) {
        if (event.key === 'Escape' && visible) {
            onCancel();
        }
    }

    onMount(() => {
        window.addEventListener('keydown', handleKeyDown);
    });

    onDestroy(() => {
        window.removeEventListener('keydown', handleKeyDown);
    });
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

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    label {
        font-size: 14px;
        font-weight: 500;
        color: #333;
    }

    input, textarea {
        padding: 8px 12px;
        border: 1px solid #ccc;
        border-radius: 4px;
        font-size: 14px;
        font-family: inherit;
    }

    input:focus, textarea:focus {
        outline: none;
        border-color: #666;
    }

    textarea {
        resize: vertical;
        min-height: 80px;
    }

    .checkbox-group {
        display: flex;
        flex-direction: column;
        gap: 10px;
        padding: 15px;
        background-color: #f9f9f9;
        border-radius: 4px;
        border: 1px solid #ddd;
    }

    .checkbox-item {
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .checkbox-item input[type="checkbox"] {
        width: 18px;
        height: 18px;
        cursor: pointer;
        accent-color: #333;
    }

    .checkbox-item label {
        margin: 0;
        cursor: pointer;
        font-weight: normal;
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

    .btn-export {
        background-color: #333;
        color: white;
        border-color: #333;
    }

    .btn-export:hover:not(:disabled) {
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
</style>

{#if visible}
    <div class="modal-overlay">
        <div class="modal-content">
            {#if mode === 'preparing'}
                <h2>Preparing Export <span class="spinner"></span></h2>
                <p class="status-text">Counting positions to export...</p>
                
                <div class="button-group">
                    <button on:click={onCancel}>Cancel</button>
                </div>
            
            {:else if mode === 'metadata'}
                <h2>Export Database</h2>
                
                <div class="summary">
                    <p><strong>{positionCount} position(s)</strong> will be exported with {exportDescription}.</p>
                </div>

                <div class="form-group">
                    <label for="export-user">User</label>
                    <input 
                        id="export-user" 
                        type="text" 
                        bind:value={metadata.user}
                        placeholder="Enter your name (optional)"
                    />
                </div>

                <div class="form-group">
                    <label for="export-description">Description</label>
                    <textarea 
                        id="export-description" 
                        bind:value={metadata.description}
                        placeholder="Enter a description for this database (optional)"
                    ></textarea>
                </div>

                <div class="form-group">
                    <label for="export-date">Creation Date</label>
                    <input 
                        id="export-date" 
                        type="date" 
                        bind:value={metadata.dateOfCreation}
                    />
                </div>

                <div class="checkbox-group">
                    <div class="checkbox-item">
                        <input 
                            type="checkbox" 
                            id="export-analysis" 
                            bind:checked={exportOptions.includeAnalysis}
                        />
                        <label for="export-analysis">Include analysis</label>
                    </div>
                    <div class="checkbox-item">
                        <input 
                            type="checkbox" 
                            id="export-comments" 
                            bind:checked={exportOptions.includeComments}
                        />
                        <label for="export-comments">Include comments</label>
                    </div>
                    <div class="checkbox-item">
                        <input 
                            type="checkbox" 
                            id="export-filter-library" 
                            bind:checked={exportOptions.includeFilterLibrary}
                        />
                        <label for="export-filter-library">Include filter library</label>
                    </div>
                </div>

                <div class="button-group">
                    <button on:click={onCancel}>Cancel</button>
                    <button class="btn-export" on:click={onExport}>Export</button>
                </div>

            {:else if mode === 'exporting'}
                <h2>Exporting Database <span class="spinner"></span></h2>
                <p class="status-text">Exporting {positionCount} position(s) to the new database...</p>
                <p class="status-text">This may take a few moments.</p>

                <div class="button-group">
                    <button on:click={onCancel}>Cancel</button>
                </div>

            {:else if mode === 'completed'}
                <h2>Export Completed</h2>
                
                <div class="summary">
                    <p><strong>Export successful!</strong> The database has been created with {positionCount} position(s).</p>
                </div>

                <div class="button-group">
                    <button on:click={onClose}>Close</button>
                </div>
            {/if}
        </div>
    </div>
{/if}
