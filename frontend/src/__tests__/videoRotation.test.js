/**
 * videoRotation.test.js — the angle a draft's video is turned to: a cycle of four, kept per
 * draft on this machine, and applied to the element of a file or of a YouTube frame.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';

const gui = vi.hoisted(() => ({ kind: 'file' }));

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    VideoSourceKind: vi.fn(() => Promise.resolve(gui.kind)),
    MediaURL: vi.fn(() => Promise.resolve('http://127.0.0.1:1/m/tok')),
    YouTubeEmbedURL: vi.fn(() => Promise.resolve('http://127.0.0.1:1/yt/abc')),
    PickTranscriptionVideo: vi.fn(() => Promise.resolve('')),
    ReleaseMedia: vi.fn(() => Promise.resolve())
}));

import VideoPane from '../components/VideoPane.svelte';
import { nextRotation, loadVideoRotation, saveVideoRotation } from '../services/videoRotation.js';

describe('videoRotation service', () => {
    beforeEach(() => localStorage.clear());
    afterEach(() => vi.unstubAllGlobals());

    test('the cycle goes 0, 90, 180, 270 and back', () => {
        expect([0, 90, 180, 270].map(nextRotation)).toEqual([90, 180, 270, 0]);
    });

    test('the angle is kept per draft, upright by default', () => {
        saveVideoRotation(4, 90);
        expect(loadVideoRotation(4)).toBe(90);
        expect(loadVideoRotation(5)).toBe(0);
        saveVideoRotation(4, 0);
        expect(loadVideoRotation(4)).toBe(0);
        expect(localStorage.getItem('blunderdb.transcription.videoRotation.4')).toBeNull();
    });

    test('a refusing or corrupt storage shows the video upright', () => {
        localStorage.setItem('blunderdb.transcription.videoRotation.9', '45');
        expect(loadVideoRotation(9)).toBe(0);
        vi.stubGlobal('localStorage', {
            getItem: () => {
                throw new Error('refused');
            },
            setItem: () => {
                throw new Error('refused');
            },
            removeItem: () => {
                throw new Error('refused');
            }
        });
        expect(loadVideoRotation(9)).toBe(0);
        expect(() => saveVideoRotation(9, 90)).not.toThrow();
    });
});

describe('VideoPane rotation', () => {
    afterEach(cleanup);

    test('a file turns its <video>; sideways, it swaps width and height', async () => {
        gui.kind = 'file';
        const { container } = render(VideoPane, { props: { source: '/v/a.mp4', rotation: 90 } });
        await vi.waitFor(() => expect(container.querySelector('video')).not.toBeNull());
        const video = /** @type {HTMLElement} */ (container.querySelector('video'));
        expect(video.style.getPropertyValue('--turn')).toBe('90deg');
        expect(video.classList.contains('sideways')).toBe(true);
    });

    test('a YouTube frame turns the same way, 180 stays upright in shape', async () => {
        gui.kind = 'youtube';
        const { container } = render(VideoPane, { props: { source: 'yt:abc', rotation: 180 } });
        await vi.waitFor(() => expect(container.querySelector('iframe')).not.toBeNull());
        const frame = /** @type {HTMLElement} */ (container.querySelector('iframe'));
        expect(frame.style.getPropertyValue('--turn')).toBe('180deg');
        expect(frame.classList.contains('sideways')).toBe(false);
    });

    test('without rotation the element is left alone', async () => {
        gui.kind = 'file';
        const { container } = render(VideoPane, { props: { source: '/v/a.mp4' } });
        await vi.waitFor(() => expect(container.querySelector('video')).not.toBeNull());
        expect(container.querySelector('video')?.classList.contains('turned')).toBe(false);
    });
});
