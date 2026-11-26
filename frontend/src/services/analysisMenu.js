// The context menu of a candidate moves table, shared by the Analysis and Eval panels: start or
// cancel a rollout, open the maps (replies to a play, the position at every score), and copy the board with the analysis — whole, as C-X C-X, and, when plays are
// selected, with those plays only.
import { get } from 'svelte/store';
import { t } from '../i18n';
import { rolloutStore, rolloutChoiceStore } from '../stores/rolloutStore.js';
import { toggleRollout, cancelRollout } from './rolloutService.js';
import { copyBoardWithAnalysisImage } from './clipboardService.js';
import { positionStore } from '../stores/positionStore.js';
import { openRollReplies, openEvalView } from './mapService.js';
import { EVAL_VIEW } from '../stores/mapStore.js';
import { positionAfterPlay, candidateMarksHits } from '../utils/moveNotation.js';

/**
 * @param {string[]} moves the plays selected; none for the whole position or a cube decision
 * @param {{ unsaved?: boolean }} [options] passed to toggleRollout
 * @param {Array<{ move?: string, analysisEngine?: string }>} [candidates] the table's plays, which
 *   say in which dialect a play is written; without them, blunderDB's own, hits starred
 * @returns {{ label: string, shortcut?: string, disabled?: boolean, onClick: () => void }[]}
 */
export function analysisMenuItems(moves, options = {}, candidates = []) {
    const tr = get(t);
    const choice = get(rolloutChoiceStore).preset;
    const preset = tr(`rollout.${choice === 'custom' || choice === 'fast' ? choice : 'standard'}`);
    /** @type {{ label: string, shortcut?: string, disabled?: boolean, onClick: () => void }[]} */
    const items = get(rolloutStore).running
        ? [{ label: tr('rollout.menuCancel'), shortcut: 'Shift+R', onClick: cancelRollout }]
        : [
              {
                  label: moves.length > 1 ? tr('rollout.menuStartMoves', { preset, n: moves.length }) : tr('rollout.menuStart', { preset }),
                  shortcut: 'Shift+R',
                  onClick: () => toggleRollout(moves, options)
              }
          ];
    const position = get(positionStore);
    const rolled = (position?.dice?.[0] ?? 0) > 0 && (position?.dice?.[1] ?? 0) > 0;
    // A notation that may leave several boards has no single position to answer: the entry stays
    // but cannot be chosen.
    if (moves.length === 1 && rolled) {
        const engine = candidates.find((c) => c?.move === moves[0])?.analysisEngine;
        const marks = engine === undefined ? true : (candidateMarksHits(candidates)[engine] ?? true);
        items.push({
            label: tr('maps.menu.replies'),
            disabled: !positionAfterPlay(position, position.dice[0], position.dice[1], moves[0], marks),
            onClick: () => openRollReplies(position, moves[0], marks)
        });
    }
    items.push({ label: tr('maps.menu.scores'), onClick: () => openEvalView(EVAL_VIEW.SCORES) });
    items.push({ label: tr('analysis.menuCopy'), shortcut: 'C-X C-X', onClick: () => copyBoardWithAnalysisImage({ moves: [] }) });
    if (moves.length) items.push({ label: tr('analysis.menuCopySelected'), onClick: () => copyBoardWithAnalysisImage({ moves }) });
    return items;
}
