/**
 * utils/keys.js — the one place letter shortcuts are matched.
 *
 * Before it existed, `event.key === 'j'` was rewritten by hand in five files,
 * without the `key.length === 1` guard keyboardService's local helper had: a
 * composition in progress (event.key 'Process') or a dead key ('Dead') is not
 * a letter and must match nothing. These tests pin the guard, the layout
 * independence (AZERTY produces the same event.key for the labelled key) and
 * the Shift / bare-letter distinction the shortcut sheet relies on
 * (j = next position, MAJ-J = previous view).
 */
import { describe, test, expect } from 'vitest';
import { isLetter, isShiftLetter, isBareLetter } from '../utils/keys.js';

function key(k, mods = {}) {
    return new KeyboardEvent('keydown', { key: k, ...mods });
}

describe('isLetter', () => {
    test('matches the letter in either case', () => {
        expect(isLetter(key('j'), 'j')).toBe(true);
        expect(isLetter(key('J'), 'j')).toBe(true);
        expect(isLetter(key('j'), 'J')).toBe(true);
        expect(isLetter(key('k'), 'j')).toBe(false);
    });

    test('is layout independent: AZERTY reports the labelled key, whatever the physical code', () => {
        // On AZERTY the key at the QWERTY "Q" position is labelled A and
        // produces 'a'; the shortcut must follow the label.
        const azertyA = key('a', { code: 'KeyQ' });
        expect(isLetter(azertyA, 'a')).toBe(true);
        expect(isLetter(azertyA, 'q')).toBe(false);
    });

    test('ignores an IME composition, a dead key and an unidentified key', () => {
        expect(isLetter(key('Process'), 'p')).toBe(false);
        expect(isLetter(key('Dead'), 'd')).toBe(false);
        expect(isLetter(key('Unidentified'), 'u')).toBe(false);
        expect(isLetter(key(''), 'j')).toBe(false);
    });

    test('does not look at modifiers: Ctrl-Shift-I still produces the letter', () => {
        expect(isLetter(key('I', { ctrlKey: true, shiftKey: true }), 'i')).toBe(true);
    });
});

describe('isShiftLetter', () => {
    test('needs the Shift modifier, not an upper-case character', () => {
        expect(isShiftLetter(key('J', { shiftKey: true }), 'j')).toBe(true);
        expect(isShiftLetter(key('j', { shiftKey: true }), 'j')).toBe(true);
        // CapsLock produces 'J' without Shift: that is a plain j.
        expect(isShiftLetter(key('J'), 'j')).toBe(false);
        expect(isShiftLetter(key('j'), 'j')).toBe(false);
    });

    test('rejects composition keys even with Shift held', () => {
        expect(isShiftLetter(key('Process', { shiftKey: true }), 'p')).toBe(false);
    });
});

describe('isBareLetter', () => {
    test('matches the letter with no modifier at all', () => {
        expect(isBareLetter(key('j'), 'j')).toBe(true);
        // CapsLock: upper-case character, no modifier — still the bare letter.
        expect(isBareLetter(key('J'), 'j')).toBe(true);
    });

    test('rejects every modifier', () => {
        expect(isBareLetter(key('j', { shiftKey: true }), 'j')).toBe(false);
        expect(isBareLetter(key('j', { ctrlKey: true }), 'j')).toBe(false);
        expect(isBareLetter(key('j', { metaKey: true }), 'j')).toBe(false);
        expect(isBareLetter(key('j', { altKey: true }), 'j')).toBe(false);
    });

    test('a bare letter and its Shift variant never both match', () => {
        for (const ev of [key('j'), key('J'), key('j', { shiftKey: true }), key('J', { shiftKey: true })]) {
            expect(isBareLetter(ev, 'j') && isShiftLetter(ev, 'j')).toBe(false);
        }
    });

    test('ignores an IME composition and a dead key', () => {
        expect(isBareLetter(key('Process'), 'p')).toBe(false);
        expect(isBareLetter(key('Dead'), 'd')).toBe(false);
    });
});
