import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const GetPanelPosition = vi.fn();
const SavePanelPosition = vi.fn(() => Promise.resolve(undefined));
const GetPanelWidth = vi.fn();
const SavePanelWidth = vi.fn(() => Promise.resolve(undefined));

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetPanelPosition: (...args) => GetPanelPosition(...args),
    SavePanelPosition: (...args) => SavePanelPosition(...args),
    GetPanelWidth: (...args) => GetPanelWidth(...args),
    SavePanelWidth: (...args) => SavePanelWidth(...args)
}));

import { panelWidthStore, DEFAULT_PANEL_WIDTH, initPanelSize, savePanelWidth } from '../stores/panelLayoutStore.js';

describe('panelLayoutStore — panel size', () => {
    beforeEach(() => {
        GetPanelWidth.mockReset();
        SavePanelWidth.mockClear();
        panelWidthStore.set(DEFAULT_PANEL_WIDTH);
    });

    test('initPanelSize loads the persisted width', async () => {
        GetPanelWidth.mockResolvedValueOnce(640);
        await initPanelSize();
        expect(get(panelWidthStore)).toBe(640);
    });

    test('initPanelSize falls back to the defaults on error', async () => {
        GetPanelWidth.mockRejectedValueOnce(new Error('boom'));
        await initPanelSize();
        expect(get(panelWidthStore)).toBe(DEFAULT_PANEL_WIDTH);
    });

    test('savePanelWidth updates the store and persists', () => {
        savePanelWidth(600);
        expect(get(panelWidthStore)).toBe(600);
        expect(SavePanelWidth).toHaveBeenCalledWith(600);
    });
});
