// Rollouts of the Analysis panel (ADR-0060): the engine and its storage live in Go and are shared
// with the CLI and the daemon; this is the GUI's side of the events and the one place that starts,
// cancels and asks.
import { get } from 'svelte/store';
import { rolloutStore, rolloutChoiceStore, idleRollout } from '../stores/rolloutStore.js';
import { positionStore } from '../stores/positionStore.js';
import { statusBarTextStore, activeTabStore } from '../stores/uiStore.js';
import { lastSearchStore } from '../stores/searchHistoryStore.js';
import { confirmAction } from './confirmService.js';
import { tMsg, t } from '../i18n';
import { logger } from '../utils/logger.js';
import { EventsOn } from '../../wailsjs/runtime/runtime.js';
import { StartRollout, StartRolloutFiltered, CancelRollout, RolloutPresets, RolloutStatus, CountRolloutFiltered } from '../../wailsjs/go/gui/App.js';

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

function failure(err) {
    return err?.message ?? String(err);
}

/** Registers the listeners once for the life of the window; safe to call from every mount. */
export function ensureRolloutEvents() {
    if (listening) return;
    listening = true;
    try {
        EventsOn('rollout:progress', (p) =>
            rolloutStore.update((s) => ({
                ...s,
                running: true,
                kind: 'position',
                positionId: p.positionId ?? 0,
                games: p.games,
                maxGames: p.maxGames,
                candidates: (p.candidates ?? []).map(normalizeCandidate),
                outcome: null
            }))
        );
        EventsOn('rollout:done', (e) =>
            rolloutStore.update((s) => ({
                ...idleRollout(),
                result: e.stored ? null : e.result,
                outcome: { type: 'done', positionId: e.positionId ?? 0, stored: !!e.stored },
                revision: s.revision + 1
            }))
        );
        EventsOn('rollout:cancelled', (e) => rolloutStore.update((s) => ({ ...idleRollout(), outcome: { type: 'cancelled', positionId: e?.positionId ?? 0 }, revision: s.revision + 1 })));
        EventsOn('rollout:error', (e) => rolloutStore.update((s) => ({ ...idleRollout(), outcome: { type: 'error', message: e?.message ?? '' }, revision: s.revision + 1 })));

        EventsOn('rollout-batch:started', (e) => rolloutStore.update((s) => ({ ...idleRollout(), running: true, kind: 'batch', total: e?.total ?? 0, revision: s.revision })));
        EventsOn('rollout-batch:progress', (p) =>
            rolloutStore.update((s) => ({
                ...s,
                running: true,
                kind: 'batch',
                done: p.done,
                total: p.total,
                positionId: p.positionId ?? 0,
                games: p.games,
                maxGames: p.maxGames
            }))
        );
        EventsOn('rollout-batch:done', (e) => rolloutStore.update((s) => ({ ...idleRollout(), outcome: { type: 'batch-done', ...e }, revision: s.revision + 1 })));
        EventsOn('rollout-batch:cancelled', (e) => rolloutStore.update((s) => ({ ...idleRollout(), outcome: { type: 'batch-cancelled', ...e }, revision: s.revision + 1 })));
        EventsOn('rollout-batch:error', (e) => rolloutStore.update((s) => ({ ...idleRollout(), outcome: { type: 'batch-error', message: e?.message ?? '' }, revision: s.revision + 1 })));
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
                  ? { ...idleRollout(), revision: s.revision + 1 }
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

/** Rolls out the position on the board. Resolves to an error message, or '' once started. */
export async function startRolloutOfCurrent(settings) {
    ensureRolloutEvents();
    const pos = get(positionStore);
    const id = pos?.id ?? 0;
    try {
        await StartRollout({ positionId: id, position: id ? undefined : pos, settings: settingsForWire(settings), moves: [], store: id !== 0 });
        rolloutStore.update((s) => ({ ...s, outcome: null, running: true, kind: 'position', positionId: id, games: 0, maxGames: Number(settings.max_games), candidates: [] }));
        return '';
    } catch (err) {
        return failure(err);
    }
}

/**
 * Rolls out the positions the current search selects, after saying how many that is.
 * Resolves to an error message, or '' (started, declined, or nothing to do).
 */
export async function startRolloutOfSearch(settings) {
    ensureRolloutEvents();
    const query = (get(lastSearchStore)?.command ?? '').trim();
    const wire = settingsForWire(settings);
    let total;
    try {
        total = await CountRolloutFiltered(query, wire);
    } catch (err) {
        return failure(err);
    }
    if (!total) {
        say('rollout.batchNothing');
        return '';
    }
    const label = get(t)(query ? 'rollout.batchScopeSearch' : 'rollout.batchScopeLibrary');
    const ok = await confirmAction(get(t)('rollout.batchConfirm', { total, scope: label, depth: `${wire.max_games}` }), { confirmLabel: get(t)('rollout.start') });
    if (!ok) return '';
    try {
        await StartRolloutFiltered(query, wire);
        rolloutStore.update((s) => ({ ...idleRollout(), running: true, kind: 'batch', total, revision: s.revision }));
        return '';
    } catch (err) {
        return failure(err);
    }
}

export function cancelRollout() {
    try {
        CancelRollout();
    } catch (err) {
        logger.error('Rollout cancel failed:', err);
    }
}

/** The panel's `r`: start with the chosen setting, or stop the one running. */
export async function toggleRollout() {
    if (get(rolloutStore).running) {
        cancelRollout();
        return '';
    }
    const available = await loadRolloutPresets();
    const settings = chosenSettings(get(rolloutChoiceStore), available);
    return settings ? startRolloutOfCurrent(settings) : '';
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
    const choice = name ? { preset: name, custom: null } : get(rolloutChoiceStore);
    const settings = chosenSettings(choice, available);
    activeTabStore.set('analysis');
    const err = await (batch ? startRolloutOfSearch(settings) : startRolloutOfCurrent(settings));
    if (err) say('rollout.error', { message: err });
}
