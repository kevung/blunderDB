/**
 * viewStore.evalTab.test.js
 *
 * L'onglet « Variante de #n » (ADR-0086 §10) : une vue neuve reçoit la position en Eval, la vue
 * d'origine garde la sienne, et chaque vue garde son contexte Eval (chemin du retour, dernier
 * plateau brouillon) quand on passe de l'une à l'autre.
 *
 * Stores, viewStore et automate réels ; seules les liaisons Wails et databaseService sont
 * mockées, comme dans modeMachine.test.js.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const loadAnalysisMock = vi.hoisted(() => vi.fn((/** @type {number} */ _id) => Promise.resolve(null)));

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListPositionIDs: vi.fn(() => Promise.resolve([])),
    CountPositions: vi.fn(() => Promise.resolve(0)),
    IndexOfPosition: vi.fn(() => Promise.resolve(-1)),
    LoadAnalysis: loadAnalysisMock,
    LoadPositionView: vi.fn(async (id) => ({ analysis: await loadAnalysisMock(id), comment: '' })),
    ComputeEPCFromPosition: vi.fn(() => Promise.resolve({})),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    GetLastVisitedMatch: vi.fn(() => Promise.resolve(null)),
    GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
    LoadComment: vi.fn(() => Promise.resolve(''))
}));

vi.mock('../services/databaseService.js', () => ({
    setStatusBarMessage: vi.fn(),
    warningMessageStore: { subscribe: vi.fn(), set: vi.fn(), update: vi.fn() }
}));

vi.mock('../services/sessionService.js', () => ({
    saveSessionState: vi.fn()
}));

/**
 * @param {number} id
 * @param {number[]} [bearoff]
 * @returns {any} a domain.Position, loosely: the stores type their own shape
 */
function makePosition(id, bearoff = [3, 3]) {
    return {
        id,
        board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff },
        cube: { owner: -1, value: 1 },
        dice: [3, 1],
        score: [5, 5],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
        has_beaver: 0
    };
}

// Fresh modules per test: viewStore and the machine's slots are singletons.
async function setup() {
    vi.resetModules();
    const ui = await import('../stores/uiStore.js');
    const ps = await import('../stores/positionStore.js');
    const { viewStore } = await import('../stores/viewStore.js');
    const machine = await import('../services/modeMachine.js');
    // Wires viewStore to the machine's Eval slots, as at start-up.
    await import('../services/positionService.js');
    (await import('../stores/databaseStore.js')).databasePathStore.set('/fake/db.sqlite');

    const lib = [makePosition(1), makePosition(2), makePosition(3)];
    ps.positionsStore.set(lib);
    ps.positionStore.set(structuredClone(lib[1]));
    ui.currentPositionIndexStore.set(1);
    ui.statusBarModeStore.set('NORMAL');
    ui.activeTabStore.set('analysis');

    // App.svelte's tab effect, which a store test does not mount.
    const openEvalTab = () => machine.enterEvalMode();
    /** Board menu « Évaluer dans un nouvel onglet ». */
    const evaluateInNewView = () => {
        const position = get(ps.positionStore);
        const id = viewStore.addView({ name: (_id, originId) => `Variante de #${originId}` });
        machine.sendPositionToEval(position);
        return { id, entered: openEvalTab() };
    };
    return { ui, ps, viewStore, machine, openEvalTab, evaluateInNewView };
}

/** @param {any} position */
const bearoffOf = (position) => position?.board?.bearoff;

beforeEach(() => {
    vi.clearAllMocks();
});

describe('viewStore — onglet « Variante de #n »', () => {
    test('la vue neuve s’appelle « Variante de #n », s’ouvre en Eval ; la vue d’origine garde sa position', async () => {
        const { ui, ps, viewStore, evaluateInNewView } = await setup();

        const { id, entered } = await evaluateInNewView();
        await entered;

        expect(id).toBe(2);
        expect(get(viewStore.views).map((v) => v.name)).toEqual(['#1', 'Variante de #1']);
        expect(get(ui.statusBarModeStore)).toBe('EVAL');
        expect(get(ps.positionStore).id).toBe(0);

        viewStore.switchTo(1);
        expect(get(ui.statusBarModeStore)).toBe('NORMAL');
        expect(get(ps.positionStore).id).toBe(2);
        expect(get(ui.currentPositionIndexStore)).toBe(1);
        expect(await ps.listedIds()).toEqual([1, 2, 3]);
    });

    test('revenir à la variante retrouve son plateau, pas celui qu’Eval a laissé dans une autre vue', async () => {
        const { ui, ps, viewStore, machine, openEvalTab, evaluateInNewView } = await setup();

        await (
            await evaluateInNewView()
        ).entered;
        ps.positionStore.set(makePosition(0, [7, 7]));

        // La vue d'origine ouvre Eval à son tour et y laisse un autre plateau.
        viewStore.switchTo(1);
        await openEvalTab();
        expect(bearoffOf(get(ps.positionStore))).not.toEqual([7, 7]);
        ps.positionStore.set(makePosition(0, [9, 9]));
        await machine.exitEvalMode();
        expect(get(ps.positionStore).id).toBe(2);

        viewStore.switchTo(2);
        expect(get(ui.statusBarModeStore)).toBe('EVAL');
        expect(get(ui.activeTabStore)).toBe('eval');
        expect(bearoffOf(get(ps.positionStore))).toEqual([7, 7]);

        // Et la vue d'origine rouvre Eval sur son propre brouillon.
        viewStore.switchTo(1);
        await openEvalTab();
        expect(bearoffOf(get(ps.positionStore))).toEqual([9, 9]);
    });

    test('quitter Eval dans la variante revient à la position étudiée, pas au brouillon', async () => {
        const { ui, ps, viewStore, machine, evaluateInNewView } = await setup();

        await (
            await evaluateInNewView()
        ).entered;
        ps.positionStore.set(makePosition(0, [7, 7]));
        viewStore.switchTo(1);
        viewStore.switchTo(2);

        await machine.exitEvalMode();
        expect(get(ui.statusBarModeStore)).toBe('NORMAL');
        expect(get(ps.positionStore).id).toBe(2);
        expect(await ps.listedIds()).toEqual([1, 2, 3]);
    });

    test('deux vues en Eval : chacune garde son brouillon et son chemin du retour', async () => {
        const { ui, ps, viewStore, machine, openEvalTab, evaluateInNewView } = await setup();

        await (
            await evaluateInNewView()
        ).entered;
        ps.positionStore.set(makePosition(0, [7, 7]));
        viewStore.switchTo(1);
        ui.currentPositionIndexStore.set(2);
        ps.positionStore.set(makePosition(3));
        await openEvalTab();
        ps.positionStore.set(makePosition(0, [9, 9]));

        viewStore.switchTo(2);
        expect(bearoffOf(get(ps.positionStore))).toEqual([7, 7]);
        viewStore.switchTo(1);
        expect(get(ui.statusBarModeStore)).toBe('EVAL');
        expect(bearoffOf(get(ps.positionStore))).toEqual([9, 9]);

        await machine.exitEvalMode();
        expect(get(ps.positionStore).id).toBe(3);
    });
});
