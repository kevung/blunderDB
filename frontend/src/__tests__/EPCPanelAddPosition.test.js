/**
 * EPCPanelAddPosition.test.js — the Eval panel's « Ajouter à la base » (#399).
 *
 * The panel is mounted for real: a prop or a rune clash only shows at mount,
 * and a test that stubs the component would never see it. The save itself is
 * saveScratchBoard()'s, tested in scratchBoardSave.test.js; what is checked
 * here is the button — where it stands, when it is disabled and why.
 */

import { describe, test, expect, vi, beforeAll, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/gui/App.js', async (importOriginal) => ({
    ...(await importOriginal()),
    EvaluatePositionImmediate: vi.fn().mockResolvedValue({ moves: [], cube: null }),
    StartEvaluationAtRest: vi.fn().mockResolvedValue(undefined),
    CancelEvaluationAtRest: vi.fn().mockResolvedValue(undefined)
}));
vi.mock('../../wailsjs/go/main/Config.js', async (importOriginal) => ({
    ...(await importOriginal()),
    GetEpcChallenge: vi.fn().mockResolvedValue(false),
    SaveEpcChallenge: vi.fn().mockResolvedValue(undefined),
    GetGammonNetDisplayPly: vi.fn().mockResolvedValue(2),
    GetGammonNetPruneK: vi.fn().mockResolvedValue(12),
    GetGammonNetCandidates: vi.fn().mockResolvedValue(10)
}));
vi.mock('../../wailsjs/runtime/runtime.js', async (importOriginal) => ({
    ...(await importOriginal()),
    EventsOn: vi.fn(() => () => {}),
    BrowserOpenURL: vi.fn()
}));
const scratch = vi.hoisted(() => ({ saveScratchBoard: vi.fn() }));
vi.mock('../services/scratchBoard.js', () => scratch);

import en from '../i18n/locales/en.json';
import { statusBarModeStore } from '../stores/uiStore.js';
import { positionStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { epcDataStore } from '../stores/epcStore.js';
import { enterEPCMode, exitEPCMode } from '../services/modeMachine.js';
import EPCPanel from '../components/EPCPanel.svelte';

function validBoard() {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    points[6] = { checkers: 5, color: 0 };
    points[19] = { checkers: 5, color: 1 };
    return {
        id: 0,
        board: { points, bearoff: [10, 10] },
        cube: { owner: -1, value: 0 },
        dice: [0, 0],
        score: [-1, -1],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
        has_beaver: 0,
        max_cube: 0
    };
}

/** @param {HTMLElement} container */
const addButton = (container) => /** @type {HTMLButtonElement} */ (container.querySelector('.badges-strip .add-position'));

// The board the Eval panel opens on the first time: the default bearoff,
// fifteen top checkers borne off. Taken before any test runs, because leaving
// the panel remembers the board it was left on (lastEPCBoard).
let defaultBoard;
beforeAll(async () => {
    statusBarModeStore.set('NORMAL');
    await enterEPCMode();
    defaultBoard = JSON.parse(JSON.stringify(get(positionStore)));
    await exitEPCMode();
});

beforeEach(async () => {
    vi.clearAllMocks();
    scratch.saveScratchBoard.mockResolvedValue({ id: 99, existed: false });
    databasePathStore.set('');
    epcDataStore.set({ bottomEPC: null, topEPC: null, race: null, error: null });
    statusBarModeStore.set('NORMAL');
    await enterEPCMode();
});

afterEach(async () => {
    cleanup();
    await exitEPCMode();
    databasePathStore.set('');
});

describe('the Eval panel’s add-to-database button (#399)', () => {
    test('leads the badge strip, with its label', async () => {
        const { container } = render(EPCPanel);
        await tick();

        const strip = container.querySelector('.badges-strip');
        expect(strip.firstElementChild).toBe(addButton(container));
        expect(addButton(container).textContent.trim()).toBe(en.eval.addPosition);
        expect(addButton(container).querySelector('svg')).not.toBeNull();
    });

    test('without a database it is disabled, and says to open one', async () => {
        positionStore.set(validBoard());
        const { container } = render(EPCPanel);
        await tick();

        expect(addButton(container).disabled).toBe(true);
        expect(addButton(container).title).toBe(en.eval.addPositionNoDatabase);
    });

    test('on the default Eval board it is disabled with the refusal the save would give', async () => {
        databasePathStore.set('/tmp/test.db');
        positionStore.set(JSON.parse(JSON.stringify(defaultBoard)));
        const { container } = render(EPCPanel);
        await tick();

        expect(defaultBoard.board.bearoff).toEqual([0, 15]);
        expect(addButton(container).disabled).toBe(true);
        expect(addButton(container).title).toBe(en.status.invalidP2BorneOff);
    });

    test('on a valid board with a database it is enabled, and the click goes to saveScratchBoard', async () => {
        databasePathStore.set('/tmp/test.db');
        positionStore.set(validBoard());
        const { container } = render(EPCPanel);
        await tick();

        expect(addButton(container).disabled).toBe(false);
        expect(addButton(container).title).toBe(en.eval.addPositionTooltip);

        await fireEvent.click(addButton(container));
        expect(scratch.saveScratchBoard).toHaveBeenCalledTimes(1);
    });

    test('it follows the board: a refused edit disables it, the fix enables it again', async () => {
        databasePathStore.set('/tmp/test.db');
        positionStore.set(validBoard());
        const { container } = render(EPCPanel);
        await tick();
        expect(addButton(container).disabled).toBe(false);

        positionStore.update((p) => ({ ...p, board: { ...p.board, points: p.board.points.map((pt, i) => (i === 19 ? { checkers: 16, color: 1 } : pt)) } }));
        await tick();
        expect(addButton(container).disabled).toBe(true);
        expect(addButton(container).title).toBe(en.status.invalidP2Over15);

        positionStore.set(validBoard());
        await tick();
        expect(addButton(container).disabled).toBe(false);
    });

    test('an evaluation error does not take the button away', async () => {
        databasePathStore.set('/tmp/test.db');
        positionStore.set(validBoard());
        epcDataStore.set({ bottomEPC: null, topEPC: null, race: null, error: 'boom' });
        const { container } = render(EPCPanel);
        await tick();

        expect(container.querySelector('.error-text').textContent).toBe('boom');
        expect(addButton(container)).not.toBeNull();
        expect(addButton(container).disabled).toBe(false);
    });
});
