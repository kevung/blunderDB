// The storage marks a request it refuses — an order no ranking serves, a
// `like` in a living collection — with this prefix; what follows names the
// refusal and is shown as is.
const REFUSAL_PREFIX = 'storage: invalid argument: ';

/**
 * The refusal an error carries, or '' when it is another failure.
 * @param {unknown} error
 */
export function refusalReason(error) {
    const text = String(error);
    const at = text.indexOf(REFUSAL_PREFIX);
    return at < 0 ? '' : text.slice(at + REFUSAL_PREFIX.length);
}
