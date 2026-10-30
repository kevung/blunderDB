/**
 * VideoCalibration.test.js
 *
 * Four clicks calibrate the board; the calibration is named, kept per draft, drawn as ghost
 * checkers, and the lost view is signalled. Mounted for real: the clicks go through the
 * pointer path, the persistence through localStorage.
 */

import { must } from './helpers/must.js';
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { loadCalibrations, pictureRect, pointerToPicture, freeName } from '../services/videoCalibration.js';
import VideoCalibration from '../components/VideoCalibration.svelte';

// A 400x200 box on a 400x200 picture; the box centre is at (200, 100) on screen.
const BOX = { w: 400, h: 200 };
const video = /** @type {HTMLVideoElement} */ ({ videoWidth: 400, videoHeight: 200 });
/** The corners of a plausible board in the picture, in click order. */
const CORNERS = [
    [40, 150],
    [360, 150],
    [320, 30],
    [80, 30]
];

beforeEach(() => {
    localStorage.clear();
    vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(BOX.w);
    vi.spyOn(HTMLElement.prototype, 'offsetHeight', 'get').mockReturnValue(BOX.h);
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue(
        /** @type {DOMRect} */ ({ left: 0, top: 0, width: BOX.w, height: BOX.h, right: BOX.w, bottom: BOX.h, x: 0, y: 0, toJSON() {} })
    );
});
afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
});

/** @param {HTMLElement | null} el @param {number[]} p */
async function click(el, p) {
    el = must(el);
    await fireEvent.pointerMove(el, { clientX: p[0], clientY: p[1] });
    await fireEvent.pointerDown(el, { clientX: p[0], clientY: p[1] });
}

/** @param {any} r */
async function calibrate(r) {
    await fireEvent.click(r.getByTestId('calibration-start'));
    for (const p of CORNERS) await click(r.getByTestId('calibration-capture'), p);
}

describe('VideoCalibration', () => {
    test('four clicks make a named calibration, drawn as ghosts and kept for the draft', async () => {
        const seen = vi.fn();
        const r = render(VideoCalibration, { props: { calibrationKey: 7, video, oncalibration: seen } });
        expect(r.getByTestId('video-calibration').dataset.state).toBe('none');
        await fireEvent.click(r.getByTestId('calibration-start'));
        expect(r.getByTestId('calibration-step').textContent).toContain('1/4');
        await click(r.getByTestId('calibration-capture'), CORNERS[0]);
        expect(r.getByTestId('calibration-step').textContent).toContain('2/4');
        expect(r.getByTestId('calibration-loupe')).toBeTruthy();
        for (const p of CORNERS.slice(1)) await click(r.getByTestId('calibration-capture'), p);

        expect(r.getByTestId('video-calibration').dataset.state).toBe('ready');
        // 24 points x 2 ghosts + 2 bar halves + 2 trays
        expect(r.getAllByTestId('calibration-ghost')).toHaveLength(52);
        const saved = loadCalibrations(7);
        expect(saved.items).toHaveLength(1);
        expect(saved.items[0].clicks[0]).toEqual([0.1, 0.75]);
        expect(seen).toHaveBeenLastCalledWith(expect.objectContaining({ name: saved.active }));
        // another draft sees nothing
        expect(loadCalibrations(8).items).toHaveLength(0);
    });

    test('the calibration is read back when the pane opens again; a second one is named apart', async () => {
        const first = render(VideoCalibration, { props: { calibrationKey: 3, video } });
        await calibrate(first);
        cleanup();
        const again = render(VideoCalibration, { props: { calibrationKey: 3, video } });
        expect(again.getByTestId('video-calibration').dataset.state).toBe('ready');
        await calibrate(again);
        const saved = loadCalibrations(3);
        expect(saved.items.map((c) => c.name)).toHaveLength(2);
        expect(new Set(saved.items.map((c) => c.name)).size).toBe(2);
        await fireEvent.input(again.getByTestId('calibration-name'), { target: { value: 'Table 2' } });
        await fireEvent.change(again.getByTestId('calibration-name'), { target: { value: 'Table 2' } });
        expect(loadCalibrations(3).active).toBe('Table 2');
        await fireEvent.click(again.getByTestId('calibration-delete'));
        expect(loadCalibrations(3).items).toHaveLength(1);
    });

    test('Escape cancels a capture; undo takes the last click back; degenerate clicks fail', async () => {
        const r = render(VideoCalibration, { props: { calibrationKey: 1, video } });
        await fireEvent.click(r.getByTestId('calibration-start'));
        await click(r.getByTestId('calibration-capture'), CORNERS[0]);
        await fireEvent.click(r.getByTestId('calibration-undo'));
        expect(r.getByTestId('calibration-step').textContent).toContain('1/4');
        await fireEvent.keyDown(window, { key: 'Escape' });
        expect(r.getByTestId('video-calibration').dataset.state).toBe('none');

        await fireEvent.click(r.getByTestId('calibration-start'));
        for (let i = 0; i < 4; i++) await click(r.getByTestId('calibration-capture'), [100, 100]);
        expect(r.getByTestId('calibration-failed')).toBeTruthy();
        expect(loadCalibrations(1).items).toHaveLength(0);
    });

    test('a lost view is signalled and offers the recalibration', async () => {
        const r = render(VideoCalibration, { props: { calibrationKey: 2, video, viewLost: true } });
        expect(r.queryByTestId('calibration-lost')).toBeNull();
        await calibrate(r);
        expect(r.getByTestId('calibration-lost')).toBeTruthy();
        expect(r.getByTestId('calibration-start').classList.contains('alert')).toBe(true);
    });

    test('without a draft nothing is kept', async () => {
        const r = render(VideoCalibration, { props: { calibrationKey: null, video } });
        await calibrate(r);
        expect(r.getByTestId('video-calibration').dataset.state).toBe('ready');
        expect(localStorage.getItem('blunderdb.transcription.videoCalibrations.null')).toBeNull();
    });
});

describe('videoCalibration service', () => {
    test('a corrupt record reads as empty', () => {
        localStorage.setItem('blunderdb.transcription.videoCalibrations.5', '{"items":[{"name":"x","clicks":[[2,0]]}]}');
        expect(loadCalibrations(5)).toEqual({ items: [], active: null });
        localStorage.setItem('blunderdb.transcription.videoCalibrations.5', 'not json');
        expect(loadCalibrations(5).items).toEqual([]);
    });

    test('the picture rect and the pointer follow the rotation', () => {
        const pic = pictureRect(400, 200, 200, 200);
        expect(pic).toEqual({ x: 100, y: 0, w: 200, h: 200 });
        expect(pointerToPicture(0, 0, 0, pic, 400, 200)).toEqual([0.5, 0.5]);
        expect(pointerToPicture(-150, 0, 0, pic, 400, 200)).toBeNull();
        // Turned 90 degrees clockwise, the picture's top is on the screen's right.
        const p = pointerToPicture(50, 0, 90, pictureRect(200, 200, 200, 200), 200, 200);
        expect(p?.[0]).toBeCloseTo(0.5);
        expect(p?.[1]).toBeCloseTo(0.25);
    });

    test('free names', () => {
        expect(freeName([{ name: 'A', clicks: [] }], 'A')).toBe('A 2');
    });
});
