import { tMsg } from '../i18n';
import { get } from 'svelte/store';
import { OpenExportDatabaseDialog, OpenExportMatDialog, ShowAlert } from '../../wailsjs/go/gui/App.js';
import {
    ExportDatabase,
    CollectionCoverage,
    LoadMetadata,
    ExportMatchMAT,
    SuggestMatFilename,
    GetAllMatches,
    GetAllCollections,
    GetAllTournaments,
    GetAllAnkiDecks,
    CountPositions,
    ListLessons,
    MatchIDsReachedBy,
    PlanExport,
    PlanExportIDs
} from '../../wailsjs/go/database/Database.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { positionsStore, browsingLibrary, listedIds, openLibrary } from '../stores/positionStore.js';
import { openMatchListInPanel } from './positionLoader.js';
import { statusBarModeStore, openModal, closeModal, MODAL } from '../stores/uiStore.js';
import { collectionsStore } from '../stores/collectionStore.js';
import { tournamentsStore } from '../stores/tournamentStore.js';
import {
    exportModalModeStore,
    exportPositionCountStore,
    exportMetadataStore,
    exportOptionsStore,
    exportMatchesStore,
    exportCollectionCoverageStore,
    exportDecksStore,
    exportScopeStore,
    resetExportState
} from '../stores/exportModalStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';

/**
 * What one scope offers: the matches to pick from, how much of each collection it covers,
 * and how many positions it exports. Read once per opening, on the first choice of it.
 * @typedef {{ matches: any[], coverage: Record<string, number>, count: number }} ScopeData
 */
/** @type {{ list?: ScopeData, library?: ScopeData }} */
let scopes = {};

/** @type {Promise<number[]> | null} */
let listed = null;
// Read once: a paged search result is read whole for this, a full scan.
const exportedIds = () => (listed ??= listedIds());

/** @param {boolean} library @returns {Promise<ScopeData>} */
async function readScope(library) {
    /** @type {any[]} */
    let matches = [];
    try {
        matches = (await GetAllMatches()) || [];
        // A list offers the matches its positions were played in; the library offers them all.
        if (!library) {
            const reached = new Set((await MatchIDsReachedBy(await exportedIds())) || []);
            matches = matches.filter((m) => reached.has(m.id));
        }
    } catch (e) {
        logger.log('Could not get matches:', e);
    }
    // How much of each collection the scope covers. The export only writes membership for
    // positions it exports, so a collection can arrive truncated; the screen says so.
    /** @type {Record<string, number>} */
    let coverage = {};
    try {
        coverage = library ? Object.fromEntries((get(collectionsStore) || []).map((/** @type {any} */ c) => [c.id, c.positionCount ?? 0])) : (await CollectionCoverage(await exportedIds())) || {};
    } catch (e) {
        logger.log('Could not measure collection coverage:', e);
    }
    // The ids the export will send, not the list's length: a search result still being
    // counted is only as long as its first page.
    const count = library ? (browsingLibrary() ? get(positionsStore).length : (await CountPositions()) || 0) : (await exportedIds()).length;
    return { matches, coverage, count };
}

/**
 * Shows what a scope exports: its matches, its coverage of the collections, its count.
 * @param {boolean} library
 */
export async function setExportScope(library) {
    const key = library ? 'library' : 'list';
    const data = (scopes[key] ??= await readScope(library));
    exportMatchesStore.set(/** @type {never[]} */ (data.matches));
    exportCollectionCoverageStore.set(data.coverage);
    exportPositionCountStore.set(data.count);
}

export async function exportDatabase() {
    logger.log('exportDatabase');

    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDbOpenedFirst'));
        return;
    }

    // The export takes ids (ExportOptions.PositionIDs), or the whole library.
    if (get(positionsStore).length === 0) {
        setStatusBarMessage(tMsg('status.noPositionsToExport'));
        await ShowAlert('No positions to export. Please load positions first.');
        return;
    }

    try {
        scopes = {};
        listed = null;
        const library = browsingLibrary();

        try {
            collectionsStore.set((await GetAllCollections()) || []);
        } catch (e) {
            logger.log('Could not get collections:', e);
        }
        try {
            tournamentsStore.set((await GetAllTournaments()) || []);
        } catch (e) {
            logger.log('Could not get tournaments:', e);
        }
        try {
            exportDecksStore.set(/** @type {never[]} */ ((await GetAllAnkiDecks()) || []));
        } catch (e) {
            logger.log('Could not get decks:', e);
            exportDecksStore.set([]);
        }
        await setExportScope(library);
        // A list may still be exported as the whole library; the library is only itself.
        exportScopeStore.set({ chooseable: !library });

        // The metadata describe the file being produced. Starting from the source's own
        // values beats an empty form: most exports keep them, and retyping them every time
        // was pure friction.
        try {
            const source = await LoadMetadata();
            exportMetadataStore.set({
                user: source.user || '',
                description: source.description || '',
                dateOfCreation: new Date().toISOString().slice(0, 10)
            });
        } catch (e) {
            logger.log('Could not read the source metadata:', e);
        }

        exportOptionsStore.update((o) => ({ ...o, ...scopeDefaults(library) }));
        exportModalModeStore.set('metadata');
        openModal(MODAL.EXPORT_DATABASE);
    } catch (error) {
        logger.error('Error during export preparation:', error);
        setStatusBarMessage(tMsg('status.errorPreparingExport', { error }));
        await ShowAlert(`Error preparing export: ${error}`);
        statusBarModeStore.set('NORMAL');
    }
}

/**
 * The families a scope starts with: the whole library ticks every one, as the daemon's
 * export carries them, with the review history (a backup); a list carries its positions
 * and their matches.
 * @param {boolean} library
 */
export function scopeDefaults(library) {
    return {
        wholeLibrary: library,
        includeFilterLibrary: library,
        includeTournaments: library,
        includeCollections: library,
        includeLessons: library,
        includeAnkiDecks: library,
        includeAnkiHistory: library
    };
}

// exportMatchMat exports one match to a .mat file: server-built file name (as
// in the CLI), written straight to the chosen path, no modal. Cancel is a
// silent no-op; a write failure raises an alert and a status message.
export async function exportMatchMat(match) {
    if (!match || match.id == null) return;
    try {
        const defaultName = await SuggestMatFilename(match.id);
        const filePath = await OpenExportMatDialog(defaultName);
        if (!filePath) {
            return; // user cancelled the dialog
        }
        await ExportMatchMAT(match.id, filePath);
        setStatusBarMessage(tMsg('match.matExported'));
    } catch (error) {
        logger.error('Error exporting match to .mat:', error);
        setStatusBarMessage(tMsg('match.errorExportingMat'));
        await ShowAlert(`Error exporting match: ${error}`);
    }
}

/**
 * The backend's ExportOptions for the choices made in the dialog, minus the file and the
 * two sealing mechanisms: what the commit sends and what the preview counts, one function so
 * that the counts are those of the file.
 * @param {any} exportOptions
 * @param {string} exportPath
 */
export async function buildExportOptions(exportOptions, exportPath) {
    const metadata = get(exportMetadataStore);
    const wholeLibrary = !!exportOptions.wholeLibrary;
    const lessons = wholeLibrary && exportOptions.includeLessons ? (await ListLessons()) || [] : [];
    // Each family: none when its box is off or its pick emptied by hand; in the whole
    // library, a pick of every item is sent as no ids, which the backend reads as all of it.
    const family = (/** @type {boolean} */ on, /** @type {number[] | undefined} */ picked, /** @type {{ id: number }[]} */ all) => {
        const ids = picked || [];
        const include = !!on && ids.length > 0;
        const everything = wholeLibrary && all.length > 0 && all.every((x) => ids.includes(x.id));
        return { include, ids: include && !everything ? ids : [] };
    };
    const matches = family(exportOptions.includeMatches, exportOptions.matchIDs, get(exportMatchesStore) || []);
    const collections = family(exportOptions.includeCollections, exportOptions.collectionIDs, get(collectionsStore) || []);
    const tournaments = family(exportOptions.includeTournaments, exportOptions.includeTournamentIDs, get(tournamentsStore) || []);
    const lessonPick = family(exportOptions.includeLessons, exportOptions.lessonIDs, lessons);
    const decks = family(exportOptions.includeAnkiDecks, exportOptions.deckIDs, get(exportDecksStore) || []);
    // A tournament carries its matches: their drafts and aliases follow them as well.
    const someMatches = matches.include || tournaments.include;
    const baseOptions = {
        exportPath,
        // The library is the daemon's whole-tenant export (ExportOptions.WholeLibrary); a
        // list goes as identifiers, not positions: the whole set can be tens of MB of JSON
        // on the UI thread.
        wholeLibrary,
        positionIDs: wholeLibrary ? [] : await listedIds(),
        metadata: {
            user: metadata.user || '',
            description: metadata.description || '',
            dateOfCreation: metadata.dateOfCreation || ''
        },
        includeAnalysis: exportOptions.includeAnalysis,
        includeComments: exportOptions.includeComments,
        includeFilterLibrary: exportOptions.includeFilterLibrary,
        includePlayedMoves: exportOptions.includePlayedMoves,
        includeMatches: matches.include,
        matchIDs: matches.ids,
        // A draft follows its match: without the match it has nothing to travel with.
        includeTranscriptions: someMatches && !!exportOptions.includeTranscriptions,
        // An alias leaves only when it names someone an exported match writes.
        includeAliases: someMatches && !!exportOptions.includeAliases,
        includeCollections: collections.include,
        collectionIDs: collections.ids,
        // The library holds every position already.
        includeCollectionPositions: !wholeLibrary && collections.include && !!exportOptions.includeCollectionPositions,
        includeTournaments: tournaments.include,
        tournamentIDs: wholeLibrary ? tournaments.ids : exportOptions.includeTournamentIDs || [],
        includeLessons: lessonPick.include,
        lessonIDs: lessonPick.ids,
        // In the library no ids means every deck; outside it, the ids are the decks.
        includeAnkiDecks: decks.include,
        deckIDs: wholeLibrary ? decks.ids : decks.include ? exportOptions.deckIDs || [] : [],
        includeAnkiHistory: decks.include && !!exportOptions.includeAnkiHistory
    };
    return baseOptions;
}

/**
 * What the file would hold for the choices in the dialog, counted by the backend on the
 * same selection the export walks.
 * @param {any} exportOptions
 */
export async function planExport(exportOptions) {
    return PlanExport(/** @type {any} */ (await buildExportOptions(exportOptions, '')));
}

/**
 * The source ids behind one count of the plan (a family of ingest.PlanExport), up to the
 * backend's page bound.
 * @param {any} exportOptions
 * @param {string} family
 * @returns {Promise<number[]>}
 */
export async function plannedIds(exportOptions, family) {
    const page = await PlanExportIDs(/** @type {any} */ (await buildExportOptions(exportOptions, '')), family, 0, 0);
    return page?.ids || [];
}

export async function handleExportCommit() {
    // The file is asked for last, once what goes into it is settled; dismissing the
    // dialog returns to the form.
    let exportPath = '';
    try {
        exportPath = await OpenExportDatabaseDialog();
    } catch (error) {
        logger.error('Error choosing the export path:', error);
    }
    if (!exportPath) {
        logger.log('No export path selected');
        return;
    }

    logger.log('Committing export to:', exportPath);
    exportModalModeStore.set('exporting');

    try {
        const exportOptions = get(exportOptionsStore);
        const baseOptions = await buildExportOptions(exportOptions, exportPath);

        // Both mechanisms are plain export options: the watermark is sealed and written by
        // the backend, and a password wraps the finished file. Neither changes anything
        // about what is exported.
        await ExportDatabase({
            ...baseOptions,
            watermark: exportOptions.watermarkEnabled ? (exportOptions.watermark || '').trim() : '',
            watermarkNote: exportOptions.watermarkEnabled ? (exportOptions.watermarkNote || '').trim() : '',
            password: exportOptions.passwordEnabled ? exportOptions.password || '' : ''
        });

        logger.log('Export completed successfully');

        // No completion screen: the status bar reports, as for every long
        // operation.
        const posCount = get(exportPositionCountStore);
        closeModal();
        resetExportState();
        setStatusBarMessage(tMsg('status.exportCompleted', { posCount }));
        statusBarModeStore.set('NORMAL');
    } catch (error) {
        logger.error('Error committing export:', error);
        closeModal();
        resetExportState();
        setStatusBarMessage(tMsg('status.errorCommittingExport', { error }));
        await ShowAlert(`Error committing export: ${error}`);
        statusBarModeStore.set('NORMAL');
    }
}

/**
 * A count opens what it counts: the export gives way to the positions of the scope — the
 * library, or the list already behind the dialog.
 * @param {boolean} library
 */
export async function openExportPositions(library) {
    handleExportCancel();
    if (library && !browsingLibrary()) await openLibrary();
}

/**
 * Opens the matches the export offers, in the Matches panel.
 * @param {number[]} ids
 */
export async function openExportMatches(ids) {
    handleExportCancel();
    await openMatchListInPanel(ids);
}

export function handleExportCancel() {
    logger.log('Export cancelled by user');

    closeModal();
    resetExportState();
    setStatusBarMessage(tMsg('status.exportCancelled'));
    statusBarModeStore.set('NORMAL');
}
