// The status bar message of a swap of a match's players that did not happen:
// the backend's two refusals (storage.ErrSwapDraftOpen, ErrSwapPlayedHere),
// recognised by their message, which is part of that contract; any other
// error is the caller's own failure message.

const REFUSALS = [
    ['a transcription draft is open on the match', 'match.swapDraftOpen'],
    ['the match was played here, under arbitration', 'match.swapPlayedHere']
];

/**
 * @param {unknown} error what SwapMatchPlayers rejected with
 * @param {string} fallback the i18n key of a failure
 * @returns {string} an i18n key
 */
export function swapPlayersErrorKey(error, fallback) {
    const message = String(error);
    const hit = REFUSALS.find(([text]) => message.includes(text));
    return hit ? hit[1] : fallback;
}
