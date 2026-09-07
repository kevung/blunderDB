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

function key(code, extra = {}) {
    const digit = /^(?:Digit|Numpad)([0-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    const produced = digit ? digit[1] : letter ? letter[1].toLowerCase() : code;
    return new KeyboardEvent('keydown', { code, key: produced, ...extra });
}

const CHECKER = { expects: 'checker' };

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

describe('cursorCommands — le clic sur une cellule', () => {
    test('remonte le Transcript d’autant de pas que d’Actions', () => {
        expect(cursorCommands(12, 9)).toEqual([{ kind: COMMAND.CURSOR_BACK }, { kind: COMMAND.CURSOR_BACK }, { kind: COMMAND.CURSOR_BACK }]);
    });

    test('le redescend de même', () => {
        expect(cursorCommands(2, 4)).toEqual([{ kind: COMMAND.CURSOR_FORWARD }, { kind: COMMAND.CURSOR_FORWARD }]);
    });

    test('la cellule où le Cursor est déjà ne demande rien', () => {
        expect(cursorCommands(3, 3)).toEqual([]);
    });
});
