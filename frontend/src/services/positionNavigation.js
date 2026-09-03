// positionNavigation.js — moving the "current position" cursor: first/previous/next/
// last, jumping to an id, and a random pick.
//
// Extracted from positionService.js (fiche D.10, #210): one of that module's six
// responsibilities. Re-exported from positionService.js so existing callers
// (keyboardService, commandProcessor, App.svelte) keep one import.
//
// Depends on showPosition, which stays in positionService.js (too many other callers
// — App.svelte, ankiService.js, modeMachine.js — to move alongside navigation alone):
// the two modules import from each other, the same pattern positionService.js already
// has with modeMachine.js. Safe here because nothing at either module's top level
// calls into the other — only the exported functions do, once both are fully loaded.
import { get } from 'svelte/store';
import { SaveLastVisitedPosition } from '../../wailsjs/go/database/Database.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { positionsStore, matchContextStore, lastVisitedMatchStore } from '../stores/positionStore.js';
import { currentPositionIndexStore, statusBarTextStore, statusBarModeStore, openModal, MODAL } from '../stores/uiStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';
import { showPosition } from './positionService.js';

function saveCurrentMatchPosition() {
    if (get(statusBarModeStore) === 'MATCH' && get(matchContextStore).isMatchMode) {
        const matchCtx = get(matchContextStore);
        const currentMovePos = matchCtx.movePositions[matchCtx.currentIndex];
        if (currentMovePos) {
            lastVisitedMatchStore.set({
                matchID: matchCtx.matchID,
                currentIndex: matchCtx.currentIndex,
                gameNumber: currentMovePos.game_number
            });
            SaveLastVisitedPosition(matchCtx.matchID, matchCtx.currentIndex).catch((e) => {
                logger.error('Error persisting last visited position:', e);
            });
        }
    }
}

export async function firstPosition() {
    if (get(statusBarModeStore) === 'EDIT') {
        setStatusBarMessage(tMsg('status.cannotBrowseEdit'));
        return;
    }
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }

    if (get(statusBarModeStore) === 'MATCH' && get(matchContextStore).isMatchMode) {
        const matchCtx = get(matchContextStore);
        const currentGameNumber = matchCtx.movePositions[matchCtx.currentIndex].game_number;

        let targetIndex = -1;
        for (let i = matchCtx.currentIndex - 1; i >= 0; i--) {
            if (matchCtx.movePositions[i].game_number < currentGameNumber) {
                targetIndex = i;
                break;
            }
        }

        if (targetIndex === -1) {
            targetIndex = 0;
        } else {
            const targetGameNumber = matchCtx.movePositions[targetIndex].game_number;
            for (let i = 0; i < matchCtx.movePositions.length; i++) {
                if (matchCtx.movePositions[i].game_number === targetGameNumber) {
                    targetIndex = i;
                    break;
                }
            }
        }

        matchContextStore.update((ctx) => ({ ...ctx, currentIndex: targetIndex }));
        const movePos = matchCtx.movePositions[targetIndex];
        await showPosition(movePos.position);
        statusBarTextStore.set(`${matchCtx.player1Name} vs ${matchCtx.player2Name}`);
        saveCurrentMatchPosition();
    } else {
        const positions = get(positionsStore);
        if (positions && positions.length > 0) {
            currentPositionIndexStore.set(0);
        }
    }
}

export async function previousPosition() {
    if (get(statusBarModeStore) === 'EDIT') {
        setStatusBarMessage(tMsg('status.cannotBrowseEdit'));
        return;
    }
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }

    if (get(statusBarModeStore) === 'MATCH' && get(matchContextStore).isMatchMode) {
        const matchCtx = get(matchContextStore);
        if (matchCtx.currentIndex > 0) {
            let newIndex = matchCtx.currentIndex - 1;
            while (newIndex >= 0) {
                const movePos = matchCtx.movePositions[newIndex];
                if (movePos.move_type === 'checker' || movePos.move_type === 'cube') break;
                newIndex--;
            }

            if (newIndex >= 0) {
                matchContextStore.update((ctx) => ({ ...ctx, currentIndex: newIndex }));
                const movePos = matchCtx.movePositions[newIndex];
                await showPosition(movePos.position);
                statusBarTextStore.set(`${matchCtx.player1Name} vs ${matchCtx.player2Name}`);
                saveCurrentMatchPosition();
            }
        }
    } else {
        const positions = get(positionsStore);
        if (positions && get(currentPositionIndexStore) > 0) {
            currentPositionIndexStore.set(get(currentPositionIndexStore) - 1);
        }
    }
}

export async function nextPosition() {
    if (get(statusBarModeStore) === 'EDIT') {
        setStatusBarMessage(tMsg('status.cannotBrowseEdit'));
        return;
    }
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }

    if (get(statusBarModeStore) === 'MATCH' && get(matchContextStore).isMatchMode) {
        const matchCtx = get(matchContextStore);
        if (matchCtx.currentIndex < matchCtx.movePositions.length - 1) {
            let newIndex = matchCtx.currentIndex + 1;
            while (newIndex < matchCtx.movePositions.length) {
                const movePos = matchCtx.movePositions[newIndex];
                if (movePos.move_type === 'checker' || movePos.move_type === 'cube') break;
                newIndex++;
            }

            if (newIndex < matchCtx.movePositions.length) {
                matchContextStore.update((ctx) => ({ ...ctx, currentIndex: newIndex }));
                const movePos = matchCtx.movePositions[newIndex];
                await showPosition(movePos.position);
                statusBarTextStore.set(`${matchCtx.player1Name} vs ${matchCtx.player2Name}`);
                saveCurrentMatchPosition();
            }
        }
    } else {
        const positions = get(positionsStore);
        if (positions && get(currentPositionIndexStore) < positions.length - 1) {
            currentPositionIndexStore.set(get(currentPositionIndexStore) + 1);
        }
    }
}

export async function lastPosition() {
    if (get(statusBarModeStore) === 'EDIT') {
        setStatusBarMessage(tMsg('status.cannotBrowseEdit'));
        return;
    }
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }

    if (get(statusBarModeStore) === 'MATCH' && get(matchContextStore).isMatchMode) {
        const matchCtx = get(matchContextStore);
        const currentGameNumber = matchCtx.movePositions[matchCtx.currentIndex].game_number;

        let targetIndex = -1;
        for (let i = matchCtx.currentIndex + 1; i < matchCtx.movePositions.length; i++) {
            if (matchCtx.movePositions[i].game_number > currentGameNumber) {
                targetIndex = i;
                break;
            }
        }

        if (targetIndex === -1) {
            const maxGameNumber = Math.max(...matchCtx.movePositions.map((p) => p.game_number));
            for (let i = 0; i < matchCtx.movePositions.length; i++) {
                if (matchCtx.movePositions[i].game_number === maxGameNumber) {
                    targetIndex = i;
                    break;
                }
            }
        }

        if (targetIndex !== -1) {
            matchContextStore.update((ctx) => ({ ...ctx, currentIndex: targetIndex }));
            const movePos = matchCtx.movePositions[targetIndex];
            await showPosition(movePos.position);
            statusBarTextStore.set(`${matchCtx.player1Name} vs ${matchCtx.player2Name}`);
            saveCurrentMatchPosition();
        }
    } else {
        const positions = get(positionsStore);
        if (positions && positions.length > 0) {
            currentPositionIndexStore.set(positions.length - 1);
        }
    }
}

export function gotoPosition() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }
    if (get(statusBarModeStore) === 'EDIT') {
        setStatusBarMessage(tMsg('status.cannotGoToEdit'));
        return;
    }
    openModal(MODAL.GO_TO_POSITION);
}

export function loadRandomPosition() {
    logger.log('loadRandomPosition');
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }
    const positions = get(positionsStore);
    if (positions && positions.length > 0) {
        let randomIndex = Math.floor(Math.random() * positions.length);
        while (randomIndex === get(currentPositionIndexStore)) {
            randomIndex = Math.floor(Math.random() * positions.length);
        }
        logger.log('Random position index:', randomIndex);
        currentPositionIndexStore.set(randomIndex);
    }
}
