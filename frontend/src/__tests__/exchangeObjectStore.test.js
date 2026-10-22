import { describe, it, expect, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { exchangeObjectStore, noteSelection } from '../stores/exchangeObjectStore.js';
import { selectedTournamentStore } from '../stores/tournamentStore.js';
import { selectedCollectionStore } from '../stores/collectionStore.js';

describe('exchangeObjectStore', () => {
    beforeEach(() => {
        selectedTournamentStore.set(null);
        selectedCollectionStore.set(null);
        exchangeObjectStore.set(null);
    });

    it('follows the last selection, whichever panel made it', () => {
        selectedTournamentStore.set(/** @type {any} */ ({ id: 4, name: 'Open' }));
        expect(get(exchangeObjectStore)).toEqual({ kind: 'tournament', id: 4, label: 'Open' });
        noteSelection('match', { id: 9, label: 'A – B' });
        expect(get(exchangeObjectStore)).toEqual({ kind: 'match', id: 9, label: 'A – B' });
    });

    it('forgets a cleared selection only when it was that one', () => {
        noteSelection('match', { id: 9, label: 'A – B' });
        selectedCollectionStore.set(null);
        expect(get(exchangeObjectStore)?.kind).toBe('match');
        noteSelection('match', null);
        expect(get(exchangeObjectStore)).toBeNull();
    });
});
