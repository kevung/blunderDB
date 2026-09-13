// A panel's deferred self-focus must never take the keyboard away from a field.
//
// Three panels give themselves keyboard focus a moment AFTER they appear —
// AnalysisPanel on the next macrotask after mounting, MatchPanel and
// TournamentPanel 100 ms after becoming visible — so that j/k and the panel's
// own shortcuts work without a click. The delay is the problem: by the time
// the callback runs the user may already have put the caret somewhere, and a
// bare `panel.focus()` took it from them.
//
// Both ways it went wrong were seen in CI (a slow runner stretches the delay
// past the next gesture):
//   - click « New tournament… », type a name, press Enter: the panel had
//     taken focus in between, Enter reached the panel, and no tournament was
//     created;
//   - Space opens the command line, whose input closes itself on blur: the
//     analysis panel mounting just after (a match review switches to it once
//     the move's analysis has loaded) blurred it, and the command line
//     vanished under the user's fingers.
//
// So the deferred focus yields to a text-entry field that already holds the
// focus — the same "typing" test keyboardService applies before it lets a
// shortcut through. Anything else (the body, a tab button just clicked, a
// table row) is still replaced by the panel, as before.

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
