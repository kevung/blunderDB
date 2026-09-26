// Jouer le coup du quiz SUR LE PLATEAU : un réducteur pur (ni DOM, ni stores,
// ni moteur), auquel `boardInteractions.js` passe le point cliqué.
//
// Aucune règle du backgammon ici : les coups légaux viennent de
// `App.LegalMoves` ; ce module garde ceux compatibles avec ce qui est joué.
//
// `LegalMoves` déduplique par position résultante et ne rend qu'UN ordre des
// pas (« 24/23 13/11 ») : filtrer par préfixe refuserait l'autre ordre, légal.
// La compatibilité se juge donc sur le MULTI-ENSEMBLE des pas. L'ordre reste
// contraint par le plateau courant (pas de 11/8 sans pion en 11), et les points
// adverses bloqués le restent : aucune séquence acceptée n'est illégale.

/** Destination d'un pion sorti. Même sentinelle que `domain.Off`. */
export const OFF = -1;

const NONE = -1;
const BLACK = 0;
const WHITE = 1;
const BLACK_BAR = 25;
const WHITE_BAR = 0;

/**
 * La barre du joueur `color`.
 * @param {number} color
 */
export function barOf(color) {
    return color === BLACK ? BLACK_BAR : WHITE_BAR;
}

/**
 * Une clé comparable pour un pas, la frappe exclue : elle est une conséquence.
 * @param {Step} step
 */
function stepKey(step) {
    return `${step.from}>${step.to}`;
}

/**
 * Le multi-ensemble des pas d'un coup, en table de comptage.
 * @param {Step[]} steps
 * @returns {Map<string, number>}
 */
function tally(steps) {
    const counts = new Map();
    for (const s of steps) counts.set(stepKey(s), (counts.get(stepKey(s)) ?? 0) + 1);
    return counts;
}

/**
 * `sub` est-il contenu dans `sup`, multiplicités comprises ?
 * @param {Map<string, number>} sup
 * @param {Map<string, number>} sub
 */
function contains(sup, sub) {
    for (const [k, n] of sub) {
        if ((sup.get(k) ?? 0) < n) return false;
    }
    return true;
}

/**
 * `plays` : les coups légaux du moteur ; `board` : le plateau APRÈS les pas
 * joués ; `steps` : les pas dans l'ordre de l'utilisateur ; `selected` : le
 * point du prochain pas, s'il est choisi.
 *
 * @typedef {{from: number, to: number, hit?: boolean}} Step
 * @typedef {{steps: Step[], notation: string, result: any}} Play
 * @typedef {{plays: Play[], mover: number, board: any, steps: Step[], selected: number|null}} PlayState
 */

/**
 * L'état de départ : la position telle qu'elle est posée, rien de joué.
 *
 * @param {any} position la position de la question
 * @param {Play[]} plays les coups légaux rendus par `App.LegalMoves`
 * @returns {PlayState}
 */
export function newPlay(position, plays) {
    return {
        plays: plays ?? [],
        mover: position?.player_on_roll ?? BLACK,
        board: cloneBoard(position?.board),
        steps: [],
        selected: null
    };
}

/**
 * Les coups encore compatibles avec ce qui a été joué.
 * @param {PlayState} state
 * @returns {Play[]}
 */
export function alivePlays(state) {
    const played = tally(state.steps);
    return state.plays.filter((p) => contains(tally(p.steps), played));
}

/**
 * La règle d'`alivePlays` (multi-ensemble, tout ordre), pour qui tient une
 * liste de coups sans état du réducteur (candidats d'une transcription,
 * ADR-0052).
 *
 * @param {Step[]} steps
 * @param {Step[]} played
 */
export function containsSteps(steps, played) {
    return contains(tally(steps ?? []), tally(played ?? []));
}

/**
 * Le coup achevé, s'il y en a un. Rend le coup du moteur, dont la position
 * résultante part au juge, jamais le plateau reconstruit ici.
 * @param {PlayState} state
 * @returns {Play|null}
 */
export function completedPlay(state) {
    return alivePlays(state).find((p) => p.steps.length === state.steps.length) ?? null;
}

/**
 * Les points d'où un pas peut encore partir.
 * @param {PlayState} state
 * @returns {Set<number>}
 */
export function sources(state) {
    const out = new Set();
    for (const step of remainingSteps(state)) {
        if (hasMoverChecker(state, step.from)) out.add(step.from);
    }
    return out;
}

/**
 * Les destinations d'un pas partant de `from`.
 * @param {PlayState} state
 * @param {number} from
 * @returns {Set<number>}
 */
export function destinationsFrom(state, from) {
    const out = new Set();
    if (!hasMoverChecker(state, from)) return out;
    for (const step of remainingSteps(state)) {
        if (step.from === from) out.add(step.to);
    }
    return out;
}

/**
 * Les pas que les coups vivants offrent encore, une fois retiré ce qui est joué.
 * @param {PlayState} state
 * @returns {Step[]}
 */
function remainingSteps(state) {
    const played = tally(state.steps);
    const out = [];
    const seen = new Set();
    for (const play of alivePlays(state)) {
        const left = new Map(tally(play.steps));
        for (const [k, n] of played) left.set(k, (left.get(k) ?? 0) - n);
        for (const step of play.steps) {
            const k = stepKey(step);
            if ((left.get(k) ?? 0) <= 0 || seen.has(k)) continue;
            seen.add(k);
            out.push(step);
        }
    }
    return out;
}

/**
 * @param {PlayState} state
 * @param {number} point
 */
function hasMoverChecker(state, point) {
    const p = state.board?.points?.[point];
    return !!p && p.checkers > 0 && p.color === state.mover;
}

/**
 * Le clic sur un point : choisit une source, en change, ou désélectionne. Un
 * point sans pas offert n'est pas sélectionnable, sauf la barre.
 * @param {PlayState} state
 * @param {number} point
 * @returns {PlayState}
 */
export function selectSource(state, point) {
    if (state.selected === point) return { ...state, selected: null };
    if (!sources(state).has(point)) return state;
    return { ...state, selected: point };
}

/**
 * Joue un pas de `from` vers `to`, ou rend l'état inchangé si aucun coup
 * vivant ne l'offre.
 * @param {PlayState} state
 * @param {number} from
 * @param {number} to
 * @returns {PlayState}
 */
export function playHop(state, from, to) {
    const step = remainingSteps(state).find((s) => s.from === from && s.to === to);
    if (!step || !hasMoverChecker(state, from)) return state;
    const board = applyStep(state.board, step, state.mover);
    return { ...state, board, steps: [...state.steps, { from, to }], selected: null };
}

/**
 * Annule le dernier pas joué, en rejouant les autres depuis le début.
 * @param {PlayState} state
 * @param {any} position
 * @returns {PlayState}
 */
export function undoLast(state, position) {
    if (state.steps.length === 0) return state;
    const kept = state.steps.slice(0, -1);
    let next = newPlay(position, state.plays);
    for (const s of kept) next = playHop(next, s.from, s.to);
    return next;
}

/**
 * Remet le plateau tel que la question le pose.
 * @param {PlayState} state
 * @param {any} position
 * @returns {PlayState}
 */
export function resetPlay(state, position) {
    return newPlay(position, state.plays);
}

// ── Le plateau affiché ───────────────────────────────────────────────────────
// Reconstruit pour l'affichage seulement ; le juge reçoit la position du moteur.

/** @param {any} board */
function cloneBoard(board) {
    return {
        points: (board?.points ?? []).map((/** @type {any} */ p) => ({ ...p })),
        bearoff: [...(board?.bearoff ?? [0, 0])]
    };
}

/**
 * Le plateau après `step`, frappe comprise. N'altère pas celui qu'on lui donne.
 * @param {any} board
 * @param {Step} step
 * @param {number} mover
 */
export function applyStep(board, step, mover) {
    const next = cloneBoard(board);
    const from = next.points[step.from];
    from.checkers -= 1;
    if (from.checkers <= 0) {
        from.checkers = 0;
        from.color = NONE;
    }

    if (step.to === OFF) {
        next.bearoff[mover] += 1;
        return next;
    }

    const to = next.points[step.to];
    const opponent = mover === BLACK ? WHITE : BLACK;
    if (to.color === opponent && to.checkers === 1) {
        const bar = next.points[barOf(opponent)];
        bar.checkers += 1;
        bar.color = opponent;
        to.checkers = 0;
        to.color = NONE;
    }
    to.checkers += 1;
    to.color = mover;
    return next;
}
