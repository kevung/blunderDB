// duelService.js — the gestures of the Duel tab (ADR-0072, ADR-0073), over the same
// duel.Service the CLI and the daemon drive (Database bindings). The panel derives nothing the
// Arbiter decides: every state comes back from it, the board included.
import { get } from 'svelte/store';
import { CreateDuel, OpenDuel, SuspendDuel, PlayDuel, FlagDuel, StopDuel, ListDuels, DuelOffer } from '../../wailsjs/go/database/Database.js';
import { LegalMoves, StartGammonNetMatchBatch } from '../../wailsjs/go/gui/App.js';
import { GetGammonNetAnalysisPly, GetGammonNetPruneK, GetDuelForm, SaveDuelForm } from '../../wailsjs/go/main/Config.js';
import { duelStore, duelListStore, duelNowStore, duelAnimatingStore, duelHoldsBoardStore, duelBoardStore } from '../stores/duelStore.js';
import { quizPlayStore } from '../stores/quizPlayStore.js';
import { positionStore } from '../stores/positionStore.js';
import { statusBarTextStore, activeTabStore, matchOpenRequestStore, matchPanelRefreshTriggerStore, dbMutationCounterStore } from '../stores/uiStore.js';
import { newPlay, completedPlay, resetPlay, playHop } from './quizPlay.js';
import { humanSide, framesBetween, clockView, normalizeForm, settingsFromForm } from './duel.js';
import { boardPress, boardContext, canValidateMove, isMine } from './duelBoard.js';
import { confirmAction } from './confirmService.js';
import { enterDuelMode, exitDuelMode } from './modeMachine.js';
import { isLetter, isBareLetter } from '../utils/keys.js';
import { logger } from '../utils/logger.js';
import { tMsg, translate } from '../i18n';

/** One frame of a Bot's move on the board, in ms. */
const FRAME_MS = 450;
const TICK_MS = 250;

/** @type {ReturnType<typeof setInterval> | null} */
let ticker = null;
/** The revision a flag was asked for, so a run-out clock is reported once. */
let flaggedAt = -1;
/** A gesture is on its way to the Arbiter: a second one would play twice. */
let busy = false;

// ── The form, remembered from one Duel to the next ───────────────────────────

/** The remembered form; the default one when nothing is stored or Config is unreachable. */
export async function loadDuelForm() {
    try {
        const saved = await GetDuelForm();
        return normalizeForm(saved ? JSON.parse(saved) : null);
    } catch {
        return normalizeForm(null);
    }
}

/** @param {any} form */
export function saveDuelForm(form) {
    Promise.resolve()
        .then(() => SaveDuelForm(JSON.stringify(normalizeForm(form))))
        .catch((error) => logger.error('could not remember the Duel form:', error));
}

/** The named Cadences and the Bot's levels. */
export async function duelOffer() {
    try {
        return await DuelOffer();
    } catch (error) {
        logger.error('could not read the Duel offer:', error);
        return { cadences: [], botLevels: [], levels: [] };
    }
}

export async function refreshDuels() {
    try {
        duelListStore.set((await ListDuels()) ?? []);
    } catch (error) {
        logger.error('could not list the Duels:', error);
        duelListStore.set([]);
    }
}

// ── Opening and leaving ──────────────────────────────────────────────────────

/**
 * @param {any} form
 * @param {any[]} cadences
 */
export async function startDuel(form, cadences) {
    saveDuelForm(form);
    const settings = settingsFromForm(form, cadences, get(positionStore));
    await gesture(() => CreateDuel(settings), { fresh: true });
}

/** @param {number} id */
export async function resumeDuel(id) {
    await gesture(() => OpenDuel(id), { fresh: true });
}

/** Puts the open Duel in suspense, its clocks stopped; the board returns to the library. */
export async function suspendDuel() {
    const duel = get(duelStore);
    if (!duel?.state) return;
    try {
        await SuspendDuel(duel.state.id);
    } catch (error) {
        logger.error('could not suspend the Duel:', error);
    }
    await leave({ restoreBoard: true });
    await refreshDuels();
}

/**
 * Arrête le Duel avant sa fin : gardé, le Match s'écrit tel qu'il est ; jeté, rien du Duel ne
 * s'écrit. Ce n'est jamais céder la partie (ADR-0072 règle 10).
 * @param {boolean} keep
 */
export async function stopDuel(keep) {
    const duel = get(duelStore);
    if (!duel?.state) return;
    await gesture(() => StopDuel(duel.state.id, duel.state.revision, keep));
}

// ── The player's decisions ───────────────────────────────────────────────────

/**
 * A decision other than a checker play: `roll`, `double`, `take`, `pass`, `resign` (with the
 * points offered, 1 to 3).
 * @param {'roll'|'double'|'take'|'pass'|'resign'} kind
 * @param {number} [level]
 */
export async function decide(kind, level = 0) {
    const duel = get(duelStore);
    if (!duel?.state?.awaiting) return;
    const side = humanSide(duel.state);
    await gesture(() => PlayDuel(duel.state.id, duel.state.revision, { side, kind, ...(level ? { level } : {}) }));
}

/** The explicit validation of the move arranged on the board; nothing is taken back after it. */
export async function validateMove() {
    const duel = get(duelStore);
    const play = get(quizPlayStore);
    const done = play ? completedPlay(play) : null;
    if (!duel?.state?.awaiting || !done) return;
    const side = humanSide(duel.state);
    await gesture(() => PlayDuel(duel.state.id, duel.state.revision, { side, kind: 'move', steps: done.steps }));
}

/** Puts the checkers back where the roll found them, before the validation. */
export function resetMove() {
    const awaiting = get(duelStore)?.state?.awaiting;
    if (!awaiting) return;
    quizPlayStore.update((state) => (state ? resetPlay(state, awaiting.position) : state));
}

// ── The board's gestures (ADR-0072: the Duel is played on the board) ──────────

/** The Duel as `duelBoard.js` reads it. */
export function duelBoardContext() {
    const state = get(duelStore)?.state;
    const board = get(duelBoardStore);
    return {
        awaiting: state && !state.ended ? (state.awaiting ?? null) : null,
        human: humanSide(state),
        animating: get(duelAnimatingStore) || busy,
        play: get(quizPlayStore),
        swapped: board.swapped,
        prompt: board.prompt
    };
}

/**
 * A left click on the board while a Duel holds it. Rend `true` when the Duel took it — always,
 * so that no other gesture of the board sees a click during a Duel.
 * @param {import('./duelBoard.js').BoardHit} hit
 */
export function duelBoardPress(hit) {
    if (!get(duelHoldsBoardStore)) return false;
    const action = boardPress(duelBoardContext(), hit);
    if (!action) return true;
    if (action.type === 'roll') decide('roll');
    else if (action.type === 'offerDouble') duelBoardStore.update((b) => ({ ...b, prompt: 'double' }));
    else if (action.type === 'swap') swapDuelDice();
    else if (action.type === 'validate') validateMove();
    else if (action.type === 'play') quizPlayStore.set(action.play);
    return true;
}

/**
 * A checker dragged from `from` to `to`: one step, if a legal play offers it.
 * @param {number} from
 * @param {number} to
 */
export function duelBoardDrop(from, to) {
    const ctx = duelBoardContext();
    if (!isMine(ctx) || ctx.prompt || ctx.awaiting.kind !== 'move' || !ctx.play) return;
    quizPlayStore.set(playHop({ ...ctx.play, selected: null }, from, to));
}

/**
 * A right click while a Duel holds the board. Rend `true` when the Duel's menu should open.
 * @param {import('./duelBoard.js').BoardHit} hit
 */
export function duelBoardContextMenu(hit) {
    const action = boardContext(duelBoardContext(), hit);
    if (action.type === 'reset') resetMove();
    else if (action.type === 'swap') swapDuelDice();
    return action.type === 'menu';
}

/** The dice change places, before any of them is played. */
export function swapDuelDice() {
    const play = get(quizPlayStore);
    if (play && play.steps.length > 0) return;
    duelBoardStore.update((b) => ({ ...b, swapped: !b.swapped }));
}

/** The on-board confirmation of a double: offered, or put back. */
export function confirmDouble() {
    duelBoardStore.update((b) => ({ ...b, prompt: null }));
    decide('double');
}

export function cancelDouble() {
    duelBoardStore.update((b) => ({ ...b, prompt: null }));
}

/**
 * Céder la partie, à 1, 2 ou 3 fois le videau, après confirmation. Une Action du jeu : seulement
 * à son tour.
 * @param {1|2|3} level
 */
export async function resignDuel(level) {
    const state = get(duelStore)?.state;
    if (!state?.awaiting || state.awaiting.side !== humanSide(state)) return;
    const points = level * (state.awaiting.position?.cube?.value || 1);
    const go = await confirmAction(translate('duel.resignConfirm', { n: points }), { confirmLabel: translate('duel.resign') });
    if (go) await decide('resign', level);
}

/**
 * Arrêter le Duel, après confirmation : gardé ou jeté.
 * @param {boolean} keep
 */
export async function confirmStopDuel(keep) {
    const go = await confirmAction(translate(keep ? 'duel.stopKeepConfirm' : 'duel.stopDiscardConfirm'), {
        confirmLabel: translate(keep ? 'duel.stopKeep' : 'duel.stopDiscard'),
        tone: keep ? 'primary' : 'danger'
    });
    if (go) await stopDuel(keep);
}

// ── The Arbiter's answers ────────────────────────────────────────────────────

/**
 * Sends one gesture and draws what comes back.
 * @param {() => Promise<any>} call
 * @param {{fresh?: boolean}} [options] `fresh`: a Duel newly opened, nothing to animate
 */
async function gesture(call, { fresh = false } = {}) {
    if (busy) return;
    busy = true;
    try {
        const before = get(duelStore);
        const result = await call();
        if (result?.conflict) statusBarTextStore.set(tMsg('duel.conflict'));
        await draw(fresh ? null : before, result);
    } catch (error) {
        logger.error('Duel gesture refused:', error);
        statusBarTextStore.set(tMsg('duel.refused', { error: String(error) }));
    } finally {
        busy = false;
    }
}

/**
 * @param {any} before the Duel as it was drawn, null for one newly opened
 * @param {any} result a `database.DuelState`
 */
async function draw(before, result) {
    const state = result?.state;
    if (!state) return;
    if (!before) await enterDuelMode();
    quizPlayStore.set(null);
    duelBoardStore.set({ swapped: false, prompt: null });

    const human = humanSide(state);
    const frames = before ? framesBetween(before.sheet, result.sheet, human) : [];
    if (frames.length) {
        duelAnimatingStore.set(true);
        try {
            for (const info of frames) {
                positionStore.set({ ...info.before, id: 0 });
                await pause(FRAME_MS);
                positionStore.set({ ...info.before, board: info.after, dice: [0, 0], id: 0 });
                await pause(FRAME_MS);
            }
        } finally {
            duelAnimatingStore.set(false);
        }
    }

    duelStore.set(result);
    if (state.ended) {
        await finish(state);
        return;
    }
    startTicker();
    const awaiting = state.awaiting;
    if (!awaiting) return;
    positionStore.set({ ...awaiting.position, id: 0 });
    if (awaiting.side === human && awaiting.kind === 'move') {
        try {
            const plays = await LegalMoves(awaiting.position);
            if (get(duelStore)?.state?.revision === state.revision) quizPlayStore.set(newPlay(awaiting.position, plays ?? []));
        } catch (error) {
            logger.error('could not list the legal plays of the Duel:', error);
        }
    }
}

/** @param {number} ms */
function pause(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * La fin : l'analyse se lance, l'onglet Matchs s'ouvre sur le Match en revue (ADR-0072
 * règle 9). Un Duel jeté rend le plateau à la bibliothèque.
 * @param {any} state
 */
async function finish(state) {
    stopTicker();
    quizPlayStore.set(null);
    const matchID = state.ended?.matchId ?? 0;
    duelStore.set(null);
    await refreshDuels();
    if (!matchID) {
        statusBarTextStore.set(tMsg('duel.discarded'));
        await exitDuelMode({ restoreBoard: true });
        return;
    }
    statusBarTextStore.set(tMsg(state.ended.overTime ? 'duel.endedOnTime' : 'duel.ended', { id: matchID }));
    await exitDuelMode({ restoreBoard: false });
    matchPanelRefreshTriggerStore.update((n) => n + 1);
    dbMutationCounterStore.update((n) => n + 1);
    matchOpenRequestStore.set(matchID);
    activeTabStore.set('matches');
    try {
        const [ply, pruneK] = await Promise.all([GetGammonNetAnalysisPly(), GetGammonNetPruneK()]);
        await StartGammonNetMatchBatch(matchID, ply, pruneK, 0);
    } catch (error) {
        logger.error('the analysis of a finished Duel failed to start:', error);
    }
}

/** @param {{restoreBoard: boolean}} options */
async function leave(options) {
    stopTicker();
    quizPlayStore.set(null);
    duelStore.set(null);
    await exitDuelMode(options);
}

/** Forgets the Duel without a word to the Arbiter: the library closed under it, suspending it. */
export function forgetDuel() {
    stopTicker();
    if (get(duelStore)) quizPlayStore.set(null);
    duelStore.set(null);
    duelListStore.set([]);
}

// ── The clocks ───────────────────────────────────────────────────────────────

function startTicker() {
    if (ticker) return;
    ticker = setInterval(tick, TICK_MS);
}

function stopTicker() {
    if (ticker) clearInterval(ticker);
    ticker = null;
}

/** The clocks move; a reserve run out is reported to the Arbiter, which alone decides. */
function tick() {
    const now = Date.now();
    duelNowStore.set(now);
    const duel = get(duelStore);
    const view = clockView(duel?.state, now);
    if (!view || view.running < 0 || view.overTime || busy) return;
    if (view.reserve[view.running] > 0 || flaggedAt === duel.state.revision) return;
    flaggedAt = duel.state.revision;
    gesture(() => FlagDuel(duel.state.id));
}

// ── The keyboard while a Duel holds the board ────────────────────────────────

/**
 * Le Duel tient le clavier : seules passent la Pile (b), le pipcount (p), l'aide (?), Échap et
 * la bascule de l'onglet (Ctrl+H) ; Entrée et Espace valident le coup, Retour arrière le remet en place. Rend `true`
 * quand la touche s'arrête ici.
 * @param {KeyboardEvent} event
 */
export function duelKeyGuard(event) {
    if (!get(duelHoldsBoardStore)) return false;
    if (event.code === 'Space' && !event.ctrlKey && !event.altKey && !event.metaKey) {
        const field = /** @type {HTMLElement|null} */ (event.target);
        // A field keeps its space; anywhere else it never scrolls the page nor presses a focused button.
        if (field && (field.tagName === 'INPUT' || field.tagName === 'SELECT' || field.tagName === 'TEXTAREA')) return true;
        event.preventDefault();
        if (canValidateMove(duelBoardContext())) validateMove();
        return true;
    }
    if (isBareLetter(event, 'b') || isBareLetter(event, 'p') || event.key === '?' || event.key === 'Escape') return false;
    if (event.ctrlKey && !event.shiftKey && isLetter(event, 'h')) return false;
    const target = /** @type {HTMLElement|null} */ (event.target);
    if (target && (target.tagName === 'INPUT' || target.tagName === 'SELECT' || target.tagName === 'TEXTAREA' || target.tagName === 'BUTTON')) return true;
    if (event.key === 'Enter' && !event.ctrlKey && !event.altKey) {
        event.preventDefault();
        validateMove();
    } else if (event.key === 'Backspace') {
        event.preventDefault();
        resetMove();
    }
    return true;
}
