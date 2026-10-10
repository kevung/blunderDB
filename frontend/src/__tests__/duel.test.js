/**
 * Le Duel côté bureau : le formulaire devenu `duel.Settings`, les horloges lues dans l'état de
 * l'Arbitre, et ce que le plateau rejoue du Bot.
 */
import { must } from './helpers/must.js';
import { describe, test, expect } from 'vitest';
import {
    normalizeForm,
    settingsFromForm,
    clockView,
    formatClock,
    framesBetween,
    humanSide,
    levelLabelParts,
    DEFAULT_FORM,
    START,
    CRAWFORD,
    POST_CRAWFORD,
    cadenceChoices,
    chooseCadence,
    saveCadence,
    deleteCadence,
    awayChoices,
    formFromBoard,
    boardFromForm,
    boardCubeDecision
} from '../services/duel.js';

const CADENCES = [
    { name: 'standard', reservePerPoint: 120, delay: 12 },
    { name: 'speed', reservePerPoint: 24, delay: 10 }
];

describe('normalizeForm', () => {
    test('completes and clamps a remembered form', () => {
        const f = normalizeForm({ matchLength: 99, side: 7, level: 'god', away: [0, 40], timeOut: 'x' });
        expect(f.matchLength).toBe(25);
        expect(f.side).toBe(0);
        expect(f.level).toBe(DEFAULT_FORM.level);
        expect(f.away).toEqual([0, 25]);
        expect(f.timeOut).toBe('continue');
        expect(f.record).toBe(true);
    });

    test('survives garbage', () => {
        expect(normalizeForm('nope')).toEqual(normalizeForm(null));
    });
});

describe('settingsFromForm', () => {
    test('the player and the Bot on the chosen Sides', () => {
        const set = settingsFromForm({ ...DEFAULT_FORM, side: 1, player: ' Kévin ', level: 'thorough' }, CADENCES);
        expect(set.sides).toEqual([
            { kind: 'bot', level: 'thorough' },
            { kind: 'external', name: 'Kévin' }
        ]);
        expect(set.matchLength).toBe(7);
        expect(set.start).toBeUndefined();
        expect(set.cadence).toBeUndefined();
    });

    test('a money session keeps Jacoby, a match never', () => {
        expect(settingsFromForm({ ...DEFAULT_FORM, money: true, jacoby: true }, []).jacoby).toBe(true);
        expect(settingsFromForm({ ...DEFAULT_FORM, money: true }, []).matchLength).toBe(0);
        expect(settingsFromForm({ ...DEFAULT_FORM, money: false, jacoby: true }, []).jacoby).toBe(false);
    });

    test('a time control is its two numbers: minutes per point, seconds of delay', () => {
        const set = settingsFromForm({ ...chooseCadence(DEFAULT_FORM, 'speed', CADENCES), timeOut: 'lose_match' }, CADENCES);
        expect(set.cadence).toEqual({ name: 'speed', reservePerPoint: 24, delay: 10, timeOut: 'lose_match' });
        const edited = settingsFromForm({ ...chooseCadence(DEFAULT_FORM, 'standard', CADENCES), minutesPerPoint: 1.5 }, CADENCES);
        expect(edited.cadence).toEqual({ name: '', reservePerPoint: 90, delay: 12, timeOut: 'continue' });
        expect(settingsFromForm({ ...chooseCadence(DEFAULT_FORM, 'speed', CADENCES), money: true }, CADENCES).cadence).toBeUndefined();
    });

    test('the Start: the board without its id, at the form score, its roll and cube choices', () => {
        const board = { id: 42, board: { points: [] }, score: [3, 3], dice: [6, 5] };
        const form = { ...DEFAULT_FORM, start: START.BOARD, away: [3, 3] };
        expect(settingsFromForm(form, [], board).start).toEqual({ board: { points: [] }, score: [3, 3], dice: [6, 5] });
        expect(settingsFromForm(form, [], board).reroll).toBeUndefined();
        expect(settingsFromForm({ ...form, reroll: true, afterCube: true }, [], board)).toMatchObject({ reroll: true, afterCube: true });
    });

    test('the opening at a score sends the Away scores, the start of the match none', () => {
        expect(settingsFromForm({ ...DEFAULT_FORM, matchLength: 5, away: [2, 5] }, []).away).toEqual([2, 5]);
        expect(settingsFromForm({ ...DEFAULT_FORM, matchLength: 5, away: [5, 5] }, []).away).toBeUndefined();
        expect(settingsFromForm({ ...DEFAULT_FORM, matchLength: 5, away: [5, 5] }, []).start).toBeUndefined();
    });

    test('a single game', () => {
        expect(settingsFromForm({ ...DEFAULT_FORM, singleGame: true }, []).singleGame).toBe(true);
        expect(settingsFromForm(DEFAULT_FORM, []).singleGame).toBeUndefined();
    });

    test('a record left unticked throws the draft away at the end', () => {
        expect(settingsFromForm({ ...DEFAULT_FORM, record: false }, []).discardAtEnd).toBe(true);
    });
});

describe('time controls', () => {
    test('two presets: standard 2 min a point + 12 s, speed 0.4 min a point + 10 s', () => {
        const presets = cadenceChoices(DEFAULT_FORM, []);
        expect(presets.map((c) => [c.name, c.minutesPerPoint, c.delay])).toEqual([
            ['standard', 2, 12],
            ['speed', 0.4, 10]
        ]);
    });

    test('one saved under a name comes back with the form, and is deleted', () => {
        let form = { ...chooseCadence(DEFAULT_FORM, 'standard', CADENCES), minutesPerPoint: 1, delay: 8 };
        form = saveCadence(form, ' Club ');
        expect(form.cadence).toBe('Club');
        const again = normalizeForm(JSON.parse(JSON.stringify(form)));
        expect(cadenceChoices(again, CADENCES).find((c) => c.name === 'Club')).toMatchObject({ minutesPerPoint: 1, delay: 8, preset: false });
        expect(chooseCadence(again, 'Club', CADENCES)).toMatchObject({ minutesPerPoint: 1, delay: 8 });
        const gone = deleteCadence(again, 'Club');
        expect(gone.cadence).toBe('');
        expect(cadenceChoices(gone, CADENCES).some((c) => c.name === 'Club')).toBe(false);
    });

    test('a preset name is not taken over; an old preset is forgotten', () => {
        expect(saveCadence(DEFAULT_FORM, 'speed').cadences).toEqual([]);
        expect(normalizeForm({ cadence: 'tournament' }).cadence).toBe('standard');
        expect(normalizeForm({ cadence: 'rapid-3+12' }).cadence).toBe('');
    });
});

describe('the score between the board and the fields', () => {
    test('the Away scores a length offers end with the two sentinels', () => {
        expect(awayChoices(3)).toEqual([3, 2, CRAWFORD, POST_CRAWFORD]);
        expect(awayChoices(1)).toEqual([CRAWFORD]);
    });

    test('the board fills the fields: a score, Crawford included, or money', () => {
        const form = { ...DEFAULT_FORM, matchLength: 5, away: [5, 5] };
        expect(formFromBoard(form, { score: [CRAWFORD, 4] })).toMatchObject({ money: false, matchLength: 5, away: [CRAWFORD, 4] });
        expect(formFromBoard(form, { score: [POST_CRAWFORD, 9] })).toMatchObject({ matchLength: 9, away: [POST_CRAWFORD, 9] });
        expect(formFromBoard(form, { score: [-1, -1] }).money).toBe(true);
        expect(formFromBoard(form, { score: [5, 5] })).toBe(form);
    });

    test('the fields go onto the board, each Away score to its player', () => {
        const pos = { score: [7, 7], player_on_roll: 1 };
        expect(boardFromForm(pos, { ...DEFAULT_FORM, away: [2, 6] }).score).toEqual([2, 6]);
        expect(boardFromForm(pos, { ...DEFAULT_FORM, money: true }).score).toEqual([-1, -1]);
        expect(boardFromForm(pos, { ...DEFAULT_FORM, away: [7, 7] })).toBe(pos);
    });

    test('the cube choice shows on a cube decision only', () => {
        const form = { ...DEFAULT_FORM, start: START.BOARD, away: [5, 5] };
        const centred = { cube: { owner: -1, value: 0 }, player_on_roll: 0, decision_type: 0, dice: [0, 0] };
        expect(boardCubeDecision(centred, form)).toBe(true);
        expect(boardCubeDecision({ ...centred, dice: [3, 1] }, form)).toBe(false);
        expect(boardCubeDecision({ ...centred, dice: [3, 1] }, { ...form, reroll: true })).toBe(true);
        expect(boardCubeDecision({ ...centred, cube: { owner: 1, value: 1 } }, form)).toBe(false);
        expect(boardCubeDecision({ ...centred, decision_type: 1, cube: { owner: 1, value: 1 } }, form)).toBe(true);
        expect(boardCubeDecision(centred, { ...form, away: [CRAWFORD, 4] })).toBe(false);
        expect(boardCubeDecision(centred, { ...form, start: START.OPENING })).toBe(false);
    });
});

describe('clockView', () => {
    const since = '2026-10-05T10:00:00.000Z';
    const t0 = Date.parse(since);
    const state = {
        awaiting: { side: 1, since },
        clock: { cadence: { delay: 12 }, reserve: [180000, 180000], turn: 0, spent: 0 }
    };

    test('the delay is free, then the running reserve goes down', () => {
        const at5 = clockView(state, t0 + 5000);
        expect(must(at5).reserve).toEqual([180000, 180000]);
        expect(must(at5).delayLeft).toBe(7000);
        const at20 = clockView(state, t0 + 20000);
        expect(must(at20).reserve).toEqual([180000, 172000]);
        expect(must(at20).running).toBe(1);
    });

    test('no Cadence, no clock; an ended Duel runs no clock', () => {
        expect(clockView({ awaiting: state.awaiting }, t0)).toBeNull();
        expect(must(clockView({ ...state, ended: { matchId: 1 } }, t0 + 60000)).running).toBe(-1);
    });

    test('formatClock', () => {
        expect(formatClock(172000)).toBe('2:52');
        expect(formatClock(-1500)).toBe('−0:02');
    });
});

describe('framesBetween and humanSide', () => {
    test('only the other Side’s checker plays are replayed', () => {
        const before = { actions: [{ side: 0, kind: 'checker', has_position: true }] };
        const after = {
            actions: [...before.actions, { side: 1, kind: 'double', has_position: true }, { side: 1, kind: 'checker', has_position: true }, { side: 0, kind: 'checker', has_position: true }]
        };
        expect(framesBetween(before, after, 0)).toEqual([{ side: 1, kind: 'checker', has_position: true }]);
    });

    test('the player is the external Side', () => {
        expect(humanSide({ sides: [{ kind: 'bot' }, { kind: 'external' }] })).toBe(1);
        expect(humanSide({ sides: [{ kind: 'external' }, { kind: 'bot' }] })).toBe(0);
    });
});

describe('levelLabelParts', () => {
    const levels = [
        { name: 'instant', ply: 0, pruneK: 0 },
        { name: 'normal', ply: 2, pruneK: 12 }
    ];
    test('takes the depth from the offer and says when the search is pruned', () => {
        expect(levelLabelParts('instant', levels)).toEqual({ key: 'duel.levelPly', params: { name: 'instant', ply: 0 } });
        expect(levelLabelParts('normal', levels)).toEqual({ key: 'duel.levelPruned', params: { name: 'normal', ply: 2 } });
    });
    test('keeps the bare name when the offer does not describe the level', () => {
        expect(levelLabelParts('thorough', levels).key).toBeNull();
        expect(levelLabelParts('thorough', undefined).key).toBeNull();
    });
});
