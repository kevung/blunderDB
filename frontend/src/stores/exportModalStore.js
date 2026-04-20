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
    collectionIDs: []
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
        collectionIDs: []
    });
    exportMatchesStore.set([]);
}
