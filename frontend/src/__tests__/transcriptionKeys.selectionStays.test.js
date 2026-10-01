/**
 * transcriptionKeys.selectionStays.test.js — `j`/`k` (et bas/haut) sont des
 * touches de la liste des candidats : sans candidat à déplacer elles restent
 * dans la transcription, au lieu de remonter au répartiteur global qui
 * parcourrait la bibliothèque.
 */

import { describe, test, expect } from 'vitest';
import { initialKeyState, pressKey } from '../services/transcriptionKeys.js';

function key(/** @type {string} */ code) {
    const letter = /^Key([A-Z])$/.exec(code);
    return new KeyboardEvent('keydown', { code, key: letter ? letter[1].toLowerCase() : code });
}

describe('j/k sans candidat à déplacer', () => {
    test.each(['KeyJ', 'KeyK', 'ArrowDown', 'ArrowUp'])('%s est prise quand aucun jet n’est tapé', (code) => {
        const result = pressKey(initialKeyState(), key(code), { expects: 'checker' });
        expect(result.handled).toBe(true);
        expect(result.commands).toEqual([]);
    });

    test.each(['KeyJ', 'KeyK'])('%s est prise quand le moteur attend une réponse à un double', (code) => {
        const result = pressKey(initialKeyState(), key(code), { expects: 'take' });
        expect(result.handled).toBe(true);
        expect(result.commands).toEqual([]);
    });
});
