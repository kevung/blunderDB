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
 *  - In EVAL, the gammonNet verdict the panel shows for this very board
 *    (evalVerdictStore) is written after the position, by
 *    SaveDisplayedEvaluation: only where no evaluation stands yet. An
 *    existing analysis (imported, deeper, a rollout) is kept whole, and the
 *    status bar says so: a verdict on screen never replaces one silently.
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
import { emptyAnalysis, evalVerdictStore } from '../stores/analysisStore.js';
import { SaveDisplayedEvaluation } from '../../wailsjs/go/database/Database.js';
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
    const onScreen = get(positionStore);
    const position = { ...JSON.parse(JSON.stringify(onScreen)), id: 0 };
    // Read before any await: the verdict must describe the board being saved.
    const shown = get(evalVerdictStore);
    const verdict = shown && shown.signature === JSON.stringify(onScreen ?? null) ? shown : null;

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
    if (verdict) await saveVerdict(id, existed, verdict);

    if (!existed) await joinLibraryBehindScratchBoard(id);
    await maybeAutoAnalyzeAfterImport();
    return { id, existed };
}

/**
 * Write the panel's verdict onto the stored position and say what became of it.
 * @param {number} id
 * @param {boolean} existed
 * @param {{ metId: number, analysis: { analysisType: string } }} verdict the engine's output (evalVerdictStore)
 */
async function saveVerdict(id, existed, verdict) {
    let outcome;
    try {
        outcome = await SaveDisplayedEvaluation(id, /** @type {any} */ (verdict.analysis), verdict.metId);
    } catch (error) {
        logger.error('saveScratchBoard: saving the evaluation failed:', error);
        setStatusBarMessage(tMsg('status.scratchBoardEvalNotSaved', { id }));
        return;
    }
    if (outcome === 'written') {
        setStatusBarMessage(tMsg(existed ? 'status.scratchBoardAlreadyStoredEvalAdded' : 'status.scratchBoardSavedWithEval', { id }));
    } else if (outcome === 'metChanged') {
        setStatusBarMessage(tMsg('status.scratchBoardEvalMetChanged', { id }));
    } else if (existed) {
        setStatusBarMessage(tMsg('status.scratchBoardAlreadyStoredAnalysisKept', { id }));
    }
}
