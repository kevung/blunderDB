// Rollouts of the Analysis panel (ADR-0060): the engine and its storage live in Go and are shared
// with the CLI and the daemon; this is the GUI's side of the events and the one place that starts,
// cancels and asks.
import { get } from 'svelte/store';
import { rolloutStore, rolloutChoiceStore, idleRollout } from '../stores/rolloutStore.js';
import { positionStore } from '../stores/positionStore.js';
import { statusBarTextStore, activeTabStore } from '../stores/uiStore.js';
import { withDisplayedPositionIDs } from './modeMachine.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { confirmAction } from './confirmService.js';
import { tMsg, t } from '../i18n';
import { logger } from '../utils/logger.js';
import { EventsOn } from '../../wailsjs/runtime/runtime.js';
import { StartRollout, StartRolloutIDs, CancelRollout, RolloutPresets, RolloutStatus, CountRolloutIDs } from '../../wailsjs/go/gui/App.js';
import { GetRolloutChoice, SaveRolloutChoice } from '../../wailsjs/go/main/Config.js';

let listening = false;
/** @type {{fast: any, standard: any} | null} */
let presets = null;

/** A stored-candidate shape for a candidate the engine reports (snake_case on the wire). */
function normalizeCandidate(c) {
    return {
        move: c.move,
        equity: c.equity,
        stdErr: c.std_err ?? c.stdErr ?? 0,
        ci95: c.ci95 ?? 0,
        games: c.games ?? 0,
        jsd: c.jsd ?? 0
    };
}

// Go refuses a rollout that would store on a library another instance holds with this text
// (database.ErrReadOnly); it is said in the reader's language.
const READ_ONLY = 'database is read-only';

/** A refusal or a failure as the reader is told it. */
export function failure(err) {
    const message = err?.message ?? String(err ?? '');
    return message.includes(READ_ONLY) ? get(t)('rollout.readOnly') : message;
}

/** Events that arrive while a start call is in flight, before the job it was given is known. */
let starting = false;
/** @type {Array<() => void>} */
let held = [];

function applyJob(change, e) {
    rolloutStore.update((s) => {
        const job = e?.job ?? 0;
        if (!job || job < s.job) return s;
        return change(s, e ?? {}, job);
    });
}

/**
 * Applies an event to the state unless it belongs to a job that was replaced since: a cancelled job
 * still reports its end after the next one has begun. While a start is in flight the new job's
 * number is not known, so the events wait for it and are then filtered against it.
 */
function onJob(change) {
    return (e) => {
        if (starting) held.push(() => applyJob(change, e));
        else applyJob(change, e);
    };
}

/**
 * Runs a start call: events wait while it is in flight, then the job it returns is posted before
 * any of them is applied. Rethrows what the call raised, after releasing the events.
 */
async function startJob(call) {
    starting = true;
    held = [];
    let job = 0;
    try {
        job = Number(await call()) || 0;
        return job;
    } finally {
        const pending = held;
        held = [];
        starting = false;
        if (job) rolloutStore.update((s) => ({ ...s, job: Math.max(s.job, job) }));
        for (const apply of pending) apply();
    }
}

/** Registers the listeners once for the life of the window; safe to call from every mount. */
export function ensureRolloutEvents() {
    if (listening) return;
    listening = true;
    try {
        EventsOn(
            'rollout:progress',
            onJob((s, p, job) => ({
                ...s,
                job,
                running: true,
                kind: 'position',
                positionId: p.positionId ?? 0,
                games: p.games,
                maxGames: p.maxGames,
                candidates: (p.candidates ?? []).map(normalizeCandidate),
                outcome: null
            }))
        );
        EventsOn(
            'rollout:done',
            onJob((s, e, job) => ({
                ...told(e.stored ? 'rollout.doneStored' : 'rollout.doneNotStored'),
                ...idleRollout(),
                job,
                result: e.stored ? null : (e.record ?? null),
                resultKey: e.stored ? '' : s.pendingKey,
                outcome: { type: 'done', positionId: e.positionId ?? 0, stored: !!e.stored },
                revision: s.revision + 1
            }))
        );
        EventsOn(
            'rollout:cancelled',
            onJob((s, e, job) => ({ ...told('rollout.cancelled'), ...idleRollout(), job, outcome: { type: 'cancelled', positionId: e.positionId ?? 0 }, revision: s.revision + 1 }))
        );
        EventsOn(
            'rollout:error',
            onJob((s, e, job) => ({ ...idleRollout(), job, outcome: { type: 'error', message: failure(e.message ?? '') }, revision: s.revision + 1 }))
        );

        EventsOn(
            'rollout-batch:started',
            onJob((s, e, job) => ({ ...idleRollout(), job, running: true, kind: 'batch', total: e.total ?? 0, revision: s.revision }))
        );
        EventsOn(
            'rollout-batch:progress',
            onJob((s, p, job) => ({
                ...s,
                job,
                running: true,
                kind: 'batch',
                done: p.done,
                total: p.total,
                positionId: p.positionId ?? 0,
                games: p.games,
                maxGames: p.maxGames
            }))
        );
        EventsOn(
            'rollout-batch:done',
            onJob((s, e, job) => ({
                ...told('rollout.batchDone', { rolledOut: e.rolledOut ?? 0, total: e.total ?? 0, refused: e.refused ?? 0, failed: e.failed ?? 0 }),
                ...idleRollout(),
                job,
                outcome: { ...e, type: 'batch-done' },
                revision: s.revision + 1
            }))
        );
        EventsOn(
            'rollout-batch:cancelled',
            onJob((s, e, job) => ({
                ...told('rollout.batchCancelled', { rolledOut: e.rolledOut ?? 0, total: e.total ?? 0 }),
                ...idleRollout(),
                job,
                outcome: { ...e, type: 'batch-cancelled' },
                revision: s.revision + 1
            }))
        );
        EventsOn(
            'rollout-batch:error',
            onJob((s, e, job) => ({ ...idleRollout(), job, outcome: { type: 'batch-error', message: failure(e.message ?? '') }, revision: s.revision + 1 }))
        );
    } catch (err) {
        listening = false;
        logger.error('Rollout events unavailable:', err);
    }
}

/** What runs now, asked of Go: a panel mounted mid-way shows it without waiting for an event. */
export async function syncRolloutStatus() {
    try {
        const st = await RolloutStatus();
        rolloutStore.update((s) =>
            st?.running
                ? {
                      ...s,
                      job: Math.max(s.job, st.job ?? 0),
                      running: true,
                      kind: st.kind || 'position',
                      positionId: st.positionId ?? 0,
                      done: st.done ?? 0,
                      total: st.total ?? 0,
                      games: st.games ?? 0,
                      maxGames: st.maxGames ?? 0,
                      outcome: null
                  }
                : s.running
                  ? { ...idleRollout(), job: s.job, pendingKey: s.pendingKey, result: s.result, resultKey: s.resultKey, revision: s.revision + 1 }
                  : s
        );
    } catch (err) {
        logger.error('Rollout status unavailable:', err);
    }
}

/** { fast, standard } as the engine defines them (null when unreachable). */
export async function loadRolloutPresets() {
    if (presets) return presets;
    try {
        const list = await RolloutPresets();
        const byName = Object.fromEntries((list ?? []).map((p) => [p.name, p.settings]));
        if (byName.fast && byName.standard) presets = { fast: byName.fast, standard: byName.standard };
    } catch (err) {
        logger.error('Rollout presets unavailable:', err);
    }
    return presets;
}

/** The settings the panel's choice stands for. */
export function chosenSettings(choice, available) {
    if (!available) return null;
    if (choice.preset === 'custom') return choice.custom ?? { ...available.standard };
    return available[choice.preset] ?? available.standard;
}

const FIELDS = ['truncation', 'min_games', 'max_games', 'jsd_limit', 'ply', 'candidates', 'seed', 'workers'];

/**
 * Why a form cannot be sent ('' when it can): every field a number, and a seed JavaScript holds
 * exactly — a larger one would be rounded on the way, and a rollout would no longer be the one
 * its signature names.
 */
export function settingsProblem(s) {
    for (const key of FIELDS) {
        const v = s[key];
        if (v === '' || v == null || !Number.isFinite(Number(v))) return get(t)('rollout.invalidSettings');
    }
    if (!Number.isSafeInteger(Number(s.seed)) || Number(s.seed) < 0) return get(t)('rollout.invalidSeed');
    return '';
}

/** A form value as Go reads it: numbers only. */
export function settingsForWire(s) {
    return {
        truncation: Number(s.truncation),
        min_games: Number(s.min_games),
        max_games: Number(s.max_games),
        jsd_limit: Number(s.jsd_limit),
        ply: Number(s.ply),
        candidates: Number(s.candidates),
        seed: Number(s.seed),
        workers: Number(s.workers ?? 0)
    };
}

function say(key, params) {
    statusBarTextStore.set(tMsg(key, params));
}

/** Says how a job ended in the status bar; spreads to nothing, so it sits inside an update. */
function told(key, params) {
    say(key, params);
    return {};
}

/** Records why a start was refused, and returns it. */
function refuse(message) {
    rolloutStore.update((s) => ({ ...s, error: message }));
    return message;
}

/** The board a result describes: a rollout of an unsaved board is shown only while it is on screen. */
export function boardKey(position) {
    return JSON.stringify([get(databasePathStore), position]);
}

/**
 * Rolls out the position on the board: the plays moves names, or its candidates (its cube
 * decision without dice) when none is named. Resolves to an error message, or '' once started.
 */
export async function startRolloutOfCurrent(settings, moves = []) {
    ensureRolloutEvents();
    const problem = settingsProblem(settings);
    if (problem) return refuse(problem);
    const pos = get(positionStore);
    const id = pos?.id ?? 0;
    try {
        rolloutStore.update((s) => ({ ...s, error: '', pendingKey: id ? '' : boardKey(pos) }));
        await startJob(() => StartRollout({ positionId: id, position: id ? undefined : pos, settings: settingsForWire(settings), moves: [...moves], store: id !== 0 }));
        rolloutStore.update((s) => ({ ...s, outcome: null, running: true, kind: 'position', positionId: id, games: 0, maxGames: Number(settings.max_games), candidates: [] }));
        await syncRolloutStatus();
        return '';
    } catch (err) {
        return refuse(failure(err));
    }
}

/**
 * Rolls out the positions of the list on screen — the search results, a match, a collection — after
 * saying how many that is. Resolves to an error message, or '' (started, declined, or nothing to do).
 */
export async function startRolloutOfSearch(settings) {
    ensureRolloutEvents();
    const problem = settingsProblem(settings);
    if (problem) return refuse(problem);
    // A paged list (a collection, search results) holds no ids of its own: read whole first.
    const ids = (await withDisplayedPositionIDs((listed) => listed)) ?? [];
    const wire = settingsForWire(settings);
    let total;
    try {
        total = ids.length ? await CountRolloutIDs(ids, wire) : 0;
    } catch (err) {
        return refuse(failure(err));
    }
    if (!total) {
        say('rollout.batchNothing');
        return '';
    }
    const ok = await confirmAction(get(t)('rollout.batchConfirm', { total, listed: ids.length, depth: `${wire.max_games}` }), { confirmLabel: get(t)('rollout.start') });
    if (!ok) return '';
    try {
        rolloutStore.update((s) => ({ ...s, error: '' }));
        await startJob(() => StartRolloutIDs(ids, wire));
        rolloutStore.update((s) => ({ ...idleRollout(), job: s.job, running: true, kind: 'batch', total, revision: s.revision }));
        await syncRolloutStatus();
        return '';
    } catch (err) {
        return refuse(failure(err));
    }
}

export function cancelRollout() {
    try {
        CancelRollout();
    } catch (err) {
        logger.error('Rollout cancel failed:', err);
    }
}

/**
 * The panel's `r` and its menu: start with the chosen setting — of the plays moves names, or of the
 * position — or stop the rollout running. Resolves to an error message, or ''.
 */
export async function toggleRollout(moves = []) {
    const now = get(rolloutStore);
    if (now.running) {
        cancelRollout();
        return '';
    }
    const available = await loadRolloutPresets();
    const settings = chosenSettings(get(rolloutChoiceStore), available);
    if (!settings) return '';
    const err = await startRolloutOfCurrent(settings, moves);
    if (err) say('rollout.error', { message: err });
    return err;
}

const PRESETS = ['fast', 'standard', 'custom'];

/** Loads the persisted setting into rolloutChoiceStore. */
export async function initRolloutChoice() {
    try {
        const saved = await GetRolloutChoice();
        if (saved && PRESETS.includes(saved.preset)) rolloutChoiceStore.set({ preset: saved.preset, custom: saved.custom ?? null });
    } catch (err) {
        logger.error('could not read the rollout setting:', err);
    }
}

/** Chooses the setting: applies it at once and persists it once every field is a number. */
export function setRolloutChoice(choice) {
    rolloutChoiceStore.set(choice);
    const custom = choice.custom && !settingsProblem(choice.custom) ? settingsForWire(choice.custom) : null;
    if (choice.preset === 'custom' && !custom) return;
    SaveRolloutChoice({ preset: choice.preset, custom }).catch((err) => logger.error('could not save the rollout setting:', err));
}

/** Whether a job in progress may give way to the new one: the person decides. */
async function mayReplaceRunning() {
    if (!get(rolloutStore).running) return true;
    return !!(await confirmAction(get(t)('rollout.replaceConfirm'), { confirmLabel: get(t)('rollout.replace') }));
}

/**
 * The command `rollout [fast|standard|stop|search [fast|standard]]` (alias `ro`): the same actions
 * as the panel, with the stored choice when no preset is named.
 */
export async function runRolloutCommand(args) {
    const words = args.trim().toLowerCase().split(/\s+/).filter(Boolean);
    const aliases = { fast: 'fast', rapide: 'fast', standard: 'standard' };
    if (words[0] === 'stop') {
        cancelRollout();
        return;
    }
    const batch = words[0] === 'search' || words[0] === 'recherche';
    const name = aliases[batch ? words[1] : words[0]];
    if ((batch ? words[1] : words[0]) && !name) {
        say('rollout.usage');
        return;
    }
    const available = await loadRolloutPresets();
    if (!available) {
        say('rollout.unavailable');
        return;
    }
    const settings = chosenSettings(name ? { preset: name, custom: null } : get(rolloutChoiceStore), available);
    activeTabStore.set('analysis');
    if (!(await mayReplaceRunning())) return;
    const err = await (batch ? startRolloutOfSearch(settings) : startRolloutOfCurrent(settings));
    if (err) say('rollout.error', { message: err });
}
