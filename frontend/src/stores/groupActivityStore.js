// Which library group is on screen, and what the background jobs of the others did (ADR-0101).
// Apart from libraryGroupStore so that a job's service can name its group without loading the
// group machinery (the photographs, the scoped stores).
import { writable, get } from 'svelte/store';

/** The groups in tab-bar order: ids only, the active one's path is databasePathStore. */
export const groupIdsStore = writable(/** @type {number[]} */ ([1]));
export const activeGroupId = writable(1);

/**
 * @typedef {{ state: 'running' | 'done', job: string, message?: import('../i18n').StatusMessage | null }} GroupActivity
 *   what a background job of a group off screen is doing: an import, the watched folder, a batch
 */
/** The groups off screen whose jobs moved since they were left, by id: their header's badge. */
export const groupActivityStore = writable(/** @type {Map<number, GroupActivity>} */ (new Map()));
/** The groups whose library a job wrote while they were off screen: read again on their return. */
const staleGroups = new Set();
/** Whether the import in flight runs one file at a time on the bound handle: no switch then. */
export const importForegroundStore = writable(false);

/**
 * Whether a job's group is the one on screen. An event that names no group comes from the
 * bound handle, which always carries the group on screen.
 * @param {number | null | undefined} group
 */
export function groupOnScreen(group) {
    return !group || group === get(activeGroupId);
}

/**
 * A background job of group moved or ended. On screen, nothing changes: the caller shows it as
 * before, and true says so. Off screen, the group's header carries a badge, and an end marks
 * its library to read again when it comes back; message is what the status bar says then.
 * @param {number | null | undefined} group
 * @param {'running' | 'done'} state
 * @param {string} job
 * @param {import('../i18n').StatusMessage | null} [message]
 */
export function noteGroupJob(group, state, job, message = null) {
    if (groupOnScreen(group)) return true;
    const id = /** @type {number} */ (group);
    if (!get(groupIdsStore).includes(id)) return false;
    if (state === 'done') staleGroups.add(id);
    groupActivityStore.update((m) => {
        const was = m.get(id);
        // A finished job's message stays until the group is seen, even if another job starts.
        const kept = state === 'running' && was?.state === 'done' ? was : null;
        return new Map(m).set(id, kept ?? { state, job, message: message ?? was?.message ?? null });
    });
    return false;
}

/** @param {number} id the group coming on screen: its badge goes, and says whether it is stale */
export function takeActivity(id) {
    const activity = get(groupActivityStore).get(id) ?? null;
    if (activity) {
        groupActivityStore.update((m) => {
            const next = new Map(m);
            next.delete(id);
            return next;
        });
    }
    const stale = staleGroups.delete(id);
    return { activity, stale };
}
