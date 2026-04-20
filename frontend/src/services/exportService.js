import { get } from 'svelte/store';
import {
    OpenExportDatabaseDialog,
    ShowAlert,
} from '../../wailsjs/go/main/App.js';
import {
    ExportDatabase,
    GetAllMatches,
    GetAllCollections,
    GetAllTournaments,
} from '../../wailsjs/go/main/Database.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { statusBarModeStore, showExportDatabaseModalStore } from '../stores/uiStore.js';
import { collectionsStore } from '../stores/collectionStore.js';
import { tournamentsStore } from '../stores/tournamentStore.js';
import {
    exportModalModeStore,
    exportPositionCountStore,
    exportMetadataStore,
    exportOptionsStore,
    exportMatchesStore,
    resetExportState,
} from '../stores/exportModalStore.js';
import { setStatusBarMessage } from './databaseService.js';

let pendingExportPath = null;

export async function exportDatabase() {
    console.log('exportDatabase');

    if (!get(databasePathStore)) {
        setStatusBarMessage('No database opened. Please open a database first.');
        return;
    }

    const positions = get(positionsStore);
    if (positions.length === 0) {
        setStatusBarMessage('No positions to export.');
        await ShowAlert('No positions to export. Please load positions first.');
        return;
    }

    try {
        const exportFilePath = await OpenExportDatabaseDialog();
        if (!exportFilePath) {
            console.log('No export path selected');
            return;
        }

        console.log('Exporting to:', exportFilePath);
        pendingExportPath = exportFilePath;

        try {
            const matches = await GetAllMatches();
            exportMatchesStore.set(matches || []);
        } catch (e) {
            console.log('Could not get matches:', e);
            exportMatchesStore.set([]);
        }

        try {
            const colls = await GetAllCollections();
            collectionsStore.set(colls || []);
        } catch (e) {
            console.log('Could not get collections:', e);
        }

        try {
            const tourns = await GetAllTournaments();
            tournamentsStore.set(tourns || []);
        } catch (e) {
            console.log('Could not get tournaments:', e);
        }

        exportPositionCountStore.set(positions.length);
        exportModalModeStore.set('metadata');
        showExportDatabaseModalStore.set(true);

    } catch (error) {
        console.error('Error during export preparation:', error);
        setStatusBarMessage(`Error preparing export: ${error}`);
        await ShowAlert(`Error preparing export: ${error}`);
        statusBarModeStore.set('NORMAL');
    }
}

export async function handleExportCommit() {
    if (!pendingExportPath) {
        console.error('No pending export path');
        return;
    }

    console.log('Committing export to:', pendingExportPath);
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

        console.log('Export completed successfully');
        exportModalModeStore.set('completed');

        const posCount = get(exportPositionCountStore);
        setStatusBarMessage(`Export completed: ${posCount} position(s) exported`);

    } catch (error) {
        console.error('Error committing export:', error);
        showExportDatabaseModalStore.set(false);
        setStatusBarMessage(`Error committing export: ${error}`);
        await ShowAlert(`Error committing export: ${error}`);
        statusBarModeStore.set('NORMAL');
    } finally {
        resetExportState();
        exportMatchesStore.set([]);
    }
}

export function handleExportCancel() {
    console.log('Export cancelled by user');

    showExportDatabaseModalStore.set(false);
    pendingExportPath = null;
    resetExportState();
    exportMatchesStore.set([]);
    setStatusBarMessage('Export cancelled');
    statusBarModeStore.set('NORMAL');
}

export function handleExportClose() {
    console.log('Export completed and closed');
    showExportDatabaseModalStore.set(false);
    pendingExportPath = null;
    resetExportState();
    exportMatchesStore.set([]);
    statusBarModeStore.set('NORMAL');
}
