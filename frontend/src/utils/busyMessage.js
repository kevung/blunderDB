// Go refuses a job that writes the library while another one does with
// "database is busy: <job>" (database.BusyError); the code names the job
// holding it, said here in the reader's language.
import { translate } from '../i18n';

const BUSY = /database is busy: (\w+)/;

/**
 * The refusal as the reader is told it, or null when message is not one.
 * @param {unknown} message
 * @returns {string | null}
 */
export function busyMessage(message) {
    const m = BUSY.exec(String(message ?? ''));
    if (!m) return null;
    return translate('jobs.busy', { job: translate(`jobs.${m[1]}`) });
}
