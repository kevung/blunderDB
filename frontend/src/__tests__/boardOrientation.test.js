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

import { boardIsMirrored, labelsFlipped, screenOfModelPoint, screenOfNotationPoint } from '../services/boardOrientation.js';

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
    test('EDIT et EVAL montrent la position telle quelle', () => {
        for (const mode of ['EDIT', 'EVAL']) {
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

describe('les deux numérotations ne se convertissent pas de la même façon', () => {
    // `domain.LegalMoves` rend des pas ABSOLUS (`Board.Points[src]`), et
    // `domain.pointLabel` écrit les notations RELATIVEMENT au camp au trait
    // (25-idx pour le joueur 2). Les deux coïncident quand le joueur 1 a le
    // trait ; le joueur 2 les sépare.
    const transcribing = (side) => ({ mode: 'TRANSCRIBE', position: { player_on_roll: side }, transcriptionSwap: false });

    test('trait au joueur 1 : les deux numérotations sont la même', () => {
        const mirrored = boardIsMirrored(transcribing(0));
        const flipped = labelsFlipped({ player_on_roll: 0 });
        expect(screenOfModelPoint(24, mirrored)).toBe(24);
        expect(screenOfNotationPoint(24, flipped)).toBe(24);
    });

    test('trait au joueur 2 : le pas ne bouge pas, la notation se retourne', () => {
        const mirrored = boardIsMirrored(transcribing(1));
        const flipped = labelsFlipped({ player_on_roll: 1 });
        // Le plateau n'est pas retourné : un pas absolu se dessine là où il est.
        expect(mirrored).toBe(false);
        expect(screenOfModelPoint(24, mirrored)).toBe(24);
        // « 24/18 » du joueur 2 nomme SES pions arriérés, en 1 et 7 du damier —
        // c'est là que la flèche doit se poser. La confondre avec le pas
        // dessinait le coup du joueur 2 sur les pions du joueur 1.
        expect(flipped).toBe(true);
        expect(screenOfNotationPoint(24, flipped)).toBe(1);
        expect(screenOfNotationPoint(18, flipped)).toBe(7);
    });

    test('plateau retourné, trait au joueur 2 : c’est l’inverse', () => {
        const mirrored = boardIsMirrored({ ...transcribing(1), transcriptionSwap: true });
        // Retourné, le joueur 2 est en bas : c'est le pas absolu qui se déplace,
        // et la notation qui tombe juste telle quelle.
        expect(screenOfModelPoint(24, mirrored)).toBe(1);
        expect(screenOfNotationPoint(24, labelsFlipped({ player_on_roll: 0 }))).toBe(24);
    });

    test('la sortie et la barre traversent sans se faire renuméroter', () => {
        expect(screenOfModelPoint(-1, true)).toBe(-1);
        expect(screenOfNotationPoint(-1, true)).toBe(-1);
        expect(screenOfModelPoint(0, true)).toBe(25);
        expect(screenOfModelPoint(25, true)).toBe(0);
    });
});
