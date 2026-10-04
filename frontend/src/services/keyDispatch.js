/**
 * keyDispatch.js — the one place a keydown reaches the application from `window`.
 *
 * Every keyboard handler that is not bound to its own element registers here, under a scope
 * declared in shortcutMap.js, instead of calling `addEventListener('keydown')` on `window` or
 * `document`. A keydown then visits the scopes in a fixed order, the tier's, not the order in
 * which components happened to mount:
 *
 *   capture phase (before any element sees the key)
 *     OVERLAY  — Escape closes the last opened overlay (escapeService.js);
 *     CAPTURE  — the Direction page's queue and grid digits;
 *   bubble phase (after the focused element and the panels bound on their own nodes)
 *     PANEL    — docked panels that own keys while focused (search, transcription, matches,
 *                collections, tournaments);
 *     LOCAL    — the board, the view tabs, an open context menu, the Direction page's undo;
 *     GLOBAL   — keyboardService.handleKeyDown, the application shortcuts.
 *
 * Two native listeners, one per phase, both installed here: capture is needed for what must
 * beat a focused element's own handler, bubble for what must come after it.
 *
 * Rules held for every scope, so no handler has to restate them:
 *   - once a handler stops propagation (stopPropagation, stopImmediatePropagation) no later
 *     scope sees the key, as a listener further up the DOM would not;
 *   - while a modal is open (isAnyModalOpen: activeModal or the confirm dialog) no bubble-phase
 *     scope runs: the modal handles its keys on its own element;
 *   - the GLOBAL tier skips a key already claimed (defaultPrevented), except Escape, whose
 *     dispatcher branch reads the claim to decide between blurring a field and leaving results.
 *
 * No handler here calls stopPropagation itself: Svelte 5 skips every delegated handler once
 * cancelBubble is set, which is why a component never uses `<svelte:window onkeydown>` for a
 * shortcut and registers here instead.
 */
import { get } from 'svelte/store';
import { isAnyModalOpen } from '../stores/uiStore.js';
import { SHORTCUTS, TIER, CAPTURE_TIERS } from './shortcutMap.js';

/** @typedef {(event: KeyboardEvent) => void} KeyHandler */
/** @typedef {{ scope: string, tier: number, handler: KeyHandler, seq: number }} Entry */

/** @type {Entry[]} */
const entries = [];
let seq = 0;
let installed = false;

/**
 * @param {Entry} a
 * @param {Entry} b
 */
function byOrder(a, b) {
    return a.tier - b.tier || a.seq - b.seq;
}

/**
 * Runs the entries of `tiers` in order, stopping where the rules above say.
 *
 * @param {KeyboardEvent} event
 * @param {(tier: number) => boolean} inPhase
 */
function run(event, inPhase) {
    // A key pressed while an input method composes text (Enter confirming a conversion, in
    // Japanese) belongs to the composition, not to a shortcut.
    if (event.isComposing) return;
    const bubble = !inPhase(TIER.OVERLAY);
    if (bubble && get(isAnyModalOpen)) return;
    // A snapshot: a handler that closes something may unregister an entry mid-dispatch.
    for (const entry of entries.filter((e) => inPhase(e.tier))) {
        if (event.cancelBubble) return;
        if (entry.tier === TIER.GLOBAL && event.defaultPrevented && event.key !== 'Escape') return;
        entry.handler(event);
    }
}

/** @param {KeyboardEvent} event */
export function dispatchCapture(event) {
    run(event, (tier) => CAPTURE_TIERS.has(tier));
}

/** @param {KeyboardEvent} event */
export function dispatchBubble(event) {
    run(event, (tier) => !CAPTURE_TIERS.has(tier));
}

function install() {
    if (installed || typeof window === 'undefined') return;
    window.addEventListener('keydown', dispatchCapture, true);
    window.addEventListener('keydown', dispatchBubble);
    installed = true;
}

/**
 * Registers the handler of a declared scope.
 *
 * @param {string} scope - a key of SHORTCUTS (shortcutMap.js), which states its tier and keys.
 * @param {KeyHandler} handler
 * @returns {() => void} removes the registration (call it on unmount).
 */
export function registerKeys(scope, handler) {
    const declared = SHORTCUTS[scope];
    if (!declared) throw new Error(`keyDispatch: undeclared shortcut scope "${scope}"`);
    install();
    /** @type {Entry} */
    const entry = { scope, tier: declared.tier, handler, seq: seq++ };
    entries.push(entry);
    entries.sort(byOrder);
    return () => {
        const i = entries.indexOf(entry);
        if (i >= 0) entries.splice(i, 1);
    };
}

/**
 * The scopes registered right now, in dispatch order. For tests and diagnostics.
 *
 * @returns {string[]}
 */
export function registeredScopes() {
    return entries.map((e) => e.scope);
}
