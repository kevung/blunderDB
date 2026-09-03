<script>
    import { untrack } from 'svelte';
    import Modal from './Modal.svelte';
    import PickList from './PickList.svelte';
    import { collectionsStore } from '../stores/collectionStore';
    import { tournamentsStore } from '../stores/tournamentStore';
    import { exportCollectionCoverageStore } from '../stores/exportModalStore.js';
    import { t } from '../i18n';

    let {
        visible = false,
        mode = 'metadata',
        positionCount = 0,
        onCancel,
        onExport,
        metadata = {
            user: '',
            description: '',
            dateOfCreation: ''
        },
        exportOptions: exportOptionsProp = {
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

    // Svelte 5 only tracks mutations made through a $state proxy. The parent hands this
    // component a plain object taken out of a store, so a checkbox writing
    // `exportOptions.includeCollections = true` updated the object but re-rendered nothing:
    // every section gated on a checkbox stayed hidden. The options are therefore mirrored in
    // local state — which the template and the handlers below use unchanged.
    //
    // The mirror is seeded ONLY when the dialog opens, and written back ONLY when the user
    // confirms. An earlier version seeded it from a reactive read of the prop and wrote back
    // from an $effect on every change; those two together can close a cycle through the
    // parent's store binding, and a component that re-renders in a loop drops focus the
    // instant you click into any of its fields. `untrack` keeps the seeding tied to
    // `visible` alone, and the write-back happens once, on confirm.
    let exportOptions = $state(untrack(() => ({ ...exportOptionsProp })));
    $effect(() => {
        if (visible) {
            untrack(() => {
                exportOptions = { ...exportOptionsProp };
            });
        }
    });

    // Hand the chosen options back to the caller, which reads them from the store object.
    function confirmExport() {
        Object.assign(exportOptionsProp, $state.snapshot(exportOptions));
        onExport?.();
    }

    let passwordVisible = $state(false);

    let collections = $derived($collectionsStore || []);

    // A ticked box with an empty field used to produce a file with no watermark and no
    // warning — the export silently did nothing about the very thing the user asked for.
    // Both mechanisms are now blocked until they are actually filled in.
    let missingOrigin = $derived(exportOptions.watermarkEnabled && !(exportOptions.watermark || '').trim());
    let missingPassword = $derived(exportOptions.passwordEnabled && !(exportOptions.password || ''));
    let cannotExport = $derived(missingOrigin || missingPassword);
    let tournaments = $derived($tournamentsStore || []);

    // How many of a collection's positions the current selection actually covers, and
    // whether that is fewer than the collection holds.
    function covered(collection) {
        const coverage = $exportCollectionCoverageStore ?? {};
        return coverage[collection.id] ?? coverage[String(collection.id)] ?? 0;
    }
    function isPartial(collection) {
        return covered(collection) < (collection.positionCount ?? 0);
    }

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
        // These two change the nature of the file produced, so they belong in the sentence
        // that says what is being produced.
        if (exportOptions.watermarkEnabled) parts.push(tr('export.descWatermarked'));
        if (exportOptions.passwordEnabled) parts.push(tr('export.descProtected'));
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
</script>

<Modal open={visible} onclose={onCancel} size="large" layer="top" closeButton={false} label={$t('export.dialogLabel')}>
    {#if mode === 'metadata' || mode === 'exporting'}
        <h2 class="modal-title">{$t('export.titleExport')}</h2>

        <div class="summary">
            <p>{$t('export.willBeExported', { count: positionCount, desc: exportDescription })}</p>
        </div>

        <!-- These describe the file being produced, not the database being
             exported from. They start from the source's values, which most
             exports keep. -->
        <p class="group-title">{$t('export.metadataTitle')}</p>

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
                <!-- A tournament is linked to its matches only when the matches
                     are exported too; without them it would arrive empty. -->
                <input type="checkbox" id="export-tournaments" bind:checked={exportOptions.includeTournaments} disabled={tournaments.length === 0 || !exportOptions.includeMatches} />
                <label for="export-tournaments">{$t('export.includeTournaments', { count: tournaments.length })}</label>
                {#if tournaments.length > 0 && !exportOptions.includeMatches}
                    <span class="info-tip" title={$t('export.tournamentsNeedMatches')} aria-label={$t('export.tournamentsNeedMatches')} role="note">?</span>
                {/if}
            </div>
            <div class="checkbox-item">
                <input type="checkbox" id="export-collections" bind:checked={exportOptions.includeCollections} disabled={collections.length === 0} />
                <label for="export-collections">{$t('export.includeCollections', { count: collections.length })}</label>
            </div>
        </div>

        <!-- Not content but working preferences: the producer's own saved searches,
         which have no business in a recipient's database. Kept apart from the
         options above for that reason, and off by default. -->
        <div class="checkbox-group issuance-toggle">
            <div class="checkbox-item">
                <input type="checkbox" id="export-filter-library" bind:checked={exportOptions.includeFilterLibrary} />
                <label for="export-filter-library">{$t('export.includeFilterLibrary')}</label>
                <span class="info-tip" title={$t('export.filterLibraryHint')} aria-label={$t('export.filterLibraryHint')} role="note">?</span>
            </div>
        </div>

        <!-- Two independent, optional mechanisms, both the producer's choice and
         both off by default: mark where the file comes from, and protect it
         with a password. Neither makes the recipient's side record anything.
         See ADR-0007. -->
        <div class="checkbox-group issuance-toggle">
            <div class="checkbox-item">
                <input type="checkbox" id="export-watermark" bind:checked={exportOptions.watermarkEnabled} />
                <label for="export-watermark">{$t('issuance.enableWatermark')}</label>
                <span class="info-tip" title={$t('issuance.watermarkNote')} aria-label={$t('issuance.watermarkNote')} role="note">?</span>
            </div>
            <div class="checkbox-item">
                <input type="checkbox" id="export-protect" bind:checked={exportOptions.passwordEnabled} />
                <label for="export-protect">{$t('issuance.enablePassword')}</label>
                <span class="info-tip" title={$t('issuance.passwordHint')} aria-label={$t('issuance.passwordHint')} role="note">?</span>
            </div>
        </div>

        {#if exportOptions.watermarkEnabled}
            <div class="issuance-section">
                <div class="form-group">
                    <label for="export-origin">{$t('issuance.originLabel')}</label>
                    <input id="export-origin" type="text" bind:value={exportOptions.watermark} placeholder={$t('issuance.originPlaceholder')} />
                    {#if missingOrigin}
                        <p class="issuance-required">{$t('issuance.originRequired')}</p>
                    {/if}
                </div>
                <div class="form-group">
                    <label for="export-watermark-note">{$t('issuance.noteLabel')}</label>
                    <input id="export-watermark-note" type="text" bind:value={exportOptions.watermarkNote} placeholder={$t('issuance.notePlaceholder')} />
                </div>
            </div>
        {/if}

        {#if exportOptions.passwordEnabled}
            <div class="issuance-section">
                <div class="form-group">
                    <label for="export-password">
                        {$t('issuance.passwordLabel')}
                        <span class="info-tip" title={$t('issuance.passwordHint')} aria-label={$t('issuance.passwordHint')} role="note">?</span>
                    </label>
                    <div class="password-row">
                        <input id="export-password" type={passwordVisible ? 'text' : 'password'} bind:value={exportOptions.password} />
                        <!-- Reveal while held, never toggled: the password goes back
                         out of sight the moment the button is released, so it
                         cannot be left showing by accident. Pointer and keyboard
                         both work. -->
                        <button
                            type="button"
                            class="reveal"
                            aria-label={$t('issuance.revealPassword')}
                            title={$t('issuance.revealPassword')}
                            onpointerdown={() => (passwordVisible = true)}
                            onpointerup={() => (passwordVisible = false)}
                            onpointerleave={() => (passwordVisible = false)}
                            onpointercancel={() => (passwordVisible = false)}
                            onkeydown={(e) => {
                                if (e.key === ' ' || e.key === 'Enter') passwordVisible = true;
                            }}
                            onkeyup={() => (passwordVisible = false)}
                            onblur={() => (passwordVisible = false)}
                        >
                            {passwordVisible ? '🙈' : '👁'}
                        </button>
                    </div>
                    {#if missingPassword}
                        <p class="issuance-required">{$t('issuance.passwordRequired')}</p>
                    {/if}
                </div>
            </div>
        {/if}

        {#if exportOptions.includeMatches && matches.length > 0}
            <PickList
                header={$t('export.selectMatches')}
                items={matches}
                isChecked={(id) => exportOptions.matchIDs.includes(id)}
                toggle={toggleMatchSelection}
                selectAll={selectAllMatches}
                selectNone={selectNoMatches}
                describe={(m) => ({ name: `${m.player1_name} vs ${m.player2_name}`, count: `(${m.game_count}g)` })}
            />
        {/if}

        {#if exportOptions.includeTournaments && tournaments.length > 0}
            <PickList
                header={$t('export.selectTournaments')}
                items={tournaments}
                isChecked={(id) => exportOptions.includeTournamentIDs.includes(id)}
                toggle={toggleTournamentSelection}
                selectAll={selectAllTournaments}
                selectNone={selectNoTournaments}
                describe={(tournament) => ({ name: tournament.name, count: `(${tournament.matchCount})` })}
            />
        {/if}

        {#if exportOptions.includeCollections && collections.length > 0}
            <!-- Covered / total. The export writes membership only for positions it
                 exports, so a partial collection arrives truncated — said here rather
                 than discovered by the recipient. -->
            <PickList
                header={$t('export.selectCollections')}
                items={collections}
                isChecked={(id) => exportOptions.collectionIDs.includes(id)}
                toggle={toggleCollectionSelection}
                selectAll={selectAllCollections}
                selectNone={selectNoCollections}
                describe={(collection) => ({ name: collection.name, count: `(${covered(collection)}/${collection.positionCount})`, partial: isPartial(collection) })}
            />
        {/if}
    {/if}

    {#if mode === 'exporting'}
        <!-- Laid over the form rather than replacing it: the dialog keeps
             exactly the same box, which is what stops WebKitGTK leaving a blank
             white rectangle when the content changes. It also covers the
             controls, so nothing can be edited while the export runs. -->
        <div class="busy-overlay">
            <h2 class="modal-title">{$t('export.exportingTitle')} <span class="spinner"></span></h2>
            <p class="status-text">{$t('export.exportingPositions', { count: positionCount })}</p>
            <p class="status-text">{$t('export.mayTakeMoments')}</p>
            <button onclick={onCancel}>{$t('common.cancel')}</button>
        </div>
    {/if}
    {#snippet footer()}
        {#if mode === 'metadata' || mode === 'exporting'}
            <button onclick={onCancel}>{$t('common.cancel')}</button>
            <button class="btn-export primary" onclick={confirmExport} disabled={cannotExport}>{$t('export.exportAction')}</button>
        {/if}
    {/snippet}
</Modal>

<style>
    .busy-overlay {
        position: absolute;
        inset: 0;
        background-color: rgba(255, 255, 255, 0.94);
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 12px;
        border-radius: 6px;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    label {
        font-size: var(--font-size-base);
        font-weight: 500;
        color: var(--color-text);
    }

    input,
    textarea {
        padding: 8px 12px;
        border: 1px solid var(--color-border);
        border-radius: 4px;
        font-size: var(--font-size-base);
        font-family: inherit;
    }

    input:focus,
    textarea:focus {
        outline: none;
        border-color: var(--color-text-muted);
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
        accent-color: var(--color-text);
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

    .reveal,
    .busy-overlay button {
        padding: 10px 20px;
        border: 1px solid var(--color-border);
        border-radius: 4px;
        font-size: var(--font-size-base);
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s ease;
        background-color: white;
        color: var(--color-text);
    }

    .busy-overlay button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .reveal:hover,
    .busy-overlay button:hover:not(:disabled) {
        background-color: #f5f5f5;
        border-color: var(--color-text-muted);
    }

    .issuance-toggle {
        border-top: 1px solid #e0e0e0;
        padding-top: 8px;
    }

    .issuance-section {
        border-left: 3px solid var(--color-primary);
        padding-left: 10px;
        margin-bottom: 8px;
    }

    .info-tip {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 15px;
        height: 15px;
        margin-left: 4px;
        border-radius: 50%;
        border: 1px solid #9aa0a6;
        color: #5f6368;
        font-size: var(--font-size-small);
        font-weight: 700;
        line-height: 1;
        cursor: help;
        user-select: none;
        flex: none;
    }

    .password-row {
        display: flex;
        gap: 6px;
        align-items: stretch;
    }

    .password-row input {
        flex: 1;
        min-width: 0;
    }

    .reveal {
        flex: none;
        padding: 0 8px;
        cursor: pointer;
        line-height: 1;
    }

    .group-title {
        margin: 6px 0 0;
        font-size: var(--font-size-small);
        font-weight: 600;
        color: var(--color-text-muted);
        text-transform: uppercase;
        letter-spacing: 0.3px;
    }

    .issuance-required {
        font-size: var(--font-size-small);
        color: #b3261e;
        margin: 2px 0 6px;
    }
</style>
