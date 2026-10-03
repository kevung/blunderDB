import { describe, test, expect } from 'vitest';
import fr from '../i18n/locales/fr.json';
import { chordLabel, toolbarHint } from '../utils/keyNames.js';
import { TOOLBAR_CHORDS, SHORTCUTS } from '../services/shortcutMap.js';

const t = (/** @type {string} */ key) => key.split('.').reduce((o, k) => o?.[k], /** @type {any} */ (fr)) ?? key;

describe('key names in the interface language', () => {
    test('a chord is spelled with the translated names', () => {
        expect(chordLabel('Ctrl+Shift+S', t)).toBe('Ctrl+Maj+S');
        expect(chordLabel('PageDown', t)).toBe('Page suiv.');
    });

    test('the toolbar hints name Gauche, Suppr and Page suiv., never the English key names', () => {
        expect(toolbarHint('previousPosition', t)).toBe('(Gauche, k)');
        expect(toolbarHint('deletePosition', t)).toBe('(Suppr)');
        expect(toolbarHint('lastPosition', t)).toBe('(Page suiv., l)');
        expect(toolbarHint('copyBoardImage', t, 2)).toBe('(Ctrl+X Ctrl+X)');
    });

    test('every key token of the toolbar has a name in fr.json', () => {
        const tokens = new Set(
            Object.values(TOOLBAR_CHORDS)
                .flat()
                .flatMap((c) => c.split('+'))
        );
        for (const token of tokens) {
            if (token.length === 1 || token === 'Ctrl') continue;
            expect(chordLabel(token, t), token).not.toBe(token);
        }
    });

    test('every chord a tooltip announces is wired in the global scope of SHORTCUTS', () => {
        // The map writes a letter shortcut in lower case and a shifted one in upper case.
        const wired = new Set(SHORTCUTS.global.keys);
        for (const [button, chords] of Object.entries(TOOLBAR_CHORDS)) {
            for (const chord of chords) {
                const tokens = chord.split('+');
                const last = tokens[tokens.length - 1];
                if (last.length === 1 && !tokens.includes('Shift')) tokens[tokens.length - 1] = last.toLowerCase();
                expect(wired.has(tokens.join('+')), `${button}: ${chord}`).toBe(true);
            }
        }
    });
});
