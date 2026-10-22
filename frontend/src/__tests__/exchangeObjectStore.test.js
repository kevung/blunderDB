import { describe, it, expect, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { exchangeObjectStore, noteSelection, forgetSelection } from '../stores/exchangeObjectStore.js';
import { selectedTournamentStore } from '../stores/tournamentStore.js';
import { selectedCollectionStore } from '../stores/collectionStore.js';

describe('exchangeObjectStore', () => {
    beforeEach(() => {
        selectedTournamentStore.set(null);
        selectedCollectionStore.set(null);
        exchangeObjectStore.set(null);
    });

    it('follows the last selection, whichever panel made it', () => {
        noteSelection('tournament', { id: 4, label: 'Open' });
        expect(get(exchangeObjectStore)).toEqual({ kind: 'tournament', id: 4, label: 'Open' });
        selectedCollectionStore.set(/** @type {any} */ ({ id: 2, name: 'Primes' }));
        expect(get(exchangeObjectStore)).toEqual({ kind: 'collection', id: 2, label: 'Primes' });
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

    it('survives the tournament browser dropping its selection when its tab is hidden', () => {
        selectedTournamentStore.set(/** @type {any} */ ({ id: 4, name: 'Open' }));
        noteSelection('tournament', { id: 4, label: 'Open' });
        selectedTournamentStore.set(null);
        expect(get(exchangeObjectStore)).toEqual({ kind: 'tournament', id: 4, label: 'Open' });
        forgetSelection('match');
        expect(get(exchangeObjectStore)?.kind).toBe('tournament');
        forgetSelection('tournament');
        expect(get(exchangeObjectStore)).toBeNull();
    });

    it('keeps the name of the tournament renamed while it is the selection', () => {
        noteSelection('tournament', { id: 4, label: 'Open' });
        selectedTournamentStore.set(/** @type {any} */ ({ id: 4, name: 'Open 2026' }));
        expect(get(exchangeObjectStore)).toEqual({ kind: 'tournament', id: 4, label: 'Open 2026' });
        noteSelection('match', { id: 9, label: 'A – B' });
        selectedTournamentStore.set(/** @type {any} */ ({ id: 4, name: 'Open 2027' }));
        expect(get(exchangeObjectStore)?.kind).toBe('match');
    });
});
