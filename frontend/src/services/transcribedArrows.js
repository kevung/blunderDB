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
 * Le coup écrit, pas par pas, chaque pas marqué `illegal` quand il ne se joue pas sur `before` :
 * pas de pion du camp au trait sur la case de départ, point d'arrivée tenu par deux pions adverses
 * ou plus, ou pion à la barre laissé là. L'ordre n'est pas celui de l'écriture : comme le Replay,
 * on joue d'abord les pas qui se jouent (« 24/18 bar/24 » se lit dans les deux sens).
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
        illegal[i] = !playable(i) || blocked;
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
    return { notation: recorded, moves: recordedSteps(recorded, info.before), tone: 'record' };
}

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
