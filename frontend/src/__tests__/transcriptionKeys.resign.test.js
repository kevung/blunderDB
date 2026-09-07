/**
 * transcriptionKeys.resign.test.js — la ligne « tout | r puis 1/2/3 » d'ux.md §3.
 *
 * Deux touches, un niveau, et une sortie : `Échap` entre les deux rend la main
 * à l'état d'avant sans avoir rien créé. C'est le seul état de la machine qui
 * détourne les chiffres des dés, et c'est ce que ce fichier tient — un `5`
 * entre `r` et son niveau ne doit surtout pas retomber sur un dé.
 */

import { describe, test, expect } from 'vitest';
import { PHASE, COMMAND, initialKeyState, pressKey, applyCandidates } from '../services/transcriptionKeys.js';

function key(code, extra = {}) {
    const digit = /^(?:Digit|Numpad)([0-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    const produced = digit ? digit[1] : letter ? letter[1].toLowerCase() : code;
    return new KeyboardEvent('keydown', { code, key: produced, ...extra });
}

function driver({ expects = 'checker', candidates = 5 } = {}) {
    let state = initialKeyState();
    const commands = [];
    let presses = 0;

    const settle = () => {
        if (!state.awaitingCandidates) return;
        const result = applyCandidates(state, candidates);
        state = result.state;
        commands.push(...result.commands);
    };

    return {
        press(code, extra) {
            presses += 1;
            const result = pressKey(state, key(code, extra), { expects });
            state = result.state;
            commands.push(...result.commands);
            settle();
            return result;
        },
        expects(next) {
            expects = next;
        },
        get state() {
            return state;
        },
        get commands() {
            return commands;
        },
        get presses() {
            return presses;
        },
        kinds() {
            return commands.map((c) => c.kind);
        }
    };
}

describe('la résignation', () => {
    test('r ouvre l’attente du niveau et ne crée rien', () => {
        const d = driver();
        const result = d.press('KeyR');
        expect(result.handled).toBe(true);
        expect(d.state.phase).toBe(PHASE.RESIGN);
        expect(d.commands).toEqual([]);
    });

    test('r puis 1, 2 ou 3 crée la résignation du niveau correspondant', () => {
        for (const [code, level] of [
            ['Digit1', 1],
            ['Digit2', 2],
            ['Digit3', 3]
        ]) {
            const d = driver();
            d.press('KeyR');
            d.press(code);
            expect(d.commands).toEqual([{ kind: COMMAND.RESIGN, value: level }]);
            expect(d.state.phase).toBe(PHASE.DICE);
        }
    });

    // ux.md §4.2 : « résignation gammon | r 2 | 2 K ».
    test('budget : résignation gammon = 2 touches', () => {
        const d = driver();
        d.press('KeyR');
        d.press('Digit2');
        expect(d.presses).toBe(2);
        expect(d.commands).toEqual([{ kind: COMMAND.RESIGN, value: 2 }]);
    });

    test('Échap annule sans effet et rend l’état d’avant', () => {
        const d = driver({ candidates: 5 });
        d.press('Digit3');
        d.press('Digit1');
        d.press('KeyJ');
        const before = d.state;
        d.press('KeyR');
        expect(d.state.phase).toBe(PHASE.RESIGN);
        d.press('Escape');
        // Le même état, à l'identique : les dés, le rang choisi, la phase.
        expect(d.state).toEqual(before);
        expect(d.kinds()).not.toContain(COMMAND.RESIGN);
    });

    test('Échap depuis un état vierge rend un état vierge', () => {
        const d = driver();
        d.press('KeyR');
        d.press('Escape');
        expect(d.state).toEqual(initialKeyState());
        expect(d.commands).toEqual([]);
    });

    // Le piège que la phase existe pour éviter : 4 à 6 sont des dés partout
    // ailleurs, et ne sont pas des niveaux.
    test('entre r et son niveau, rien ne retombe sur les dés', () => {
        for (const code of ['Digit4', 'Digit5', 'Digit6']) {
            const d = driver();
            d.press('KeyR');
            const result = d.press(code);
            expect(result.handled).toBe(true);
            expect(d.state.phase).toBe(PHASE.RESIGN);
            expect(d.commands).toEqual([]);
        }
        const d = driver();
        d.press('KeyR');
        d.press('KeyJ');
        d.press('Enter');
        d.press('KeyD');
        expect(d.commands).toEqual([]);
        expect(d.state.phase).toBe(PHASE.RESIGN);
    });

    test('r se presse dans tous les états, y compris devant une offre de videau', () => {
        for (const expects of ['checker', 'opening', 'dance', 'take']) {
            const d = driver({ expects });
            d.press('KeyR');
            d.press('Digit1');
            expect(d.commands).toEqual([{ kind: COMMAND.RESIGN, value: 1 }]);
        }
    });

    test('R majuscule, Ctrl-R et Alt-R ne sont pas la touche de la résignation', () => {
        for (const extra of [{ shiftKey: true }, { ctrlKey: true }, { altKey: true }, { metaKey: true }]) {
            const d = driver();
            expect(d.press('KeyR', extra).handled).toBe(false);
            expect(d.state.phase).toBe(PHASE.DICE);
        }
    });
});
