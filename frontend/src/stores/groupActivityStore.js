// Which library group is on screen, and what the background jobs of the others did (ADR-0101).
// Apart from libraryGroupStore so that a job's service can name its group without loading the
// group machinery (the photographs, the scoped stores).
import { writable, get } from 'svelte/store';
import { libraryChanged } from './databaseStore.js';
import { matchPanelRefreshTriggerStore, dbMutationCounterStore } from './uiStore.js';
import { refreshLibraryCounts } from './libraryCountsStore.js';
import { elsewhereStore, idleElsewhere } from './elsewhereStore.js';

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
/**
 * Whether a switch of group is in flight. Between its first await and the state put back, the
 * screen still shows the group left while the backend may already be bound to the one coming:
 * a second switch would photograph the wrong group, and a write would land in the wrong library.
 */
export const groupSwitchingStore = writable(false);

/** The writes to the library on screen still awaiting the backend's answer. */
const writesInFlight = new Set();

/**
 * Runs fn, a write that names rows of the library on screen by id; undefined, without running
 * it, while a switch holds the lock. A switch begun meanwhile waits for it to be answered.
 * @template T
 * @param {() => Promise<T>} fn
 * @returns {Promise<T | undefined>}
 */
export async function writingOnScreen(fn) {
    if (get(groupSwitchingStore)) return undefined;
    const write = fn();
    writesInFlight.add(write);
    try {
        return await write;
    } finally {
        writesInFlight.delete(write);
    }
}

/**
 * Runs fn holding the switch lock; undefined, without running it, when a switch holds it already.
 * @template T
 * @param {() => Promise<T>} fn
 * @returns {Promise<T | undefined>}
 */
export async function holdingSwitch(fn) {
    if (get(groupSwitchingStore)) return undefined;
    groupSwitchingStore.set(true);
    try {
        // Each Wails call runs in its own goroutine: a write sent before the switch could
        // otherwise reach the backend after it, and land in the library coming.
        await Promise.allSettled([...writesInFlight]);
        return await fn();
    } finally {
        groupSwitchingStore.set(false);
        // A write that ended during the switch was kept for later: if its group is the one on
        // screen now (the switch failed, or it ended after the group came), it is read now.
        if (staleGroups.delete(get(activeGroupId))) readOnScreenAgain();
    }
}

/** Counts the libraries the bound handle has carried: a read knows which one it was sent to. */
let libraryEpoch = 0;

/** The bound handle now carries another library (a switch, an open): reads begun before belong to the one left. */
export function libraryReplaced() {
    libraryEpoch++;
}

/** Never settles: what a read of the library left would have said. */
const dropped = () => new Promise(() => {});

/**
 * Runs read, a read of the library on screen, and gives its answer only if that library is still
 * the one on screen: an answer of the library left must not be shown in the one coming, and a
 * read of the new one follows the switch anyway.
 * @template T
 * @param {() => Promise<T>} read
 * @returns {Promise<T>}
 */
export function readOfLibrary(read) {
    const at = libraryEpoch;
    return read().then(
        (value) => (at === libraryEpoch ? value : dropped()),
        (error) => (at === libraryEpoch ? Promise.reject(error) : dropped())
    );
}

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

/**
 * Bumped by every write a job made to a library, whichever group: what the group on screen
 * shows of the others (the « Autres bases » panel) reads them again.
 */
export const librariesWrittenStore = writable(0);

/** Every panel listing the library on screen reads it again. */
function readOnScreenAgain() {
    libraryChanged();
    matchPanelRefreshTriggerStore.update((n) => n + 1);
    dbMutationCounterStore.update((n) => n + 1);
    refreshLibraryCounts();
}

/**
 * A job wrote the library of group: the one place that tells the panels. On screen, every panel
 * listing it reads it again; off screen, or while a switch is in flight (the backend may be
 * bound to either group), it is read when it is on screen again. Says whether it was read now.
 * @param {number | null | undefined} group
 */
export function libraryWritten(group) {
    librariesWrittenStore.update((n) => n + 1);
    elsewhereStore.set(idleElsewhere());
    const id = group || get(activeGroupId);
    if (!get(groupSwitchingStore) && groupOnScreen(id)) {
        readOnScreenAgain();
        return true;
    }
    if (get(groupIdsStore).includes(id)) staleGroups.add(id);
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
