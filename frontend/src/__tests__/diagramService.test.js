/**
 * diagramService.test.js — dessiner une position hors de l'écran (#279).
 *
 * Le point vérifié est celui qui justifie le module : le diagramme sort du
 * MÊME dessinateur que le plateau. On ne teste donc pas à quoi il ressemble —
 * un test de pixels serait faux à la première retouche — mais qu'il produit un
 * document SVG autonome, avec son fond, sans rien attacher au document.
 */

import { describe, test, expect } from 'vitest';
import { renderPositionSVG, DIAGRAM_WIDTH } from '../services/diagramService.js';
import { BOARD_BACKGROUND } from '../services/boardSnapshot.js';

function startingPosition() {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    // Une position de départ suffit : le module dessine ce qu'on lui donne.
    points[1] = { checkers: 2, color: 1 };
    points[12] = { checkers: 5, color: 1 };
    points[17] = { checkers: 3, color: 1 };
    points[19] = { checkers: 5, color: 1 };
    points[24] = { checkers: 2, color: 0 };
    points[13] = { checkers: 5, color: 0 };
    points[8] = { checkers: 3, color: 0 };
    points[6] = { checkers: 5, color: 0 };
    return {
        board: { points, bearoff: [0, 0] },
        cube: { owner: -1, value: 0 },
        dice: [3, 1],
        score: [7, 7],
        player_on_roll: 0,
        decision_type: 0
    };
}

describe('le diagramme hors écran', () => {
    test('rend un document SVG autonome', () => {
        const svg = renderPositionSVG(startingPosition());
        expect(svg).toContain('<svg');
        expect(svg).toContain('http://www.w3.org/2000/svg');
        expect(svg).toContain(`width="${DIAGRAM_WIDTH}"`);
    });

    test('peint son propre fond', () => {
        expect(renderPositionSVG(startingPosition())).toContain(BOARD_BACKGROUND);
    });

    // Un diagramme qui apparaîtrait une fraction de seconde à l'écran serait
    // une nuisance visible : rien ne doit être attaché au document.
    test("n'attache rien au document", () => {
        const before = document.body.childElementCount;
        renderPositionSVG(startingPosition());
        expect(document.body.childElementCount).toBe(before);
    });

    test('respecte la taille demandée', () => {
        const svg = renderPositionSVG(startingPosition(), { width: 300, height: 220 });
        expect(svg).toContain('width="300"');
        expect(svg).toContain('height="220"');
    });
});
