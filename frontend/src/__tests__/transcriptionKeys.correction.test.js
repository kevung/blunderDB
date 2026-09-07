/**
 * transcriptionKeys.correction.test.js — les budgets de correction d'ux.md §4.3,
 * COMPTÉS.
 *
 * Une transcription se corrige constamment : un dé mal lu, un coup oublié, un
 * coup en double, un camp inversé. Le tableau §4.3 chiffre chacun de ces cas en
 * touches, et un chiffre n'est tenu que s'il est mesuré — d'où ce fichier, qui
 * presse les touches une par une à travers `pressKey` (fonction pure) et échoue
 * dès qu'un cas coûte une touche de plus que son budget.
 *
 * Ce qu'il ne mesure pas : le temps. K = 0,28 s est la constante KLM d'ux.md §1,
 * et la conversion est une multiplication, pas une propriété du code. Ce qu'il
 * mesure est le NOMBRE de touches, qui est la seule chose que le logiciel décide.
 */

import { describe, test, expect } from 'vitest';
import { PHASE, COMMAND, initialKeyState, pressKey, applyCandidates } from '../services/transcriptionKeys.js';

function key(code, extra = {}) {
    const digit = /^(?:Digit|Numpad)([0-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    const produced = digit ? digit[1] : letter ? letter[1].toLowerCase() : code;
    return new KeyboardEvent('keydown', { code, key: produced, ...extra });
}

const CHECKER = { expects: 'checker' };

/**
 * Presse une suite de touches et rend ce qui en sort : l'état final, les gestes
 * demandés, et le NOMBRE de touches qui ont porté (une touche ignorée par la
 * machine ne compte pas comme un geste réussi — elle compte comme un échec).
 */
function press(codes, { state = initialKeyState(), context = CHECKER } = {}) {
    const commands = [];
    let count = 0;
    for (const code of codes) {
        const result = pressKey(state, key(code), context);
        expect(result.handled, `la touche ${code} n'a pas été prise par le panneau`).toBe(true);
        state = result.state;
        commands.push(...result.commands);
        count += 1;
    }
    return { state, commands, keys: count };
}

/** Le jet vient de tomber : le moteur a rendu `n` candidats, le premier posé. */
function armed(n = 4) {
    const rolled = press(['Digit6', 'Digit3']).state;
    return applyCandidates(rolled, n).state;
}

const kinds = (commands) => commands.map((c) => c.kind);

describe('ux.md §4.3 — dé mal lu, vu aussitôt', () => {
    test('deux touches, et le jet est repris depuis le premier dé', () => {
        // Le budget compte la CORRECTION, à partir de l'état « jet corrigeable ».
        const { commands, keys } = press(['Digit4', 'Digit1'], { state: armed() });
        expect(keys).toBeLessThanOrEqual(2);
        expect(kinds(commands)).toEqual([COMMAND.DIE, COMMAND.DIE]);
        expect(commands.map((c) => c.value)).toEqual([4, 1]);
    });
});

describe('ux.md §4.3 — candidat voisin, vu aussitôt', () => {
    test('une touche', () => {
        const { state, commands, keys } = press(['KeyJ'], { state: armed() });
        expect(keys).toBeLessThanOrEqual(1);
        expect(kinds(commands)).toEqual([COMMAND.SELECT]);
        expect(state.phase).toBe(PHASE.CANDIDATE);
    });
});

describe('ux.md §4.3 — erreur vue k tours plus tard', () => {
    // Retour arrière (abandonner le jet en cours), `h`×k, `j`/`k`×m, `l`×k.
    test.each([
        [1, 1],
        [5, 1],
        [5, 3]
    ])('k=%i tours, m=%i candidats : 1 + 2k + m touches', (k, m) => {
        const back = Array.from({ length: k }, () => 'KeyH');
        const walk = Array.from({ length: m }, () => 'KeyJ');
        const forward = Array.from({ length: k }, () => 'KeyL');

        // Le Retour arrière part d'un jet à moitié tapé, ce qui est le cas décrit.
        // Seule la touche d'effacement compte dans le budget : le chiffre qui la
        // précède est la saisie fautive, pas la correction.
        const abandoned = press(['Digit4', 'Backspace']);
        const erase = 1;

        const walked = press(back, { state: abandoned.state });
        // Le panneau réarme la machine sur l'Action visée : le moteur a rechargé
        // ses dés et son coup, ce que `applyCandidates` reproduit ici.
        const rearmed = applyCandidates({ ...initialKeyState(), phase: PHASE.ROLL, awaitingCandidates: true }, 4).state;
        const chosen = press(walk, { state: rearmed });
        const home = press(forward, { state: chosen.state });

        const total = erase + walked.keys + chosen.keys + home.keys;
        expect(total).toBeLessThanOrEqual(1 + 2 * k + m);
        expect(kinds(walked.commands)).toEqual(back.map(() => COMMAND.CURSOR_BACK));
        expect(kinds(home.commands)).toEqual(forward.map(() => COMMAND.CURSOR_FORWARD));
        // Le choix a bien changé de candidat, sans quoi le budget serait tenu
        // pour rien.
        expect(kinds(chosen.commands).filter((x) => x === COMMAND.SELECT).length).toBe(m);
    });
});

describe('ux.md §4.3 — coup oublié', () => {
    test.each([
        [1, 0],
        [5, 1]
    ])('k=%i, m=%i : 2k + 3 + m touches', (k, m) => {
        const back = Array.from({ length: k }, () => 'KeyH');
        const forward = Array.from({ length: k }, () => 'KeyL');
        const walk = Array.from({ length: m }, () => 'KeyJ');

        const walked = press(back);
        const inserted = press(['KeyI'], { state: walked.state });
        expect(kinds(inserted.commands)).toEqual([COMMAND.INSERT_BEFORE]);

        const rolled = press(['Digit3', 'Digit1'], { state: inserted.state });
        const chosen = press(walk, { state: applyCandidates(rolled.state, 4).state });
        const home = press(forward, { state: chosen.state });

        const total = walked.keys + inserted.keys + rolled.keys + chosen.keys + home.keys;
        expect(total).toBeLessThanOrEqual(2 * k + 3 + m);
    });

    test('`a` insère derrière, pour le coup oublié qui suit celui du Cursor', () => {
        const { commands } = press(['KeyA']);
        expect(kinds(commands)).toEqual([COMMAND.INSERT_AFTER]);
    });
});

describe('ux.md §4.3 — coup en double, et camp faux', () => {
    test.each([
        ['coup en double', 'KeyX', COMMAND.DELETE],
        ['coup en double (Suppr)', 'Delete', COMMAND.DELETE],
        ['camp faux', 'KeyS', COMMAND.FLIP_SIDE]
    ])('%s : 2k + 1 touches', (_name, code, command) => {
        const k = 5;
        const walked = press(Array.from({ length: k }, () => 'KeyH'));
        const fixed = press([code], { state: walked.state });
        const home = press(
            Array.from({ length: k }, () => 'KeyL'),
            { state: fixed.state }
        );
        expect(walked.keys + fixed.keys + home.keys).toBeLessThanOrEqual(2 * k + 1);
        expect(kinds(fixed.commands)).toEqual([command]);
    });
});

describe('les gestes de correction partent de tout état (ux.md §3, ligne « tout »)', () => {
    const states = {
        'dés attendus': initialKeyState(),
        'un dé saisi': press(['Digit6']).state,
        'jet corrigeable': armed(),
        'candidat choisi': press(['KeyJ'], { state: armed() }).state
    };

    for (const [name, state] of Object.entries(states)) {
        test.each([
            ['KeyI', COMMAND.INSERT_BEFORE],
            ['KeyA', COMMAND.INSERT_AFTER],
            ['KeyX', COMMAND.DELETE],
            ['KeyS', COMMAND.FLIP_SIDE]
        ])(`depuis « ${name} », %s`, (code, command) => {
            const result = pressKey(state, key(code), CHECKER);
            expect(result.handled).toBe(true);
            expect(kinds(result.commands)).toEqual([command]);
            // La machine repart à zéro : le panneau la réarme sur ce que le
            // moteur a rendu, elle ne garde rien du jet abandonné.
            expect(result.state).toEqual(initialKeyState());
        });
    }

    test('mais pas pendant une résignation, qui est modale entre `r` et son niveau', () => {
        const resigning = press(['KeyR']).state;
        expect(resigning.phase).toBe(PHASE.RESIGN);
        const result = pressKey(resigning, key('KeyX'), CHECKER);
        expect(result.handled).toBe(true);
        expect(result.commands).toEqual([]);
    });

    test('une lettre modifiée n’est pas un geste de correction : CTRL-S reste global', () => {
        const result = pressKey(initialKeyState(), key('KeyS', { ctrlKey: true }), CHECKER);
        expect(result.handled).toBe(false);
    });
});

describe('CTRL-Z reste au répartiteur global', () => {
    test('la machine à touches ne la prend pas : une combinaison CTRL est toujours globale', () => {
        expect(pressKey(initialKeyState(), key('KeyZ', { ctrlKey: true }), CHECKER).handled).toBe(false);
        expect(pressKey(initialKeyState(), key('KeyZ', { ctrlKey: true, shiftKey: true }), CHECKER).handled).toBe(false);
    });
});
