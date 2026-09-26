/**
 * scratchBoard.js — writing a scratch board (CONTEXT.md) to the library. The
 * board has no identity: saving it individually imports a Position (ADR-0001,
 * ADR-0002), and it stays a scratch board afterwards. Hence:
 *
 *  - The analysis is emptyAnalysis() plus the board's xgid. analysisStore is
 *    never read: it describes the position studied before, and SaveAnalysis
 *    would merge its moves, cube analyses and players into this one.
 *  - A position already stored receives only its provenance flag: an empty
 *    analysis would blank its players and engine and promote an empty cube
 *    block.
 *  - Mode, tab and index do not move; the position joins the list behind the
 *    board only when that list is the whole library
 *    (modeMachine.joinLibraryBehindScratchBoard).
 *
 * SCRATCH_SAVE_MODES: EDIT and EVAL. Every gesture (CTRL-S, `w`, toolbar, Eval
 * button) comes here; in other modes the board is a record or a draft's, and
 * the save is refused.
 */

import { get } from 'svelte/store';
import { databasePathStore } from '../stores/databaseStore.js';
import { positionStore } from '../stores/positionStore.js';
import { emptyAnalysis } from '../stores/analysisStore.js';
import { statusBarModeStore } from '../stores/uiStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { generateXGID } from './xgid.js';
import { positionRefusal } from './positionRefusal.js';
import { savePositionAndAnalysis, maybeAutoAnalyzeAfterImport } from './importService.js';
import { joinLibraryBehindScratchBoard } from './modeMachine.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

/** The modes whose board saveScratchBoard() writes: Search (EDIT) and Eval (EVAL). */
export const SCRATCH_SAVE_MODES = Object.freeze(['EDIT', 'EVAL']);

/**
 * Write the board on screen as an individually imported Position, and report
 * the outcome in the status bar.
 *
 * @returns {Promise<{ id: number, existed: boolean } | null>} the stored
 *   position, or null when nothing was written (no database, a mode without a
 *   scratch board, a refused board, a backend error).
 */
export async function saveScratchBoard() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return null;
    }
    if (!SCRATCH_SAVE_MODES.includes(get(statusBarModeStore))) {
        setStatusBarMessage(tMsg('status.saveOnlyEdit'));
        return null;
    }

    // A copy: the write path stamps the new id on the object it is given, and
    // the board must not become a library record. The id the board may carry
    // (EDIT blanks the studied position in place) is not its own either.
    const position = { ...JSON.parse(JSON.stringify(get(positionStore))), id: 0 };

    const refusal = positionRefusal(position);
    if (refusal) {
        setStatusBarMessage(tMsg(refusal));
        return null;
    }

    const analysis = { ...emptyAnalysis(), xgid: generateXGID(position) };

    let existed = false;
    const id = await savePositionAndAnalysis(
        position,
        analysis,
        (saved) => {
            existed = saved.existed;
            return tMsg(saved.existed ? 'status.scratchBoardAlreadyStored' : 'status.scratchBoardSaved', { id: saved.id });
        },
        { reload: false, mergeIntoExisting: false }
    );
    if (id == null) return null;
    logger.log('saveScratchBoard:', id, existed ? '(already stored)' : '(new)');

    if (!existed) await joinLibraryBehindScratchBoard(id);
    await maybeAutoAnalyzeAfterImport();
    return { id, existed };
}
