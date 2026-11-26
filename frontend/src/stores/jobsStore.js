import { writable, derived, get } from 'svelte/store';
import { ListJobs, CancelJob, PauseJob, ResumeJob, MoveJob, RemoveJob } from '../../wailsjs/go/gui/App.js';
import { EventsOn } from '../../wailsjs/runtime/runtime.js';
import { logger } from '../utils/logger.js';

/** @typedef {import('../../wailsjs/go/models').jobs.Info} JobInfo */

// The registry of durable tasks, followed on jobs:changed / jobs:removed: the one source the
// Calculs chip and panel read.
export const jobsStore = writable(/** @type {JobInfo[]} */ ([]));

const ENDED = new Set(['done', 'cancelled', 'error']);

/** @param {JobInfo} job */
export const isEnded = (job) => ENDED.has(job.state);

/** A paused task still holding a place in a queue belongs with the waiting ones. */
/** @param {JobInfo} job */
export const isWaiting = (job) => job.state === 'waiting' || (job.state === 'paused' && job.position > 0);

/** @param {JobInfo} job */
export const isRunning = (job) => job.state === 'running' || (job.state === 'paused' && !(job.position > 0));

export const runningJobs = derived(jobsStore, ($jobs) => $jobs.filter(isRunning));
export const waitingJobs = derived(jobsStore, ($jobs) => $jobs.filter(isWaiting).sort((a, b) => a.library - b.library || a.position - b.position));
export const endedJobs = derived(jobsStore, ($jobs) => $jobs.filter(isEnded));

/** Overall progress of the tasks not ended, 0..1, or null when no one knows its total. */
export const jobsProgress = derived(jobsStore, ($jobs) => {
    let done = 0;
    let total = 0;
    for (const job of $jobs) {
        if (isEnded(job) || !(job.total > 0)) continue;
        done += Math.min(job.done, job.total);
        total += job.total;
    }
    return total > 0 ? done / total : null;
});

/** @param {JobInfo} job */
function upsert(job) {
    jobsStore.update(($jobs) => {
        const i = $jobs.findIndex((j) => j.id === job.id);
        if (i < 0) return [...$jobs, job];
        const next = $jobs.slice();
        next[i] = job;
        return next;
    });
}

/** Loads the registry, then follows its events. Returns the function that stops following. */
export function startJobsFeed() {
    /** @type {Set<number>} */
    const removed = new Set();
    const offs = [
        EventsOn('jobs:changed', (/** @type {JobInfo} */ job) => job && upsert(job)),
        EventsOn('jobs:removed', (/** @type {{id: number}} */ e) => e && (removed.add(e.id), jobsStore.update(($jobs) => $jobs.filter((j) => j.id !== e.id))))
    ];
    ListJobs().then(
        // An event received before the list is newer than the list: a change stays, a removal holds.
        (list) => (list || []).forEach((job) => removed.has(job.id) || get(jobsStore).some((j) => j.id === job.id) || upsert(job)),
        (error) => logger.error('could not list the tasks:', error)
    );
    return () => offs.forEach((off) => off && off());
}

/** @param {Promise<unknown>} call */
const settle = (call) => call.catch((error) => logger.error('task action failed:', error));

export const pauseJob = (/** @type {number} */ id) => settle(PauseJob(id));
export const resumeJob = (/** @type {number} */ id) => settle(ResumeJob(id));
export const cancelJob = (/** @type {number} */ id) => settle(CancelJob(id));
export const removeJob = (/** @type {number} */ id) => settle(RemoveJob(id));
export const moveJob = (/** @type {number} */ id, /** @type {number} */ index) => settle(MoveJob(id, index));

/** Everything running or waiting stops at its next checkpoint, or, when all is already paused, goes on. */
export function toggleAllPaused() {
    const live = get(jobsStore).filter((j) => !isEnded(j));
    const pausing = live.some((j) => j.state !== 'paused' && (j.pausable || j.state === 'waiting'));
    return Promise.all(live.filter((j) => (pausing ? j.state !== 'paused' && (j.pausable || j.state === 'waiting') : j.state === 'paused')).map((j) => (pausing ? pauseJob(j.id) : resumeJob(j.id))));
}

/** True when every task not ended is paused. */
export const allPaused = derived(jobsStore, ($jobs) => {
    const live = $jobs.filter((j) => !isEnded(j));
    return live.length > 0 && live.every((j) => j.state === 'paused' || !(j.pausable || j.state === 'waiting'));
});
