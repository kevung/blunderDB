/**
 * vision.available.pane.test.js — the vision switch seen from the real player and the shortcut
 * table. Cut, the player offers no calibration and a saved one stays unread; on, both are there.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent, screen } from '@testing-library/svelte';

const visionSwitch = vi.hoisted(() => ({ on: false }));
vi.mock('../vision/available.js', () => ({
    get VISION_AVAILABLE() {
        return visionSwitch.on;
    }
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    MediaURL: vi.fn(() => Promise.resolve('http://127.0.0.1:1/media/one')),
    ReleaseMedia: vi.fn(() => Promise.resolve()),
    CheckVideoMux: vi.fn(() => Promise.resolve({ checked: true, faulty: [], command: '' })),
    VideoSourceKind: vi.fn(() => Promise.resolve('file')),
    YouTubeEmbedURL: vi.fn(() => Promise.resolve('')),
    PickTranscriptionVideo: vi.fn(() => Promise.resolve(''))
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardSetText: vi.fn(() => Promise.resolve(true)) }));

import { calibrateBoard } from '../vision/geometry/index.js';
import { saveCalibrations } from '../services/videoCalibration.js';
import VideoPane from '../components/VideoPane.svelte';

const KEY = 21;

/** Mounts the player on a file until its calibration bar may show. */
async function readyPane(/** @type {(c: any) => void} */ oncalibration) {
    const { container } = render(VideoPane, { props: { source: '/v/a.mp4', calibrationKey: KEY, oncalibration } });
    await vi.waitFor(() => expect(container.querySelector('video')?.getAttribute('src')).toBeTruthy());
    await fireEvent(/** @type {HTMLVideoElement} */ (container.querySelector('video')), new Event('loadedmetadata'));
    return container;
}

function saveOne() {
    const clicks = /** @type {[number, number][]} */ ([
        [0.1, 0.75],
        [0.9, 0.75],
        [0.8, 0.15],
        [0.2, 0.15]
    ]);
    const board = calibrateBoard(clicks);
    if (!board.ok || !board.modelToImage) throw new Error('fixture: a plausible board calibrates');
    saveCalibrations(KEY, { items: [{ name: 'a', clicks, modelToImage: Array.from(board.modelToImage) }], active: 'a' });
}

beforeEach(() => {
    localStorage.clear();
});
afterEach(cleanup);

describe('the player', () => {
    test('the switch cut: no calibration bar, a saved calibration reports nothing', async () => {
        visionSwitch.on = false;
        saveOne();
        const seen = vi.fn();
        const container = await readyPane(seen);
        expect(container.querySelector('video')).not.toBeNull();
        expect(screen.queryByTestId('video-calibration')).toBeNull();
        expect(screen.queryByTestId('calibration-start')).toBeNull();
        expect(seen).not.toHaveBeenCalled();
    });

    test('the switch on: the bar shows, the saved calibration is reported', async () => {
        visionSwitch.on = true;
        saveOne();
        const seen = vi.fn();
        await readyPane(seen);
        await vi.waitFor(() => expect(screen.queryByTestId('video-calibration')).not.toBeNull());
        await vi.waitFor(() => expect(seen).toHaveBeenCalledWith(expect.objectContaining({ name: 'a' })));
    });
});

describe('the shortcut table', () => {
    test.each([
        [false, []],
        [true, ['g', 'Shift+G']]
    ])('with the switch at %s, the transcription scope declares %j among g and Shift+G', async (on, expected) => {
        visionSwitch.on = on;
        vi.resetModules();
        const { SHORTCUTS } = await import('../services/shortcutMap.js');
        expect(SHORTCUTS.transcription.keys.filter((/** @type {string} */ k) => k === 'g' || k === 'Shift+G')).toEqual(expected);
    });
});
