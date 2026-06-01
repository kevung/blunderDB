import { tMsg } from '../i18n';
import { get } from 'svelte/store';
import { OpenExportDatabaseDialog, ShowAlert } from '../../wailsjs/go/gui/App.js';
import { ExportDatabase, GetAllMatches, GetAllCollections, GetAllTournaments } from '../../wailsjs/go/database/Database.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { statusBarModeStore, openModal, closeModal, MODAL } from '../stores/uiStore.js';
import { collectionsStore } from '../stores/collectionStore.js';
import { tournamentsStore } from '../stores/tournamentStore.js';
import { exportModalModeStore, exportPositionCountStore, exportMetadataStore, exportOptionsStore, exportMatchesStore, resetExportState } from '../stores/exportModalStore.js';
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

        await ExportDatabase({
            exportPath: pendingExportPath,
            positions: positions,
            metadata: {
                user: metadata.user || '',
                description: metadata.description || '',
                dateOfCreation: metadata.dateOfCreation || ''
            },
            includeAnalysis: exportOptions.includeAnalysis,
            includeComments: exportOptions.includeComments,
            includeFilterLibrary: exportOptions.includeFilterLibrary,
            includePlayedMoves: exportOptions.includePlayedMoves,
            includeMatches: exportOptions.includeMatches,
            includeCollections: exportOptions.includeCollections,
            collectionIDs: exportOptions.collectionIDs || [],
            matchIDs: exportOptions.matchIDs || [],
            tournamentIDs: exportOptions.includeTournamentIDs || []
        });

        logger.log('Export completed successfully');
        exportModalModeStore.set('completed');

        const posCount = get(exportPositionCountStore);
        setStatusBarMessage(tMsg('status.exportCompleted', { posCount }));
    } catch (error) {
        logger.error('Error committing export:', error);
        closeModal();
        setStatusBarMessage(tMsg('status.errorCommittingExport', { error }));
        await ShowAlert(`Error committing export: ${error}`);
        statusBarModeStore.set('NORMAL');
    } finally {
        resetExportState();
        exportMatchesStore.set([]);
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

export function handleExportClose() {
    logger.log('Export completed and closed');
    closeModal();
    pendingExportPath = null;
    resetExportState();
    exportMatchesStore.set([]);
    statusBarModeStore.set('NORMAL');
}
