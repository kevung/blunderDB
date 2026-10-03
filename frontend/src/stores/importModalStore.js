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

// The end-of-import report (PR, worst decisions, flagged and unjudged counts). null when an import
// could not record a batch: a convenience whose absence must never look like a failure.
export const fileImportReportStore = writable(null);

// The batch's per-file journal, one line per file with its last outcome (new, duplicate, enriched,
// error). Empty when no batch could be recorded.
export const fileImportJournalStore = writable([]);
// {batchID, files} of an import the user stopped, what "Resume" needs; null otherwise.
export const fileImportInterruptedStore = writable(null);

// The pipeline's progress (files, positions, bytes, rate, ETA), null before the first event.
export const fileImportProgressStore = writable(null);
// The import goes on behind a status-bar chip while the user keeps working.
export const fileImportMinimizedStore = writable(false);
