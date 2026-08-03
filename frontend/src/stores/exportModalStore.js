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
    // Issuance: off by default, so an ordinary export stays exactly what it was. A
    // watermark can only be posed here, at export, because nothing else in the product
    // states who a copy is for. See ADR-0007.
    watermark: false,
    distribution: '',
    recipients: '',
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
        watermark: false,
        distribution: '',
        recipients: '',
        password: ''
    });
    exportMatchesStore.set([]);
}
