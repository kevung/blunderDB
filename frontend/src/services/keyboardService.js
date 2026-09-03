import { get } from 'svelte/store';
import { isAnyModalOpen, showCommandInputStore, activeModal, MODAL, activeTabStore } from '../stores/uiStore.js';
import { ankiViewModeStore, ankiReviewActionStore, showAnkiAnswer } from '../stores/ankiStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { viewStore } from '../stores/viewStore.js';
import { isLetter, isShiftLetter, isBareLetter } from '../utils/keys.js';

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
    toggleCollectionPanelAction,
    toggleMatchPanel,
    toggleTournamentPanel,
    toggleStatsPanel,
    toggleSearchPanel,
    toggleEPCMode,
    togglePipcount,
    reloadAllPositions,
    loadRandomPosition,
    showDatesAndMetadata
} from './positionService.js';
import { importDatabase, importPosition, importFolder, pastePosition } from './importService.js';
import { exportDatabase } from './exportService.js';
import { copyPosition, copyBoardImage, copyBoardWithAnalysisImage } from './clipboardService.js';

let lastCtrlXTime = 0;

// Ctrl-combos the WebView implements on an editable field: clipboard, select
// all, undo/redo, and word-wise navigation or deletion. blunderDB binds some of
// the same combos to board actions (Ctrl-C copies the position, Ctrl-Delete
// would delete it), so while a field has focus these have to go to the field.
// Letters are matched by the character produced, like every other letter
// shortcut here, so the rule holds on AZERTY and QWERTZ too.
const TEXT_EDITING_LETTERS = new Set(['a', 'c', 'v', 'x', 'y', 'z']);
const TEXT_EDITING_KEYS = new Set(['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End', 'Backspace', 'Delete', 'Insert']);

function isTextEditingCombo(event) {
    if (event.key.length === 1) return TEXT_EDITING_LETTERS.has(event.key.toLowerCase());
    return TEXT_EDITING_KEYS.has(event.key);
}

const EDITABLE_FIELD_SELECTOR = 'input, textarea, [contenteditable]';

// Position-browsing keys some panels forward to the board instead of using
// for their own list navigation (see the allowNavKeys option below).
const NAVIGATION_KEYS = new Set(['j', 'k', 'h', 'l', 'ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'PageUp', 'PageDown']);

// The keys that browse positions on the board: bare h/j/k/l (Shift-J/K switch
// views instead — see the dispatch below), arrows and PageUp/PageDown. Docked
// panels that hold a selection keep these for their own list.
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
 * Docked panels (MatchPanel, TournamentPanel, CollectionPanel, …) install
 * their own document-level keydown handler for list navigation (j/k, Esc by
 * levels, Delete…) and must stop that handler from swallowing a fixed set of
 * keys the rest of the app always needs, no matter which panel has focus:
 * Ctrl/Meta combos, Space (opens the command line), '?' (opens help), and
 * any keystroke while the user is typing in an editable field.
 *
 * This centralizes that "always let it through" test — it used to be
 * reimplemented ad hoc per panel and had drifted (see fiche-09): some panels
 * forgot the editable-field check, others forgot Space/'?'.
 *
 * @param {KeyboardEvent} event
 * @param {{allowNavKeys?: boolean}} [options] - also let position-browsing
 *   keys (j/k/h/l/arrows/PageUp/PageDown) through, for panels that don't use
 *   them for their own in-panel navigation and forward them to the board.
 * @returns {boolean} true if the panel must return without handling the event.
 */
export function panelKeyGuard(event, { allowNavKeys = false } = {}) {
    if (event.ctrlKey || event.metaKey) return true;
    if (event.code === 'Space') return true;
    if (event.key === '?') return true;
    if (event.target instanceof Element && event.target.matches(EDITABLE_FIELD_SELECTOR)) return true;
    if (allowNavKeys && NAVIGATION_KEYS.has(event.key)) return true;
    return false;
}

// Bare Tab only opens the search panel (#204) while focus sits on the board:
// nothing inside `.scrollable-content` (App.svelte) is itself a focus target
// (Board.svelte sets no tabindex), so this is effectively "focus is on
// <body>, or on the board's container" — the default browsing state right
// after startup, or after a click on the board/background. Before this
// guard, plain Tab was hijacked everywhere, unconditionally: standard
// keyboard focus navigation (buttons, links, form fields outside the one
// SearchPanel field a document-level stopPropagation happened to protect —
// see its handleKeyDown) did not exist anywhere in the app. Once focus has
// genuinely moved to a real element, Tab must behave like it does in any
// other application.
function isFocusOnBoard() {
    const active = document.activeElement;
    if (!active || active === document.body) return true;
    return !!active.closest('.scrollable-content');
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

// A toggle, not a plain "show" (#202, raccourcis.rst's Ctrl-F "Afficher/
// cacher"): pressing it again while the search tab is already active
// switches back to whatever was showing before, via toggleSearchPanel /
// TAB_TOGGLES.search in positionService.js.
export const focusSearchTab = toggleSearchPanel;

export function handleKeyDown(event) {
    event.stopPropagation();

    // Match a letter shortcut by the character produced (event.key), not the
    // physical key position (event.code). This keeps letter shortcuts on the
    // labeled key across keyboard layouts (AZERTY, QWERTZ, Dvorak, …) instead of
    // mapping to the US-QWERTY physical position. Non-letter keys (Space, Tab,
    // Delete, arrows, digits) stay positional below. `letter` is the shared
    // helper from utils/keys.js, bound to this event for the Ctrl-combos.
    const letter = (ch) => isLetter(event, ch);

    if (get(isAnyModalOpen)) return;

    // During Anki review on the Anki tab, route review keys
    if (get(ankiViewModeStore) === 'review' && !event.ctrlKey && get(activeTabStore) === 'anki') {
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
            // Show the answer (ADR-0025 rule 3). Space is free in this branch —
            // it opens the command line everywhere else, and this guard returns
            // before that. It deliberately gets no second meaning once the
            // answer is shown, unlike real Anki where Space then grades "Good":
            // a double tap would enter a grade the user never meant, and a
            // false grade durably pollutes the schedule.
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

    // Text editing wins over the board shortcuts. While focus sits in an
    // editable field, the clipboard/selection/undo combos belong to the field:
    // Ctrl-C there copies the selected text, not the position on the board.
    // Returning without preventDefault() is the point — the WebView performs its
    // own copy/cut/paste/select-all/undo, which the panel guard further down
    // used to suppress for every Ctrl combo it saw.
    if (inTextField && event.ctrlKey && !event.altKey && isTextEditingCombo(event)) {
        return;
    }

    // Allow normal typing in input fields
    if (inTextField && !event.ctrlKey && event.key !== 'Escape' && event.key !== 'Tab') {
        return;
    }

    // Comment panel: while focus is anywhere inside the panel, suppress single-key
    // shortcuts (navigation h/j/k/l, p, space, …) so they never conflict with
    // typing or editing comments. Ctrl-combos, Escape (blur) and Tab still pass.
    if (document.activeElement.closest('.comment-panel') && !event.ctrlKey && event.key !== 'Escape' && event.key !== 'Tab') {
        return;
    }

    // Analysis panel focus handling. The panel focuses itself when it opens, so
    // this branch is what the user hits first: it must let through the same
    // app-wide escape hatches every other panel guarantees via panelKeyGuard()
    // — Ctrl/Meta combos, Space (opens the command line) and '?' (opens help).
    if (document.activeElement.closest('.analysis-panel')) {
        if (event.ctrlKey || event.metaKey || event.key === 'Escape' || event.key === 'Tab' || event.code === 'Space' || event.key === '?') {
            // Let shortcut through
        } else {
            if (isBoardNavigationKey(event) && !get(selectedMoveStore)) {
                // No move selected - allow position navigation
            } else {
                return;
            }
        }
    }

    // Panel focus handling. There is no PANEL entry for the comment tab (see
    // uiStore.js's PANEL comment) — the active tab is the only live signal that
    // CommentPanel is the one TabbedPanel currently has mounted.
    const showComment = get(activeTabStore) === 'comments';
    if (document.activeElement.closest('.match-panel') || document.activeElement.closest('.collection-panel') || document.activeElement.closest('.tournament-panel') || showComment) {
        if (event.ctrlKey) {
            event.preventDefault();
        } else if (event.key === 'Escape' || event.key === 'Tab') {
            // Allow
        } else if (event.code === 'Space') {
            // Allow command line to open
        } else if (event.key === '?') {
            // Allow help modal to open
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
        event.preventDefault();
        event.stopPropagation();
        if (document.activeElement && document.activeElement.matches('input, textarea, [contenteditable]')) {
            /** @type {HTMLElement} */ (document.activeElement).blur();
        }
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
        // Ctrl-Tab is not a focus-navigation combo (unlike bare Tab below) and
        // is documented as "Afficher/cacher" like the other seven Ctrl+letter
        // panel toggles (raccourcis.rst) — toggleMatchPanel() is the same
        // toggle-back-if-already-showing behaviour #202 gave the other seven;
        // this one used to still call the pre-#202 plain `.set()`.
        event.preventDefault();
        toggleMatchPanel();
    } else if (!event.ctrlKey && event.code === 'Tab' && isFocusOnBoard()) {
        event.preventDefault();
        activeTabStore.set('search');
    } else if (!event.ctrlKey && event.code === 'Space') {
        event.preventDefault();
        showCommandInputStore.set(true);
    } else if (event.ctrlKey && letter('l')) {
        event.preventDefault();
        if (showComment) toggleCommentPanel();
        toggleAnalysisPanel();
    } else if (event.ctrlKey && letter('p')) {
        event.preventDefault();
        toggleCommentPanel();
    } else if (event.ctrlKey && letter('f')) {
        focusSearchTab();
    } else if (!event.ctrlKey && event.key === '?') {
        toggleHelpModal();
    } else if (event.ctrlKey && letter('m')) {
        toggleMetadataPanel();
    } else if (event.ctrlKey && letter('k')) {
        toggleAnkiPanel();
    } else if (event.ctrlKey && letter('t')) {
        event.preventDefault();
        viewStore.addView();
    } else if (event.ctrlKey && letter('w')) {
        event.preventDefault();
        viewStore.closeView(get(viewStore.activeViewId));
    } else if (event.ctrlKey && letter('y')) {
        event.preventDefault();
        toggleTournamentPanel();
    } else if (event.ctrlKey && letter('d')) {
        event.preventDefault();
        toggleStatsPanel();
    } else if (event.ctrlKey && letter('e')) {
        event.preventDefault();
        toggleEPCMode();
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
