// A reviewed match for the e2e suite: six decisions, their losses and the
// study summary the backend serves for them (storage.MatchReview).
import { matchMovePositions } from './fixtures.js';

export const reviewMoves = matchMovePositions.map((mp, i) => ({ ...mp, move_id: 100 + i, player_on_roll: i % 2 }));

export const reviewLosses = reviewMoves.map((mp, i) => ({
    move_id: mp.move_id,
    game_number: mp.game_number,
    move_number: mp.move_number,
    player: i % 2,
    decision_type: 'checker',
    mwc_loss: i === 2 ? null : [0.012, 0, null, 0.034, 0.005, 0.021][i],
    difficulty: i === 2 ? null : [0.004, 0, null, 0.006, 0.004, 0.002][i],
    avoidable: i === 3
}));

const player = (/** @type {number} */ seat) => ({
    pr: seat === 0 ? 4.82 : 7.35,
    pr_interval: { available: true, low: seat === 0 ? 2.9 : 4.6, high: seat === 0 ? 6.7 : 10.1, units: 3 },
    decisions: 3,
    mwc7: {
        available: true,
        loss: seat === 0 ? 0.021 : 0.058,
        has_interval: true,
        low: 0.01,
        high: 0.09,
        elo: seat === 0 ? 1920 : 1780,
        elo_low: 1700,
        elo_high: 2000,
        elo_floored: false,
        matches: 1
    },
    mwc_loss: seat === 0 ? 0.017 : 0.055,
    to_review: reviewLosses
        .filter((d) => d.player === seat && d.mwc_loss)
        .map((d) => ({
            move_id: d.move_id,
            game_number: d.game_number,
            move_number: d.move_number,
            decision_type: 'checker',
            mwc_loss: d.mwc_loss,
            difficulty: d.difficulty,
            avoidable: d.avoidable,
            avoidable_loss: d.avoidable ? d.mwc_loss : 0
        })),
    pace: { hasty: seat, deliberate: 1, unknown: 1 - seat, hasty_loss: seat ? 0.021 : 0, deliberate_loss: 0.012 },
    luck:
        seat === 0
            ? { available: true, result: 1, luck: 0.31, adjusted: 0.52, error_balance: 0.04, rolls_measured: 3, rolls: 3 }
            : { available: true, result: -1, luck: -0.31, adjusted: -0.52, error_balance: -0.04, rolls_measured: 3, rolls: 3 },
    difficulty: {
        decisions: 3,
        loss: seat === 0 ? 0.017 : 0.055,
        difficulty: seat === 0 ? 0.008 : 0.008,
        excess: seat === 0 ? 0.009 : 0.047,
        ratio: seat === 0 ? 2.1 : 6.9,
        avoidable: seat === 0 ? 0 : 1
    }
});

export const reviewSummary = { match_id: 7, players: [player(0), player(1)] };
