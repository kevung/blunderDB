import { tMsg } from '../i18n';
import { GetLikeMaxOrder } from '../../wailsjs/go/main/Config.js';

// The storage marks a request it refuses — an order no ranking serves, a
// `like` in a living collection — with this prefix; what follows names the
// refusal.
const REFUSAL_PREFIX = 'storage: invalid argument: ';

// The refusals of `like` head their text with a code the backend keeps
// stable (searchquery.ErrLikeUnranked, searchquery.ErrLikeStarred,
// reach.ErrOrder): they are worded here in the user's language, never from
// the English sentence that follows the code.
const UNRANKED = /(?:^|: )like-unranked(?::|$)/;
const STARRED = /(?:^|: )like-starred: (\S+?):/;
const ORDER = /(?:^|: )like-order(?::|$)/;

/**
 * The refusal an error carries, in the user's language when the backend
 * coded it, as the backend worded it otherwise; '' when it is another
 * failure.
 * @param {unknown} error
 * @returns {Promise<string | import('../i18n').StatusMessage>}
 */
export async function refusalReason(error) {
    const text = String(error);
    const at = text.indexOf(REFUSAL_PREFIX);
    if (at < 0) return '';
    const reason = text.slice(at + REFUSAL_PREFIX.length);
    // Every caller sets the reason inside a sentence of its own: no worded
    // reason ends with a full stop.
    return (await wordLikeRefusal(reason)) ?? reason;
}

/**
 * A coded refusal of `like` in the user's language, or null.
 * @param {string} reason
 * @returns {Promise<import('../i18n').StatusMessage | null>}
 */
async function wordLikeRefusal(reason) {
    if (UNRANKED.test(reason)) return tMsg('similar.unrankedRefused');
    const starred = STARRED.exec(reason);
    if (starred) return tMsg('similar.widenedRefused', { token: starred[1] });
    if (ORDER.test(reason)) {
        try {
            const max = await GetLikeMaxOrder();
            if (Number.isInteger(max)) return tMsg('similar.orderRefused', { max });
        } catch {
            // Without the backend's maximum, its own words still name it.
        }
    }
    return null;
}
