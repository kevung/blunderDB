// Jouer le coup du quiz SUR LE PLATEAU : un réducteur pur (ni DOM, ni stores,
// ni moteur), auquel `boardInteractions.js` passe le point cliqué.
//
// Aucune règle du backgammon ici : les coups légaux viennent de
// `App.LegalMoves` ; ce module garde ceux compatibles avec ce qui est joué.
//
// `LegalMoves` déduplique par position résultante et ne rend qu'UN ordre des
// pas (« 24/23 13/11 ») : filtrer par préfixe refuserait l'autre ordre, légal.
// La compatibilité se juge donc sur le MULTI-ENSEMBLE des pas, puis l'ORDRE
// joué est éprouvé sur les plateaux intermédiaires : la suite des pas doit être
// le début d'un ordre jouable d'un coup du moteur (`orderFits`). Un pion à la
// barre entre avant tout autre ; un pion ne sort que si tous sont dans le jan
// à ce pas-là, et d'un dé plus fort que la distance seulement s'il n'en reste
// aucun plus loin. Les points adverses bloqués le restent : aucune séquence
// acceptée n'est illégale.

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
 * `start` : le plateau avant tout pas ; `dice` : le jet de la position (le
 * `rolled` d'une transcription le remplace), qui borne les sorties.
 *
 * @typedef {{plays: Play[], mover: number, board: any, steps: Step[], selected: number|null, start?: any, dice?: number[]|null, rolled?: number[]|null}} PlayState
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
        start: cloneBoard(position?.board),
        dice: position?.dice ? [position.dice[0], position.dice[1]] : null,
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
            if (!orderFits(state, play, step)) continue;
            seen.add(k);
            out.push(step);
        }
    }
    return out;
}

// ── L'ordre des pas ──────────────────────────────────────────────────────────
// `LegalMoves` rend chaque coup dans UN ordre ; le joueur peut en jouer un
// autre, pourvu que chaque pas soit jouable sur le plateau où il tombe.

/**
 * Les dés du jet, quatre pour un double ; null quand le jet est inconnu (une
 * transcription sans dés saisis) : seules la barre et le jan sont alors vérifiés.
 * @param {PlayState} state
 * @returns {number[]|null}
 */
function rollDice(state) {
    const d = state.rolled ?? state.dice;
    if (!d || !(d[0] >= 1) || !(d[1] >= 1)) return null;
    return d[0] === d[1] ? [d[0], d[0], d[0], d[0]] : [d[0], d[1]];
}

/**
 * Le joueur a-t-il un pion sur l'un des points de `points` ?
 * @param {any} board
 * @param {number} mover
 * @param {(p: number) => boolean} where
 */
function anyChecker(board, mover, where) {
    const pts = board?.points ?? [];
    for (let i = 0; i < pts.length; i++) {
        if (where(i) && pts[i]?.checkers > 0 && pts[i].color === mover) return true;
    }
    return false;
}

/**
 * Les dés (valeurs) qui jouent `step` sur `board` parmi `left` ; `[0]` quand le
 * jet est inconnu et que la barre et le jan le permettent ; vide sinon.
 * @param {any} board
 * @param {number} mover
 * @param {Step} step
 * @param {number[]|null} left
 * @returns {number[]}
 */
function diceFor(board, mover, step, left) {
    const p = board?.points?.[step.from];
    if (!p || p.checkers <= 0 || p.color !== mover) return [];
    const bar = barOf(mover);
    if (step.from !== bar && anyChecker(board, mover, (i) => i === bar)) return [];
    if (step.to !== OFF) {
        if (!left) return [0];
        const d = Math.abs(step.to - step.from);
        return left.includes(d) ? [d] : [];
    }
    // Sortie : tous les pions dans le jan, celui-ci compris.
    const outside = mover === BLACK ? (/** @type {number} */ i) => i > 6 : (/** @type {number} */ i) => i < 19;
    if (anyChecker(board, mover, outside)) return [];
    if (!left) return [0];
    const d = mover === BLACK ? step.from : 25 - step.from;
    const farther = anyChecker(board, mover, mover === BLACK ? (i) => i > step.from : (i) => i < step.from);
    return [...new Set(left)].filter((die) => die === d || (die > d && !farther));
}

/**
 * Les pas `seq`, dans cet ordre, puis `rest` dans un ordre quelconque, se
 * jouent-ils de `board`, chaque pas d'un dé encore libre ?
 * @param {any} board
 * @param {number} mover
 * @param {Step[]} seq
 * @param {Step[]} rest
 * @param {number[]|null} left
 * @returns {boolean}
 */
function playable(board, mover, seq, rest, left) {
    const [step, ...more] = seq.length ? seq : rest;
    if (!step) return true;
    const tries = seq.length ? [[step, more, rest]] : rest.map((s, i) => [s, [], [...rest.slice(0, i), ...rest.slice(i + 1)]]);
    for (const [st, nextSeq, nextRest] of /** @type {[Step, Step[], Step[]][]} */ (tries)) {
        for (const die of diceFor(board, mover, st, left)) {
            const nextLeft = left ? [...left.slice(0, left.indexOf(die)), ...left.slice(left.indexOf(die) + 1)] : null;
            if (playable(applyStep(board, st, mover), mover, nextSeq, nextRest, nextLeft)) return true;
        }
    }
    return false;
}

/**
 * Jouer `step` après les pas déjà joués laisse-t-il un début d'ordre jouable
 * de `play` ? Un état sans plateau de départ ne juge que le pas sur le plateau
 * courant.
 * @param {PlayState} state
 * @param {Play} play
 * @param {Step} step
 */
function orderFits(state, play, step) {
    const seq = [...state.steps, step];
    const rest = [...play.steps];
    for (const s of seq) {
        const i = rest.findIndex((r) => stepKey(r) === stepKey(s));
        if (i < 0) return false;
        rest.splice(i, 1);
    }
    if (!state.start) return diceFor(state.board, state.mover, step, null).length > 0;
    return playable(state.start, state.mover, seq, rest, rollDice(state));
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
 * Joue des pas écrits (une notation tapée) dans un ordre légal, s'il en existe
 * un : l'ordre d'écriture ne compte pas, seul celui des gestes au plateau est
 * imposé. L'ordre écrit est essayé d'abord ; sans ordre qui les joue tous,
 * rend l'état qui en joue le plus.
 * @param {PlayState} state
 * @param {{from: number, to: number}[]} steps
 * @returns {{ state: PlayState, all: boolean }}
 */
export function playStepsInAnyOrder(state, steps) {
    let best = state;
    /**
     * @param {PlayState} at
     * @param {{from: number, to: number}[]} left
     * @returns {boolean}
     */
    const walk = (at, left) => {
        if (at.steps.length > best.steps.length) best = at;
        if (left.length === 0) return true;
        const tried = new Set();
        for (let i = 0; i < left.length; i++) {
            const key = `${left[i].from}>${left[i].to}`;
            if (tried.has(key)) continue;
            tried.add(key);
            const next = playHop(at, left[i].from, left[i].to);
            if (next !== at && walk(next, [...left.slice(0, i), ...left.slice(i + 1)])) return true;
        }
        return false;
    };
    const all = walk(state, steps);
    return { state: best, all };
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
