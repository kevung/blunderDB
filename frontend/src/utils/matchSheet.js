// What the match sheet shows at a glance and what it remembers: the final score
// read from the games, and which of its tabs the user left open. The tab is a
// per-viewer convenience, so a storage that throws (private window, blocked
// site data) only means the sheet opens on its transcript.

/** The sheet's tabs, in the order of its tab bar. */
/** @type {readonly string[]} */
export const MATCH_TABS = ['transcript', 'charts', 'review', 'details', 'info', 'stats'];

const TAB_KEY = 'blunderdb.matchTab';

/**
 * The tab the user left the sheet on; the transcript by default.
 * @returns {string}
 */
export function rememberedTab() {
    try {
        const id = globalThis.localStorage?.getItem(TAB_KEY) ?? fromFoldedSections();
        return id && MATCH_TABS.includes(id) ? id : 'transcript';
    } catch {
        return 'transcript';
    }
}

// The sheet once had folded sections, each remembered open under its own key.
// The first one a user left open names the tab to start on; the keys are then
// dropped, so this is read once.
const SECTION_KEY = 'blunderdb.matchSection.';
/** @type {[string, string][]} */
const SECTION_TABS = [
    ['review', 'details'],
    ['info', 'info'],
    ['origin', 'info'],
    ['stats', 'stats']
];

/** @returns {string | null} */
function fromFoldedSections() {
    const storage = globalThis.localStorage;
    if (!storage) return null;
    const open = SECTION_TABS.find(([section]) => storage.getItem(SECTION_KEY + section) === '1');
    for (const [section] of SECTION_TABS) storage.removeItem(SECTION_KEY + section);
    if (!open) return null;
    storage.setItem(TAB_KEY, open[1]);
    return open[1];
}

/** @param {string} id */
export function rememberTab(id) {
    try {
        globalThis.localStorage?.setItem(TAB_KEY, id);
    } catch {
        // Nothing to keep: the sheet opens on its transcript next time.
    }
}

const SHARE_KEY = 'blunderdb.matchSheetShare';
const DEFAULT_SHEET_SHARE = 0.55;
const MIN_PANE_WIDTH = 200;

/**
 * The sheet's width once bounded: the list and the sheet each keep a readable width, or half of
 * a panel too narrow for both.
 *
 * @param {number} width the wanted width, in CSS pixels of the zoomed UI
 * @param {number} extent the width the list and the sheet share, in the same pixels
 * @returns {number}
 */
export function clampSheetWidth(width, extent) {
    const min = Math.min(MIN_PANE_WIDTH, extent / 2);
    return Math.max(min, Math.min(extent - min, width));
}

/**
 * The part of the panel the user left to the sheet. A share rather than a width: the same panel
 * moves between the bottom and the side of the window, and the interface scale changes its pixels.
 *
 * @returns {number} strictly between 0 and 1
 */
export function rememberedSheetShare() {
    try {
        const raw = globalThis.localStorage?.getItem(SHARE_KEY);
        const share = raw == null ? NaN : Number(raw);
        return share > 0 && share < 1 ? share : DEFAULT_SHEET_SHARE;
    } catch {
        return DEFAULT_SHEET_SHARE;
    }
}

/** @param {number} share */
export function rememberSheetShare(share) {
    try {
        globalThis.localStorage?.setItem(SHARE_KEY, String(share));
    } catch {
        // Nothing to keep: the sheet opens at its default width next time.
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
