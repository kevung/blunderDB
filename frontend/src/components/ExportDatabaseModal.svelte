<script>
    import { onMount, onDestroy } from 'svelte';
    import { collectionsStore } from '../stores/collectionStore';
    import { tournamentsStore } from '../stores/tournamentStore';

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
        includeFilterLibrary: false,
        includePlayedMoves: true,
        includeMatches: true,
        matchIDs: [],
        includeTournaments: false,
        includeTournamentIDs: [],
        includeCollections: false,
        collectionIDs: []
    };

    export let matches = [];

    let collections = [];

    collectionsStore.subscribe(value => {
        collections = value || [];
    });

    let tournaments = [];

    tournamentsStore.subscribe(value => {
        tournaments = value || [];
    });

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

    // Auto-select all matches when includeMatches is toggled on (only if not manually modified)
    let matchesManuallyModified = false;
    $: if (exportOptions.includeMatches && matches.length > 0 && exportOptions.matchIDs.length === 0 && !matchesManuallyModified) {
        exportOptions.matchIDs = matches.map(m => m.id);
    }

    // Clear matchIDs when includeMatches is toggled off
    $: if (!exportOptions.includeMatches) {
        exportOptions.matchIDs = [];
        matchesManuallyModified = false;
    }

    // Auto-select all collections when includeCollections is toggled on (only if not manually modified)
    let collectionsManuallyModified = false;
    $: if (exportOptions.includeCollections && collections.length > 0 && exportOptions.collectionIDs.length === 0 && !collectionsManuallyModified) {
        exportOptions.collectionIDs = collections.map(c => c.id);
    }

    // Clear collectionIDs when includeCollections is toggled off
    $: if (!exportOptions.includeCollections) {
        exportOptions.collectionIDs = [];
        collectionsManuallyModified = false;
    }

    // Auto-select all tournaments when includeTournaments is toggled on (only if not manually modified)
    let tournamentsManuallyModified = false;
    $: if (exportOptions.includeTournaments && tournaments.length > 0 && exportOptions.includeTournamentIDs.length === 0 && !tournamentsManuallyModified) {
        exportOptions.includeTournamentIDs = tournaments.map(t => t.id);
    }

    // Clear tournamentIDs when includeTournaments is toggled off
    $: if (!exportOptions.includeTournaments) {
        exportOptions.includeTournamentIDs = [];
        tournamentsManuallyModified = false;
    }

    // Computed description of what will be exported
    $: exportDescription = (() => {
        let parts = [];
        if (exportOptions.includeAnalysis) parts.push('analysis');
        if (exportOptions.includeComments) parts.push('comments');
        if (exportOptions.includeFilterLibrary) parts.push('filter library');
        if (exportOptions.includePlayedMoves) parts.push('played moves');
        if (exportOptions.includeMatches && exportOptions.matchIDs.length > 0) parts.push(`${exportOptions.matchIDs.length} match${exportOptions.matchIDs.length > 1 ? 'es' : ''}`);
        if (exportOptions.includeTournaments && exportOptions.includeTournamentIDs.length > 0) parts.push(`${exportOptions.includeTournamentIDs.length} tournament${exportOptions.includeTournamentIDs.length > 1 ? 's' : ''}`);
        if (exportOptions.includeCollections && exportOptions.collectionIDs.length > 0) parts.push(`${exportOptions.collectionIDs.length} collection${exportOptions.collectionIDs.length > 1 ? 's' : ''}`);
        
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

    function toggleMatchSelection(matchId) {
        matchesManuallyModified = true;
        if (exportOptions.matchIDs.includes(matchId)) {
            exportOptions.matchIDs = exportOptions.matchIDs.filter(id => id !== matchId);
        } else {
            exportOptions.matchIDs = [...exportOptions.matchIDs, matchId];
        }
    }

    function selectAllMatches() {
        matchesManuallyModified = true;
        exportOptions.matchIDs = matches.map(m => m.id);
    }

    function selectNoMatches() {
        matchesManuallyModified = true;
        exportOptions.matchIDs = [];
    }

    function toggleCollectionSelection(collectionId) {
        collectionsManuallyModified = true;
        if (exportOptions.collectionIDs.includes(collectionId)) {
            exportOptions.collectionIDs = exportOptions.collectionIDs.filter(id => id !== collectionId);
        } else {
            exportOptions.collectionIDs = [...exportOptions.collectionIDs, collectionId];
        }
    }

    function toggleTournamentSelection(tournamentId) {
        tournamentsManuallyModified = true;
        if (exportOptions.includeTournamentIDs.includes(tournamentId)) {
            exportOptions.includeTournamentIDs = exportOptions.includeTournamentIDs.filter(id => id !== tournamentId);
        } else {
            exportOptions.includeTournamentIDs = [...exportOptions.includeTournamentIDs, tournamentId];
        }
    }

    function selectAllCollections() {
        collectionsManuallyModified = true;
        exportOptions.collectionIDs = collections.map(c => c.id);
    }

    function selectNoCollections() {
        collectionsManuallyModified = true;
        exportOptions.collectionIDs = [];
    }

    function selectAllTournaments() {
        tournamentsManuallyModified = true;
        exportOptions.includeTournamentIDs = tournaments.map(t => t.id);
    }

    function selectNoTournaments() {
        tournamentsManuallyModified = true;
        exportOptions.includeTournamentIDs = [];
    }

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

    .checkbox-item input[type="checkbox"]:disabled {
        cursor: not-allowed;
        opacity: 0.5;
    }

    .checkbox-item label {
        margin: 0;
        cursor: pointer;
        font-weight: normal;
    }

    .checkbox-item input[type="checkbox"]:disabled + label {
        cursor: not-allowed;
        opacity: 0.5;
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

    .collections-section {
        border: 1px solid #ddd;
        border-radius: 4px;
        overflow: hidden;
    }

    .collections-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 8px 12px;
        background-color: #f9f9f9;
        border-bottom: 1px solid #ddd;
        font-size: 13px;
        font-weight: 500;
    }

    .collections-buttons {
        display: flex;
        gap: 4px;
    }

    .small-btn {
        font-size: 11px;
        padding: 2px 8px;
    }

    .collections-list {
        max-height: 120px;
        overflow-y: auto;
        padding: 4px;
    }

    .collection-checkbox {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 8px;
        cursor: pointer;
        font-size: 13px;
    }

    .collection-checkbox:hover {
        background-color: #f5f5f5;
    }

    .collection-checkbox input[type="checkbox"] {
        width: 16px;
        height: 16px;
        cursor: pointer;
        accent-color: #333;
    }

    .coll-name {
        flex: 1;
    }

    .coll-count {
        color: #888;
        font-size: 12px;
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
                    <div class="checkbox-item">
                        <input 
                            type="checkbox" 
                            id="export-played-moves" 
                            bind:checked={exportOptions.includePlayedMoves}
                            disabled={!exportOptions.includeAnalysis}
                        />
                        <label for="export-played-moves">Include played moves</label>
                    </div>
                    <div class="checkbox-item">
                        <input 
                            type="checkbox" 
                            id="export-matches" 
                            bind:checked={exportOptions.includeMatches}
                            disabled={matches.length === 0}
                        />
                        <label for="export-matches">Include matches ({matches.length})</label>
                    </div>
                    <div class="checkbox-item">
                        <input 
                            type="checkbox" 
                            id="export-tournaments" 
                            bind:checked={exportOptions.includeTournaments}
                            disabled={tournaments.length === 0}
                        />
                        <label for="export-tournaments">Include tournaments ({tournaments.length})</label>
                    </div>
                    <div class="checkbox-item">
                        <input 
                            type="checkbox" 
                            id="export-collections" 
                            bind:checked={exportOptions.includeCollections}
                            disabled={collections.length === 0}
                        />
                        <label for="export-collections">Include collections ({collections.length})</label>
                    </div>
                </div>

                {#if exportOptions.includeMatches && matches.length > 0}
                    <div class="collections-section">
                        <div class="collections-header">
                            <span>Select matches to export</span>
                            <div class="collections-buttons">
                                <button type="button" class="small-btn" on:click={selectAllMatches}>All</button>
                                <button type="button" class="small-btn" on:click={selectNoMatches}>None</button>
                            </div>
                        </div>
                        <div class="collections-list">
                            {#each matches as match}
                                <label class="collection-checkbox">
                                    <input 
                                        type="checkbox" 
                                        checked={exportOptions.matchIDs.includes(match.id)}
                                        on:change={() => toggleMatchSelection(match.id)}
                                    />
                                    <span class="coll-name">{match.player1_name} vs {match.player2_name}</span>
                                    <span class="coll-count">({match.game_count}g)</span>
                                </label>
                            {/each}
                        </div>
                    </div>
                {/if}

                {#if exportOptions.includeTournaments && tournaments.length > 0}
                    <div class="collections-section">
                        <div class="collections-header">
                            <span>Select tournaments to export</span>
                            <div class="collections-buttons">
                                <button type="button" class="small-btn" on:click={selectAllTournaments}>All</button>
                                <button type="button" class="small-btn" on:click={selectNoTournaments}>None</button>
                            </div>
                        </div>
                        <div class="collections-list">
                            {#each tournaments as tournament}
                                <label class="collection-checkbox">
                                    <input 
                                        type="checkbox" 
                                        checked={exportOptions.includeTournamentIDs.includes(tournament.id)}
                                        on:change={() => toggleTournamentSelection(tournament.id)}
                                    />
                                    <span class="coll-name">{tournament.name}</span>
                                    <span class="coll-count">({tournament.matchCount})</span>
                                </label>
                            {/each}
                        </div>
                    </div>
                {/if}

                {#if exportOptions.includeCollections && collections.length > 0}
                    <div class="collections-section">
                        <div class="collections-header">
                            <span>Select collections to export</span>
                            <div class="collections-buttons">
                                <button type="button" class="small-btn" on:click={selectAllCollections}>All</button>
                                <button type="button" class="small-btn" on:click={selectNoCollections}>None</button>
                            </div>
                        </div>
                        <div class="collections-list">
                            {#each collections as collection}
                                <label class="collection-checkbox">
                                    <input 
                                        type="checkbox" 
                                        checked={exportOptions.collectionIDs.includes(collection.id)}
                                        on:change={() => toggleCollectionSelection(collection.id)}
                                    />
                                    <span class="coll-name">{collection.name}</span>
                                    <span class="coll-count">({collection.positionCount})</span>
                                </label>
                            {/each}
                        </div>
                    </div>
                {/if}

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
