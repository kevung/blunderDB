/**
 * transcriptionKeys.cube.test.js — les lignes « videau » d'ux.md §3, et leur
 * budget de §4.2.
 *
 * Un double, une prise, une passe : trois touches, trois Actions, et le compte
 * qui va avec — « double + prise = 2 K ». Ce fichier tient les deux, parce que
 * le budget d'un geste de videau n'est pas une impression mais un nombre de
 * `pressKey`.
 *
 * Ce qu'il tient AUSSI, et qui est la moitié de l'affaire : `d` ne juge pas.
 * Doubler sans posséder le videau ou en partie Crawford reste une frappe qui
 * crée une Action — l'Incohérence est du Replay, marquée et jamais refusée
 * (ADR-0044, fonctionnel.md §1.4).
 */

import { describe, test, expect } from 'vitest';
import { PHASE, COMMAND, initialKeyState, pressKey, applyCandidates } from '../services/transcriptionKeys.js';

function key(code, extra = {}) {
    const digit = /^(?:Digit|Numpad)([0-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    const produced = digit ? digit[1] : letter ? letter[1].toLowerCase() : code;
    return new KeyboardEvent('keydown', { code, key: produced, ...extra });
}

/**
 * Un pilote qui compte les touches. `expects` est ce que le document attend —
 * `annotated.next.expects` — et il change quand le moteur répond : après un
 * `double`, c'est une réponse ; après une prise ou une passe, des dés ou une
 * ouverture. Le pilote le fait à la main, comme le panneau le reçoit du Go.
 */
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

describe('le videau, ligne à ligne d’ux.md §3', () => {
    test('d depuis « dés attendus » → un double, et rien d’autre', () => {
        const d = driver();
        const result = d.press('KeyD');
        expect(result.handled).toBe(true);
        expect(d.kinds()).toEqual([COMMAND.DOUBLE]);
        expect(d.state.phase).toBe(PHASE.DICE);
    });

    // La règle de la fiche : UNE touche, pas deux. Le `validate` du candidat en
    // attente est fait par le moteur (transcript.cubeGesture), donc la machine
    // n'en émet pas — sans quoi le coup serait validé deux fois.
    test('d depuis « candidat choisi » → un seul geste, le double', () => {
        const d = driver({ candidates: 5 });
        d.press('Digit3');
        d.press('Digit1');
        d.press('KeyJ');
        expect(d.state.phase).toBe(PHASE.CANDIDATE);
        d.press('KeyD');
        expect(d.kinds().at(-1)).toBe(COMMAND.DOUBLE);
        expect(d.kinds().filter((k) => k === COMMAND.VALIDATE)).toHaveLength(0);
    });

    test('d au milieu d’un jet abandonne la saisie et double quand même', () => {
        const d = driver();
        d.press('Digit4');
        expect(d.state.phase).toBe(PHASE.DIE1);
        d.press('KeyD');
        expect(d.kinds().at(-1)).toBe(COMMAND.DOUBLE);
        expect(d.state.dice).toEqual([0, 0]);
    });

    test('t prend, p passe — mais seulement en face d’une offre', () => {
        const answer = driver({ expects: 'take' });
        answer.press('KeyT');
        expect(answer.kinds()).toEqual([COMMAND.TAKE]);

        const refusal = driver({ expects: 'take' });
        refusal.press('KeyP');
        expect(refusal.kinds()).toEqual([COMMAND.PASS]);

        // Sans double en face, la touche n'est pas à nous : elle remonte au
        // répartiteur global, qui en fait ce qu'il en faisait déjà.
        const idle = driver({ expects: 'checker' });
        expect(idle.press('KeyT').handled).toBe(false);
        expect(idle.press('KeyP').handled).toBe(false);
        expect(idle.commands).toEqual([]);
    });

    test('une réponse attendue n’est pas une saisie de dés', () => {
        const d = driver({ expects: 'take' });
        expect(d.press('Digit3').handled).toBe(false);
        expect(d.press('KeyJ').handled).toBe(false);
        expect(d.commands).toEqual([]);
    });

    // ux.md §4.2, mesuré et non estimé.
    test('budget : double + prise = 2 touches', () => {
        const d = driver();
        d.press('KeyD');
        d.expects('take');
        d.press('KeyT');
        expect(d.presses).toBe(2);
        expect(d.kinds()).toEqual([COMMAND.DOUBLE, COMMAND.TAKE]);
    });

    test('budget : double + passe = 2 touches', () => {
        const d = driver();
        d.press('KeyD');
        d.expects('take');
        d.press('KeyP');
        expect(d.presses).toBe(2);
        expect(d.kinds()).toEqual([COMMAND.DOUBLE, COMMAND.PASS]);
    });

    test('budget : redouble = 2 touches, la partie reprenant après la prise', () => {
        const d = driver({ candidates: 5 });
        d.press('KeyD');
        d.expects('take');
        d.press('KeyT');
        d.expects('checker');
        expect(d.presses).toBe(2);
        // Et le tour repart sur des dés : le doubleur rejoue.
        d.press('Digit6');
        expect(d.state.phase).toBe(PHASE.DIE1);
    });

    // ADR-0044 : les rôles du moteur sont de classer, reconnaître et signaler,
    // jamais d'interdire. La machine à touches n'a donc aucune précondition de
    // videau à faire respecter — elle n'en connaît d'ailleurs pas le
    // propriétaire.
    test('d ne juge pas : la même frappe, dans tous les états, crée un double', () => {
        for (const expects of ['checker', 'opening', 'dance', 'take']) {
            const d = driver({ expects });
            expect(d.press('KeyD').handled).toBe(true);
            expect(d.kinds()).toEqual([COMMAND.DOUBLE]);
        }
    });

    test('D majuscule, Ctrl-D et Alt-D ne sont pas la touche du double', () => {
        for (const extra of [{ shiftKey: true }, { ctrlKey: true }, { altKey: true }, { metaKey: true }]) {
            const d = driver();
            expect(d.press('KeyD', extra).handled).toBe(false);
        }
    });
});
