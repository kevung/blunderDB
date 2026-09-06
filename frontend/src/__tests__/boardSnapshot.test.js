/**
 * boardSnapshot.test.js — un seul rendu du plateau (#278, fiche I.22).
 *
 * Ce qui est vérifié : la copie est autonome. Un SVG sérialisé quitte le
 * document, donc ses feuilles de style ; sans la recopie des styles calculés,
 * il s'ouvre en noir et blanc — c'est le défaut qu'on ne voit qu'après avoir
 * collé l'image quelque part.
 */

import { describe, test, expect, beforeEach, afterEach } from 'vitest';
import { snapshotBoardSVG, boardImageFilename, BOARD_BACKGROUND } from '../services/boardSnapshot.js';

function mountBoard() {
    document.body.innerHTML = `
        <div id="backgammon-board">
            <svg width="400" height="300" xmlns="http://www.w3.org/2000/svg">
                <circle id="pion" cx="10" cy="10" r="5"></circle>
            </svg>
        </div>`;
    // jsdom renders nothing, so getComputedStyle returns the initial values;
    // the point of the test is that they are COPIED, not what they are.
    document.getElementById('pion').style.fill = 'rgb(1, 2, 3)';
}

beforeEach(() => {
    document.body.innerHTML = '';
});
afterEach(() => {
    document.body.innerHTML = '';
});

describe('la copie du plateau', () => {
    test("rend null quand il n'y a pas de plateau à l'écran", () => {
        expect(snapshotBoardSVG()).toBeNull();
    });

    test("rend null quand le conteneur n'a pas de SVG", () => {
        document.body.innerHTML = '<div id="backgammon-board"></div>';
        expect(snapshotBoardSVG()).toBeNull();
    });

    test('porte ses dimensions et son espace de noms', () => {
        mountBoard();
        const snap = snapshotBoardSVG();
        expect(snap.width).toBe(400);
        expect(snap.height).toBe(300);
        expect(snap.svg).toContain('http://www.w3.org/2000/svg');
    });

    // Le fond est peint DANS le SVG : un fichier ouvert dans un navigateur ou
    // glissé dans un traitement de texte n'a pas de « fond du plateau » à lui
    // prêter.
    test('peint son propre fond', () => {
        mountBoard();
        const snap = snapshotBoardSVG();
        expect(snap.svg).toContain(BOARD_BACKGROUND);
    });

    // Les styles calculés doivent atterrir sur les éléments : c'est ce qui
    // rend la copie autonome.
    test('recopie les styles calculés dans le clone', () => {
        mountBoard();
        const snap = snapshotBoardSVG();
        expect(snap.svg).toContain('rgb(1, 2, 3)');
    });

    test('propose un nom de fichier daté et lisible', () => {
        const name = boardImageFilename('svg');
        expect(name).toMatch(/^blunderdb-\d{8}-\d{4}\.svg$/);
    });
});
