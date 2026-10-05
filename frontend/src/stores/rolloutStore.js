import { writable } from 'svelte/store';

/**
 * @typedef {{ move: string, equity: number, stdErr: number, ci95: number, games: number, jsd: number }} RolloutCandidateRow
 * @typedef {{ type: string, positionId?: number, stored?: boolean, message?: string, [key: string]: any }} RolloutOutcome
 * @typedef {{
 *   running: boolean, job: number, kind: string, positionId: number, games: number, maxGames: number,
 *   done: number, total: number, candidates: RolloutCandidateRow[], result: any, resultKey: string,
 *   pendingKey: string, outcome: RolloutOutcome | null, error: string, revision: number
 * }} RolloutState
 * @typedef {{ preset: string, custom: import('../../wailsjs/go/models.js').rollout.Settings | null }} RolloutChoiceState
 */

// What the rollout panel shows, fed by services/rolloutService.js from the
// rollout:* and rollout-batch:* events and from RolloutStatus() when the
// panel is mounted again (the job outlives the view).
/** @returns {RolloutState} */
export function idleRollout() {
    return {
        running: false,
        // The engine's number for the job in hand: an event of a lower one belongs to a job that
        // was replaced and is ignored.
        job: 0,
        // 'position' | 'batch' | '' while idle
        kind: '',
        positionId: 0,
        games: 0,
        maxGames: 0,
        done: 0,
        total: 0,
        // The candidates as they stand after the last batch of games, in the shape of a stored one.
        candidates: [],
        // The last finished rollout of a board that is not in the database (never stored), as a
        // stored rollout, and the board it describes.
        result: null,
        resultKey: '',
        pendingKey: '',
        // { type: 'done' | 'cancelled' | 'error' | 'batch-done' | 'batch-cancelled' | 'batch-error', ... }
        outcome: null,
        // Why the last start was refused ('' when it was not).
        error: '',
        // Bumped whenever something may have been written: readers reload what is stored.
        revision: 0
    };
}

export const rolloutStore = writable(/** @type {RolloutState} */ (idleRollout()));

// The setting chosen in the panel: 'fast' | 'standard' | 'custom'. The custom
// settings start from the preset last in use and are kept across mounts.
export const rolloutChoiceStore = writable(/** @type {RolloutChoiceState} */ ({ preset: 'standard', custom: null }));
