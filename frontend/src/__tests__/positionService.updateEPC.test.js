/**
 * The EPC refresh follows every edit of the Eval board. It clears its own
 * error from the status bar, never the message the edit itself has just
 * posted (a paste, a copy).
 */
import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const ComputeEPCFromPosition = vi.fn();
vi.mock('../../wailsjs/go/database/Database.js', () => ({ ComputeEPCFromPosition: (/** @type {any[]} */ ...a) => ComputeEPCFromPosition(...a) }));

const { updateEPC } = await import('../services/positionService.js');
const { statusBarTextStore } = await import('../stores/uiStore.js');
const { tMsg } = await import('../i18n');

beforeEach(() => {
    ComputeEPCFromPosition.mockReset();
});

describe('updateEPC and the status bar', () => {
    test.each([
        ['with race data', { bottom: { epc: { epc: 8.1 } }, top: null, race: null }],
        ['without', {}]
    ])('%s: a message just posted stays up', async (_, result) => {
        ComputeEPCFromPosition.mockResolvedValue(result);
        statusBarTextStore.set(tMsg('status.positionPastedClipboard'));
        await updateEPC({});
        expect(get(statusBarTextStore)).toEqual(tMsg('status.positionPastedClipboard'));
    });

    test('its own earlier error is cleared once the refresh succeeds', async () => {
        ComputeEPCFromPosition.mockRejectedValueOnce(new Error('boom'));
        await updateEPC({});
        expect(get(statusBarTextStore)).toEqual(tMsg('commands.epcErrorComputing'));
        ComputeEPCFromPosition.mockResolvedValue({});
        await updateEPC({});
        expect(get(statusBarTextStore)).toBe('');
    });
});
