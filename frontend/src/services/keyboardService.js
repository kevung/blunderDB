import { tMsg } from '../i18n';
import { get } from 'svelte/store';
import { isAnyModalOpen, showCommandInputStore, activeModal, MODAL, openPanels, PANEL, activeTabStore } from '../stores/uiStore.js';
import { ankiViewModeStore, ankiReviewActionStore } from '../stores/ankiStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { viewStore } from '../stores/viewStore.js';

import { newDatabase, openDatabase, exitApp, setStatusBarMessage } from './databaseService.js';
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
    toggleTournamentPanel,
    toggleStatsPanel,
    toggleEPCMode,
    togglePipcount,
    loadAllPositions,
    loadRandomPosition
} from './positionService.js';
import { importDatabase, importPosition, importFolder, pastePosition } from './importService.js';
import { exportDatabase } from './exportService.js';
import { copyPosition, copyBoardImage, copyBoardWithAnalysisImage } from './clipboardService.js';

let lastCtrlXTime = 0;

export function toggleHelpModal() {
    const wasOpen = get(activeModal) === MODAL.HELP;
    if (wasOpen) {
        activeModal.set(null);
        setTimeout(() => {
            if (get(activeModal) === MODAL.COMMAND) {
                const el = document.querySelector('.command-input');
                if (el) /** @type {HTMLElement} */ (el).focus();
            } else if (get(openPanels).has(PANEL.COMMENT)) {
                const el = document.getElementById('commentsTextArea');
                if (el) el.focus();
            }
        }, 0);
    } else {
        activeModal.set(MODAL.HELP);
    }
}

export function toggleSearchHistoryPanel() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.searchHistoryRequiresDb'));
        return;
    }
    activeTabStore.set('search');
}

export function handleKeyDown(event) {
    event.stopPropagation();

    // Match a letter shortcut by the character produced (event.key), not the
    // physical key position (event.code). This keeps letter shortcuts on the
    // labeled key across keyboard layouts (AZERTY, QWERTZ, Dvorak, …) instead of
    // mapping to the US-QWERTY physical position. Non-letter keys (Space, Tab,
    // Delete, arrows, digits) stay positional below.
    const letter = (ch) => event.key.length === 1 && event.key.toLowerCase() === ch;

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
        } else if (event.code === 'Escape') {
            event.preventDefault();
            ankiReviewActionStore.set('back');
        } else if (letter('p')) {
            togglePipcount();
        }
        return;
    }

    // Allow normal typing in input fields
    if (document.activeElement.matches('input, textarea, [contenteditable]') && !event.ctrlKey && event.key !== 'Escape' && event.key !== 'Tab') {
        return;
    }

    // Comment panel: while focus is anywhere inside the panel, suppress single-key
    // shortcuts (navigation h/j/k/l, p, space, …) so they never conflict with
    // typing or editing comments. Ctrl-combos, Escape (blur) and Tab still pass.
    if (document.activeElement.closest('.comment-panel') && !event.ctrlKey && event.key !== 'Escape' && event.key !== 'Tab') {
        return;
    }

    // Analysis panel focus handling
    if (document.activeElement.closest('.analysis-panel')) {
        if (event.ctrlKey || event.key === 'Escape' || event.key === 'Tab') {
            // Let shortcut through
        } else {
            const isNavigationKey =
                event.key === 'j' ||
                event.key === 'k' ||
                event.key === 'ArrowLeft' ||
                event.key === 'ArrowRight' ||
                event.key === 'h' ||
                event.key === 'l' ||
                event.key === 'PageUp' ||
                event.key === 'PageDown';
            if (isNavigationKey && !get(selectedMoveStore)) {
                // No move selected - allow position navigation
            } else {
                return;
            }
        }
    }

    // Panel focus handling
    const showComment = get(openPanels).has(PANEL.COMMENT);
    if (
        document.activeElement.closest('.filter-library-panel') ||
        document.activeElement.closest('.search-history-panel') ||
        document.activeElement.closest('.match-panel') ||
        document.activeElement.closest('.collection-panel') ||
        document.activeElement.closest('.tournament-panel') ||
        showComment
    ) {
        if (event.ctrlKey) {
            event.preventDefault();
        } else if (event.key === 'Escape' || event.key === 'Tab') {
            // Allow
        } else if (event.code === 'Space') {
            // Allow command line to open
        } else if (event.key === '?') {
            // Allow help modal to open
        } else {
            const isNavigationKey =
                event.key === 'j' ||
                event.key === 'k' ||
                event.key === 'ArrowLeft' ||
                event.key === 'ArrowRight' ||
                event.key === 'h' ||
                event.key === 'l' ||
                event.key === 'PageUp' ||
                event.key === 'PageDown';
            if (isNavigationKey) {
                const filterLibraryHasSelection = document.querySelector('.filter-library-panel tr.highlight');
                const searchHistoryHasSelection = document.querySelector('.search-history-panel tr.selected');
                const matchPanelHasSelection = document.querySelector('.match-panel tr.selected');
                if (filterLibraryHasSelection || searchHistoryHasSelection || matchPanelHasSelection) return;
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
    } else if (!event.ctrlKey && event.key === 'h') {
        if (!showComment) firstPosition();
    } else if (!event.ctrlKey && event.key === 'ArrowLeft') {
        if (!showComment && !get(selectedMoveStore)) {
            event.preventDefault();
            previousPosition();
        }
    } else if (!event.ctrlKey && event.key === 'k') {
        if (!showComment && !get(selectedMoveStore)) previousPosition();
    } else if (!event.ctrlKey && event.key === 'ArrowRight') {
        if (!showComment && !get(selectedMoveStore)) {
            event.preventDefault();
            nextPosition();
        }
    } else if (!event.ctrlKey && event.key === 'j') {
        if (!showComment && !get(selectedMoveStore)) nextPosition();
    } else if (!event.ctrlKey && event.key === 'PageDown') {
        if (!showComment) {
            event.preventDefault();
            lastPosition();
        }
    } else if (!event.ctrlKey && event.key === 'l') {
        if (!showComment) lastPosition();
    } else if (event.ctrlKey && letter('b')) {
        event.preventDefault();
        toggleCollectionPanelAction();
    } else if (event.ctrlKey && letter('r')) {
        loadAllPositions();
    } else if (event.ctrlKey && event.code === 'Tab') {
        event.preventDefault();
        activeTabStore.set('matches');
    } else if (!event.ctrlKey && event.code === 'Tab') {
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
        toggleSearchHistoryPanel();
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
    } else if ((event.ctrlKey && event.key === 'PageUp') || (!event.ctrlKey && event.key === 'J')) {
        event.preventDefault();
        viewStore.selectPreviousView();
    } else if ((event.ctrlKey && event.key === 'PageDown') || (!event.ctrlKey && event.key === 'K')) {
        event.preventDefault();
        viewStore.selectNextView();
    } else if (!event.ctrlKey && letter('p')) {
        togglePipcount();
    } else if (!event.ctrlKey && letter('r')) {
        loadRandomPosition();
    }
}
