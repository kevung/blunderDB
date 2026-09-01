// Letter shortcuts are matched by the character the key produced (event.key),
// never by its physical position (event.code): that keeps "j" on the key
// labelled J across AZERTY, QWERTZ, Dvorak… Digits and non-letter keys stay
// positional (event.code) — see keyboardService.js and the keyboard-layout
// convention it documents.
//
// Every helper below guards on `key.length === 1`. During an IME composition
// event.key is 'Process', a dead key reports 'Dead', an unknown key
// 'Unidentified': none of those is a letter, and a bare `event.key === 'j'`
// written by hand does not need the guard — but a case-insensitive comparison
// does, and hand-written copies drift (MatchPanel had 'j' but not 'J', the
// service had both). Match through these helpers instead.

/**
 * The key produced the letter `ch`, in either case. Modifiers are not
 * inspected: callers combine it with event.ctrlKey / event.shiftKey the same
 * way they would with any other key.
 *
 * @param {KeyboardEvent} event
 * @param {string} ch - one ASCII letter, any case
 * @returns {boolean}
 */
export function isLetter(event, ch) {
    const key = event.key;
    return typeof key === 'string' && key.length === 1 && key.toLowerCase() === ch.toLowerCase();
}

/**
 * The letter `ch` with Shift held — the "MAJ-J" / "MAJ-K" of the shortcut
 * sheet. Tested on the modifier, not on the case of the character produced,
 * so CapsLock does not turn "j" into a view switch.
 *
 * @param {KeyboardEvent} event
 * @param {string} ch
 * @returns {boolean}
 */
export function isShiftLetter(event, ch) {
    return event.shiftKey === true && isLetter(event, ch);
}

/**
 * The letter `ch` on its own: no Ctrl, Meta, Alt or Shift. This is what the
 * single-letter shortcuts (h/j/k/l navigation, p, r, …) mean, and it is what
 * keeps them apart from the Shift and Ctrl variants of the same letter.
 *
 * @param {KeyboardEvent} event
 * @param {string} ch
 * @returns {boolean}
 */
export function isBareLetter(event, ch) {
    return !event.ctrlKey && !event.metaKey && !event.altKey && !event.shiftKey && isLetter(event, ch);
}
