<script>
    import { untrack } from 'svelte';
    import Modal from './Modal.svelte';
    import PickList from './PickList.svelte';
    import PanelHeader from './panels/PanelHeader.svelte';
    import FormGrid from './panels/FormGrid.svelte';
    import FormRow from './panels/FormRow.svelte';
    import CountLink from './panels/CountLink.svelte';
    import { collectionsStore } from '../stores/collectionStore';
    import { tournamentsStore } from '../stores/tournamentStore';
    import { exportCollectionCoverageStore } from '../stores/exportModalStore.js';
    import { ListLessons } from '../../wailsjs/go/database/Database.js';
    import { logger } from '../utils/logger.js';
    import { t } from '../i18n';

    let {
        visible = false,
        mode = 'metadata',
        positionCount = 0,
        onCancel,
        onExport,
        // A list can be exported as the whole library; onScope shows what the scope chosen exports.
        chooseScope = false,
        onScope = undefined,
        // A count opens what it counts (ADR-0085 rule 7).
        onOpenPositions = undefined,
        onOpenMatches = undefined,
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
            includeTranscriptions: true,
            includeAliases: true,
            matchIDs: [],
            includeTournaments: false,
            includeTournamentIDs: [],
            includeCollections: false,
            collectionIDs: [],
            includeLessons: false,
            lessonIDs: []
        },
        matches = [],
        decks = [],
        // The backend's preview: planFor(options) counts what the file would hold, and
        // planIdsFor(options, family) gives the ids behind one count.
        planFor = undefined,
        planIdsFor = undefined
    } = $props();

    /**
     * @typedef {object} ExportOptions
     * @property {boolean} [wholeLibrary]
     * @property {boolean} [includeAnalysis]
     * @property {boolean} [includeComments]
     * @property {boolean} [includeFilterLibrary]
     * @property {boolean} [includePlayedMoves]
     * @property {boolean} [includeMatches]
     * @property {boolean} [includeTranscriptions]
     * @property {boolean} [includeAliases]
     * @property {number[]} matchIDs
     * @property {boolean} [includeTournaments]
     * @property {number[]} includeTournamentIDs
     * @property {boolean} [includeCollections]
     * @property {number[]} collectionIDs
     * @property {boolean} [includeCollectionPositions]
     * @property {boolean} [includeLessons]
     * @property {number[]} [lessonIDs]
     * @property {boolean} [includeAnkiDecks]
     * @property {number[]} [deckIDs]
     * @property {boolean} [includeAnkiHistory]
     * @property {boolean} [watermarkEnabled]
     * @property {string} [watermark]
     * @property {string} [watermarkNote]
     * @property {boolean} [passwordEnabled]
     * @property {string} [password]
     */

    // The prop is a plain object Svelte 5 cannot track, so options are mirrored in
    // $state: seeded only on open (`untrack`, tied to `visible`) and written back
    // only on confirm — a two-way sync loops through the parent's store and drops focus.
    let exportOptions = $state(/** @type {ExportOptions} */ (untrack(() => ({ ...exportOptionsProp }))));
    $effect(() => {
        if (visible) {
            untrack(() => {
                exportOptions = /** @type {ExportOptions} */ ({ ...exportOptionsProp });
            });
        }
    });

    // Hand the chosen options back to the caller, which reads them from the store object.
    function confirmExport() {
        Object.assign(exportOptionsProp, $state.snapshot(exportOptions));
        onExport?.(plan);
    }

    let passwordVisible = $state(false);

    /** @typedef {{ positions: number, comments: number, matches: number, tournaments: number, directions: number, rencontres: number, collections: number, lessons: number, ankiDecks: number, transcriptions: number, aliases: number, filters: number, frozenCollections: {id: number, name: string}[], degradedAnkiDecks: {id: number, name: string}[] }} Plan */
    // What the file will hold, read from the backend after every change of the choices. A
    // slower answer to an earlier choice is dropped.
    let plan = $state(/** @type {Plan | null} */ (null));
    let planTicket = 0;
    $effect(() => {
        if (!visible) {
            plan = null;
            planTicket++;
        }
        if (!visible || mode !== 'metadata' || !planFor) return;
        // Read here so that every choice (and the lists they apply to) re-runs the preview.
        const snapshot = $state.snapshot(exportOptions);
        void [matches, decks, lessons, collections, tournaments];
        const ticket = ++planTicket;
        // A count of an earlier choice is never left shown, nor clickable.
        plan = null;
        const timer = setTimeout(() => {
            Promise.resolve()
                .then(() => planFor(snapshot))
                .then((p) => {
                    if (ticket === planTicket) plan = p;
                })
                .catch((error) => {
                    if (ticket === planTicket) plan = null;
                    logger.error('could not preview the export:', error);
                });
        }, 150);
        return () => clearTimeout(timer);
    });

    // A count opens what it counts: the ids come from the same plan.
    async function openPlanned(/** @type {string} */ family, /** @type {(ids: number[]) => void} */ open) {
        if (!planIdsFor) return;
        open(await planIdsFor($state.snapshot(exportOptions), family));
    }

    let frozenIDs = $derived(new Set((plan?.frozenCollections ?? []).map((c) => c.id)));
    let degradedDecks = $derived(plan?.degradedAnkiDecks ?? []);
    const shown = (/** @type {number | undefined} */ planned, /** @type {number} */ available) => planned ?? available;

    let collections = $derived($collectionsStore || []);

    // Lessons have no store of their own: the list is read each time the dialog opens.
    /** @type {{id: number, name: string, stepCount: number}[]} */
    let lessons = $state([]);
    $effect(() => {
        if (!visible) return;
        // Through a promise: without a backend the binding throws synchronously.
        Promise.resolve()
            .then(() => ListLessons())
            .then((list) => {
                lessons = list || [];
            })
            .catch((error) => {
                logger.error('could not list the lessons:', error);
                lessons = [];
            });
    });

    // Ticking the box selects every Lesson, as for collections; an emptied selection
    // exports none.
    let lessonsManuallyModified = $state(false);
    $effect(() => {
        if (exportOptions.includeLessons && lessons.length > 0 && (exportOptions.lessonIDs ?? []).length === 0 && !lessonsManuallyModified) {
            exportOptions.lessonIDs = lessons.map((l) => l.id);
        }
    });
    $effect(() => {
        if (!exportOptions.includeLessons) {
            exportOptions.lessonIDs = [];
            lessonsManuallyModified = false;
        }
    });

    function toggleLessonSelection(/** @type {number} */ lessonId) {
        lessonsManuallyModified = true;
        const ids = exportOptions.lessonIDs ?? [];
        exportOptions.lessonIDs = ids.includes(lessonId) ? ids.filter((id) => id !== lessonId) : [...ids, lessonId];
    }

    // A ticked mechanism with an empty field blocks the export rather than silently doing nothing.
    let missingOrigin = $derived(exportOptions.watermarkEnabled && !(exportOptions.watermark || '').trim());
    let missingPassword = $derived(exportOptions.passwordEnabled && !(exportOptions.password || ''));
    let cannotExport = $derived(missingOrigin || missingPassword);
    let tournaments = $derived($tournamentsStore || []);

    // How many of a collection's positions the current selection actually covers, and
    // whether that is fewer than the collection holds.
    function covered(/** @type {{ id: number, positionCount?: number }} */ collection) {
        const coverage = /** @type {Record<string, number>} */ ($exportCollectionCoverageStore ?? {});
        return coverage[collection.id] ?? coverage[String(collection.id)] ?? 0;
    }
    function isPartial(/** @type {{ id: number, positionCount?: number }} */ collection) {
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
    // Every match the scope offers is picked until one is unticked by hand; a change of
    // scope offers other matches and picks them all again.
    let matchesManuallyModified = $state(false);
    $effect(() => {
        const all = matches.map((/** @type {{ id: number }} */ m) => m.id);
        if (exportOptions.includeMatches && !untrack(() => matchesManuallyModified)) {
            exportOptions.matchIDs = all;
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
    // Decks are picked whole when ticked, as the other families.
    let decksManuallyModified = $state(false);
    $effect(() => {
        if (exportOptions.includeAnkiDecks && decks.length > 0 && (exportOptions.deckIDs ?? []).length === 0 && !decksManuallyModified) {
            exportOptions.deckIDs = decks.map((/** @type {{ id: number }} */ d) => d.id);
        }
    });
    $effect(() => {
        if (!exportOptions.includeAnkiDecks) {
            exportOptions.deckIDs = [];
            decksManuallyModified = false;
        }
    });
    function toggleDeckSelection(/** @type {number} */ deckId) {
        decksManuallyModified = true;
        const ids = exportOptions.deckIDs ?? [];
        exportOptions.deckIDs = ids.includes(deckId) ? ids.filter((id) => id !== deckId) : [...ids, deckId];
    }

    // A tournament brings its matches: their drafts and aliases are offered with them.
    let carriesMatches = $derived(!!exportOptions.includeMatches || !!exportOptions.includeTournaments);

    // The whole library ticks every family, with the review history; a list starts from its
    // positions and their matches.
    function setScope(/** @type {boolean} */ library) {
        if (!!exportOptions.wholeLibrary === library) return;
        Object.assign(exportOptions, {
            wholeLibrary: library,
            includeFilterLibrary: library,
            includeTournaments: library,
            includeCollections: library,
            includeLessons: library,
            includeAnkiDecks: library,
            includeAnkiHistory: library
        });
        matchesManuallyModified = false;
        onScope?.(library);
    }

    function toggleMatchSelection(/** @type {number} */ matchId) {
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

    function toggleCollectionSelection(/** @type {number} */ collectionId) {
        collectionsManuallyModified = true;
        if (exportOptions.collectionIDs.includes(collectionId)) {
            exportOptions.collectionIDs = exportOptions.collectionIDs.filter((id) => id !== collectionId);
        } else {
            exportOptions.collectionIDs = [...exportOptions.collectionIDs, collectionId];
        }
    }

    function toggleTournamentSelection(/** @type {number} */ tournamentId) {
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
        <PanelHeader
            title={$t('export.titleExport')}
            count={$t('export.positionCount', { count: plan?.positions ?? positionCount })}
            onCount={onOpenPositions ? () => onOpenPositions(!!exportOptions.wholeLibrary) : null}
        >
            {#snippet actions()}
                <button type="button" onclick={onCancel}>{$t('common.cancel')}</button>
                <button type="button" class="btn-export primary" onclick={confirmExport} disabled={cannotExport}>{$t('export.exportAction')}</button>
            {/snippet}
        </PanelHeader>

        <FormGrid>
            {#if chooseScope}
                <FormRow label={$t('export.scope')}>
                    <label class="check"
                        ><input type="radio" id="export-scope-list" name="export-scope" checked={!exportOptions.wholeLibrary} onchange={() => setScope(false)} />{$t('export.scopeList')}</label
                    >
                    <label class="check"
                        ><input type="radio" id="export-scope-library" name="export-scope" checked={!!exportOptions.wholeLibrary} onchange={() => setScope(true)} />{$t('export.scopeLibrary')}</label
                    >
                </FormRow>
            {/if}
            <FormRow label={$t('export.rowPositions')} for="export-analysis">
                <label class="check"><input type="checkbox" id="export-analysis" bind:checked={exportOptions.includeAnalysis} />{$t('export.analysis')}</label>
                <label class="check"
                    ><input type="checkbox" id="export-played-moves" bind:checked={exportOptions.includePlayedMoves} disabled={!exportOptions.includeAnalysis} />{$t('export.playedMoves')}</label
                >
                <label class="check"
                    ><input type="checkbox" id="export-comments" bind:checked={exportOptions.includeComments} />{$t('export.comments')}{#if plan}
                        <span class="count" data-testid="export-comments-count">{plan.comments}</span>{/if}</label
                >
            </FormRow>
            <FormRow label={$t('export.rowMatches')} for="export-matches">
                <input type="checkbox" id="export-matches" bind:checked={exportOptions.includeMatches} disabled={matches.length === 0} />
                {#if onOpenMatches && (plan ? plan.matches > 0 : matches.length > 0)}
                    <CountLink
                        testid="export-matches-count"
                        label={shown(plan?.matches, matches.length)}
                        onclick={() => (plan ? openPlanned('matches', (ids) => onOpenMatches(ids)) : onOpenMatches(matches.map((/** @type {{ id: number }} */ m) => m.id)))}
                    />
                {:else}
                    <span class="count" data-testid="export-matches-count">{shown(plan?.matches, matches.length)}</span>
                {/if}
                <label class="check"
                    ><input type="checkbox" id="export-transcriptions" bind:checked={exportOptions.includeTranscriptions} disabled={!carriesMatches} />{$t('export.transcriptions')}{#if plan}
                        <span class="count" data-testid="export-transcriptions-count">{plan.transcriptions}</span>{/if}</label
                >
                <label class="check"
                    ><input type="checkbox" id="export-aliases" bind:checked={exportOptions.includeAliases} disabled={!carriesMatches} />{$t('export.aliases')}{#if plan}
                        <span class="count" data-testid="export-aliases-count">{plan.aliases}</span>{/if}</label
                >
            </FormRow>
            <FormRow label={$t('export.rowTournaments')} for="export-tournaments">
                <!-- A tournament always leaves with its matches, so this box does not depend on the matches one. -->
                <input type="checkbox" id="export-tournaments" bind:checked={exportOptions.includeTournaments} disabled={tournaments.length === 0} />
                <span class="count" data-testid="export-tournaments-count">{shown(plan?.tournaments, tournaments.length)}</span>
                {#if plan && plan.tournaments > 0 && (plan.directions > 0 || plan.rencontres > 0)}
                    <span class="count">{$t('export.tournamentsDetail', { directions: plan.directions, rencontres: plan.rencontres })}</span>
                {/if}
            </FormRow>
            <FormRow label={$t('export.rowCollections')} for="export-collections">
                <input type="checkbox" id="export-collections" bind:checked={exportOptions.includeCollections} disabled={collections.length === 0} />
                <span class="count" data-testid="export-collections-count">{shown(plan?.collections, collections.length)}</span>
                {#if !exportOptions.wholeLibrary}
                    <label class="check"
                        ><input type="checkbox" id="export-collection-positions" bind:checked={exportOptions.includeCollectionPositions} disabled={!exportOptions.includeCollections} />{$t(
                            'export.collectionPositions'
                        )}</label
                    >
                {/if}
            </FormRow>
            <FormRow label={$t('export.rowLessons')} for="export-lessons">
                <input type="checkbox" id="export-lessons" bind:checked={exportOptions.includeLessons} disabled={lessons.length === 0} />
                <span class="count" data-testid="export-lessons-count">{shown(plan?.lessons, lessons.length)}</span>
            </FormRow>
            <FormRow label={$t('export.rowDecks')} for="export-decks">
                <input type="checkbox" id="export-decks" bind:checked={exportOptions.includeAnkiDecks} disabled={decks.length === 0} />
                <span class="count" data-testid="export-decks-count">{shown(plan?.ankiDecks, decks.length)}</span>
                <label class="check"
                    ><input type="checkbox" id="export-anki-history" bind:checked={exportOptions.includeAnkiHistory} disabled={!exportOptions.includeAnkiDecks} />{$t('export.ankiHistory')}</label
                >
            </FormRow>
            <FormRow label={$t('export.rowFilters')} for="export-filter-library">
                <!-- The producer's saved searches: working preferences, off for a list. -->
                <input type="checkbox" id="export-filter-library" bind:checked={exportOptions.includeFilterLibrary} />
                {#if plan}<span class="count" data-testid="export-filters-count">{plan.filters}</span>{/if}
            </FormRow>

            <!-- Describe the produced file; seeded from the source's values. -->
            <FormRow label={$t('export.user')} for="export-user">
                <input id="export-user" type="text" bind:value={metadata.user} />
            </FormRow>
            <FormRow label={$t('export.description')} for="export-description">
                <textarea id="export-description" bind:value={metadata.description}></textarea>
            </FormRow>
            <FormRow label={$t('export.creationDate')} for="export-date">
                <input id="export-date" type="date" bind:value={metadata.dateOfCreation} />
            </FormRow>

            <!-- Watermark and password, both optional and off by default; nothing is
                 recorded on the recipient's side (ADR-0007). -->
            <FormRow label={$t('export.rowWatermark')} for="export-watermark">
                <input type="checkbox" id="export-watermark" bind:checked={exportOptions.watermarkEnabled} title={$t('issuance.watermarkNote')} />
                {#if exportOptions.watermarkEnabled}
                    <input
                        id="export-origin"
                        type="text"
                        class:invalid={missingOrigin}
                        bind:value={exportOptions.watermark}
                        placeholder={$t('issuance.originPlaceholder')}
                        aria-label={$t('issuance.originLabel')}
                    />
                    <input id="export-watermark-note" type="text" bind:value={exportOptions.watermarkNote} placeholder={$t('issuance.notePlaceholder')} aria-label={$t('issuance.noteLabel')} />
                {/if}
            </FormRow>
            <FormRow label={$t('export.rowPassword')} for="export-protect">
                <input type="checkbox" id="export-protect" bind:checked={exportOptions.passwordEnabled} title={$t('issuance.passwordHint')} />
                {#if exportOptions.passwordEnabled}
                    <span class="password-row">
                        <input
                            id="export-password"
                            type={passwordVisible ? 'text' : 'password'}
                            class:invalid={missingPassword}
                            bind:value={exportOptions.password}
                            aria-label={$t('issuance.passwordLabel')}
                        />
                        <!-- Reveal while held, never toggled, so it cannot be left showing. -->
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
                    </span>
                {/if}
            </FormRow>
        </FormGrid>

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
            <!-- Covered / total: a partial collection arrives truncated, unless its positions come along. -->
            <PickList
                header={$t('export.selectCollections')}
                items={collections}
                isChecked={(id) => exportOptions.collectionIDs.includes(id)}
                toggle={toggleCollectionSelection}
                selectAll={selectAllCollections}
                selectNone={selectNoCollections}
                describe={(collection) => ({
                    name: collection.name,
                    count: `(${covered(collection)}/${collection.positionCount})${frozenIDs.has(collection.id) ? ` ${$t('export.frozenMark')}` : ''}`,
                    partial: (isPartial(collection) && !exportOptions.includeCollectionPositions) || frozenIDs.has(collection.id)
                })}
            />
            {#if frozenIDs.size > 0}
                <p class="note" data-testid="export-frozen-note">{$t('export.frozenNote', { count: frozenIDs.size })}</p>
            {/if}
        {/if}

        {#if exportOptions.includeLessons && lessons.length > 0}
            <!-- A Lesson carries the positions and collections its steps show. -->
            <PickList
                header={$t('export.selectLessons')}
                items={lessons}
                isChecked={(id) => (exportOptions.lessonIDs ?? []).includes(id)}
                toggle={toggleLessonSelection}
                selectAll={() => {
                    lessonsManuallyModified = true;
                    exportOptions.lessonIDs = lessons.map((l) => l.id);
                }}
                selectNone={() => {
                    lessonsManuallyModified = true;
                    exportOptions.lessonIDs = [];
                }}
                describe={(lesson) => ({ name: lesson.name, count: `(${lesson.stepCount})` })}
            />
        {/if}

        {#if exportOptions.includeAnkiDecks && decks.length > 0}
            <!-- A deck brings its positions. -->
            <PickList
                header={$t('export.selectDecks')}
                items={decks}
                isChecked={(id) => (exportOptions.deckIDs ?? []).includes(id)}
                toggle={toggleDeckSelection}
                selectAll={() => {
                    decksManuallyModified = true;
                    exportOptions.deckIDs = decks.map((/** @type {{ id: number }} */ d) => d.id);
                }}
                selectNone={() => {
                    decksManuallyModified = true;
                    exportOptions.deckIDs = [];
                }}
                describe={(deck) => ({ name: deck.name, count: `(${deck.cardCount})` })}
            />
            {#if degradedDecks.length > 0}
                <p class="note" data-testid="export-degraded-note">{$t('export.degradedNote', { count: degradedDecks.length })}</p>
            {/if}
        {/if}
    {/if}

    {#if mode === 'exporting'}
        <!-- Overlaid, not replacing the form: WebKitGTK leaves a blank rectangle
             when the box changes, and the controls stay covered during export. -->
        <div class="busy-overlay">
            <h2 class="modal-title">{$t('export.exportingTitle')} <span class="spinner"></span></h2>
            <p class="status-text">{$t('export.exportingPositions', { count: positionCount })}</p>
            <button onclick={onCancel}>{$t('common.cancel')}</button>
        </div>
    {/if}
</Modal>

<style>
    .busy-overlay {
        position: absolute;
        inset: 0;
        background-color: color-mix(in srgb, var(--color-surface) 94%, transparent);
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: var(--space-2);
        border-radius: var(--radius);
    }

    input[type='text'],
    input[type='password'],
    input[type='date'],
    textarea {
        padding: var(--space-1) var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
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
        width: 100%;
    }

    /* An empty field its ticked box needs: the export waits for it. */
    .invalid {
        border-color: var(--color-danger);
    }

    .check {
        display: inline-flex;
        align-items: center;
        gap: var(--space-1);
        cursor: pointer;
    }

    input[type='checkbox'],
    input[type='radio'] {
        accent-color: var(--color-text);
        cursor: pointer;
    }

    input:disabled,
    .check:has(input:disabled) {
        cursor: not-allowed;
        opacity: 0.5;
    }

    .note {
        margin: var(--space-1) var(--space-3);
        color: var(--color-text-muted);
    }

    .count {
        font-variant-numeric: tabular-nums;
        color: var(--color-text-muted);
    }

    .password-row {
        display: inline-flex;
        gap: var(--space-1);
        align-items: stretch;
    }

    .reveal,
    .busy-overlay button {
        padding: var(--space-1) var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background-color: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }

    .reveal:hover,
    .busy-overlay button:hover {
        background-color: var(--color-surface-alt);
    }
</style>
