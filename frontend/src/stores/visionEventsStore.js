// The events the vision core reported for the video on screen, in arrival order. They are
// the panel's view of the video (timeline markers, snap of a Repère, indicators), kept in
// memory only: nothing here is a datum of the draft.
import { writable } from 'svelte/store';

/** More than a long match yields (a few events per roll); the oldest go first. */
export const VISION_EVENTS_MAX = 5000;

/** @type {import('svelte/store').Writable<readonly import('../vision/contract.js').VisionEvent[]>} */
export const visionEventsStore = writable([]);

/** @param {import('../vision/contract.js').VisionEvent} event */
export function pushVisionEvent(event) {
    visionEventsStore.update((all) => (all.length >= VISION_EVENTS_MAX ? [...all.slice(all.length - VISION_EVENTS_MAX + 1), event] : [...all, event]));
}

/** A new draft or source starts a new flow. */
export function resetVisionEvents() {
    visionEventsStore.set([]);
}
