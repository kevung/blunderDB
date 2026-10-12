/**
 * theatreBoard.js — ce que le mini-plateau du théâtre dessine : la position du plateau
 * principal en transcription, telle que Board.svelte la dessine (coup en cours au plateau,
 * plateau retourné), avec les flèches du candidat sélectionné.
 */

import { boardMetrics, checkerPointAndCountAt, mirrorPosition, parseMoveNotation } from '../utils/boardGeometry.js';
import { defaultBoardConfig } from '../utils/boardConfig.js';
import { hitTestBearoffTray, hitTestSideControls } from '../utils/boardInteractions.js';
import { OFF } from './quizPlay.js';
import { boardIsMirrored, labelsFlipped, screenOfNotationPoint } from './boardOrientation.js';
import { diceShade, orderedDice } from './boardMove.js';
import { arrowsHiddenBy } from './boardMoveInput.js';

/**
 * `armed` : le coup au plateau suit la grammaire d'ADR-0086 ; ses dés se dessinent alors dans
 * l'ordre choisi et se grisent à mesure, comme sur le plateau principal. `mirrored` dit au clic
 * de rendre le point du modèle (25 - p).
 * @param {{ position: any, play?: any, swap?: boolean, selectedMove?: string | null, armed?: boolean }} params
 * @returns {{ position: any, onRoll: number, flip: boolean, mirrored: boolean, moves: import('../utils/boardGeometry.js').StepMove[], diceUsed: any } | null}
 */
export function theatreScene({ position, play = null, swap = false, selectedMove = null, armed = false }) {
    if (!position?.board) return null;
    // Seul le damier suit le coup en cours ; le reste vient de la position.
    let shown = play?.board ? { ...position, board: play.board } : position;
    const grammar = armed && !!play;
    if (grammar) shown = { ...shown, dice: orderedDice(play.rolled ?? position.dice, !!play.swapped) };
    const diceUsed = grammar ? diceShade(play, shown.dice) : null;
    const mirrored = boardIsMirrored({ mode: 'TRANSCRIBE', position: shown, transcriptionSwap: swap });
    if (mirrored) shown = mirrorPosition(shown);
    // La notation est numérotée depuis le camp au trait : `flip`, pas le miroir.
    const flip = labelsFlipped(shown);
    // Le coup qu'on saisit retire les flèches du candidat, comme sur le plateau principal.
    const notation = arrowsHiddenBy(play) ? '' : (selectedMove ?? '');
    const moves = parseMoveNotation(notation).map((m) => (flip ? { ...m, from: screenOfNotationPoint(m.from, true), to: screenOfNotationPoint(m.to, true) } : m));
    return { position: shown, onRoll: position.player_on_roll, flip, mirrored, moves, diceUsed };
}

/**
 * Ce que vise un point du dessin du mini-plateau, par les tests de touche du plateau principal
 * (utils/boardInteractions.js) sur la même géométrie que le dessin (diagramService.js) :
 * un dé, un point du MODÈLE (25 - p quand le plateau est retourné), le plateau de sortie du camp
 * au trait (cible d'un glissé), ou rien.
 * @param {number} x
 * @param {number} y
 * @param {{ position: any, onRoll: number, mirrored: boolean }} scene
 * @param {{ width: number, height: number, orientation: 'left' | 'right' }} drawing
 * @returns {{ kind: 'die' } | { kind: 'point', point: number } | { kind: 'off' } | { kind: 'none' }}
 */
export function theatreHit(x, y, scene, { width, height, orientation }) {
    const cfg = { ...defaultBoardConfig(), orientation };
    const geom = boardMetrics(width, height, cfg.widthFactor);
    const roller = scene.position.player_on_roll;
    if (hitTestSideControls(x, y, geom, cfg, roller).die !== null) return { kind: 'die' };
    if (hitTestBearoffTray(x, y, geom, cfg, scene.onRoll, roller)) return { kind: 'off' };
    const { checkerPoint } = checkerPointAndCountAt(x, y, width, height, cfg.widthFactor, orientation);
    if (checkerPoint < 0 || checkerPoint > 25) return { kind: 'none' };
    return { kind: 'point', point: scene.mirrored ? 25 - checkerPoint : checkerPoint };
}

/**
 * Le point du modèle où un glissé se termine : un point, le plateau de sortie (OFF), sinon null.
 * @param {ReturnType<typeof theatreHit>} hit
 */
export function dropTarget(hit) {
    if (hit.kind === 'point') return hit.point;
    return hit.kind === 'off' ? OFF : null;
}
