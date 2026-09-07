/**
 * transcriptionKeys.cubeMouse.test.js — T2.5 : le videau et la correction à la
 * souris.
 *
 * Deux choses se mesurent ici, et ni l'une ni l'autre dans un composant.
 *
 * La première est l'ÉQUIVALENCE : un bouton de la rangée `[D] [T] [P] [R]`, le
 * videau cliqué sur le plateau, une entrée du menu contextuel du Transcript
 * doivent produire exactement ce que produisent `d`, `t`, `p`, `r` et
 * `i`/`a`/`x`/`s` — mêmes gestes, dans le même ordre, même état final. Un
 * second chemin qui ressemblerait au premier finirait par en différer sur un
 * état que personne ne teste ; le test le dit, geste par geste.
 *
 * La seconde est le BUDGET d'ux.md §4.2 : « double puis prise à la souris,
 * H P B B, P B B, H = 3,4 s ». Un budget est un COMPTE — ici deux clics et deux
 * changements de main — et il se compte en appelant la machine autant de fois
 * que l'utilisateur clique. Les constantes sont celles d'ux.md §1.
 */

import { describe, test, expect } from 'vitest';
import { COMMAND, PHASE, initialKeyState, pressKey, applyCandidates, cubeGesture, beginResign, resignWithLevel, cancelResign, menuCommands, cursorCommands } from '../services/transcriptionKeys.js';

function key(code, extra = {}) {
    const digit = /^(?:Digit|Numpad)([0-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    const produced = digit ? digit[1] : letter ? letter[1].toLowerCase() : code;
    return new KeyboardEvent('keydown', { code, key: produced, ...extra });
}

/** Ce que la touche fait, dépouillé du `handled` que la souris n'a pas. */
function typed(state, code, expects) {
    const result = pressKey(state, key(code), { expects });
    return { state: result.state, commands: result.commands };
}

// ux.md §1, valeurs de Card, Moran & Newell.
const K = 0.28;
const P = 1.1;
const B = 0.1;
const H = 0.4;
const CLICK = P + B + B;

describe('un clic de videau vaut sa touche', () => {
    test('[D] fait ce que fait d, depuis n’importe quel état de saisie', () => {
        for (const expects of ['checker', 'opening', 'dance']) {
            const clicked = cubeGesture(initialKeyState(), COMMAND.DOUBLE);
            const pressed = typed(initialKeyState(), 'KeyD', expects);
            expect(clicked.commands, expects).toEqual(pressed.commands);
            expect(clicked.state, expects).toEqual(pressed.state);
        }
    });

    // La règle qui fait le budget : UNE commande, pas deux. Le `validate` du
    // candidat en attente est fait par le moteur (transcript.cubeGesture).
    test('[D] depuis un candidat choisi n’émet pas de validate non plus', () => {
        let state = typed(initialKeyState(), 'Digit3', 'checker').state;
        state = typed(state, 'Digit1', 'checker').state;
        state = applyCandidates(state, 17).state;
        state = typed(state, 'KeyJ', 'checker').state;
        expect(state.phase).toBe(PHASE.CANDIDATE);

        const clicked = cubeGesture(state, COMMAND.DOUBLE);
        const pressed = typed(state, 'KeyD', 'checker');
        expect(clicked.commands).toEqual([{ kind: COMMAND.DOUBLE }]);
        expect(clicked.commands).toEqual(pressed.commands);
        expect(clicked.state).toEqual(pressed.state);
    });

    test('[T] et [P] font ce que font t et p devant une offre', () => {
        for (const [kind, code] of [
            [COMMAND.TAKE, 'KeyT'],
            [COMMAND.PASS, 'KeyP']
        ]) {
            const clicked = cubeGesture(initialKeyState(), kind);
            const pressed = typed(initialKeyState(), code, 'take');
            expect(clicked.commands, kind).toEqual(pressed.commands);
            expect(clicked.state, kind).toEqual(pressed.state);
        }
    });

    // ADR-0044 : le geste ne juge pas. Ce qui s'éteint est le BOUTON, dans le
    // panneau, parce qu'une cible qui ne répond à rien vaut moins que pas de
    // cible ; la machine, elle, produit le double d'où qu'il vienne.
    test('le geste ne juge pas davantage que la touche', () => {
        const clicked = cubeGesture(initialKeyState(), COMMAND.DOUBLE);
        expect(clicked.commands).toEqual([{ kind: COMMAND.DOUBLE }]);
    });
});

describe('la résignation, deux gestes à la souris comme au clavier', () => {
    test('[R] mène au même état que r, l’état d’avant mis de côté', () => {
        const before = typed(initialKeyState(), 'Digit4', 'checker').state;
        const clicked = beginResign(before);
        const pressed = typed(before, 'KeyR', 'checker');
        expect(clicked.state).toEqual(pressed.state);
        expect(clicked.state.phase).toBe(PHASE.RESIGN);
        expect(clicked.commands).toEqual([]);
    });

    test('un niveau cliqué fait ce que fait son chiffre', () => {
        for (const level of [1, 2, 3]) {
            const armed = beginResign(initialKeyState()).state;
            const clicked = resignWithLevel(armed, level);
            const pressed = typed(armed, `Digit${level}`, 'checker');
            expect(clicked.commands, level).toEqual([{ kind: COMMAND.RESIGN, value: level }]);
            expect(clicked.commands, level).toEqual(pressed.commands);
            expect(clicked.state, level).toEqual(pressed.state);
        }
    });

    test('« Annuler » rend l’état d’avant, comme Échap, sans rien enregistrer', () => {
        const before = typed(initialKeyState(), 'Digit4', 'checker').state;
        const armed = beginResign(before).state;
        const clicked = cancelResign(armed);
        const pressed = pressKey(armed, key('Escape'), { expects: 'checker' });
        expect(clicked.commands).toEqual([]);
        expect(clicked.state).toEqual(before);
        expect(clicked.state).toEqual(pressed.state);
    });

    // Un niveau qui n'en est pas un ne crée rien : la rangée n'en offre que
    // trois, mais la fonction est publique et le dit elle-même.
    test('un niveau hors de 1-2-3 ne produit aucun geste', () => {
        const armed = beginResign(initialKeyState()).state;
        for (const level of [0, 4, -1]) {
            expect(resignWithLevel(armed, level).commands, String(level)).toEqual([]);
        }
    });
});

describe('le menu contextuel du Transcript vaut sa relecture au clavier', () => {
    // ux.md §4.3, ligne « coup en double » : `h`×k puis `x`. Le menu fait les
    // deux d'un clic droit et d'une entrée choisie, et le moteur reçoit
    // exactement la même suite de gestes.
    test('« supprimer » sur une cellule lointaine = h×k puis x', () => {
        const byMenu = menuCommands(12, 7, COMMAND.DELETE);

        let state = initialKeyState();
        const byKeys = [];
        for (let i = 0; i < 5; i++) {
            const step = pressKey(state, key('KeyH'), { expects: 'checker' });
            state = step.state;
            byKeys.push(...step.commands);
        }
        byKeys.push(...pressKey(state, key('KeyX'), { expects: 'checker' }).commands);

        expect(byMenu).toEqual(byKeys);
        expect(byMenu).toHaveLength(6);
    });

    test('les quatre entrées sont les quatre touches i, a, x, s', () => {
        for (const [kind, code] of [
            [COMMAND.INSERT_BEFORE, 'KeyI'],
            [COMMAND.INSERT_AFTER, 'KeyA'],
            [COMMAND.DELETE, 'KeyX'],
            [COMMAND.FLIP_SIDE, 'KeyS']
        ]) {
            expect(menuCommands(3, 3, kind), kind).toEqual(pressKey(initialKeyState(), key(code), { expects: 'checker' }).commands);
        }
    });

    test('sur la cellule du Cursor, le menu ne déplace rien', () => {
        expect(menuCommands(4, 4, COMMAND.FLIP_SIDE)).toEqual([{ kind: COMMAND.FLIP_SIDE }]);
    });

    test('vers l’avant, le Cursor avance — le menu n’a pas d’autre sens de marche', () => {
        expect(menuCommands(2, 5, COMMAND.DELETE)).toEqual([...cursorCommands(2, 5), { kind: COMMAND.DELETE }]);
    });
});

describe('le budget d’ux.md §4.2, à la souris', () => {
    /** Un pilote qui compte les CLICS, comme celui des touches compte les frappes. */
    function driver({ expects = 'checker' } = {}) {
        let state = initialKeyState();
        const commands = [];
        let clicks = 0;
        return {
            clickCube(kind) {
                clicks += 1;
                const result = cubeGesture(state, kind);
                state = result.state;
                commands.push(...result.commands);
            },
            expects(next) {
                expects = next;
                void expects;
            },
            get clicks() {
                return clicks;
            },
            kinds() {
                return commands.map((c) => c.kind);
            }
        };
    }

    // « H P B B, P B B, H » : la main quitte le clavier, deux clics, la main y
    // revient. Un clic par Action, et rien entre les deux — c'est de là que
    // vient le 3,4 s, et c'est le compte de clics qui le tient.
    test('double puis prise : deux clics, 3,4 s', () => {
        const d = driver();
        d.clickCube(COMMAND.DOUBLE);
        d.expects('take');
        d.clickCube(COMMAND.TAKE);

        expect(d.clicks).toBe(2);
        expect(d.kinds()).toEqual([COMMAND.DOUBLE, COMMAND.TAKE]);

        const seconds = H + d.clicks * CLICK + H;
        expect(seconds).toBeCloseTo(3.4, 2);
    });

    test('double puis passe : le même compte, le même budget', () => {
        const d = driver();
        d.clickCube(COMMAND.DOUBLE);
        d.expects('take');
        d.clickCube(COMMAND.PASS);
        expect(d.clicks).toBe(2);
        expect(H + d.clicks * CLICK + H).toBeCloseTo(3.4, 2);
    });

    // La résignation coûte deux gestes des deux côtés : `r` `2` au clavier,
    // [R] puis le niveau à la souris.
    test('la résignation : deux gestes à la souris, deux touches au clavier', () => {
        let state = initialKeyState();
        let clicks = 0;
        state = beginResign(state).state;
        clicks += 1;
        const done = resignWithLevel(state, 2);
        clicks += 1;
        expect(clicks).toBe(2);
        expect(done.commands).toEqual([{ kind: COMMAND.RESIGN, value: 2 }]);
        expect(H + clicks * CLICK + H).toBeCloseTo(3.4, 2);
    });

    // Le clavier reste deux fois plus rapide : la rangée est une entrée pour la
    // souris, jamais un remplacement (ux.md §4.2, « d t = 2 K = 0,56 s »).
    test('le clavier garde son avance', () => {
        expect(2 * K).toBeLessThan(H + 2 * CLICK + H);
        expect(2 * K).toBeCloseTo(0.56, 2);
    });
});
