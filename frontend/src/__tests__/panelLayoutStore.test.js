import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const GetPanelPosition = vi.fn();
const SavePanelPosition = vi.fn(() => Promise.resolve(undefined));
const GetPanelHeight = vi.fn();
const SavePanelHeight = vi.fn(() => Promise.resolve(undefined));
const GetPanelWidth = vi.fn();
const SavePanelWidth = vi.fn(() => Promise.resolve(undefined));

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetPanelPosition: (...args) => GetPanelPosition(...args),
    SavePanelPosition: (...args) => SavePanelPosition(...args),
    GetPanelHeight: (...args) => GetPanelHeight(...args),
    SavePanelHeight: (...args) => SavePanelHeight(...args),
    GetPanelWidth: (...args) => GetPanelWidth(...args),
    SavePanelWidth: (...args) => SavePanelWidth(...args)
}));

import { panelHeightStore, panelWidthStore, DEFAULT_PANEL_HEIGHT, DEFAULT_PANEL_WIDTH, initPanelSize, savePanelHeight, savePanelWidth } from '../stores/panelLayoutStore.js';

describe('panelLayoutStore — panel size', () => {
    beforeEach(() => {
        GetPanelHeight.mockReset();
        GetPanelWidth.mockReset();
        SavePanelHeight.mockClear();
        SavePanelWidth.mockClear();
        panelHeightStore.set(DEFAULT_PANEL_HEIGHT);
        panelWidthStore.set(DEFAULT_PANEL_WIDTH);
    });

    test('initPanelSize loads the persisted height and width', async () => {
        GetPanelHeight.mockResolvedValueOnce(520);
        GetPanelWidth.mockResolvedValueOnce(640);
        await initPanelSize();
        expect(get(panelHeightStore)).toBe(520);
        expect(get(panelWidthStore)).toBe(640);
    });

    test('initPanelSize falls back to the defaults on error', async () => {
        GetPanelHeight.mockRejectedValueOnce(new Error('boom'));
        GetPanelWidth.mockRejectedValueOnce(new Error('boom'));
        await initPanelSize();
        expect(get(panelHeightStore)).toBe(DEFAULT_PANEL_HEIGHT);
        expect(get(panelWidthStore)).toBe(DEFAULT_PANEL_WIDTH);
    });

    test('savePanelHeight updates the store and persists', () => {
        savePanelHeight(444);
        expect(get(panelHeightStore)).toBe(444);
        expect(SavePanelHeight).toHaveBeenCalledWith(444);
    });

    test('savePanelWidth updates the store and persists', () => {
        savePanelWidth(600);
        expect(get(panelWidthStore)).toBe(600);
        expect(SavePanelWidth).toHaveBeenCalledWith(600);
    });
});
