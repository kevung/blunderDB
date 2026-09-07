/**
 * transcriptionKeys.opening.test.js — la dernière ligne du tableau d'ux.md §3.
 *
 * « ouverture | 1–6, 1–6 | dé J1, dé J2 ; égalité → « relance », rejouer | jet
 * corrigeable (gagnant) ». Une transition par test, et le budget d'ux.md §4.5 :
 * l'ouverture coûte DEUX touches, pas trois — le second dé valide, on ne
 * confirme pas.
 *
 * Les tours de pions (jet corrigeable, candidats, j/k, Entrée, danse) sont
 * T1.3 ; ici la machine les laisse remonter au répartiteur global.
 */

import { describe, test, expect } from 'vitest';
import { PHASE, COMMAND, initialKeyState, pressKey, dieOf } from '../services/transcriptionKeys.js';

const OPENING = { expects: 'opening' };

/** Une frappe. Les chiffres sont positionnels, les autres touches nommées. */
function key(code, extra = {}) {
    const digit = /^Digit([1-9])$/.exec(code);
    return new KeyboardEvent('keydown', { code, key: digit ? digit[1] : code, ...extra });
}

/** Enchaîne des frappes et rend l'état final avec tous les gestes émis. */
function type(codes, context = OPENING, start = initialKeyState()) {
    let state = start;
    const commands = [];
    let presses = 0;
    for (const code of codes) {
        presses += 1;
        const result = pressKey(state, key(code), context);
        state = result.state;
        commands.push(...result.commands);
    }
    return { state, commands, presses };
}

describe('la saisie des dés', () => {
    test('un chiffre remplit le premier dé', () => {
        const { state, commands } = type(['Digit6']);
        expect(state.phase).toBe(PHASE.DIE1);
        expect(state.dice).toEqual([6, 0]);
        expect(commands).toEqual([{ kind: COMMAND.DIE, value: 6 }]);
    });

    test('le second chiffre remplit le second dé et valide l’ouverture', () => {
        const { state, commands } = type(['Digit6', 'Digit3']);
        expect(commands).toEqual([{ kind: COMMAND.DIE, value: 6 }, { kind: COMMAND.DIE, value: 3 }, { kind: COMMAND.VALIDATE }]);
        // Le gagnant joue les DEUX dés : ils restent affichés, plus fort d'abord.
        expect(state.dice).toEqual([6, 3]);
        expect(state.phase).toBe(PHASE.ROLL);
        expect(state.awaitingCandidates).toBe(true);
        expect(state.tie).toBe(false);
    });

    test('le dé du joueur 2 peut être le plus fort : le jet reste trié', () => {
        const { state } = type(['Digit2', 'Digit5']);
        expect(state.dice).toEqual([5, 2]);
    });

    test('une égalité affiche « relance » et attend une nouvelle ouverture', () => {
        const { state, commands } = type(['Digit4', 'Digit4']);
        // L'Action `opening` est créée quand même : rien n'est refusé, rien
        // n'est supprimé (fonctionnel.md §1.2).
        expect(commands).toEqual([{ kind: COMMAND.DIE, value: 4 }, { kind: COMMAND.DIE, value: 4 }, { kind: COMMAND.VALIDATE }]);
        expect(state.tie).toBe(true);
        expect(state.phase).toBe(PHASE.DICE);
        expect(state.dice).toEqual([0, 0]);
        expect(state.awaitingCandidates).toBe(false);
    });

    test('la relance repart d’une saisie vide', () => {
        const first = type(['Digit4', 'Digit4']);
        const second = type(['Digit6', 'Digit1'], OPENING, first.state);
        expect(second.state.tie).toBe(false);
        expect(second.state.dice).toEqual([6, 1]);
    });

    test('Retour arrière efface les deux dés', () => {
        const { state, commands } = type(['Digit6', 'Backspace']);
        expect(commands).toEqual([{ kind: COMMAND.DIE, value: 6 }, { kind: COMMAND.CLEAR }]);
        expect(state.phase).toBe(PHASE.DICE);
        expect(state.dice).toEqual([0, 0]);
    });

    // Ailleurs Retour arrière réinitialise le plateau ; ici le plateau est celui
    // du brouillon, donc la touche est prise même quand il n'y a rien à effacer.
    test('Retour arrière est pris même sans dé saisi, mais n’émet rien', () => {
        const result = pressKey(initialKeyState(), key('Backspace'), OPENING);
        expect(result.handled).toBe(true);
        expect(result.commands).toEqual([]);
    });

    // Échap ferme le panneau partout ailleurs : elle n'est prise que s'il y a
    // une saisie à abandonner.
    test('Échap abandonne la saisie en cours, et rien d’autre', () => {
        expect(pressKey(initialKeyState(), key('Escape'), OPENING).handled).toBe(false);
        const started = type(['Digit3']).state;
        const result = pressKey(started, key('Escape'), OPENING);
        expect(result.handled).toBe(true);
        expect(result.state.dice).toEqual([0, 0]);
    });

    test('les touches qui ne sont pas des dés remontent au répartiteur', () => {
        for (const code of ['KeyJ', 'KeyP', 'Enter', 'Digit7', 'Digit9']) {
            expect(pressKey(initialKeyState(), key(code), OPENING).handled).toBe(false);
        }
    });

    // Le tour de pions est T1.3 : jusque-là la machine ne prend rien hors
    // ouverture, plutôt que d'avaler des touches en silence.
    test('hors ouverture, aucune touche n’est prise', () => {
        expect(pressKey(initialKeyState(), key('Digit3'), { expects: 'checker' }).handled).toBe(false);
    });
});

describe('les dés sont positionnels', () => {
    // Convention du dépôt : les chiffres se lisent sur event.code, pour que la
    // rangée du haut d'un AZERTY marche sans Maj (« & » y est produit par 1).
    test('Digit3 et Numpad3 valent 3, quel que soit le caractère produit', () => {
        expect(dieOf(new KeyboardEvent('keydown', { code: 'Digit3', key: '"' }))).toBe(3);
        expect(dieOf(new KeyboardEvent('keydown', { code: 'Numpad3', key: '3' }))).toBe(3);
    });

    test('7, 8, 9, 0 et les combinaisons Ctrl ne sont pas des dés', () => {
        expect(dieOf(new KeyboardEvent('keydown', { code: 'Digit7', key: '7' }))).toBe(0);
        expect(dieOf(new KeyboardEvent('keydown', { code: 'Digit0', key: '0' }))).toBe(0);
        // CTRL-1 … CTRL-9 changent de vue (raccourcis.rst) et restent globales.
        expect(dieOf(new KeyboardEvent('keydown', { code: 'Digit3', key: '3', ctrlKey: true }))).toBe(0);
    });
});

describe('le budget d’ux.md §4.5', () => {
    test('une ouverture coûte deux touches', () => {
        const { presses, commands } = type(['Digit6', 'Digit3']);
        expect(presses).toBe(2);
        // Et elle est enregistrée : la validation ne demande pas de troisième touche.
        expect(commands.some((c) => c.kind === COMMAND.VALIDATE)).toBe(true);
    });
});
