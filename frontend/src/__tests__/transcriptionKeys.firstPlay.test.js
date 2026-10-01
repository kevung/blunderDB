/**
 * transcriptionKeys.firstPlay.test.js — le premier coup d'une partie au clavier.
 *
 * Ses deux dés sont le jet d'ouverture : dé du joueur 1, puis dé du joueur 2.
 * C'est le moteur qui donne le coup au plus fort (GestureEnterDie) ; la
 * machine à touches les saisit comme n'importe quel jet, sans validation au
 * second dé ni relance : un double est saisi tel quel et le moteur le marque.
 *
 * Les tours de pions (jet corrigeable, candidats, j/k, Entrée, danse) ont leur
 * propre fichier, transcriptionKeys.turn.test.js.
 */

import { describe, test, expect } from 'vitest';
import { PHASE, COMMAND, initialKeyState, pressKey, dieOf } from '../services/transcriptionKeys.js';

const FIRST_PLAY = { expects: 'checker' };

/**
 * Une frappe. Les chiffres sont positionnels (`event.code`), les lettres se
 * lisent au caractère produit (`event.key`) : le pilote pose les deux, comme le
 * navigateur le fait.
 */
function key(/** @type {string} */ code, extra = {}) {
    const digit = /^(?:Digit|Numpad)([0-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    const produced = digit ? digit[1] : letter ? letter[1].toLowerCase() : code;
    return new KeyboardEvent('keydown', { code, key: produced, ...extra });
}

/** Enchaîne des frappes et rend l'état final avec tous les gestes émis. */
function type(/** @type {any} */ codes, context = FIRST_PLAY, start = initialKeyState()) {
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

    test('le second chiffre remplit le second dé et attend les candidats, sans rien valider', () => {
        const { state, commands } = type(['Digit6', 'Digit3']);
        expect(commands).toEqual([
            { kind: COMMAND.DIE, value: 6 },
            { kind: COMMAND.DIE, value: 3 }
        ]);
        expect(state.phase).toBe(PHASE.ROLL);
        expect(state.awaitingCandidates).toBe(true);
    });

    test('les dés restent dans l’ordre tapé : c’est lui qui dit qui commence', () => {
        const { state } = type(['Digit2', 'Digit5']);
        expect(state.dice).toEqual([2, 5]);
    });

    test('un double est saisi tel quel : aucune relance n’est transcrite', () => {
        const { state, commands } = type(['Digit4', 'Digit4']);
        expect(commands).toEqual([
            { kind: COMMAND.DIE, value: 4 },
            { kind: COMMAND.DIE, value: 4 }
        ]);
        expect(state.phase).toBe(PHASE.ROLL);
        expect(state.dice).toEqual([4, 4]);
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
        const result = pressKey(initialKeyState(), key('Backspace'), FIRST_PLAY);
        expect(result.handled).toBe(true);
        expect(result.commands).toEqual([]);
    });

    // Échap ferme le panneau partout ailleurs : elle n'est prise que s'il y a
    // une saisie à abandonner.
    test('Échap abandonne la saisie en cours, et rien d’autre', () => {
        expect(pressKey(initialKeyState(), key('Escape'), FIRST_PLAY).handled).toBe(false);
        const started = type(['Digit3']).state;
        const result = pressKey(started, key('Escape'), FIRST_PLAY);
        expect(result.handled).toBe(true);
        expect(result.state.dice).toEqual([0, 0]);
    });

    test('les touches qui ne sont pas des dés remontent au répartiteur', () => {
        for (const code of ['KeyP', 'Enter', 'Digit7', 'Digit9']) {
            expect(pressKey(initialKeyState(), key(code), FIRST_PLAY).handled).toBe(false);
        }
    });

    // Une réponse à un double n'est pas une saisie de dés : ses touches sont
    // `t`/`p` (T1.4), et rien n'est avalé ici en attendant.
    test('une réponse au videau n’est pas une saisie de dés', () => {
        expect(pressKey(initialKeyState(), key('Digit3'), { expects: 'take' }).handled).toBe(false);
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
    test('le jet d’ouverture coûte deux touches, la validation vient avec le tour suivant', () => {
        const { presses, commands } = type(['Digit6', 'Digit3', 'Digit5']);
        expect(presses).toBe(3);
        expect(commands).toContainEqual({ kind: COMMAND.VALIDATE });
    });
});
