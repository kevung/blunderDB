// xgid.js — encoding a blunderDB Position as an XGID string.
//
// Extracted from positionService.js (fiche D.10, #210): one of that module's six
// responsibilities, and self-contained (no store, no backend call). Fiche D.11 (#211)
// is where this encoding itself gets fixed (match length/Crawford/Jacoby losses); this
// split only moves the existing code, so it lands first and independently.

// generateXGID re-encodes a blunderDB Position as an XGID string (see
// pkg/blunderdb/domain/xgid.go for the field layout and domain.DecodeXGID,
// its inverse). blunderDB's Position only ever stores the AWAY score per
// side, never the match's absolute length or either player's points-so-far —
// so at a match score this is necessarily a re-encoding, not a byte-for-byte
// copy of whatever XGID the position first arrived with: it reconstructs the
// smallest match length consistent with the away scores shown (an away pair
// of [2, 4] becomes "4pt match, 2-0" here even if the position was first
// imported from a 5pt match at 3-1). That reconstruction still round-trips
// through domain.DecodeXGID to the exact same away scores, cube and board —
// the position it describes is unchanged, only the cosmetic match-length/
// score pair differs — and it is the same choice the corpus documents and
// pins (testdata/xgid_corpus.json). What must NOT be lost, because nothing
// else recovers it once dropped, is the ruleset a money game is actually
// played under: Jacoby and Beaver. Field 7 carries the Crawford flag in
// match play but the Jacoby/Beaver bitmask in a money game (bit 0 = Jacoby,
// bit 1 = Beaver) — the same dual meaning domain.DecodeXGID documents — so
// which one it emits is decided by the game type, never both.
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
    const matchLength = isMoneyGame ? 0 : Math.max(score[0], score[1]);
    const actualScore1 = isMoneyGame ? 0 : matchLength - score[0];
    const actualScore2 = isMoneyGame ? 0 : matchLength - score[1];
    const isCrawford = !isMoneyGame && (score[0] === 1 || score[1] === 1) ? 1 : 0;
    const field7 = isMoneyGame ? (has_jacoby ? 1 : 0) | (has_beaver ? 2 : 0) : isCrawford;
    const playerOnRoll = player_on_roll === 0 ? 1 : -1;
    // Field 9 (max cube): the ceiling the SOURCE stated, carried back out
    // unchanged (#271). The evaluator still does not model a capped cube —
    // this field is reported, never acted on — but dropping it on the way out
    // would silently rewrite the ruleset of a position the user pasted in. 0
    // is XGID's own "no ceiling stated", which is what a position that never
    // carried one keeps.
    const maxCube = max_cube || 0;

    return `${positionPart}:${cubeValue}:${cubeOwner}:${playerOnRoll}:${dicePart}:${actualScore1}:${actualScore2}:${field7}:${matchLength}:${maxCube}`;
}
