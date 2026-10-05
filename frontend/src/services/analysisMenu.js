// The context menu of a candidate moves table, shared by the Analysis and Eval panels: start or
// cancel a rollout, and copy the board with the analysis — whole, as C-X C-X, and, when plays are
// selected, with those plays only.
import { get } from 'svelte/store';
import { t } from '../i18n';
import { rolloutStore, rolloutChoiceStore } from '../stores/rolloutStore.js';
import { toggleRollout, cancelRollout } from './rolloutService.js';
import { copyBoardWithAnalysisImage } from './clipboardService.js';

/**
 * @param {string[]} moves the plays selected; none for the whole position or a cube decision
 * @param {{ unsaved?: boolean }} [options] passed to toggleRollout
 * @returns {{ label: string, shortcut?: string, onClick: () => void }[]}
 */
export function analysisMenuItems(moves, options = {}) {
    const tr = get(t);
    const choice = get(rolloutChoiceStore).preset;
    const preset = tr(`rollout.${choice === 'custom' || choice === 'fast' ? choice : 'standard'}`);
    /** @type {{ label: string, shortcut?: string, onClick: () => void }[]} */
    const items = get(rolloutStore).running
        ? [{ label: tr('rollout.menuCancel'), shortcut: 'R', onClick: cancelRollout }]
        : [
              {
                  label: moves.length > 1 ? tr('rollout.menuStartMoves', { preset, n: moves.length }) : tr('rollout.menuStart', { preset }),
                  shortcut: 'R',
                  onClick: () => toggleRollout(moves, options)
              }
          ];
    items.push({ label: tr('analysis.menuCopy'), shortcut: 'C-X C-X', onClick: () => copyBoardWithAnalysisImage({ moves: [] }) });
    if (moves.length) items.push({ label: tr('analysis.menuCopySelected'), onClick: () => copyBoardWithAnalysisImage({ moves }) });
    return items;
}
