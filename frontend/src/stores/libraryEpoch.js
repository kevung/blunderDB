// Which library the bound handle carries, as a count, and whether a switch is rebinding it: a
// leaf, with no store of its own to import, so that a service any panel loads (the confirmation)
// can tell whether the library on screen changed during a wait.
import { writable, get } from 'svelte/store';

/**
 * Whether a switch of group is in flight. Between its first await and the state put back, the
 * screen still shows the group left while the backend may already be bound to the one coming:
 * a second switch would photograph the wrong group, and a write would land in the wrong library.
 */
export const groupSwitchingStore = writable(false);

/** Counts the libraries the bound handle has carried: a read knows which one it was sent to. */
let libraryEpoch = 0;

/** The bound handle now carries another library (a switch, an open): reads begun before belong to the one left. */
export function libraryReplaced() {
    libraryEpoch++;
}

/**
 * A mark of the library on screen, for a gesture that spans a wait (a confirmation): it still
 * acts on that library only if libraryStillOnScreen(mark) when the wait ends.
 */
export function libraryOnScreenMark() {
    return libraryEpoch;
}

/** @param {number} mark from libraryOnScreenMark: no switch or open since, none in flight */
export function libraryStillOnScreen(mark) {
    return mark === libraryEpoch && !get(groupSwitchingStore);
}
