import { writable } from 'svelte/store';

export const showImportProgressModalStore = writable(false);
export const importModalModeStore = writable('analyzing'); // 'analyzing', 'preview', 'committing', 'completed'
export const importAnalysisStore = writable({
    toAdd: 0,
    toMerge: 0,
    toSkip: 0,
    total: 0,
    importPath: ''
});
export const importResultStore = writable({
    added: 0,
    merged: 0,
    skipped: 0,
    total: 0
});

export const showFileImportModalStore = writable(false);
export const fileImportModeStore = writable('idle'); // 'idle', 'importing', 'completed'
export const fileImportTotalFilesStore = writable(0);
export const fileImportCurrentIndexStore = writable(0);
export const fileImportCurrentFileStore = writable('');
export const fileImportResultsStore = writable({ succeeded: 0, failed: 0, skipped: 0, errors: [] });
