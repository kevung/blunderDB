// A panel's deferred self-focus (AnalysisPanel after mount, MatchPanel and TournamentPanel 100 ms
// after showing, so j/k work without a click) must never take the keyboard away from a field
// the user has focused in the meantime: Enter would reach the panel instead of a form, and the
// command line, which closes on blur, would vanish. So it yields to a text-entry field, the same
// "typing" test keyboardService applies. It also yields to a focus already inside the panel (a row
// just created and focused): the panel receives that row's keys by bubbling anyway. And it yields
// to a modal open over it (the help opened within the delay): a modal takes its keys on its own
// element, so a panel focused behind it would receive them instead. Anything else outside the
// panel is still replaced.

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
    const active = document.activeElement;
    if (isTypingTarget(active)) return false;
    if (active && active !== panel && panel.contains(active)) return false;
    const modal = document.querySelector('[aria-modal="true"]');
    if (modal && !modal.contains(panel)) return false;
    panel.focus();
    return true;
}
