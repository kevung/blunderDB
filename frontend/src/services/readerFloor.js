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

/**
 * The refusal of a database another blunderDB holds for writing while its schema is not the one
 * this blunderDB reads, worded in the user's language, or null for any other error. The backend
 * names both schema versions; closing the other blunderDB lets this one open and migrate it.
 * @param {unknown} error
 */
export function heldElsewhereMessage(error) {
    const m = /another blunderDB instance holds this database.*?its schema is (\S+?), this blunderDB reads (\S+?);/s.exec(String(error));
    return m ? tMsg('commands.databaseHeldElsewhere', { schema: m[1], reads: m[2] }) : null;
}
