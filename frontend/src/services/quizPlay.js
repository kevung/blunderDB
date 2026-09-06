// Jouer le coup du quiz SUR LE PLATEAU (#294, fiche J.4).
//
// Ce module est un réducteur pur : un état, des fonctions qui en rendent un
// autre. Il ne touche ni au DOM, ni aux magasins, ni au moteur — ce qui le
// rend testable sans plateau, et ce qui permet à `boardInteractions.js` de
// n'avoir qu'à lui passer le point cliqué.
//
// **Aucune règle du backgammon n'est écrite ici.** Les coups légaux viennent
// de `App.LegalMoves`, donc de `domain.LegalMoves` : ce module ne fait que
// choisir, parmi ces coups, ceux qui restent compatibles avec ce que
// l'utilisateur a déjà joué.
//
// Le piège que ce choix évite : `LegalMoves` déduplique par position
// résultante, donc pour « 24/23 13/11 » elle ne rend QU'UN ordre des deux
// pas. Filtrer sur le préfixe des pas bloquerait l'utilisateur qui joue
// l'autre ordre — un coup parfaitement légal, refusé par un artefact de
// déduplication. La compatibilité se juge donc sur le MULTI-ENSEMBLE des pas :
// un coup reste vivant tant que ce qui a été joué est contenu dans ses pas,
// dans n'importe quel ordre.
//
// L'ordre reste contraint là où il doit l'être, et gratuitement : un pas ne
// part que d'un point qui porte un pion du joueur SUR LE PLATEAU COURANT, donc
// 11/8 avant 13/11 est impossible faute de pion en 11. Et les points bloqués
// par l'adversaire le restent pendant tout le coup — ses pions ne bougent que
// pour être frappés —, donc aucune séquence acceptée ici n'est illégale.

/** Destination d'un pion sorti. Même sentinelle que `domain.Off`. */
export const OFF = -1;

const NONE = -1;
const BLACK = 0;
const WHITE = 1;
const BLACK_BAR = 25;
const WHITE_BAR = 0;

/** La barre du joueur `color`. */
export function barOf(color) {
    return color === BLACK ? BLACK_BAR : WHITE_BAR;
}

/** Une clé comparable pour un pas, la frappe exclue : elle est une conséquence. */
function stepKey(step) {
    return `${step.from}>${step.to}`;
}

/** Le multi-ensemble des pas d'un coup, en table de comptage. */
function tally(steps) {
    const counts = new Map();
    for (const s of steps) counts.set(stepKey(s), (counts.get(stepKey(s)) ?? 0) + 1);
    return counts;
}

/** `sub` est-il contenu dans `sup`, multiplicités comprises ? */
function contains(sup, sub) {
    for (const [k, n] of sub) {
        if ((sup.get(k) ?? 0) < n) return false;
    }
    return true;
}

/**
 * L'état d'un coup en cours de saisie.
 *
 * @typedef {{from: number, to: number, hit?: boolean}} Step
 * @typedef {{steps: Step[], notation: string, result: object}} Play
 * @typedef {{
 *   plays: Play[],       les coups légaux, tels que le moteur les rend
 *   mover: number,       le joueur au trait (0 = noir, 1 = blanc)
 *   board: object,       le plateau tel qu'il est APRÈS les pas déjà joués
 *   steps: Step[],       ce que l'utilisateur a joué, dans son ordre à lui
 *   selected: number|null  le point d'où part le prochain pas, s'il est choisi
 * }} PlayState
 */

/**
 * L'état de départ : la position telle qu'elle est posée, rien de joué.
 *
 * @param {object} position la position de la question
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

/** Les coups encore compatibles avec ce qui a été joué. */
export function alivePlays(state) {
    const played = tally(state.steps);
    return state.plays.filter((p) => contains(tally(p.steps), played));
}

/**
 * Le coup achevé, s'il y en a un : tous ses pas sont joués.
 *
 * Rend le coup du moteur, et non le plateau reconstruit ici — c'est SA
 * position résultante qui part au juge, donc une erreur de reconstruction ne
 * peut pas se glisser dans la réponse.
 */
export function completedPlay(state) {
    return alivePlays(state).find((p) => p.steps.length === state.steps.length) ?? null;
}

/** Les points d'où un pas peut encore partir. */
export function sources(state) {
    const out = new Set();
    for (const step of remainingSteps(state)) {
        if (hasMoverChecker(state, step.from)) out.add(step.from);
    }
    return out;
}

/**
 * Les destinations qu'un pas partant de `from` peut atteindre. Vide quand
 * `from` n'a plus rien à donner.
 */
export function destinationsFrom(state, from) {
    const out = new Set();
    if (!hasMoverChecker(state, from)) return out;
    for (const step of remainingSteps(state)) {
        if (step.from === from) out.add(step.to);
    }
    return out;
}

/** Les pas que les coups vivants offrent encore, une fois retiré ce qui est joué. */
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

function hasMoverChecker(state, point) {
    const p = state.board?.points?.[point];
    return !!p && p.checkers > 0 && p.color === state.mover;
}

/**
 * Le clic sur un point : il choisit une source, en change, ou déselectionne.
 * Un point qui n'offre aucun pas ne devient pas une sélection — la barre
 * mise à part, où l'on est obligé d'entrer, et où le joueur clique d'abord.
 */
export function selectSource(state, point) {
    if (state.selected === point) return { ...state, selected: null };
    if (!sources(state).has(point)) return state;
    return { ...state, selected: point };
}

/**
 * Joue un pas de `from` vers `to`. Rend l'état inchangé si le pas n'est offert
 * par aucun coup vivant : l'interface n'a alors rien à annuler ni à expliquer,
 * le pion n'a simplement pas bougé.
 */
export function playHop(state, from, to) {
    const step = remainingSteps(state).find((s) => s.from === from && s.to === to);
    if (!step || !hasMoverChecker(state, from)) return state;
    const board = applyStep(state.board, step, state.mover);
    return { ...state, board, steps: [...state.steps, { from, to }], selected: null };
}

/** Annule le dernier pas joué, en rejouant les autres depuis le début. */
export function undoLast(state, position) {
    if (state.steps.length === 0) return state;
    const kept = state.steps.slice(0, -1);
    let next = newPlay(position, state.plays);
    for (const s of kept) next = playHop(next, s.from, s.to);
    return next;
}

/** Remet le plateau tel que la question le pose. */
export function resetPlay(state, position) {
    return newPlay(position, state.plays);
}

// ── Le plateau affiché ───────────────────────────────────────────────────────
//
// Reconstruit pas à pas pour que l'utilisateur VOIE son coup. La réponse
// envoyée au juge, elle, est la position résultante du moteur (completedPlay),
// jamais celle-ci.

function cloneBoard(board) {
    return {
        points: (board?.points ?? []).map((p) => ({ ...p })),
        bearoff: [...(board?.bearoff ?? [0, 0])]
    };
}

/** Le plateau après `step`, frappe comprise. N'altère pas celui qu'on lui donne. */
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
