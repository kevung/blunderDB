import { isOnBoard } from './boardArea.js';
import { get } from 'svelte/store';
import { rankNeighboursOfCurrentPosition } from './rankService.js';
import { isAnyModalOpen, showCommandInputStore, activeModal, MODAL, activeTabStore } from '../stores/uiStore.js';
import { ankiViewModeStore, ankiReviewActionStore, showAnkiAnswer } from '../stores/ankiStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { viewStore } from '../stores/viewStore.js';
import { isLetter, isShiftLetter, isBareLetter } from '../utils/keys.js';
import { directionOwnsKey } from './directionKeys.js';
import { trainingHoldsBoardStore } from '../stores/trainingTabStore.js';

import { newDatabase, openDatabase, exitApp } from './databaseService.js';
import {
    firstPosition,
    previousPosition,
    nextPosition,
    lastPosition,
    saveCurrentPosition,
    updatePosition,
    deletePosition,
    toggleAnalysisPanel,
    toggleCommentPanel,
    toggleMetadataPanel,
    toggleAnkiPanel,
    toggleTrainingPanel,
    toggleCollectionPanelAction,
    toggleMatchPanel,
    toggleTournamentPanel,
    toggleStatsPanel,
    toggleTranscriptionPanel,
    toggleSearchPanel,
    toggleEvalMode,
    togglePipcount,
    reloadAllPositions,
    leaveSubSearchResults,
    loadRandomPosition,
    showDatesAndMetadata
} from './positionService.js';
import { importDatabase, importPosition, importFolder, pastePosition } from './importService.js';
import { undoTranscription } from './transcriptionService.js';
import { exportDatabase } from './exportService.js';
import { copyPosition, copyBoardImage, copyBoardWithAnalysisImage } from './clipboardService.js';
import { runPinnedFilter } from './filterLibraryService.js';
import { toggleCommandPalette } from './commandPalette.js';

let lastCtrlXTime = 0;

// Ctrl-combos the WebView implements in an editable field (clipboard, select
// all, undo/redo, word navigation). Some are also board actions (Ctrl-C copies
// the position), so a focused field must win. Letters by event.key (AZERTY).
const TEXT_EDITING_LETTERS = new Set(['a', 'c', 'v', 'x', 'y', 'z']);
const TEXT_EDITING_KEYS = new Set(['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End', 'Backspace', 'Delete', 'Insert']);

/** @param {KeyboardEvent} event */
function isTextEditingCombo(event) {
    if (event.key.length === 1) return TEXT_EDITING_LETTERS.has(event.key.toLowerCase());
    return TEXT_EDITING_KEYS.has(event.key);
}

const EDITABLE_FIELD_SELECTOR = 'input, textarea, [contenteditable]';

// Position-browsing keys some panels forward to the board instead of using
// for their own list navigation (see the allowNavKeys option below).
const NAVIGATION_KEYS = new Set(['j', 'k', 'h', 'l', 'ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'PageUp', 'PageDown']);

// Position-browsing keys: bare h/j/k/l (Shift-J/K switch views), arrows,
// PageUp/PageDown. Panels holding a selection keep them for their own list.
function isBoardNavigationKey(event) {
    return (
        isBareLetter(event, 'j') ||
        isBareLetter(event, 'k') ||
        isBareLetter(event, 'h') ||
        isBareLetter(event, 'l') ||
        event.key === 'ArrowLeft' ||
        event.key === 'ArrowRight' ||
        event.key === 'PageUp' ||
        event.key === 'PageDown'
    );
}

/**
 * Called first by a docked panel's own keydown handler: whether it must let
 * the event through untouched — isAlwaysGlobal() keys, or any key typed in an
 * editable field.
 *
 * @param {KeyboardEvent} event
 * @param {{allowNavKeys?: boolean}} [options] - also let position-browsing
 *   keys (j/k/h/l/arrows/PageUp/PageDown) through, for panels that don't use
 *   them for their own in-panel navigation and forward them to the board.
 * @returns {boolean} true if the panel must return without handling the event.
 */
export function panelKeyGuard(event, { allowNavKeys = false } = {}) {
    if (isAlwaysGlobal(event)) return true;
    if (event.target instanceof Element && event.target.matches(EDITABLE_FIELD_SELECTOR)) return true;
    if (allowNavKeys && NAVIGATION_KEYS.has(event.key)) return true;
    return false;
}

/**
 * The keys no panel may ever swallow, stated once: both panelKeyGuard() and the
 * dispatcher's per-panel branches read it, so a global shortcut added here
 * works whichever panel is open.
 *
 * @param {KeyboardEvent} event
 * @returns {boolean}
 */
export function isAlwaysGlobal(event) {
    if (event.ctrlKey || event.metaKey) return true;
    if (event.code === 'Space') return true;
    if (event.key === '?') return true;
    // Switching views: SHIFT-J / SHIFT-K are the bare-Shift spelling of
    // Ctrl-PageUp / Ctrl-PageDown, and must reach the dispatcher from
    // anywhere, exactly as the Ctrl form does. No panel binds Shift+letter.
    if (isShiftLetter(event, 'j') || isShiftLetter(event, 'k')) return true;
    // Running a pinned filter: ALT-1 … ALT-9, one gesture from anywhere.
    if (pinnedFilterDigit(event)) return true;
    return false;
}

/**
 * ALT-1 … ALT-9 run the library's pinned filters. Positional (event.code): on
 * AZERTY the top row produces "&é\"'(". Keypad excluded: Alt+keypad types a
 * character code on Windows.
 *
 * @param {KeyboardEvent} event
 * @returns {number} the pin's rank, 1 to 9, or 0.
 */
export function pinnedFilterDigit(event) {
    if (!event.altKey || event.ctrlKey || event.metaKey || event.shiftKey) return 0;
    const m = /^Digit([1-9])$/.exec(event.code ?? '');
    return m ? Number(m[1]) : 0;
}

// Bare Tab opens the search panel only while focus is on the board (<body> or
// the board's container — Board.svelte sets no tabindex). Once focus is on a
// real element, Tab must do standard focus navigation.
function isFocusOnBoard() {
    const active = document.activeElement;
    if (!active || active === document.body) return true;
    return isOnBoard(active);
}

export function toggleHelpModal() {
    const wasOpen = get(activeModal) === MODAL.HELP;
    if (wasOpen) {
        activeModal.set(null);
        setTimeout(() => {
            if (get(activeModal) === MODAL.COMMAND) {
                const el = document.querySelector('.command-input');
                if (el) /** @type {HTMLElement} */ (el).focus();
            } else if (get(activeTabStore) === 'comments') {
                const el = document.getElementById('commentsTextArea');
                if (el) el.focus();
            }
        }, 0);
    } else {
        activeModal.set(MODAL.HELP);
    }
}

// A toggle (raccourcis.rst: Ctrl-F "Afficher/cacher"): pressed again on the
// search tab, it returns to what was showing before.
export const focusSearchTab = toggleSearchPanel;

/**
 * The global dispatcher, on window in the bubble phase (App.svelte).
 *
 * Escape goes to the first of these with something to close, and no further:
 *   1. an open modal (Modal.svelte stops it; this dispatcher returns);
 *   2. the last opened overlay registered with escapeService.closeOnEscape()
 *      (window CAPTURE listener, so before everything below);
 *   3. a panel's own tiers — it claims the press with preventDefault() or stops
 *      it before window;
 *   4. here: leave the focused text field, else leave `ss` results.
 * A new overlay registers itself (step 2), never via `<svelte:window
 * onkeydown>`, which runs after this dispatcher.
 *
 * Never call event.stopPropagation() here: Svelte 5 skips every declarative
 * handler once cancelBubble is set, silencing each `<svelte:window onkeydown>`
 * mounted after App.
 *
 * @param {KeyboardEvent} event
 */
export function handleKeyDown(event) {
    // Letters by event.key, so shortcuts follow the key label on any layout;
    // non-letter keys stay positional (event.code).
    /** @param {string} ch */
    const letter = (ch) => isLetter(event, ch);

    if (get(isAnyModalOpen)) return;

    // Under the Direction page, bare J / K / ↓ / ↑ / Enter belong to the proposal queue
    // (services/directionKeys.js). The queue has claimed them already, or something open above it
    // keeps them — either way they browse nothing on the board the page hides.
    if (directionOwnsKey(event)) return;

    // A board-surface training question holds the board, revealed or not:
    // browsing would put another position under the answer. Focus is
    // irrelevant (the clicked button has left the DOM).
    if (isBoardNavigationKey(event) && get(trainingHoldsBoardStore)) return;

    // During Anki review on the Anki tab, route review keys
    if (get(ankiViewModeStore) === 'review' && !event.ctrlKey && get(activeTabStore) === 'anki') {
        // A review keeps the board: ALT-1 must neither grade the card nor
        // run a pinned filter under it.
        if (event.altKey) return;
        if (event.code === 'Digit1' || event.code === 'Numpad1') {
            event.preventDefault();
            ankiReviewActionStore.set(1);
        } else if (event.code === 'Digit2' || event.code === 'Numpad2') {
            event.preventDefault();
            ankiReviewActionStore.set(2);
        } else if (event.code === 'Digit3' || event.code === 'Numpad3') {
            event.preventDefault();
            ankiReviewActionStore.set(3);
        } else if (event.code === 'Digit4' || event.code === 'Numpad4') {
            event.preventDefault();
            ankiReviewActionStore.set(4);
        } else if (event.code === 'Space') {
            // Show the answer (ADR-0025 rule 3). Unlike Anki, Space gets no
            // second meaning once shown: a double tap would record an
            // unintended grade and pollute the schedule.
            event.preventDefault();
            showAnkiAnswer();
        } else if (event.code === 'Escape') {
            event.preventDefault();
            ankiReviewActionStore.set('back');
        } else if (letter('p')) {
            togglePipcount();
        }
        return;
    }

    const inTextField = document.activeElement.matches('input, textarea, [contenteditable]');

    // In an editable field the clipboard/selection/undo combos belong to the
    // field: return without preventDefault() so the WebView performs them.
    if (inTextField && event.ctrlKey && !event.altKey && isTextEditingCombo(event)) {
        return;
    }

    // Allow normal typing in input fields
    if (inTextField && !event.ctrlKey && event.key !== 'Escape' && event.key !== 'Tab') {
        return;
    }

    // Comment panel: suppress single-key shortcuts while focused inside it;
    // Ctrl-combos, Escape (blur) and Tab still pass.
    if (document.activeElement.closest('.comment-panel') && !isAlwaysGlobal(event) && event.key !== 'Escape' && event.key !== 'Tab') {
        return;
    }

    // The analysis panel focuses itself on open, so this branch must pass the
    // same isAlwaysGlobal() keys as panelKeyGuard().
    if (document.activeElement.closest('.analysis-panel')) {
        if (isAlwaysGlobal(event) || event.key === 'Escape' || event.key === 'Tab') {
            // Let shortcut through
        } else {
            if (isBoardNavigationKey(event) && !get(selectedMoveStore)) {
                // No move selected - allow position navigation
            } else {
                return;
            }
        }
    }

    // The comment tab has no PANEL entry: the active tab is the only signal
    // that CommentPanel is mounted.
    const showComment = get(activeTabStore) === 'comments';
    if (document.activeElement.closest('.match-panel') || document.activeElement.closest('.collection-panel') || document.activeElement.closest('.tournament-panel') || showComment) {
        if (event.ctrlKey) {
            event.preventDefault();
        } else if (event.key === 'Escape' || event.key === 'Tab' || isAlwaysGlobal(event)) {
            // Allow: Escape, Tab, and every always-global shortcut (Space opens
            // the command line, '?' the help, SHIFT-J/K switch views).
        } else {
            if (isBoardNavigationKey(event)) {
                const matchPanelHasSelection = document.querySelector('.match-panel tr.selected');
                if (matchPanelHasSelection) return;
            } else {
                return;
            }
        }
    }

    // Key dispatch
    if (event.key === 'Escape') {
        // Steps 3 and 4 above. Read before claiming the event: a panel with
        // something to close has already called preventDefault().
        const claimedByPanel = event.defaultPrevented;
        event.preventDefault();
        if (document.activeElement && document.activeElement.matches('input, textarea, [contenteditable]')) {
            /** @type {HTMLElement} */ (document.activeElement).blur();
        } else if (!claimedByPanel) {
            // Nothing claimed the Escape: leave `ss` results for their
            // collection or match.
            leaveSubSearchResults();
        }
    } else if (pinnedFilterDigit(event)) {
        event.preventDefault();
        runPinnedFilter(pinnedFilterDigit(event));
    } else if (event.ctrlKey && letter('n')) {
        newDatabase();
    } else if (event.ctrlKey && letter('o')) {
        openDatabase();
    } else if (event.ctrlKey && letter('q')) {
        exitApp();
    } else if (event.ctrlKey && event.shiftKey && letter('i')) {
        importDatabase();
    } else if (event.ctrlKey && event.shiftKey && letter('f')) {
        importFolder();
    } else if (event.ctrlKey && letter('i')) {
        importPosition();
    } else if (event.ctrlKey && letter('c')) {
        copyPosition();
    } else if (event.ctrlKey && letter('x')) {
        event.preventDefault();
        const now = Date.now();
        if (now - lastCtrlXTime < 500) {
            lastCtrlXTime = 0;
            copyBoardWithAnalysisImage();
        } else {
            lastCtrlXTime = now;
            copyBoardImage();
        }
    } else if (event.ctrlKey && letter('v')) {
        pastePosition();
    } else if (event.ctrlKey && event.shiftKey && letter('s')) {
        exportDatabase();
    } else if (event.ctrlKey && letter('s')) {
        saveCurrentPosition();
    } else if (event.ctrlKey && letter('u')) {
        updatePosition();
    } else if (event.code === 'Delete') {
        deletePosition();
    } else if (!event.ctrlKey && event.key === 'PageUp') {
        if (!showComment) {
            event.preventDefault();
            firstPosition();
        }
    } else if (isBareLetter(event, 'h')) {
        if (!showComment) firstPosition();
    } else if (!event.ctrlKey && event.key === 'ArrowLeft') {
        if (!showComment && !get(selectedMoveStore)) {
            event.preventDefault();
            previousPosition();
        }
    } else if (isBareLetter(event, 'k')) {
        if (!showComment && !get(selectedMoveStore)) previousPosition();
    } else if (!event.ctrlKey && event.key === 'ArrowRight') {
        if (!showComment && !get(selectedMoveStore)) {
            event.preventDefault();
            nextPosition();
        }
    } else if (isBareLetter(event, 'j')) {
        if (!showComment && !get(selectedMoveStore)) nextPosition();
    } else if (!event.ctrlKey && event.key === 'PageDown') {
        if (!showComment) {
            event.preventDefault();
            lastPosition();
        }
    } else if (isBareLetter(event, 'l')) {
        if (!showComment) lastPosition();
    } else if (event.ctrlKey && letter('b')) {
        event.preventDefault();
        toggleCollectionPanelAction();
    } else if (event.ctrlKey && letter('r')) {
        reloadAllPositions();
    } else if (event.ctrlKey && event.code === 'Tab') {
        // Ctrl-Tab is a panel toggle like the other Ctrl+letter ones
        // (raccourcis.rst), unlike bare Tab below.
        event.preventDefault();
        toggleMatchPanel();
    } else if (!event.ctrlKey && event.code === 'Tab' && isFocusOnBoard()) {
        event.preventDefault();
        activeTabStore.set('search');
    } else if (!event.ctrlKey && event.code === 'Space') {
        event.preventDefault();
        showCommandInputStore.set(true);
    } else if (event.ctrlKey && event.shiftKey && letter('l')) {
        // Ctrl+Maj+L : les voisines de la position courante (ADR-0043). Testé
        // AVANT Ctrl+L, que la branche suivante capte — sans quoi le panneau
        // d'analyse s'ouvrirait à la place.
        event.preventDefault();
        rankNeighboursOfCurrentPosition();
    } else if (event.ctrlKey && letter('l')) {
        event.preventDefault();
        if (showComment) toggleCommentPanel();
        toggleAnalysisPanel();
    } else if (event.ctrlKey && event.shiftKey && letter('p')) {
        // Ctrl+Maj+P : la palette (convention VS Code ; Ctrl+K est Anki ici).
        // Testé avant Ctrl+P, qui n'exclut pas Maj.
        event.preventDefault();
        toggleCommandPalette();
    } else if (event.ctrlKey && letter('p')) {
        event.preventDefault();
        toggleCommentPanel();
    } else if (event.ctrlKey && letter('f')) {
        focusSearchTab();
    } else if (!event.ctrlKey && event.key === '?') {
        toggleHelpModal();
    } else if (event.ctrlKey && letter('m')) {
        toggleMetadataPanel();
    } else if (event.ctrlKey && letter('j')) {
        event.preventDefault();
        toggleTrainingPanel();
    } else if (event.ctrlKey && letter('k')) {
        toggleAnkiPanel();
    } else if (event.ctrlKey && event.shiftKey && letter('z')) {
        // Before the Ctrl-Z branch, which does not exclude Shift.
        event.preventDefault();
        undoTranscription(true);
    } else if (event.ctrlKey && letter('z')) {
        // Transcription undo/redo (ux.md §3): a Ctrl combo is always global, so
        // it lives here and reaches the panel through a store. No-op without a
        // draft; in a text field the isTextEditingCombo guard returned first.
        event.preventDefault();
        undoTranscription(false);
    } else if (event.ctrlKey && event.shiftKey && letter('t')) {
        // Before the Ctrl-T branch, which does not exclude Shift (else a new
        // view would open).
        event.preventDefault();
        toggleTranscriptionPanel();
    } else if (event.ctrlKey && letter('t')) {
        event.preventDefault();
        viewStore.addView();
    } else if (event.ctrlKey && letter('w')) {
        event.preventDefault();
        viewStore.closeView(get(viewStore.activeViewId));
    } else if (event.ctrlKey && letter('y')) {
        event.preventDefault();
        toggleTournamentPanel();
    } else if (event.ctrlKey && !event.shiftKey && letter('d')) {
        // Sans MAJ seulement : CTRL-MAJ-D ne doit pas retomber sur Stats.
        event.preventDefault();
        toggleStatsPanel();
    } else if (event.ctrlKey && letter('e')) {
        event.preventDefault();
        toggleEvalMode();
    } else if (event.ctrlKey && letter('g')) {
        event.preventDefault();
        showDatesAndMetadata();
    } else if ((event.ctrlKey && event.key === 'PageUp') || (!event.ctrlKey && isShiftLetter(event, 'j'))) {
        event.preventDefault();
        viewStore.selectPreviousView();
    } else if ((event.ctrlKey && event.key === 'PageDown') || (!event.ctrlKey && isShiftLetter(event, 'k'))) {
        event.preventDefault();
        viewStore.selectNextView();
    } else if (!event.ctrlKey && letter('p')) {
        togglePipcount();
    } else if (!event.ctrlKey && letter('r')) {
        loadRandomPosition();
    }
}
