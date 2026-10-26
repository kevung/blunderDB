import { tMsg } from '../i18n';

/**
 * The refusal of a database that needs a newer blunderDB, worded in the user's language under
 * the given message key, or null for any other error. The backend names the schema version the
 * file requires.
 * @param {unknown} error
 * @param {string} [key] the message to word it with; it takes a `{version}` parameter
 */
export function tooNewMessage(error, key = 'commands.databaseTooNew') {
    const m = /requires a newer blunderDB.*?reads schema (\S+) or later/s.exec(String(error));
    return m ? tMsg(key, { version: m[1] }) : null;
}
