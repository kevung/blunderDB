/**
 * theatreBoard.js — ce que le mini-plateau du théâtre dessine : la position du plateau
 * principal en transcription, telle que Board.svelte la dessine (coup en cours au plateau,
 * plateau retourné), avec les flèches du candidat sélectionné.
 */

import { mirrorPosition, parseMoveNotation } from '../utils/boardGeometry.js';
import { boardIsMirrored, labelsFlipped, screenOfNotationPoint } from './boardOrientation.js';

/**
 * @param {{ position: any, play?: { board?: any } | null, swap?: boolean, selectedMove?: string | null }} params
 * @returns {{ position: any, flip: boolean, moves: import('../utils/boardGeometry.js').StepMove[] } | null}
 */
export function theatreScene({ position, play = null, swap = false, selectedMove = null }) {
    if (!position?.board) return null;
    // Seul le damier suit le coup en cours ; le reste vient de la position.
    let shown = play?.board ? { ...position, board: play.board } : position;
    if (boardIsMirrored({ mode: 'TRANSCRIBE', position: shown, transcriptionSwap: swap })) shown = mirrorPosition(shown);
    // La notation est numérotée depuis le camp au trait : `flip`, pas le miroir.
    const flip = labelsFlipped(shown);
    const moves = parseMoveNotation(selectedMove ?? '').map((m) => (flip ? { ...m, from: screenOfNotationPoint(m.from, true), to: screenOfNotationPoint(m.to, true) } : m));
    return { position: shown, flip, moves };
}
