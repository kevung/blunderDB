// The name of a key as the user reads it, in the interface language. Tokens follow the chord
// notation of services/shortcutMap.js (KeyboardEvent names, "Ctrl" and "Shift" as modifiers); a
// plain letter or symbol is shown as is. The names live in the `keyNames` section of each locale.
import { TOOLBAR_CHORDS } from '../services/shortcutMap.js';

/** @type {Record<string, string>} */
const KEY_NAME_KEYS = {
    Ctrl: 'ctrl',
    Shift: 'shift',
    Alt: 'alt',
    ArrowLeft: 'left',
    ArrowRight: 'right',
    ArrowUp: 'up',
    ArrowDown: 'down',
    Delete: 'delete',
    PageUp: 'pageUp',
    PageDown: 'pageDown',
    Home: 'home',
    End: 'end',
    Enter: 'enter',
    Escape: 'escape',
    Space: 'space',
    Backspace: 'backspace',
    Tab: 'tab'
};

/**
 * @param {string} token
 * @param {(key: string) => string} t
 */
export function keyName(token, t) {
    const key = KEY_NAME_KEYS[token];
    return key ? t(`keyNames.${key}`) : token;
}

/**
 * "Ctrl+Shift+S" as "Ctrl+Maj+S" (fr).
 * @param {string} chord
 * @param {(key: string) => string} t
 */
export function chordLabel(chord, t) {
    return chord
        .split('+')
        .map((token) => keyName(token, t))
        .join('+');
}

/**
 * The parenthesised keys of a toolbar button's tooltip, "(Gauche, k)"; `repeat` presses the
 * chord that many times in a row ("Ctrl+X Ctrl+X").
 * @param {string} button a key of TOOLBAR_CHORDS
 * @param {(key: string) => string} t
 * @param {number} [repeat]
 */
export function toolbarHint(button, t, repeat = 1) {
    const chords = TOOLBAR_CHORDS[button] ?? [];
    const one = chords.map((c) => chordLabel(c, t));
    return `(${repeat > 1 ? Array(repeat).fill(one[0]).join(' ') : one.join(', ')})`;
}
