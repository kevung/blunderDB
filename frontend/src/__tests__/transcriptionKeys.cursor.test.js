/**
 * transcriptionKeys.cursor.test.js — la ligne « tout | h/l, ←/→ » d'ux.md §3.
 *
 * Le Cursor est une CELLULE du Transcript et les deux colonnes sont les deux
 * camps : le déplacer d'une Action, c'est passer d'une cellule à l'autre, donc
 * d'un camp à l'autre — d'où l'axe horizontal, quand `j`/`k` gardent le
 * vertical pour la liste des candidats.
 *
 * Ce que ce fichier tient : le geste part de TOUT état (c'est la ligne
 * « tout » du tableau, et la relecture est ce qui gouverne l'usage), il coûte
 * une touche par Action (le budget d'ux.md §4.3, `h`×k), et un clic sur une
 * cellule vaut le même chemin sans le compter en touches.
 */

import { describe, test, expect } from 'vitest';
import { PHASE, COMMAND, initialKeyState, pressKey, cursorDelta, cursorCommands } from '../services/transcriptionKeys.js';

function key(/** @type {string} */ code, extra = {}) {
    const digit = /^(?:Digit|Numpad)([0-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    const produced = digit ? digit[1] : letter ? letter[1].toLowerCase() : code;
    return new KeyboardEvent('keydown', { code, key: produced, ...extra });
}

const CHECKER = { expects: 'checker' };

describe('match gagné : plus rien n’est attendu au bout du document', () => {
    const OVER = { expects: '' };

    test.each(['Digit6', 'KeyD', 'KeyR', 'KeyF', 'KeyT', 'KeyJ'])('%s ne saisit rien', (code) => {
        const result = pressKey(initialKeyState(), key(code), OVER);
        expect(result.handled).toBe(false);
        expect(result.commands).toEqual([]);
        expect(result.state).toEqual(initialKeyState());
    });

    test('la relecture reste', () => {
        expect(pressKey(initialKeyState(), key('KeyH'), OVER).commands).toEqual([{ kind: COMMAND.CURSOR_BACK }]);
    });
});

describe('cursorDelta', () => {
    test.each([
        ['KeyH', -1],
        ['ArrowLeft', -1],
        ['KeyL', 1],
        ['ArrowRight', 1],
        ['KeyJ', 0],
        ['ArrowDown', 0]
    ])('%s vaut %i', (code, delta) => {
        expect(cursorDelta(key(code))).toBe(delta);
    });

    test('une lettre modifiée n’est pas un déplacement : Ctrl+L appartient au dispatcheur', () => {
        expect(cursorDelta(key('KeyL', { ctrlKey: true }))).toBe(0);
    });
});

describe('h/l depuis tout état', () => {
    test.each([
        ['KeyH', COMMAND.CURSOR_BACK],
        ['ArrowLeft', COMMAND.CURSOR_BACK],
        ['KeyL', COMMAND.CURSOR_FORWARD],
        ['ArrowRight', COMMAND.CURSOR_FORWARD]
    ])('%s demande %s', (code, command) => {
        const result = pressKey(initialKeyState(), key(code), CHECKER);
        expect(result.handled).toBe(true);
        expect(result.commands).toEqual([{ kind: command }]);
    });

    test('un jet à moitié tapé n’empêche pas la relecture, et la machine repart à zéro', () => {
        let state = pressKey(initialKeyState(), key('Digit3'), CHECKER).state;
        expect(state.phase).toBe(PHASE.DIE1);

        const result = pressKey(state, key('KeyH'), CHECKER);
        expect(result.commands).toEqual([{ kind: COMMAND.CURSOR_BACK }]);
        // Le panneau réarme la machine sur l'Action visée, dont le moteur
        // recharge les dés et le coup : elle ne garde donc rien du jet abandonné.
        expect(result.state).toEqual(initialKeyState());
    });

    test('même devant une réponse au videau, qui n’est pas une saisie de dés', () => {
        const result = pressKey(initialKeyState(), key('KeyL'), { expects: 'take' });
        expect(result.handled).toBe(true);
        expect(result.commands).toEqual([{ kind: COMMAND.CURSOR_FORWARD }]);
    });

    test('k Actions en arrière coûtent k touches (ux.md §4.3)', () => {
        let state = initialKeyState();
        const commands = [];
        for (let i = 0; i < 5; i++) {
            const result = pressKey(state, key('KeyH'), CHECKER);
            state = result.state;
            commands.push(...result.commands);
        }
        expect(commands.length).toBe(5);
        expect(commands.every((c) => c.kind === COMMAND.CURSOR_BACK)).toBe(true);
    });
});

/** Un document de `n` Actions, le Cursor en `cursor`, sans trou. */
const doc = (/** @type {number} */ n, /** @type {number} */ cursor) => ({ cursor, actions: Array.from({ length: n }, () => ({ kind: 'checker' })) });

describe('cursorCommands — le clic sur une cellule', () => {
    test('un clic lointain est un seul geste, quelle que soit la distance', () => {
        expect(cursorCommands(doc(325, 324), 2)).toEqual([{ kind: COMMAND.CURSOR_TO, index: 2, hole: false }]);
        expect(cursorCommands(doc(325, 0), 325)).toEqual([{ kind: COMMAND.CURSOR_TO, index: 325, hole: false }]);
    });

    test('le trou devant une Action est un arrêt à part', () => {
        const ann = doc(6, 0);
        ann.actions[4] = { kind: 'checker', inconsistencies: [{ kind: 'double_turn' }] };
        expect(cursorCommands(ann, 4)).toEqual([{ kind: COMMAND.CURSOR_TO, index: 4, hole: false }]);
        expect(cursorCommands(ann, 4, true)).toEqual([{ kind: COMMAND.CURSOR_TO, index: 4, hole: true }]);
    });

    test('la cellule où le Cursor est déjà ne demande rien', () => {
        expect(cursorCommands(doc(8, 3), 3)).toEqual([]);
    });
});
