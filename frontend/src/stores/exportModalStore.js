import { writable } from 'svelte/store';

export const exportModalModeStore = writable('preparing'); // 'preparing', 'metadata', 'exporting', 'completed'
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
    exportModalModeStore.set('preparing');
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
