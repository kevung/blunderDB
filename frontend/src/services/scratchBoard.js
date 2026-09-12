/**
 * scratchBoard.js — writing a scratch board to the library (#400).
 *
 * A scratch board is a board the user composes rather than reads from the
 * library (CONTEXT.md, *Scratch board*). What stands on it has no identity,
 * and nothing known about the position it was copied from travels with it:
 * saving it is individually importing a Position (ADR-0001, ADR-0002), and the
 * board is still a scratch board afterwards.
 *
 * Three consequences shape saveScratchBoard():
 *
 *  - The analysis starts from emptyAnalysis() with only the board's own xgid.
 *    analysisStore is never read: while a scratch mode is on it still
 *    describes the position studied before (modeMachine.js header), and
 *    SaveAnalysis MERGES played moves, cube analyses and player names — the
 *    old save carried another position's into the new one.
 *  - A position already stored receives nothing but its provenance flag. The
 *    board has no analysis to add, and an empty one sent over a stored
 *    analysis blanks its players and engine and makes an empty cube block
 *    the primary one (measured against database.SaveAnalysis).
 *  - Nothing the user is looking at moves: not the mode, not the tab, not the
 *    index. The position joins the list behind the board only when that list
 *    is the whole library (modeMachine.joinLibraryBehindScratchBoard), so that
 *    leaving the panel finds exactly what was being studied.
 *
 * The modes it serves are listed in SCRATCH_SAVE_MODES: Search's EDIT board
 * and the Eval panel's EPC board (#399). Every gesture reaches this one
 * function — CTRL-S, `w`, the toolbar button and the Eval panel's own button —
 * and the mode machine knows the list behind both boards. Elsewhere (NORMAL,
 * MATCH, COLLECTION, TRANSCRIBE) the board is a record or a draft's, and the
 * save is refused.
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

/** The modes whose board saveScratchBoard() writes: Search (EDIT) and Eval (EPC). */
export const SCRATCH_SAVE_MODES = Object.freeze(['EDIT', 'EPC']);

/**
 * Write the board on screen as an individually imported Position.
 *
 * Says the outcome in the status bar (the refusal, or the position's number)
 * and returns it, so that a caller has nothing to say itself.
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
