/**
 * fuzzy.test.js — the approximate matching of the command palette (#287).
 */

import { describe, test, expect } from 'vitest';
import { fold, fuzzyMatch, highlightSegments } from '../utils/fuzzy.js';

describe('fold', () => {
    test('lower-cases and strips accents, one character for one', () => {
        expect(fold('Équité Kazaross')).toBe('equite kazaross');
        expect(fold('Ёлка')).toBe('елка');
        expect(fold('Équité').length).toBe('Équité'.length);
    });
});

describe('fuzzyMatch', () => {
    test('finds the characters in order, not side by side', () => {
        expect(fuzzyMatch('trsc', 'Transcription')).not.toBeNull();
        expect(fuzzyMatch('csrt', 'Transcription')).toBeNull();
    });

    test('ignores case and accents', () => {
        expect(fuzzyMatch('equite', "Table d'équité de match")).not.toBeNull();
        expect(fuzzyMatch('ÉQUITÉ', "table d'equite")).not.toBeNull();
    });

    test('an empty query matches everything with a neutral score', () => {
        expect(fuzzyMatch('', 'anything')).toEqual({ score: 0, indices: [] });
    });

    test('a substring beats a scattered match, a word start beats a middle', () => {
        const substring = fuzzyMatch('stat', 'Statistiques');
        const scattered = fuzzyMatch('stat', 'Supprimer toute la table');
        expect(substring.score).toBeGreaterThan(scattered.score);
        const atStart = fuzzyMatch('mat', 'Matrice du videau');
        const inside = fuzzyMatch('mat', 'Automatique');
        expect(atStart.score).toBeGreaterThan(inside.score);
    });

    test('never trades a match for a word start that loses the rest', () => {
        expect(fuzzyMatch('μτρκβ', 'Μήτρα του κύβου')).not.toBeNull();
    });

    test('prefers word starts for the scattered characters', () => {
        const m = fuzzyMatch('bd', 'Nouvelle base de données');
        expect(m.indices).toEqual([9, 14]);
    });
});

describe('highlightSegments', () => {
    test('splits the text into matched and unmatched runs', () => {
        expect(highlightSegments('Anki', [0, 1])).toEqual([
            { text: 'An', hit: true },
            { text: 'ki', hit: false }
        ]);
    });
});
