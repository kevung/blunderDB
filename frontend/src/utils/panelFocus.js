// A panel's deferred self-focus (AnalysisPanel after mount, MatchPanel and TournamentPanel 100 ms
// after showing, so j/k work without a click) must never take the keyboard away from a field
// the user has focused in the meantime: Enter would reach the panel instead of a form, and the
// command line, which closes on blur, would vanish. So it yields to a text-entry field, the same
// "typing" test keyboardService applies; anything else is still replaced.

const TYPING_TARGET = 'input, textarea, select, [contenteditable]';

/**
 * Whether `el` is a field the user types into.
 *
 * @param {Element|null|undefined} el
 * @returns {boolean}
 */
export function isTypingTarget(el) {
    return !!el && typeof el.matches === 'function' && el.matches(TYPING_TARGET);
}

/**
 * Focus `panel`, unless the user is typing in a field right now.
 *
 * @param {HTMLElement|null|undefined} panel
 * @returns {boolean} whether the panel took the focus
 */
export function focusPanelUnlessTyping(panel) {
    if (!panel) return false;
    if (isTypingTarget(document.activeElement)) return false;
    panel.focus();
    return true;
}
