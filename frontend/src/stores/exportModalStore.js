import { writable } from 'svelte/store';

// Two screens only: the form, and the progress laid over it. A 'preparing' state existed
// before the native file dialog — which is blocking, so it was never painted — and a
// 'completed' one that asked for a click to acknowledge success the status bar already
// reports. Both are gone.
export const exportModalModeStore = writable('metadata'); // 'metadata', 'exporting'
export const exportPositionCountStore = writable(0);
export const exportMetadataStore = writable({
    user: '',
    description: '',
    dateOfCreation: ''
});
export const exportOptionsStore = writable({
    includeAnalysis: true,
    includeComments: true,
    includeFilterLibrary: false,
    includePlayedMoves: true,
    includeMatches: true,
    matchIDs: [],
    includeTournaments: false,
    includeTournamentIDs: [],
    includeCollections: false,
    collectionIDs: [],
    // Two optional, independent mechanisms, both off by default so an ordinary export
    // stays exactly what it was: mark where the file comes from, and protect it with a
    // password. See ADR-0007.
    watermarkEnabled: false,
    watermark: '',
    watermarkNote: '',
    passwordEnabled: false,
    password: ''
});
export const exportMatchesStore = writable([]);

export function resetExportState() {
    exportModalModeStore.set('metadata');
    exportMetadataStore.set({ user: '', description: '', dateOfCreation: '' });
    exportOptionsStore.set({
        includeAnalysis: true,
        includeComments: true,
        includeFilterLibrary: false,
        includePlayedMoves: true,
        includeMatches: true,
        matchIDs: [],
        includeTournaments: false,
        includeTournamentIDs: [],
        includeCollections: false,
        collectionIDs: [],
        watermarkEnabled: false,
        watermark: '',
        watermarkNote: '',
        passwordEnabled: false,
        password: ''
    });
    exportMatchesStore.set([]);
}
