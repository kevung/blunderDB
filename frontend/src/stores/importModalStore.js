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

// The end-of-import report (#257): what the import just brought in — its PR,
// its worst decisions, how many positions the source tool had flagged and how
// many nothing has judged yet. null while no import has produced one, and
// after every one that could not record a batch: the report is a convenience,
// and its absence must never look like a failed import.
export const fileImportReportStore = writable(null);
