/**
 * positionRefusal.js — why a position cannot be written to the library.
 *
 * Pure: no store, no status bar, no backend. It answers with the i18n key of
 * the first rule the position breaks, or null when it breaks none, so that a
 * caller can say it (isValidPosition writes it in the status bar) or merely
 * act on it (a panel disabling its save button while the board is refused).
 *
 * The rules are checked in a fixed order and only the first broken one is
 * named: a board with sixteen checkers of one side is reported for that,
 * whatever else is wrong with it.
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
