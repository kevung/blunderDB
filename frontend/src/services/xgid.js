// xgid.js — encoding a blunderDB Position as an XGID string.

// generateXGID re-encodes a Position as an XGID (field layout in
// pkg/blunderdb/domain/xgid.go; inverse: domain.DecodeXGID). A Position stores
// only away scores, so at a match score this rebuilds the smallest consistent
// match length ([2, 4] away → "4pt match, 2-0"), which round-trips to the same
// position (testdata/xgid_corpus.json). What must not be lost is a money
// game's Jacoby/Beaver: field 7 is the Crawford flag in a match but the
// Jacoby/Beaver bitmask (bit 0, bit 1) in a money game, chosen by game type.
import { pointsAway } from '../utils/awayScore.js';

export function generateXGID(position) {
    const { board, cube, dice, score, player_on_roll, decision_type, has_jacoby, has_beaver, max_cube } = position;

    let positionPart = '';
    for (let i = 0; i < 26; i++) {
        const point = board.points[i];
        if (point.checkers > 0) {
            const charCode = point.color === 0 ? 'A'.charCodeAt(0) : 'a'.charCodeAt(0);
            positionPart += String.fromCharCode(charCode + point.checkers - 1);
        } else {
            positionPart += '-';
        }
    }

    const cubeValue = cube.value;
    const cubeOwner = cube.owner === 0 ? 1 : cube.owner === 1 ? -1 : 0;
    const dicePart = decision_type === 1 ? '00' : dice.join('');
    const isMoneyGame = score[0] === -1 || score[1] === -1;
    // Distances, so the post-Crawford sentinel is decoded first: raw 0 would
    // emit a score equal to the match length, a match already won.
    const away1 = isMoneyGame ? 0 : pointsAway(score[0]);
    const away2 = isMoneyGame ? 0 : pointsAway(score[1]);
    // Field 7 reads the RAW score, not the distance: `1` is what says this is
    // the Crawford game, and `0` is precisely what says it is not.
    const isCrawford = !isMoneyGame && (score[0] === 1 || score[1] === 1) ? 1 : 0;
    // A 1-point match IS the Crawford game (DecodeXGID reads it so), so two
    // post-Crawford sentinels [0, 0] go out as a 2-point match at 1-1.
    const smallest = isMoneyGame ? 0 : Math.max(away1, away2);
    const matchLength = smallest === 1 && !isCrawford ? 2 : smallest;
    const actualScore1 = isMoneyGame ? 0 : matchLength - away1;
    const actualScore2 = isMoneyGame ? 0 : matchLength - away2;
    const field7 = isMoneyGame ? (has_jacoby ? 1 : 0) | (has_beaver ? 2 : 0) : isCrawford;
    const playerOnRoll = player_on_roll === 0 ? 1 : -1;
    // Field 9 (max cube): carried back out unchanged — reported, never acted
    // on, but dropping it would rewrite the pasted ruleset. 0 = none stated.
    const maxCube = max_cube || 0;

    return `${positionPart}:${cubeValue}:${cubeOwner}:${playerOnRoll}:${dicePart}:${actualScore1}:${actualScore2}:${field7}:${matchLength}:${maxCube}`;
}
