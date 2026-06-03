import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const GetBoardColors = vi.fn();
const SaveBoardColors = vi.fn(() => Promise.resolve(undefined));

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetBoardColors: (...args) => GetBoardColors(...args),
    SaveBoardColors: (...args) => SaveBoardColors(...args)
}));

import { boardColorsStore, DEFAULT_BOARD_COLORS, initBoardColors, setBoardColor, resetBoardColors } from '../stores/boardColorsStore.js';

describe('boardColorsStore', () => {
    beforeEach(() => {
        GetBoardColors.mockReset();
        SaveBoardColors.mockClear();
        boardColorsStore.set({ ...DEFAULT_BOARD_COLORS });
    });

    test('defaults match the historical palette', () => {
        expect(DEFAULT_BOARD_COLORS.background).toBe('#f0f0f0');
        expect(DEFAULT_BOARD_COLORS.checker2).toBe('#ffffff');
    });

    test('initBoardColors loads persisted colours', async () => {
        GetBoardColors.mockResolvedValueOnce({ ...DEFAULT_BOARD_COLORS, background: '#000000' });
        await initBoardColors();
        expect(get(boardColorsStore).background).toBe('#000000');
    });

    test('initBoardColors fills missing/blank fields with defaults', async () => {
        GetBoardColors.mockResolvedValueOnce({ background: '#111111', border: '' });
        await initBoardColors();
        const colors = get(boardColorsStore);
        expect(colors.background).toBe('#111111');
        expect(colors.border).toBe(DEFAULT_BOARD_COLORS.border); // blank -> default
        expect(colors.checker1).toBe(DEFAULT_BOARD_COLORS.checker1); // missing -> default
    });

    test('initBoardColors falls back to defaults on error', async () => {
        GetBoardColors.mockRejectedValueOnce(new Error('boom'));
        await initBoardColors();
        expect(get(boardColorsStore)).toEqual(DEFAULT_BOARD_COLORS);
    });

    test('setBoardColor updates one key and persists the whole palette', () => {
        setBoardColor('checker1', '#ff0000');
        expect(get(boardColorsStore).checker1).toBe('#ff0000');
        expect(SaveBoardColors).toHaveBeenCalledTimes(1);
        expect(SaveBoardColors.mock.calls[0][0].checker1).toBe('#ff0000');
    });

    test('setBoardColor ignores unknown keys', () => {
        setBoardColor('bogus', '#ff0000');
        expect(SaveBoardColors).not.toHaveBeenCalled();
    });

    test('resetBoardColors restores and persists defaults', () => {
        boardColorsStore.set({ ...DEFAULT_BOARD_COLORS, background: '#123456' });
        resetBoardColors();
        expect(get(boardColorsStore)).toEqual(DEFAULT_BOARD_COLORS);
        expect(SaveBoardColors).toHaveBeenCalledWith(DEFAULT_BOARD_COLORS);
    });
});
