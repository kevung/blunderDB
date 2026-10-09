// What the match sheet shows at a glance and what it remembers: the final score
// read from the games, and which of its folded sections the user left open.
// The open state is a per-viewer convenience, so a storage that throws (private
// window, blocked site data) only means every section starts folded.

const SECTION_KEY = 'blunderdb.matchSection.';

/**
 * Whether the user left a section of the match sheet open; folded by default.
 * @param {string} id
 * @returns {boolean}
 */
export function isSectionOpen(id) {
    try {
        return globalThis.localStorage?.getItem(SECTION_KEY + id) === '1';
    } catch {
        return false;
    }
}

/**
 * @param {string} id
 * @param {boolean} open
 */
export function rememberSectionOpen(id, open) {
    try {
        if (open) globalThis.localStorage?.setItem(SECTION_KEY + id, '1');
        else globalThis.localStorage?.removeItem(SECTION_KEY + id);
    } catch {
        // Nothing to keep: the section folds again next time.
    }
}

/**
 * The score after the last game, and who took the match when one player
 * reached its length. A game's `winner` is 1 for player 1, -1 for player 2,
 * 0 while unfinished (domain.WinnerPlayer1).
 *
 * @param {readonly { game_number: number, initial_score?: number[], winner: number, points_won: number }[] | null | undefined} games
 * @param {number} matchLength 0 for money play, which has no winner
 * @returns {{ score: [number, number], winner: 0 | 1 | null } | null}
 */
export function finalScore(games, matchLength) {
    if (!games?.length) return null;
    const last = games.reduce((a, g) => (g.game_number > a.game_number ? g : a));
    /** @type {[number, number]} */
    const score = [last.initial_score?.[0] ?? 0, last.initial_score?.[1] ?? 0];
    if (last.winner === 1) score[0] += last.points_won;
    else if (last.winner === -1) score[1] += last.points_won;
    /** @type {0 | 1 | null} */
    let winner = null;
    if (matchLength > 0 && score[0] >= matchLength) winner = 0;
    else if (matchLength > 0 && score[1] >= matchLength) winner = 1;
    return { score, winner };
}
