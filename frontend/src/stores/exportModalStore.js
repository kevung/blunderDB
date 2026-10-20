import { writable } from 'svelte/store';

// Two screens only: the form, and the progress laid over it. A 'preparing' state existed
// before the native file dialog — which is blocking, so it was never painted — and a
// 'completed' one that asked for a click to acknowledge success the status bar already
// reports. Both are gone.
export const exportModalModeStore = writable('metadata'); // 'metadata', 'exporting'
// How many positions the chosen scope exports.
export const exportPositionCountStore = writable(0);
export const exportMetadataStore = writable({
    user: '',
    description: '',
    dateOfCreation: ''
});

// The folder an incremental export last went to: kept across openings of the dialog, so
// that synchronising again is one tick (ADR-0096). The cursor itself lives in the library.
let lastSyncFolder = '';
/** @param {string} folder */
export function rememberSyncFolder(folder) {
    lastSyncFolder = folder;
}

function defaultOptions() {
    return {
        // The current list, or the whole library (the daemon's export).
        wholeLibrary: false,
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
        // Brings what the exported collections show into a list export, so that none
        // arrives truncated and a living one finds its positions in the file.
        includeCollectionPositions: false,
        includeLessons: false,
        lessonIDs: [],
        includeAnkiDecks: false,
        deckIDs: [],
        // Off: a deck leaves as a fresh study copy.
        includeAnkiHistory: false,
        // Two optional, independent mechanisms, both off by default so an ordinary export
        // stays exactly what it was: mark where the file comes from, and protect it with a
        // password. See ADR-0007.
        watermarkEnabled: false,
        watermark: '',
        watermarkNote: '',
        passwordEnabled: false,
        password: '',
        // Only what entered the library since the last incremental export to syncFolder.
        incremental: false,
        syncFolder: lastSyncFolder
    };
}

export const exportOptionsStore = writable(defaultOptions());
// The matches the chosen scope offers.
export const exportMatchesStore = writable([]);
export const exportDecksStore = writable([]);
// Per collection id, how many of its positions the chosen scope exports.
export const exportCollectionCoverageStore = writable({});
// Whether the scope can be chosen: a list can be exported as the whole library.
export const exportScopeStore = writable({ chooseable: false });

export function resetExportState() {
    exportModalModeStore.set('metadata');
    exportMetadataStore.set({ user: '', description: '', dateOfCreation: '' });
    exportOptionsStore.set(defaultOptions());
    exportMatchesStore.set([]);
    exportDecksStore.set([]);
    exportCollectionCoverageStore.set({});
    exportScopeStore.set({ chooseable: false });
}
