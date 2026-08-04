import { tMsg } from '../i18n';
import { get } from 'svelte/store';
import { OpenExportDatabaseDialog, OpenExportMatDialog, ShowAlert } from '../../wailsjs/go/gui/App.js';
import { ExportDatabase, CollectionCoverage, LoadMetadata, ExportMatchMAT, SuggestMatFilename, GetAllMatches, GetAllCollections, GetAllTournaments } from '../../wailsjs/go/database/Database.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { positionsStore } from '../stores/positionStore.js';
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

let pendingExportPath = null;

export async function exportDatabase() {
    logger.log('exportDatabase');

    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDbOpenedFirst'));
        return;
    }

    const positions = get(positionsStore);
    if (positions.length === 0) {
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

        try {
            const matches = await GetAllMatches();
            exportMatchesStore.set(matches || []);
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
            exportCollectionCoverageStore.set((await CollectionCoverage(positions.map((p) => p.id))) || {});
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

        exportPositionCountStore.set(positions.length);
        exportModalModeStore.set('metadata');
        openModal(MODAL.EXPORT_DATABASE);
    } catch (error) {
        logger.error('Error during export preparation:', error);
        setStatusBarMessage(tMsg('status.errorPreparingExport', { error }));
        await ShowAlert(`Error preparing export: ${error}`);
        statusBarModeStore.set('NORMAL');
    }
}

// exportMatchMat exports a single match to a Jellyfish/gnubg .mat file. The save
// dialog is pre-filled with a name built server-side (the same helper the CLI
// batch uses), and the file is written straight to the chosen path — a .mat has
// nothing to configure, so there is no modal. Cancelling the dialog is a silent
// no-op; a write failure raises a native alert and a status-bar message.
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
        const positions = get(positionsStore);

        const baseOptions = {
            exportPath: pendingExportPath,
            // Send identifiers, not positions. Shipping the whole set was 73 MB of JSON for
            // a real database — seconds of JSON.stringify on the browser's only thread,
            // which left the progress dialog blank, then the same volume across the bridge
            // and decoded again. The exporter reads them from the database it already has
            // open. See ExportOptions.PositionIDs.
            positionIDs: positions.map((p) => p.id),
            metadata: {
                user: metadata.user || '',
                description: metadata.description || '',
                dateOfCreation: metadata.dateOfCreation || ''
            },
            includeAnalysis: exportOptions.includeAnalysis,
            includeComments: exportOptions.includeComments,
            includeFilterLibrary: exportOptions.includeFilterLibrary,
            includePlayedMoves: exportOptions.includePlayedMoves,
            // The backend treats "IncludeMatches && empty MatchIDs" as "export ALL
            // matches" (the CLI's --match-ids empty=all convention). In the GUI the
            // modal always auto-fills every match id when the section is enabled, so
            // an empty selection here can only mean the user explicitly clicked "None"
            // (or unchecked every match) — which must export no matches, not all.
            // Collapse that to includeMatches=false so "None" means none.
            includeMatches: exportOptions.includeMatches && (exportOptions.matchIDs || []).length > 0,
            includeCollections: exportOptions.includeCollections,
            collectionIDs: exportOptions.collectionIDs || [],
            matchIDs: exportOptions.matchIDs || [],
            tournamentIDs: exportOptions.includeTournamentIDs || []
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

        // No completion screen: the dialog closes and the status bar says what happened.
        // Acknowledging "it worked" with a click is friction, and it is how every other
        // long operation in blunderDB reports — the .mat export has never had a dialog at
        // all.
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
