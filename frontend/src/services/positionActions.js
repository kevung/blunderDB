/**
 * positionActions.js — the handlers of actionCatalog.js's position actions: each one is the
 * function the toolbar, a shortcut or a command already runs, on the position on the board. The
 * board's menu and a position row's menu (the row is displayed first) both build from here.
 */

import { get } from 'svelte/store';
import { tick } from 'svelte';
import { positionStore } from '../stores/positionStore';
import { statusBarModeStore, activeTabStore, commandTextStore, showCommandInputStore } from '../stores/uiStore';
import { databasePathStore } from '../stores/databaseStore';
import { ankiDecksStore } from '../stores/ankiStore.js';
import { analysisChallengeStore, toggleAnalysisChallenge } from '../stores/analysisChallengeStore.js';
import { viewStore } from '../stores/viewStore.js';
import { sendPositionToEval, evaluateInNewView, loadPositionsByFilters, saveCurrentPosition, deletePosition } from './positionService.js';
import { copyBoardImage, copyBoardWithAnalysisImage, copyPosition, copyXGID, exportBoardImage } from './clipboardService.js';
import { rankNeighboursOfCurrentPosition } from './rankService.js';
import { runRolloutCommand } from './rolloutService.js';
import { setPositionStudied, knownStudied, refreshStudied } from './studyQueueService.js';
import { toggleCommentPanel } from './tabToggles.js';
import { setStatusBarMessage } from './databaseService.js';
import * as anki from './ankiService.js';
import { mirrorPosition } from '../utils/boardGeometry.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

/**
 * @param {{ id: number, name: string }} deck
 * @param {number} positionId
 */
export async function addPositionToAnkiDeck(deck, positionId) {
    try {
        await anki.addPositionToDeck(deck.id, positionId);
        setStatusBarMessage(tMsg('status.positionAddedToDeck', { deck: deck.name }));
    } catch (err) {
        logger.error('Failed to add the position to the Anki deck:', err);
        setStatusBarMessage(tMsg('status.failedAddToDeck', { err }));
    }
}

/** Opens the comment of the position, its field focused. */
async function commentPosition() {
    if (get(activeTabStore) !== 'comments') toggleCommentPanel();
    await tick();
    document.getElementById('commentTextArea')?.focus();
}

// The studied mark is read as soon as a position is on the board, so the menu, built
// synchronously, offers the one of the two actions that applies.
positionStore.subscribe((position) => {
    if (position?.id) refreshStudied(position.id);
});

/**
 * The context of the position actions: the position on the board, and the handlers that act on it.
 *
 * @param {{ displayPosition?: () => any, positionId?: number }} [options] displayPosition: the
 *   position as drawn (mirrored, a play in progress), for the evaluations; the stored one
 *   otherwise. positionId: the position a list row names, while the board is still loading it.
 * @returns {import('./actionCatalog.js').ActionContext}
 */
export function positionContext({ displayPosition = () => get(positionStore), positionId = get(positionStore)?.id ?? 0 } = {}) {
    return {
        mode: get(statusBarModeStore),
        positionId,
        dbOpen: !!get(databasePathStore),
        studiedMark: knownStudied(positionId),
        challengeOn: !!get(analysisChallengeStore),
        ankiDecks: get(ankiDecksStore),
        do: {
            evaluate: () => sendPositionToEval(displayPosition()),
            evaluateMirror: () => sendPositionToEval(mirrorPosition(displayPosition())),
            evaluateInNewView: () => evaluateInNewView(displayPosition()),
            copyImage: () => copyBoardImage(),
            copyXGID: () => copyXGID(),
            copyPositionText: () => copyPosition(),
            copyImageWithAnalysis: () => copyBoardWithAnalysisImage(),
            saveImageSVG: () => exportBoardImage('svg'),
            saveImagePNG: () => exportBoardImage('png'),
            comment: () => commentPosition(),
            // The command line opens on `#`: Enter adds the tags as the `#` command does.
            addTag: () => {
                commandTextStore.set('#');
                showCommandInputStore.set(true);
            },
            markStudied: () => setPositionStudied(positionId, true),
            unmarkStudied: () => setPositionStudied(positionId, false),
            neighbours: () => rankNeighboursOfCurrentPosition(),
            // A structure search on this board, as `s` on the search board asks it.
            sameStructure: () => loadPositionsByFilters({ queryBoard: get(positionStore), searchCommand: 's' }),
            newView: () => viewStore.addView(),
            rolloutFast: () => runRolloutCommand('fast'),
            rolloutStandard: () => runRolloutCommand('standard'),
            addToAnkiDeck: (deck) => addPositionToAnkiDeck(deck, positionId),
            challenge: () => toggleAnalysisChallenge(),
            save: () => saveCurrentPosition(),
            trash: () => deletePosition()
        }
    };
}
