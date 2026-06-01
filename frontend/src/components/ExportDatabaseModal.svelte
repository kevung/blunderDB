<script>
    import { onMount, onDestroy } from 'svelte';
    import { trapFocus } from '../utils/focusTrap.js';
    import { collectionsStore } from '../stores/collectionStore';
    import { tournamentsStore } from '../stores/tournamentStore';
    import { t } from '../i18n';

    let {
        visible = false,
        mode = 'preparing',
        positionCount = 0,
        onCancel,
        onExport,
        onClose,
        metadata = {
            user: '',
            description: '',
            dateOfCreation: ''
        },
        exportOptions = {
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
        },
        matches = []
    } = $props();

    let collections = $derived($collectionsStore || []);
    let tournaments = $derived($tournamentsStore || []);

    // Get current date in YYYY-MM-DD format
    function getCurrentDate() {
        const now = new Date();
        const year = now.getFullYear();
        const month = String(now.getMonth() + 1).padStart(2, '0');
        const day = String(now.getDate()).padStart(2, '0');
        return `${year}-${month}-${day}`;
    }

    // Initialize date when modal becomes visible in metadata mode
    $effect(() => {
        if (visible && mode === 'metadata' && !metadata.dateOfCreation) {
            metadata.dateOfCreation = getCurrentDate();
        }
    });
    // Auto-select all matches when includeMatches is toggled on (only if not manually modified)
    let matchesManuallyModified = $state(false);
    $effect(() => {
        if (exportOptions.includeMatches && matches.length > 0 && exportOptions.matchIDs.length === 0 && !matchesManuallyModified) {
            exportOptions.matchIDs = matches.map((m) => m.id);
        }
    });
    // Clear matchIDs when includeMatches is toggled off
    $effect(() => {
        if (!exportOptions.includeMatches) {
            exportOptions.matchIDs = [];
            matchesManuallyModified = false;
        }
    });
    // Auto-select all collections when includeCollections is toggled on (only if not manually modified)
    let collectionsManuallyModified = $state(false);
    $effect(() => {
        if (exportOptions.includeCollections && collections.length > 0 && exportOptions.collectionIDs.length === 0 && !collectionsManuallyModified) {
            exportOptions.collectionIDs = collections.map((c) => c.id);
        }
    });
    // Clear collectionIDs when includeCollections is toggled off
    $effect(() => {
        if (!exportOptions.includeCollections) {
            exportOptions.collectionIDs = [];
            collectionsManuallyModified = false;
        }
    });
    // Auto-select all tournaments when includeTournaments is toggled on (only if not manually modified)
    let tournamentsManuallyModified = $state(false);
    $effect(() => {
        if (exportOptions.includeTournaments && tournaments.length > 0 && exportOptions.includeTournamentIDs.length === 0 && !tournamentsManuallyModified) {
            exportOptions.includeTournamentIDs = tournaments.map((t) => t.id);
        }
    });
    // Clear tournamentIDs when includeTournaments is toggled off
    $effect(() => {
        if (!exportOptions.includeTournaments) {
            exportOptions.includeTournamentIDs = [];
            tournamentsManuallyModified = false;
        }
    });
    // Computed description of what will be exported
    let exportDescription = $derived.by(() => {
        const tr = $t;
        let parts = [];
        if (exportOptions.includeAnalysis) parts.push(tr('export.descAnalysis'));
        if (exportOptions.includeComments) parts.push(tr('export.descComments'));
        if (exportOptions.includeFilterLibrary) parts.push(tr('export.descFilterLibrary'));
        if (exportOptions.includePlayedMoves) parts.push(tr('export.descPlayedMoves'));
        if (exportOptions.includeMatches && exportOptions.matchIDs.length > 0)
            parts.push(exportOptions.matchIDs.length > 1 ? tr('export.descMatchesPlural', { count: exportOptions.matchIDs.length }) : tr('export.descMatch', { count: exportOptions.matchIDs.length }));
        if (exportOptions.includeTournaments && exportOptions.includeTournamentIDs.length > 0)
            parts.push(
                exportOptions.includeTournamentIDs.length > 1
                    ? tr('export.descTournamentsPlural', { count: exportOptions.includeTournamentIDs.length })
                    : tr('export.descTournament', { count: exportOptions.includeTournamentIDs.length })
            );
        if (exportOptions.includeCollections && exportOptions.collectionIDs.length > 0)
            parts.push(
                exportOptions.collectionIDs.length > 1
                    ? tr('export.descCollectionsPlural', { count: exportOptions.collectionIDs.length })
                    : tr('export.descCollection', { count: exportOptions.collectionIDs.length })
            );

        if (parts.length === 0) {
            return tr('export.descPositionsOnly');
        } else if (parts.length === 1) {
            return `${parts[0]}`;
        } else if (parts.length === 2) {
            return tr('export.descTwo', { a: parts[0], b: parts[1] });
        } else {
            return tr('export.descMany', { list: parts.slice(0, -1).join(', '), last: parts[parts.length - 1] });
        }
    });

    function toggleMatchSelection(matchId) {
        matchesManuallyModified = true;
        if (exportOptions.matchIDs.includes(matchId)) {
            exportOptions.matchIDs = exportOptions.matchIDs.filter((id) => id !== matchId);
        } else {
            exportOptions.matchIDs = [...exportOptions.matchIDs, matchId];
        }
    }

    function selectAllMatches() {
        matchesManuallyModified = true;
        exportOptions.matchIDs = matches.map((m) => m.id);
    }

    function selectNoMatches() {
        matchesManuallyModified = true;
        exportOptions.matchIDs = [];
    }

    function toggleCollectionSelection(collectionId) {
        collectionsManuallyModified = true;
        if (exportOptions.collectionIDs.includes(collectionId)) {
            exportOptions.collectionIDs = exportOptions.collectionIDs.filter((id) => id !== collectionId);
        } else {
            exportOptions.collectionIDs = [...exportOptions.collectionIDs, collectionId];
        }
    }

    function toggleTournamentSelection(tournamentId) {
        tournamentsManuallyModified = true;
        if (exportOptions.includeTournamentIDs.includes(tournamentId)) {
            exportOptions.includeTournamentIDs = exportOptions.includeTournamentIDs.filter((id) => id !== tournamentId);
        } else {
            exportOptions.includeTournamentIDs = [...exportOptions.includeTournamentIDs, tournamentId];
        }
    }

    function selectAllCollections() {
        collectionsManuallyModified = true;
        exportOptions.collectionIDs = collections.map((c) => c.id);
    }

    function selectNoCollections() {
        collectionsManuallyModified = true;
        exportOptions.collectionIDs = [];
    }

    function selectAllTournaments() {
        tournamentsManuallyModified = true;
        exportOptions.includeTournamentIDs = tournaments.map((t) => t.id);
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

{#if visible}
    <div class="modal-overlay" role="dialog" aria-modal="true" aria-label={$t('export.dialogLabel')} use:trapFocus>
        <div class="modal-content">
            {#if mode === 'preparing'}
                <h2>{$t('export.preparing')} <span class="spinner"></span></h2>
                <p class="status-text">{$t('export.countingPositions')}</p>

                <div class="button-group">
                    <button onclick={onCancel}>{$t('common.cancel')}</button>
                </div>
            {:else if mode === 'metadata'}
                <h2>{$t('export.titleExport')}</h2>

                <div class="summary">
                    <p>{$t('export.willBeExported', { count: positionCount, desc: exportDescription })}</p>
                </div>

                <div class="form-group">
                    <label for="export-user">{$t('export.user')}</label>
                    <input id="export-user" type="text" bind:value={metadata.user} placeholder={$t('export.userPlaceholder')} />
                </div>

                <div class="form-group">
                    <label for="export-description">{$t('export.description')}</label>
                    <textarea id="export-description" bind:value={metadata.description} placeholder={$t('export.descriptionPlaceholder')}></textarea>
                </div>

                <div class="form-group">
                    <label for="export-date">{$t('export.creationDate')}</label>
                    <input id="export-date" type="date" bind:value={metadata.dateOfCreation} />
                </div>

                <div class="checkbox-group">
                    <div class="checkbox-item">
                        <input type="checkbox" id="export-analysis" bind:checked={exportOptions.includeAnalysis} />
                        <label for="export-analysis">{$t('export.includeAnalysis')}</label>
                    </div>
                    <div class="checkbox-item">
                        <input type="checkbox" id="export-comments" bind:checked={exportOptions.includeComments} />
                        <label for="export-comments">{$t('export.includeComments')}</label>
                    </div>
                    <div class="checkbox-item">
                        <input type="checkbox" id="export-filter-library" bind:checked={exportOptions.includeFilterLibrary} />
                        <label for="export-filter-library">{$t('export.includeFilterLibrary')}</label>
                    </div>
                    <div class="checkbox-item">
                        <input type="checkbox" id="export-played-moves" bind:checked={exportOptions.includePlayedMoves} disabled={!exportOptions.includeAnalysis} />
                        <label for="export-played-moves">{$t('export.includePlayedMoves')}</label>
                    </div>
                    <div class="checkbox-item">
                        <input type="checkbox" id="export-matches" bind:checked={exportOptions.includeMatches} disabled={matches.length === 0} />
                        <label for="export-matches">{$t('export.includeMatches', { count: matches.length })}</label>
                    </div>
                    <div class="checkbox-item">
                        <input type="checkbox" id="export-tournaments" bind:checked={exportOptions.includeTournaments} disabled={tournaments.length === 0} />
                        <label for="export-tournaments">{$t('export.includeTournaments', { count: tournaments.length })}</label>
                    </div>
                    <div class="checkbox-item">
                        <input type="checkbox" id="export-collections" bind:checked={exportOptions.includeCollections} disabled={collections.length === 0} />
                        <label for="export-collections">{$t('export.includeCollections', { count: collections.length })}</label>
                    </div>
                </div>

                {#if exportOptions.includeMatches && matches.length > 0}
                    <div class="collections-section">
                        <div class="collections-header">
                            <span>{$t('export.selectMatches')}</span>
                            <div class="collections-buttons">
                                <button type="button" class="small-btn" onclick={selectAllMatches}>{$t('export.all')}</button>
                                <button type="button" class="small-btn" onclick={selectNoMatches}>{$t('export.none')}</button>
                            </div>
                        </div>
                        <div class="collections-list">
                            {#each matches as match (match.id)}
                                <label class="collection-checkbox">
                                    <input type="checkbox" checked={exportOptions.matchIDs.includes(match.id)} onchange={() => toggleMatchSelection(match.id)} />
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
                            <span>{$t('export.selectTournaments')}</span>
                            <div class="collections-buttons">
                                <button type="button" class="small-btn" onclick={selectAllTournaments}>{$t('export.all')}</button>
                                <button type="button" class="small-btn" onclick={selectNoTournaments}>{$t('export.none')}</button>
                            </div>
                        </div>
                        <div class="collections-list">
                            {#each tournaments as tournament (tournament.id)}
                                <label class="collection-checkbox">
                                    <input type="checkbox" checked={exportOptions.includeTournamentIDs.includes(tournament.id)} onchange={() => toggleTournamentSelection(tournament.id)} />
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
                            <span>{$t('export.selectCollections')}</span>
                            <div class="collections-buttons">
                                <button type="button" class="small-btn" onclick={selectAllCollections}>{$t('export.all')}</button>
                                <button type="button" class="small-btn" onclick={selectNoCollections}>{$t('export.none')}</button>
                            </div>
                        </div>
                        <div class="collections-list">
                            {#each collections as collection (collection.id)}
                                <label class="collection-checkbox">
                                    <input type="checkbox" checked={exportOptions.collectionIDs.includes(collection.id)} onchange={() => toggleCollectionSelection(collection.id)} />
                                    <span class="coll-name">{collection.name}</span>
                                    <span class="coll-count">({collection.positionCount})</span>
                                </label>
                            {/each}
                        </div>
                    </div>
                {/if}

                <div class="button-group">
                    <button onclick={onCancel}>{$t('common.cancel')}</button>
                    <button class="btn-export" onclick={onExport}>{$t('export.exportAction')}</button>
                </div>
            {:else if mode === 'exporting'}
                <h2>{$t('export.exportingTitle')} <span class="spinner"></span></h2>
                <p class="status-text">{$t('export.exportingPositions', { count: positionCount })}</p>
                <p class="status-text">{$t('export.mayTakeMoments')}</p>

                <div class="button-group">
                    <button onclick={onCancel}>{$t('common.cancel')}</button>
                </div>
            {:else if mode === 'completed'}
                <h2>{$t('export.completedTitle')}</h2>

                <div class="summary">
                    <p>{$t('export.exportSuccessDetail', { count: positionCount })}</p>
                </div>

                <div class="button-group">
                    <button onclick={onClose}>{$t('common.close')}</button>
                </div>
            {/if}
        </div>
    </div>
{/if}

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

    input,
    textarea {
        padding: 8px 12px;
        border: 1px solid #ccc;
        border-radius: 4px;
        font-size: 14px;
        font-family: inherit;
    }

    input:focus,
    textarea:focus {
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

    .checkbox-item input[type='checkbox'] {
        width: 18px;
        height: 18px;
        cursor: pointer;
        accent-color: #333;
    }

    .checkbox-item input[type='checkbox']:disabled {
        cursor: not-allowed;
        opacity: 0.5;
    }

    .checkbox-item label {
        margin: 0;
        cursor: pointer;
        font-weight: normal;
    }

    .checkbox-item input[type='checkbox']:disabled + label {
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
        0% {
            transform: rotate(0deg);
        }
        100% {
            transform: rotate(360deg);
        }
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

    .collection-checkbox input[type='checkbox'] {
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
