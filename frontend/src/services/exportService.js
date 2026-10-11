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
    ListLessons,
    MatchIDsReachedBy
} from '../../wailsjs/go/database/Database.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { positionsStore, browsingLibrary, listedIds } from '../stores/positionStore.js';
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
    resetExportState
} from '../stores/exportModalStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';

/** @type {string | null} */
let pendingExportPath = null;

export async function exportDatabase() {
    logger.log('exportDatabase');

    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDbOpenedFirst'));
        return;
    }

    // The export takes ids (ExportOptions.PositionIDs), or AllPositions for the library.
    if (get(positionsStore).length === 0) {
        setStatusBarMessage(tMsg('status.noPositionsToExport'));
        await ShowAlert('No positions to export. Please load positions first.');
        return;
    }

    try {
        const exportFilePath = await OpenExportDatabaseDialog();
        if (!exportFilePath) {
            logger.log('No export path selected');
            return;
        }

        logger.log('Exporting to:', exportFilePath);
        pendingExportPath = exportFilePath;

        // Read once: a paged search result is read whole for this, a full scan.
        /** @type {Promise<number[]> | null} */
        let listed = null;
        const exportedIds = () => (listed ??= listedIds());

        // A list offers the matches its positions were played in; the library offers them all.
        try {
            /** @type {any[]} */
            let matches = (await GetAllMatches()) || [];
            if (!browsingLibrary()) {
                const reached = new Set((await MatchIDsReachedBy(await exportedIds())) || []);
                matches = matches.filter((m) => reached.has(m.id));
            }
            exportMatchesStore.set(/** @type {never[]} */ (matches));
        } catch (e) {
            logger.log('Could not get matches:', e);
            exportMatchesStore.set([]);
        }

        try {
            const colls = await GetAllCollections();
            collectionsStore.set(colls || []);
        } catch (e) {
            logger.log('Could not get collections:', e);
        }

        try {
            const tourns = await GetAllTournaments();
            tournamentsStore.set(tourns || []);
        } catch (e) {
            logger.log('Could not get tournaments:', e);
        }

        // How much of each collection the current selection actually covers. The export only
        // writes membership for positions it exports, so a collection can arrive truncated;
        // the screen says so rather than letting it happen quietly.
        try {
            // The whole library covers every collection: no id list to send.
            const coverage = browsingLibrary()
                ? Object.fromEntries((get(collectionsStore) || []).map((/** @type {any} */ c) => [c.id, c.positionCount ?? 0]))
                : (await CollectionCoverage(await exportedIds())) || {};
            exportCollectionCoverageStore.set(coverage);
        } catch (e) {
            logger.log('Could not measure collection coverage:', e);
            exportCollectionCoverageStore.set({});
        }

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

        // The ids the export will send, not the list's length: a search result still being
        // counted is only as long as its first page.
        exportPositionCountStore.set(browsingLibrary() ? get(positionsStore).length : (await exportedIds()).length);
        // The whole library starts from what the daemon's export carries: every family ticked.
        if (browsingLibrary()) {
            exportOptionsStore.update((o) => ({ ...o, includeFilterLibrary: true, includeTournaments: true, includeCollections: true, includeLessons: true }));
        }
        exportModalModeStore.set('metadata');
        openModal(MODAL.EXPORT_DATABASE);
    } catch (error) {
        logger.error('Error during export preparation:', error);
        setStatusBarMessage(tMsg('status.errorPreparingExport', { error }));
        await ShowAlert(`Error preparing export: ${error}`);
        statusBarModeStore.set('NORMAL');
    }
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

export async function handleExportCommit() {
    if (!pendingExportPath) {
        logger.error('No pending export path');
        return;
    }

    logger.log('Committing export to:', pendingExportPath);
    exportModalModeStore.set('exporting');

    try {
        const metadata = get(exportMetadataStore);
        const exportOptions = get(exportOptionsStore);
        const wholeLibrary = browsingLibrary();
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
        const baseOptions = {
            exportPath: pendingExportPath,
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
            includeTranscriptions: matches.include && !!exportOptions.includeTranscriptions,
            includeCollections: collections.include,
            collectionIDs: collections.ids,
            includeTournaments: tournaments.include,
            tournamentIDs: wholeLibrary ? tournaments.ids : exportOptions.includeTournamentIDs || [],
            includeLessons: lessonPick.include,
            lessonIDs: lessonPick.ids,
            // No deck picker yet: the whole library carries every deck, as the daemon's does.
            includeAnkiDecks: wholeLibrary
        };

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
        exportMatchesStore.set([]);
        pendingExportPath = null;
        setStatusBarMessage(tMsg('status.exportCompleted', { posCount }));
        statusBarModeStore.set('NORMAL');
    } catch (error) {
        logger.error('Error committing export:', error);
        closeModal();
        resetExportState();
        exportMatchesStore.set([]);
        pendingExportPath = null;
        setStatusBarMessage(tMsg('status.errorCommittingExport', { error }));
        await ShowAlert(`Error committing export: ${error}`);
        statusBarModeStore.set('NORMAL');
    }
}

export function handleExportCancel() {
    logger.log('Export cancelled by user');

    closeModal();
    pendingExportPath = null;
    resetExportState();
    exportMatchesStore.set([]);
    setStatusBarMessage(tMsg('status.exportCancelled'));
    statusBarModeStore.set('NORMAL');
}
