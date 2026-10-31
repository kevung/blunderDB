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
import { loadCalibrations, saveCalibrations, pictureRect, pointerToPicture, freeName, overlayInFrame } from '../services/videoCalibration.js';
import { closeOnEscape } from '../services/escapeService.js';
import VideoCalibration from '../components/VideoCalibration.svelte';

// The pane is 400x200 and its centre (200, 100) on screen. A video turned by 90 or 270 degrees
// is as wide as the pane is high, so the element is 200x400 then.
const PANE = { w: 400, h: 200 };
const layout = { box: { w: 400, h: 200 }, media: { w: 400, h: 200 }, rotation: 0 };
const video = { videoWidth: 400, videoHeight: 200 };
/** Fractions of the picture: a plausible board, in click order. */
const CORNERS = [
    [0.1, 0.75],
    [0.9, 0.75],
    [0.8, 0.15],
    [0.2, 0.15]
];

beforeEach(() => {
    localStorage.clear();
    layout.box = { w: 400, h: 200 };
    layout.media = { w: 400, h: 200 };
    layout.rotation = 0;
    video.videoWidth = 400;
    video.videoHeight = 200;
    vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockImplementation(() => layout.box.w);
    vi.spyOn(HTMLElement.prototype, 'offsetHeight', 'get').mockImplementation(() => layout.box.h);
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue(
        /** @type {DOMRect} */ ({ left: 0, top: 0, width: PANE.w, height: PANE.h, right: PANE.w, bottom: PANE.h, x: 0, y: 0, toJSON() {} })
    );
});
afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
});

/**
 * Where a fraction of the picture lies on screen: the picture inside its element
 * (object-fit: contain), the element turned clockwise about the pane's centre.
 * Written the other way round from the component, so a sign or a unit error shows.
 * @param {number[]} f
 */
function onScreen(f) {
    const { box, media, rotation } = layout;
    const s = Math.min(box.w / media.w, box.h / media.h);
    const w = media.w * s;
    const h = media.h * s;
    const lx = (box.w - w) / 2 + f[0] * w - box.w / 2;
    const ly = (box.h - h) / 2 + f[1] * h - box.h / 2;
    const a = (rotation * Math.PI) / 180;
    return [PANE.w / 2 + lx * Math.cos(a) - ly * Math.sin(a), PANE.h / 2 + lx * Math.sin(a) + ly * Math.cos(a)];
}

/** @param {HTMLElement | null} el @param {number[]} p screen position */
async function click(el, p) {
    el = must(el);
    await fireEvent.pointerMove(el, { clientX: p[0], clientY: p[1] });
    await fireEvent.pointerDown(el, { clientX: p[0], clientY: p[1], button: 0 });
}

/** @param {any} r */
async function calibrate(r) {
    await fireEvent.click(r.getByTestId('calibration-start'));
    for (const p of CORNERS) await click(r.getByTestId('calibration-capture'), onScreen(p));
}

describe('VideoCalibration', () => {
    test('four clicks make a named calibration, drawn as ghosts and kept for the draft', async () => {
        const seen = vi.fn();
        const r = render(VideoCalibration, { props: { calibrationKey: 7, video: /** @type {any} */ (video), oncalibration: seen } });
        expect(r.getByTestId('video-calibration').dataset.state).toBe('none');
        await fireEvent.click(r.getByTestId('calibration-start'));
        expect(r.getByTestId('calibration-step').textContent).toContain('1/4');
        await click(r.getByTestId('calibration-capture'), onScreen(CORNERS[0]));
        expect(r.getByTestId('calibration-step').textContent).toContain('2/4');
        expect(r.getByTestId('calibration-loupe')).toBeTruthy();
        for (const p of CORNERS.slice(1)) await click(r.getByTestId('calibration-capture'), onScreen(p));

        expect(r.getByTestId('video-calibration').dataset.state).toBe('ready');
        // the opening position: 15 checkers a side
        expect(r.getAllByTestId('calibration-ghost')).toHaveLength(30);
        const saved = loadCalibrations(7);
        expect(saved.items).toHaveLength(1);
        expect(saved.items[0].modelToImage).toHaveLength(9);
        expect(saved.items[0].clicks[0]).toEqual(CORNERS[0]);
        expect(seen).toHaveBeenLastCalledWith(expect.objectContaining({ name: saved.active }));
        // another draft sees nothing
        expect(loadCalibrations(8).items).toHaveLength(0);
    });

    test('the calibration is read back when the pane opens again; a second one is named apart', async () => {
        const first = render(VideoCalibration, { props: { calibrationKey: 3, video: /** @type {any} */ (video) } });
        await calibrate(first);
        cleanup();
        const again = render(VideoCalibration, { props: { calibrationKey: 3, video: /** @type {any} */ (video) } });
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
        const r = render(VideoCalibration, { props: { calibrationKey: 1, video: /** @type {any} */ (video) } });
        await fireEvent.click(r.getByTestId('calibration-start'));
        await click(r.getByTestId('calibration-capture'), onScreen(CORNERS[0]));
        await fireEvent.click(r.getByTestId('calibration-undo'));
        expect(r.getByTestId('calibration-step').textContent).toContain('1/4');
        await fireEvent.keyDown(window, { key: 'Escape' });
        expect(r.getByTestId('video-calibration').dataset.state).toBe('none');

        await fireEvent.click(r.getByTestId('calibration-start'));
        for (let i = 0; i < 4; i++) await click(r.getByTestId('calibration-capture'), [200, 100]);
        expect(r.getByTestId('calibration-failed')).toBeTruthy();
        expect(loadCalibrations(1).items).toHaveLength(0);
    });

    test('a lost view is signalled and offers the recalibration', async () => {
        const r = render(VideoCalibration, { props: { calibrationKey: 2, video: /** @type {any} */ (video), viewLost: true } });
        expect(r.queryByTestId('calibration-lost')).toBeNull();
        await calibrate(r);
        expect(r.getByTestId('calibration-lost')).toBeTruthy();
        expect(r.getByTestId('calibration-start').classList.contains('alert')).toBe(true);
    });

    test('without a draft nothing is kept', async () => {
        const r = render(VideoCalibration, { props: { calibrationKey: null, video: /** @type {any} */ (video) } });
        await calibrate(r);
        expect(r.getByTestId('video-calibration').dataset.state).toBe('ready');
        expect(localStorage.getItem('blunderdb.transcription.videoCalibrations.null')).toBeNull();
    });
});

describe('the overlay zone', () => {
    // Fractions exact in binary, so the expected rectangle is written in the literal.
    const ZONE = [
        [0.25, 0.5],
        [0.75, 0.75]
    ];
    /** @param {any} r @param {number[][]} [corners] */
    async function trace(r, corners = ZONE) {
        await fireEvent.click(r.getByTestId('calibration-overlay'));
        for (const p of corners) await click(r.getByTestId('calibration-capture'), onScreen(p));
    }

    test('two clicks trace the zone on the active calibration; it is kept, drawn and handed to the caller', async () => {
        const seen = vi.fn();
        const r = render(VideoCalibration, { props: { calibrationKey: 11, video: /** @type {any} */ (video), oncalibration: seen } });
        await calibrate(r);
        expect(r.queryByTestId('calibration-overlay-zone')).toBeNull();
        await fireEvent.click(r.getByTestId('calibration-overlay'));
        expect(r.getByTestId('calibration-overlay-step').textContent).toContain('1/2');
        await click(r.getByTestId('calibration-capture'), onScreen(ZONE[0]));
        expect(r.getByTestId('calibration-overlay-step').textContent).toContain('2/2');
        await click(r.getByTestId('calibration-capture'), onScreen(ZONE[1]));

        expect(r.getByTestId('video-calibration').dataset.state).toBe('ready');
        expect(loadCalibrations(11).items[0].overlay).toEqual({ x: 0.25, y: 0.5, w: 0.5, h: 0.25 });
        expect(seen).toHaveBeenLastCalledWith(expect.objectContaining({ overlay: { x: 0.25, y: 0.5, w: 0.5, h: 0.25 } }));
        const zone = r.getByTestId('calibration-overlay-zone');
        expect([zone.getAttribute('x'), zone.getAttribute('y'), zone.getAttribute('width'), zone.getAttribute('height')]).toEqual(['100', '100', '200', '50']);
    });

    test('the corners may be given in any order', async () => {
        const r = render(VideoCalibration, { props: { calibrationKey: 12, video: /** @type {any} */ (video) } });
        await calibrate(r);
        await trace(r, [ZONE[1], ZONE[0]]);
        expect(loadCalibrations(12).items[0].overlay).toEqual({ x: 0.25, y: 0.5, w: 0.5, h: 0.25 });
    });

    test('forgetting the zone removes it and the caller no longer receives one', async () => {
        const seen = vi.fn();
        const r = render(VideoCalibration, { props: { calibrationKey: 13, video: /** @type {any} */ (video), oncalibration: seen } });
        await calibrate(r);
        await trace(r);
        await fireEvent.click(r.getByTestId('calibration-overlay-clear'));
        expect(loadCalibrations(13).items[0].overlay).toBeUndefined();
        expect(seen.mock.lastCall?.[0]).not.toHaveProperty('overlay');
        expect(r.queryByTestId('calibration-overlay-clear')).toBeNull();
    });

    test('Escape drops a half-traced zone; a zone under the hash grid is refused', async () => {
        const r = render(VideoCalibration, { props: { calibrationKey: 14, video: /** @type {any} */ (video) } });
        await calibrate(r);
        await fireEvent.click(r.getByTestId('calibration-overlay'));
        await click(r.getByTestId('calibration-capture'), onScreen(ZONE[0]));
        await fireEvent.keyDown(window, { key: 'Escape' });
        expect(r.getByTestId('video-calibration').dataset.state).toBe('ready');
        expect(loadCalibrations(14).items[0].overlay).toBeUndefined();

        // 3×3 px of a 400×200 picture is 2×3 px of the 320×180 frame
        await trace(r, [
            [0.5, 0.5],
            [0.5075, 0.515]
        ]);
        expect(r.getByTestId('overlay-too-small')).toBeTruthy();
        expect(loadCalibrations(14).items[0].overlay).toBeUndefined();
    });

    test('a zone belongs to its calibration: a new one starts without', async () => {
        const r = render(VideoCalibration, { props: { calibrationKey: 15, video: /** @type {any} */ (video) } });
        await calibrate(r);
        await trace(r);
        await calibrate(r);
        const [first, second] = loadCalibrations(15).items;
        expect(first.overlay).toBeDefined();
        expect(second.overlay).toBeUndefined();
    });

    test('a stored zone outside the picture or empty is dropped on reading', () => {
        const base = loadCalibrations(0);
        expect(base.items).toHaveLength(0);
        const cal = { name: 'a', clicks: CORNERS.map((c) => [c[0], c[1]]), modelToImage: [1, 0, 0, 0, 1, 0, 0, 0, 1] };
        const read = (/** @type {any} */ overlay) => {
            localStorage.clear();
            saveCalibrations(5, { items: [{ ...cal, ...(overlay ? { overlay } : {}) }], active: 'a' });
            return loadCalibrations(5).items[0]?.overlay;
        };
        expect(read({ x: 0.5, y: 0.5, w: 0.5, h: 0.5 })).toEqual({ x: 0.5, y: 0.5, w: 0.5, h: 0.5 });
        expect(read({ x: 0.5, y: 0.5, w: 0.6, h: 0.1 })).toBeUndefined();
        expect(read({ x: 0.5, y: 0.5, w: 0.1, h: 0.6 })).toBeUndefined();
        expect(read({ x: -0.1, y: 0.1, w: 0.1, h: 0.1 })).toBeUndefined();
        expect(read({ x: 0.1, y: 0.1, w: 0, h: 0.1 })).toBeUndefined();
        expect(read({ x: 0.1, y: 0.1, w: 0.1, h: 0 })).toBeUndefined();
        expect(read({ x: 0.1, y: 0.1, w: 'a', h: 0.1 })).toBeUndefined();
    });

    test('whole pixels of the reduced frame, outer ones included, and nothing under 9×8', () => {
        expect(overlayInFrame({ x: 0.25, y: 0.5, w: 0.5, h: 0.25 }, 320, 180)).toEqual({ x: 80, y: 90, w: 160, h: 45 });
        // 100.5 → 100, 157.5 → 158
        expect(overlayInFrame({ x: 100.5 / 320, y: 0.5, w: 49.75 / 320, h: 67.5 / 180 }, 320, 180)).toEqual({ x: 100, y: 90, w: 51, h: 68 });
        expect(overlayInFrame({ x: 0.5, y: 0.5, w: 9 / 320, h: 8 / 180 }, 320, 180)).toMatchObject({ w: 9, h: 8 });
        expect(overlayInFrame({ x: 0.5, y: 0.5, w: 8 / 320, h: 8 / 180 }, 320, 180)).toBeNull();
        expect(overlayInFrame({ x: 0.5, y: 0.5, w: 9 / 320, h: 7 / 180 }, 320, 180)).toBeNull();
        expect(overlayInFrame(undefined, 320, 180)).toBeNull();
    });
});

/** @param {string} points */
function centroid(points) {
    const q = points.split(' ').map((v) => v.split(',').map(Number));
    return [q.reduce((a, b) => a + b[0], 0) / q.length, q.reduce((a, b) => a + b[1], 0) / q.length];
}

describe('VideoCalibration geometry', () => {
    /** @type {[string, number, [number, number], [number, number]][]} */
    const CASES = [
        ['upright', 0, [400, 200], [400, 200]],
        ['letterboxed', 0, [400, 200], [200, 200]],
        ['turned 90', 90, [200, 400], [400, 200]],
        ['turned 180', 180, [400, 200], [400, 200]],
        ['turned 270', 270, [200, 400], [400, 200]],
        ['turned 90, letterboxed', 90, [200, 400], [100, 300]]
    ];
    test.each(CASES)('%s: the clicks land on the picture fractions that were aimed at', async (_name, rotation, box, media) => {
        layout.rotation = rotation;
        layout.box = { w: box[0], h: box[1] };
        layout.media = { w: media[0], h: media[1] };
        video.videoWidth = media[0];
        video.videoHeight = media[1];
        const r = render(VideoCalibration, { props: { calibrationKey: 9, video: /** @type {any} */ (video), rotation } });
        await calibrate(r);
        const saved = loadCalibrations(9).items[0];
        saved.clicks.forEach((c, i) => {
            expect(c[0]).toBeCloseTo(CORNERS[i][0], 5);
            expect(c[1]).toBeCloseTo(CORNERS[i][1], 5);
        });
    });

    test('a click outside the picture is ignored', async () => {
        layout.media = { w: 200, h: 200 };
        video.videoWidth = 200;
        video.videoHeight = 200;
        const r = render(VideoCalibration, { props: { calibrationKey: 9, video: /** @type {any} */ (video) } });
        await fireEvent.click(r.getByTestId('calibration-start'));
        await click(r.getByTestId('calibration-capture'), [50, 100]);
        expect(r.getByTestId('calibration-step').textContent).toContain('1/4');
    });

    test('the homography handed over maps the model corners to the clicks', async () => {
        const seen = vi.fn();
        const r = render(VideoCalibration, { props: { calibrationKey: 4, video: /** @type {any} */ (video), oncalibration: seen } });
        await calibrate(r);
        const c = must(seen.mock.calls.at(-1))[0];
        // model corners (0,0), (13,0), (13,13), (0,13): 1-point, 12-point, 13-point, 24-point
        const model = [
            [0, 0],
            [13, 0],
            [13, 13],
            [0, 13]
        ];
        model.forEach((m, i) => {
            const w = c.modelToImage[6] * m[0] + c.modelToImage[7] * m[1] + c.modelToImage[8];
            expect((c.modelToImage[0] * m[0] + c.modelToImage[1] * m[1] + c.modelToImage[2]) / w).toBeCloseTo(CORNERS[i][0], 5);
            expect((c.modelToImage[3] * m[0] + c.modelToImage[4] * m[1] + c.modelToImage[5]) / w).toBeCloseTo(CORNERS[i][1], 5);
        });
        const id = c.imageToModel;
        const w0 = id[6] * CORNERS[2][0] + id[7] * CORNERS[2][1] + id[8];
        expect((id[0] * CORNERS[2][0] + id[1] * CORNERS[2][1] + id[2]) / w0).toBeCloseTo(13, 4);
    });

    test('the ghosts are the opening position, where the board says they are', async () => {
        // an upright rectangle: the model-to-picture map is then a plain scale and shift
        const rect = [
            [0.1, 0.75],
            [0.9, 0.75],
            [0.9, 0.15],
            [0.1, 0.15]
        ];
        const r = render(VideoCalibration, { props: { calibrationKey: 5, video: /** @type {any} */ (video) } });
        await fireEvent.click(r.getByTestId('calibration-start'));
        for (const p of rect) await click(r.getByTestId('calibration-capture'), onScreen(p));
        const ghosts = r.getAllByTestId('calibration-ghost');
        expect(ghosts.filter((g) => g.dataset.side === 'near')).toHaveLength(15);
        expect(ghosts.filter((g) => g.dataset.side === 'far')).toHaveLength(15);
        // pixels of the 400x200 picture: u in 0..13 -> x 40..360, v in 0..13 -> y 150..30
        const px = (/** @type {number} */ u, /** @type {number} */ v) => /** @type {[number, number]} */ ([40 + (u * 320) / 13, 150 - (v * 120) / 13]);
        const at = (/** @type {[number, number]} */ p) => ghosts.filter((g) => Math.hypot(...centroid(must(g.getAttribute('points'))).map((c, i) => c - p[i])) < 0.6);
        // the bottom player's lone 24-point checkers: the first against the near... far edge
        expect(at(px(0.5, 12.5)).map((g) => g.dataset.side)).toEqual(['near']);
        // his 6-point stack of five rises from the near edge, the 1-point's two checkers are the opponent's
        expect(at(px(5.5, 0.5)).map((g) => g.dataset.side)).toEqual(['near']);
        expect(at(px(5.5, 4.5))).toHaveLength(1);
        expect(at(px(0.5, 1.5)).map((g) => g.dataset.side)).toEqual(['far']);
        expect(at(px(0.5, 2.5))).toHaveLength(0);
        // the same mirrored left to right holds no checker: the picture tells the orientation
        expect(at(px(7.5, 0.5))).toHaveLength(0);
    });

    test('only the primary button places a click', async () => {
        const r = render(VideoCalibration, { props: { calibrationKey: 6, video: /** @type {any} */ (video) } });
        await fireEvent.click(r.getByTestId('calibration-start'));
        await fireEvent.pointerDown(must(r.getByTestId('calibration-capture')), { clientX: 200, clientY: 100, button: 2 });
        expect(r.getByTestId('calibration-step').textContent).toContain('1/4');
    });

    test('Escape closes the topmost overlay first, then the capture', async () => {
        const other = vi.fn();
        const r = render(VideoCalibration, { props: { calibrationKey: 1, video: /** @type {any} */ (video) } });
        await fireEvent.click(r.getByTestId('calibration-start'));
        const off = closeOnEscape(other);
        await fireEvent.keyDown(window, { key: 'Escape' });
        expect(other).toHaveBeenCalledTimes(1);
        expect(r.getByTestId('video-calibration').dataset.state).toBe('capturing');
        off();
        await fireEvent.keyDown(window, { key: 'Escape' });
        expect(r.getByTestId('video-calibration').dataset.state).toBe('none');
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
        expect(freeName([{ name: 'A', clicks: [], modelToImage: [] }], 'A')).toBe('A 2');
    });
});
