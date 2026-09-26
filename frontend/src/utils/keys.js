// Letter shortcuts match the character produced (event.key), never the physical position, so "j"
// stays on the key labelled J across AZERTY, QWERTZ…; digits stay positional (event.code), see
// keyboardService.js. The helpers guard `key.length === 1` ('Process' during IME, 'Dead',
// 'Unidentified') and fold case: use them rather than hand-written comparisons.

/**
 * The key produced the letter `ch`, in either case. Modifiers are not inspected.
 *
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
 * The letter `ch` with Shift held ("MAJ-J"). Tested on the modifier, not the case, so CapsLock
 * does not turn "j" into a view switch.
 *
 * @param {KeyboardEvent} event
 * @param {string} ch
 * @returns {boolean}
 */
export function isShiftLetter(event, ch) {
    return event.shiftKey === true && isLetter(event, ch);
}

/**
 * The letter `ch` alone, no Ctrl, Meta, Alt or Shift: the single-letter shortcuts.
 *
 * @param {KeyboardEvent} event
 * @param {string} ch
 * @returns {boolean}
 */
export function isBareLetter(event, ch) {
    return !event.ctrlKey && !event.metaKey && !event.altKey && !event.shiftKey && isLetter(event, ch);
}
