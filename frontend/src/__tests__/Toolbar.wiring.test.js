/**
 * Toolbar.svelte imports the singleton service actions itself (it used to
 * receive them as two dozen props from App.svelte). This test mounts the
 * toolbar alone and clicks every button in DOM order, checking each one
 * reaches the service action it stands for — the only thing the move could
 * have broken.
 */
import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';

const stub = vi.hoisted(() => (names) => Object.fromEntries(names.map((n) => [n, vi.fn()])));
vi.mock('../services/databaseService.js', () => stub(['newDatabase', 'openDatabase', 'exitApp']));
vi.mock('../services/importService.js', () => stub(['importDatabase', 'importPosition', 'importFolder', 'pastePosition']));
vi.mock('../services/exportService.js', () => stub(['exportDatabase']));
vi.mock('../services/clipboardService.js', () => stub(['copyPosition', 'copyBoardImage']));
vi.mock('../services/positionService.js', () =>
    stub([
        'saveCurrentPosition',
        'updatePosition',
        'deletePosition',
        'firstPosition',
        'previousPosition',
        'nextPosition',
        'lastPosition',
        'gotoPosition',
        'togglePipcount',
        'loadRandomPosition',
        'reloadAllPositions'
    ])
);
vi.mock('../services/keyboardService.js', () => stub(['toggleHelpModal']));

import Toolbar from '../components/Toolbar.svelte';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeModal, activeTabStore, MODAL } from '../stores/uiStore.js';
import * as databaseService from '../services/databaseService.js';
import * as importService from '../services/importService.js';
import * as exportService from '../services/exportService.js';
import * as clipboardService from '../services/clipboardService.js';
import * as positionService from '../services/positionService.js';
import * as keyboardService from '../services/keyboardService.js';

// One entry per toolbar button, in DOM order.
const EXPECTED = [
    databaseService.newDatabase,
    databaseService.openDatabase,
    importService.importDatabase,
    exportService.exportDatabase,
    databaseService.exitApp,
    importService.importPosition,
    importService.importFolder,
    clipboardService.copyPosition,
    importService.pastePosition,
    positionService.saveCurrentPosition,
    positionService.updatePosition,
    positionService.deletePosition,
    positionService.reloadAllPositions,
    positionService.firstPosition,
    positionService.previousPosition,
    positionService.nextPosition,
    positionService.lastPosition,
    positionService.gotoPosition,
    positionService.togglePipcount,
    positionService.loadRandomPosition,
    clipboardService.copyBoardImage,
    MODAL.CONFIG,
    MODAL.TOUR,
    keyboardService.toggleHelpModal
];

beforeEach(() => {
    databasePathStore.set('/tmp/some.db'); // enables the database-bound buttons
    activeTabStore.set('search'); // enables save / update (search tab only)
    activeModal.set(null);
});

afterEach(() => {
    cleanup();
    databasePathStore.set('');
    activeTabStore.set('matches');
    activeModal.set(null);
    vi.clearAllMocks();
});

describe('Toolbar — service wiring', () => {
    test('mounts without props and has one button per action', () => {
        const { container } = render(Toolbar);
        expect(container.querySelectorAll('.toolbar button').length).toBe(EXPECTED.length);
    });

    test.each(EXPECTED.map((target, i) => [i, target]))('button %i reaches its action', async (i, target) => {
        const { container } = render(Toolbar);
        const button = container.querySelectorAll('.toolbar button')[i];
        expect(button.disabled).toBe(false);
        await fireEvent.click(button);
        if (typeof target === 'string') {
            expect(get(activeModal)).toBe(target);
        } else {
            expect(target).toHaveBeenCalledTimes(1);
            const others = EXPECTED.filter((f) => typeof f === 'function' && f !== target);
            others.forEach((f) => expect(f).not.toHaveBeenCalled());
        }
    });
});
