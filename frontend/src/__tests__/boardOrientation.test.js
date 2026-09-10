/**
 * Le plateau oscillait à chaque demi-coup.
 *
 * Hors transcription, le camp au trait est toujours en bas et rien ne bouge :
 * une position de la bibliothèque est enregistrée normalisée, le camp au trait
 * EST le joueur 0. Un brouillon, lui, est une partie qui se déroule — le trait
 * change à chaque demi-coup — et la même règle y faisait basculer le damier
 * d'un tour sur l'autre.
 *
 * L'oracle est la RÈGLE, pas un pixel : `Board.svelte` n'a pas de test de rendu
 * (two.js dessine sur un canvas que jsdom n'implémente pas), et c'est pour cela
 * que la décision vit dans un module à part.
 */
import { describe, test, expect } from 'vitest';

import { boardIsMirrored, labelsFlipped } from '../services/boardOrientation.js';

/** Une position dont seul le camp au trait nous intéresse ici. */
function pos(playerOnRoll) {
    return { player_on_roll: playerOnRoll };
}

/** Le contexte du mode Match, à l'index d'un coup joué par `playerOnRoll`. */
function matchAt(playerOnRoll) {
    return { isMatchMode: true, currentIndex: 0, movePositions: [{ player_on_roll: playerOnRoll }] };
}

describe('en transcription, le joueur 1 reste en bas', () => {
    test('le trait ne retourne rien, ni dans un sens ni dans l’autre', () => {
        for (const side of [0, 1]) {
            expect(boardIsMirrored({ mode: 'TRANSCRIBE', position: pos(side), transcriptionSwap: false }), `trait au joueur ${side + 1}`).toBe(false);
        }
    });

    test('l’inversion demandée retourne le plateau, elle aussi quel que soit le trait', () => {
        for (const side of [0, 1]) {
            expect(boardIsMirrored({ mode: 'TRANSCRIBE', position: pos(side), transcriptionSwap: true })).toBe(true);
        }
    });

    test('sans plateau retourné, les points se renumérotent quand le joueur 2 a le trait', () => {
        // Le jan du bas appartient au joueur 1 ; celui du camp au trait est en
        // haut, et ses points se comptent depuis là.
        expect(labelsFlipped(pos(1))).toBe(true);
        expect(labelsFlipped(pos(0))).toBe(false);
    });
});

describe('les autres modes ne bougent pas', () => {
    test('EDIT et EPC montrent la position telle quelle', () => {
        for (const mode of ['EDIT', 'EPC']) {
            expect(boardIsMirrored({ mode, position: pos(1) }), mode).toBe(false);
        }
    });

    test('en mode Match, le joueur 1 reste en bas', () => {
        expect(boardIsMirrored({ mode: 'NORMAL', position: pos(0), matchContext: matchAt(1) })).toBe(true);
        expect(boardIsMirrored({ mode: 'NORMAL', position: pos(0), matchContext: matchAt(0) })).toBe(false);
    });

    test('hors match, le camp au trait descend', () => {
        expect(boardIsMirrored({ mode: 'NORMAL', position: pos(1) })).toBe(true);
        expect(boardIsMirrored({ mode: 'NORMAL', position: pos(0) })).toBe(false);
    });
});
