/**
 * transcriptionKeys.mouse.test.js — T2.1 : le jet donné à la souris.
 *
 * Deux choses sont tenues ici, et elles ne se mesurent qu'en dehors d'un
 * composant : qu'un clic sur une case du triangle produise EXACTEMENT ce que
 * produisent ses deux chiffres — mêmes gestes, dans le même ordre, même état
 * final — et que le budget d'ux.md §4.1 reste vrai, c'est-à-dire que la souris
 * ne rattrape pas le clavier et que le triangle batte la grille de 36.
 *
 * Les constantes de temps sont celles d'ux.md §1 ; le P du triangle est celui
 * MESURÉ le 2026-09-07 sur le prototype jetable, dans une fenêtre de 1024 px
 * (voir l'en-tête de DiceTriangle.svelte), et non celui que le document avait
 * prédit — la prédiction était optimiste de 0,07 s.
 */

import { describe, test, expect } from 'vitest';
import { COMMAND, PHASE, initialKeyState, pressKey, applyCandidates, selectCandidate, enterDicePair, enterSingleDie } from '../services/transcriptionKeys.js';

function key(code) {
    const digit = /^(?:Digit|Numpad)([0-9])$/.exec(code);
    return new KeyboardEvent('keydown', { code, key: digit ? digit[1] : code });
}

/** Les deux frappes, telles qu'un utilisateur les presse. */
function typed(state, d1, d2, expects) {
    const first = pressKey(state, key(`Digit${d1}`), { expects });
    const second = pressKey(first.state, key(`Digit${d2}`), { expects });
    return { state: second.state, commands: [...first.commands, ...second.commands] };
}

// ux.md §1, valeurs de Card, Moran & Newell.
const K = 0.28;
const B = 0.1;
const H = 0.4;
// P mesurés au prototype (loi de Fitts, b = 0,15 s/bit, D du centre du plateau
// au centre de la case, pire cas des 21 ou des 36 cibles).
const P_TRIANGLE = 0.612;
const P_GRID36 = 0.658;

const click = (p) => H + p + B + B;

describe('un clic vaut ses deux touches', () => {
    test('la case (3,1) fait ce que font les touches 3 puis 1', () => {
        const clicked = enterDicePair(initialKeyState(), 3, 1, { expects: 'checker' });
        const pressed = typed(initialKeyState(), 3, 1, 'checker');
        expect(clicked.commands).toEqual(pressed.commands);
        expect(clicked.state).toEqual(pressed.state);
        expect(clicked.state.phase).toBe(PHASE.ROLL);
        expect(clicked.state.dice).toEqual([3, 1]);
        expect(clicked.state.awaitingCandidates).toBe(true);
    });

    // Depuis « candidat choisi », un chiffre VALIDE avant d'ouvrir le jet
    // suivant (ux.md §3). Le clic hérite de la règle sans la réécrire — c'est
    // tout l'intérêt de passer par la même machine.
    test('depuis un candidat choisi, le clic valide puis ouvre le jet suivant', () => {
        let state = typed(initialKeyState(), 3, 1, 'checker').state;
        state = applyCandidates(state, 17).state;
        state = selectCandidate(state, 3).state;
        expect(state.phase).toBe(PHASE.CANDIDATE);

        const clicked = enterDicePair(state, 6, 5, { expects: 'checker' });
        const pressed = typed(state, 6, 5, 'checker');
        expect(clicked.commands).toEqual(pressed.commands);
        expect(clicked.commands[0]).toEqual({ kind: COMMAND.VALIDATE });
        expect(clicked.state.dice).toEqual([6, 5]);
    });

    // « Jet corrigeable » : le clic recommence le jet, comme le chiffre.
    test('depuis un jet corrigeable, le clic recommence le jet sans rien valider', () => {
        let state = typed(initialKeyState(), 3, 1, 'checker').state;
        state = applyCandidates(state, 17).state;
        expect(state.phase).toBe(PHASE.ROLL);

        const clicked = enterDicePair(state, 4, 2, { expects: 'checker' });
        expect(clicked.commands.map((c) => c.kind)).not.toContain(COMMAND.VALIDATE);
        expect(clicked.state.dice).toEqual([4, 2]);
    });

    // L'ouverture demande un dé par camp : la rangée de six rend un dé, et le
    // second valide l'Action `opening` comme le ferait la seconde touche.
    test('les deux dés de l-ouverture, un clic chacun, valident l-ouverture', () => {
        const first = enterSingleDie(initialKeyState(), 6, { expects: 'opening' });
        const second = enterSingleDie(first.state, 3, { expects: 'opening' });
        const pressed = typed(initialKeyState(), 6, 3, 'opening');

        expect([...first.commands, ...second.commands]).toEqual(pressed.commands);
        expect(second.commands.map((c) => c.kind)).toContain(COMMAND.VALIDATE);
        // Le gagnant joue les deux dés, dé fort d'abord (fonctionnel.md §1.2).
        expect(second.state.dice).toEqual([6, 3]);
    });

    test('l-égalité à l-ouverture se lit « relance » au clic comme à la touche', () => {
        const first = enterSingleDie(initialKeyState(), 4, { expects: 'opening' });
        const second = enterSingleDie(first.state, 4, { expects: 'opening' });
        expect(second.state.tie).toBe(true);
    });
});

describe('le budget d’ux.md §4.1, ligne « dés à la souris »', () => {
    // Un clic = un geste, et un seul : c'est ce que compte T3.5.
    test('le jet coûte UN clic, contre deux touches au clavier', () => {
        const clicked = enterDicePair(initialKeyState(), 3, 1, { expects: 'checker' });
        expect(clicked.commands.filter((c) => c.kind === COMMAND.DIE)).toHaveLength(2);
        // Une seule décision de pointage, donc un seul H et un seul P.
        expect(click(P_TRIANGLE)).toBeLessThanOrEqual(1.25);
    });

    test('le clavier reste le plus rapide : la souris ne le remplace pas', () => {
        expect(2 * K).toBeLessThan(click(P_TRIANGLE));
        // Un facteur deux environ, ce que dit le document : c'est la raison
        // pour laquelle le triangle est posé À CÔTÉ du clavier.
        expect(click(P_TRIANGLE) / (2 * K)).toBeGreaterThan(1.9);
    });

    test('le triangle bat la grille de 36, et de quinze cibles à lire', () => {
        expect(click(P_TRIANGLE)).toBeLessThan(click(P_GRID36));
        expect(21).toBeLessThan(36);
    });
});
