/**
 * positionRefusal.js — why a position cannot be written to the library. Pure:
 * returns the i18n key of the first broken rule (fixed order), or null, for
 * the caller to display or act on.
 */

/**
 * @param {number} color
 * @param {{ board: { points: { color: number, checkers: number }[] } }} position
 */
function checkersOf(color, position) {
    return position.board.points.reduce((acc, point) => acc + (point.color === color ? point.checkers : 0), 0);
}

/**
 * @param {any} position the board as positionStore holds it
 * @returns {string | null} an i18n key under `status.`, or null
 */
export function positionRefusal(position) {
    const player1Checkers = checkersOf(0, position);
    const player2Checkers = checkersOf(1, position);

    if (player1Checkers > 15) return 'status.invalidP1Over15';
    if (player2Checkers > 15) return 'status.invalidP2Over15';
    if (player1Checkers === 0) return 'status.invalidP1BorneOff';
    if (player2Checkers === 0) return 'status.invalidP2BorneOff';

    if (position.decision_type === 1) {
        if (position.cube.owner !== position.player_on_roll && position.cube.owner !== -1) {
            return 'status.invalidCubeUnavailable';
        }
        if (position.score[position.player_on_roll] === 1) {
            return 'status.invalidCrawford';
        }
    }

    if ((position.score[0] === -1 && position.score[1] !== -1) || (position.score[1] === -1 && position.score[0] !== -1)) {
        return 'status.invalidUnlimitedScore';
    }

    return null;
}
