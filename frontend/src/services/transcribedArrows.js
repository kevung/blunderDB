/**
 * transcribedArrows.js — les flèches que le plateau dessine en transcription.
 *
 * Par défaut, le déplacement ÉCRIT dans l'Action sous le Cursor, rejoué sur la position que le
 * Replay lui donne : une correction en amont la change, et un pas qui ne se joue plus (case vide,
 * point bloqué, pion à la barre oublié) reste dessiné depuis la case écrite, marqué illégal.
 * Le coup du moteur ne se dessine que pour une ligne de l'analyse que l'utilisateur désigne
 * (survol, sélection) ou pour la saisie d'un nouveau coup, dans un ton à part.
 */

import { parseMoveNotation } from '../utils/boardGeometry.js';
import { screenOfNotationPoint } from './boardOrientation.js';

/** @typedef {import('../utils/boardGeometry.js').StepMove} StepMove */
/** @typedef {{ notation: string, moves: StepMove[], tone: 'record' | 'engine' }} TranscriptionArrows */

/**
 * L'index du modèle d'un point de notation, numérotée depuis le camp `mover`. `parseMoveNotation`
 * rend la barre comme 25 et la sortie comme -1 ; la barre du camp 0 est le point 25 du modèle,
 * celle du camp 1 le point 0.
 * @param {number} point
 * @param {number} mover
 */
function modelPoint(point, mover) {
    if (point === -1) return -1;
    return mover === 1 ? 25 - point : point;
}

/**
 * Les pas qu'aucun dé du jet ne paie, comme `diceCoherent` du Replay : chaque pas prend un dé, ou
 * plusieurs par des points intermédiaires libres (« 24/14 » sur 6-4) ; une sortie se paie d'un dé
 * égal ou supérieur à la distance. Un jet corrigé sous un coup gardé laisse des pas de la mauvaise
 * longueur : ce sont eux. Parmi les affectations, celle qui paie le plus de pas, les premiers d'abord.
 *
 * @param {StepMove[]} moves
 * @param {{ board: { points: { color: number, checkers: number }[] }, player_on_roll: number, dice?: number[] }} before
 * @returns {boolean[]} vrai pour un pas impayé ; rien n'est impayé sans jet connu
 */
function unpaidSteps(moves, before) {
    const [d1, d2] = before.dice ?? [0, 0];
    if (!(d1 >= 1 && d1 <= 6 && d2 >= 1 && d2 <= 6)) return moves.map(() => false);
    const dice = d1 === d2 ? [d1, d1, d1, d1] : [d1, d2];
    const mover = before.player_on_roll;
    const points = before.board.points;
    const open = (/** @type {number} */ p) => {
        const c = points[modelPoint(p, mover)];
        return !(c.color === 1 - mover && c.checkers >= 2);
    };
    const used = dice.map(() => false);
    /** @type {number[]} */
    let best = [];

    /** Couvre le pas depuis `at` avec des dés libres, puis continue par `next`. */
    const chain = (/** @type {StepMove} */ step, /** @type {number} */ at, /** @type {() => void} */ next) => {
        dice.forEach((die, j) => {
            if (used[j]) return;
            used[j] = true;
            const to = at - die;
            if (step.to === -1 ? die >= at : to === step.to) next();
            else if (to > Math.max(step.to, 0) && open(to)) chain(step, to, next);
            used[j] = false;
        });
    };
    const assign = (/** @type {number} */ i, /** @type {number[]} */ paid) => {
        if (paid.length + moves.length - i <= best.length) return;
        if (i === moves.length) {
            best = paid;
            return;
        }
        chain(moves[i], moves[i].from, () => assign(i + 1, [...paid, i]));
        assign(i + 1, paid);
    };
    assign(0, []);
    return moves.map((_, i) => !best.includes(i));
}

/**
 * Le coup écrit, pas par pas, chaque pas marqué `illegal` quand il ne se joue pas sur `before` :
 * pas de pion du camp au trait sur la case de départ, point d'arrivée tenu par deux pions adverses
 * ou plus, pion à la barre laissé là, sortie avec un autre pion hors du jan, ou longueur qu'aucun
 * dé ne paie. L'ordre n'est pas celui de l'écriture : comme le Replay, on joue d'abord les pas qui
 * se jouent (« 24/18 bar/24 » se lit dans les deux sens).
 *
 * @param {string | null | undefined} notation - numérotée depuis le camp au trait
 * @param {{ board?: { points: { color: number, checkers: number }[] }, player_on_roll: number } | null | undefined} before
 * @returns {StepMove[]} dans l'ordre de `parseMoveNotation(notation)`
 */
export function recordedSteps(notation, before) {
    const moves = parseMoveNotation(notation);
    const points = before?.board?.points;
    if (!moves.length || !points) return moves;
    const mover = before.player_on_roll;
    const opponent = 1 - mover;
    const bar = mover === 1 ? 0 : 25;
    const board = points.map((p) => ({ color: p.color, checkers: p.checkers }));
    const holds = (/** @type {number} */ i) => i >= 0 && i <= 25 && board[i].color === mover && board[i].checkers > 0;
    // Pions du camp hors du jan (barre comprise), en dehors de celui qui sort.
    const outside = (/** @type {number} */ leaving) => {
        let n = 0;
        for (let p = 7; p <= 25; p++) {
            const m = modelPoint(p, mover);
            if (holds(m)) n += board[m].checkers;
        }
        return leaving >= 7 ? n - 1 : n;
    };
    const unpaid = unpaidSteps(moves, before);

    /** @type {boolean[]} */
    const illegal = moves.map(() => false);
    const remaining = moves.map((_, i) => i);
    while (remaining.length) {
        const barred = holds(bar);
        const playable = (/** @type {number} */ i) => {
            const from = modelPoint(moves[i].from, mover);
            return holds(from) && (!barred || from === bar);
        };
        let at = remaining.findIndex(playable);
        if (at < 0) at = remaining.findIndex((i) => holds(modelPoint(moves[i].from, mover)));
        if (at < 0) at = 0;
        const i = remaining.splice(at, 1)[0];
        const from = modelPoint(moves[i].from, mover);
        const to = modelPoint(moves[i].to, mover);
        const blocked = to >= 0 && to <= 25 && board[to].color === opponent && board[to].checkers >= 2;
        const early = to === -1 && outside(moves[i].from) > 0;
        illegal[i] = !playable(i) || blocked || early || unpaid[i];
        if (holds(from)) {
            board[from].checkers--;
            if (board[from].checkers === 0) board[from].color = -1;
        }
        if (to >= 0 && to <= 25) {
            const hit = board[to].color === opponent;
            board[to] = { color: mover, checkers: hit ? 1 : board[to].checkers + 1 };
        }
    }
    return moves.map((m, i) => (illegal[i] ? { ...m, illegal: true } : m));
}

/**
 * Ce que le plateau dessine en transcription. `engineMove` est la ligne de l'analyse que
 * l'utilisateur désigne ; elle l'emporte sur le coup écrit, sauf à être ce coup même. Sinon,
 * l'Action sous le Cursor : seul un coup de pions écrit se dessine — ni danse, ni coup non
 * enregistré, ni videau.
 *
 * @param {{ info?: any, engineMove?: string | null }} params
 * @returns {TranscriptionArrows | null}
 */
export function transcriptionArrows({ info = null, engineMove = null }) {
    const recorded = info?.kind === 'checker' && info.before && info.notation ? info.notation : null;
    if (engineMove && engineMove !== recorded) {
        return { notation: engineMove, moves: parseMoveNotation(engineMove), tone: 'engine' };
    }
    if (!recorded) return null;
    const moves = recordedSteps(recorded, info.before);
    // Le Replay juge le coup entier : une faute qu'aucun pas ne montre seul (un dé laissé
    // injouer) met tous les pas en alerte, pour qu'un coup illégal ne se dessine jamais normal.
    const faulted = (info.inconsistencies ?? []).some((/** @type {any} */ f) => PLAY_FAULTS.has(f.rule));
    const flagged = faulted && !moves.some((m) => m.illegal) ? moves.map((m) => ({ ...m, illegal: true })) : moves;
    return { notation: recorded, moves: flagged, tone: 'record' };
}

// Les règles du Replay qui rendent un coup écrit illégal (transcript/replay.go).
const PLAY_FAULTS = new Set(['unreachable_board', 'dice_unused']);

/**
 * Des pas en notation vers les points dessinés : la notation est numérotée depuis le camp au
 * trait, donc `flipped` (joueur dessiné en haut), pas le miroir.
 * @param {StepMove[]} moves
 * @param {boolean} flipped
 * @returns {StepMove[]}
 */
export function screenMoves(moves, flipped) {
    if (!flipped) return moves;
    return moves.map((m) => ({ ...m, from: screenOfNotationPoint(m.from, true), to: screenOfNotationPoint(m.to, true) }));
}
