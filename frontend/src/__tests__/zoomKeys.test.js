import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const SaveUIScale = vi.fn(() => Promise.resolve());
vi.mock('../../wailsjs/go/main/Config.js', () => ({ GetUIScale: vi.fn(() => Promise.resolve(100)), SaveUIScale }));
const setStatusBarMessage = vi.fn();
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage }));

const { handleZoomKey } = await import('../services/zoomKeys.js');
const { uiScaleStore } = await import('../stores/uiScaleStore.js');

/** @param {Partial<KeyboardEvent>} init */
function press(init) {
    const event = new KeyboardEvent('keydown', { ctrlKey: true, cancelable: true, ...init });
    handleZoomKey(event);
    return event;
}

describe('zoom keys', () => {
    beforeEach(() => {
        uiScaleStore.set(100);
        SaveUIScale.mockClear();
        setStatusBarMessage.mockClear();
    });

    test.each([
        [{ key: '=' }, 110],
        [{ key: '+', shiftKey: true }, 110],
        [{ key: '+', code: 'NumpadAdd' }, 110],
        [{ key: '-' }, 90],
        [{ key: '-', code: 'NumpadSubtract' }, 90]
    ])('%j moves the scale to %i and saves it', (init, expected) => {
        const event = press(init);
        expect(get(uiScaleStore)).toBe(expected);
        expect(SaveUIScale).toHaveBeenCalledWith(expected);
        expect(setStatusBarMessage).toHaveBeenCalledWith(`${expected} %`);
        expect(event.defaultPrevented).toBe(true);
    });

    test('Ctrl+0 returns to 100 %', () => {
        uiScaleStore.set(150);
        press({ key: '0' });
        expect(get(uiScaleStore)).toBe(100);
        expect(SaveUIScale).toHaveBeenCalledWith(100);
    });

    test('bounds hold at 50 and 200', () => {
        uiScaleStore.set(200);
        press({ key: '=' });
        expect(get(uiScaleStore)).toBe(200);
        uiScaleStore.set(50);
        press({ key: '-' });
        expect(get(uiScaleStore)).toBe(50);
        expect(SaveUIScale).not.toHaveBeenCalled();
    });

    test('other keys and a missing Ctrl do nothing', () => {
        expect(press({ key: 'a' }).defaultPrevented).toBe(false);
        expect(press({ key: '=', ctrlKey: false }).defaultPrevented).toBe(false);
        expect(get(uiScaleStore)).toBe(100);
    });
});
