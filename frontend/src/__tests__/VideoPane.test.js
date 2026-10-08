/**
 * VideoPane.test.js
 *
 * The pane serves a file through the loopback URL, answers the caller's seek, and says why a
 * file does not play (its container, the webview's verdict, the packages to install) instead
 * of leaving a black frame.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

const gui = vi.hoisted(() => ({ kind: 'file', mediaError: null }));

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    VideoSourceKind: vi.fn(() => Promise.resolve(gui.kind)),
    MediaURL: vi.fn(() => (gui.mediaError ? Promise.reject(gui.mediaError) : Promise.resolve('http://127.0.0.1:1/m/tok'))),
    YouTubeEmbedURL: vi.fn(() => Promise.resolve('http://127.0.0.1:1/yt/abc')),
    PickTranscriptionVideo: vi.fn(() => Promise.resolve('/new/place.mp4')),
    ReleaseMedia: vi.fn(() => Promise.resolve())
}));

import { MediaURL, ReleaseMedia } from '../../wailsjs/go/gui/App.js';
import VideoPane from '../components/VideoPane.svelte';

describe('VideoPane', () => {
    beforeEach(() => {
        gui.kind = 'file';
        gui.mediaError = null;
    });
    afterEach(cleanup);

    test('closing releases its own URL, a new source releases the old one', async () => {
        vi.mocked(MediaURL).mockClear();
        vi.mocked(ReleaseMedia).mockClear();
        vi.mocked(MediaURL).mockResolvedValueOnce('http://127.0.0.1:1/media/one').mockResolvedValueOnce('http://127.0.0.1:1/media/two');
        const { container, rerender, unmount } = render(VideoPane, { props: { source: '/v/a.mp4' } });
        await vi.waitFor(() => expect(container.querySelector('video')?.getAttribute('src')).toBe('http://127.0.0.1:1/media/one'));
        await rerender({ source: '/v/b.mp4' });
        await vi.waitFor(() => expect(ReleaseMedia).toHaveBeenCalledWith('http://127.0.0.1:1/media/one'));
        await vi.waitFor(() => expect(container.querySelector('video')?.getAttribute('src')).toBe('http://127.0.0.1:1/media/two'));
        unmount();
        expect(ReleaseMedia).toHaveBeenLastCalledWith('http://127.0.0.1:1/media/two');
    });

    test('an answer that arrives after the pane closed is released at once', async () => {
        vi.mocked(ReleaseMedia).mockClear();
        let answer;
        vi.mocked(MediaURL).mockImplementationOnce(() => new Promise((r) => (answer = r)));
        const { unmount } = render(VideoPane, { props: { source: '/v/late.mp4' } });
        await vi.waitFor(() => expect(answer).toBeDefined());
        unmount();
        answer('http://127.0.0.1:1/media/late');
        await vi.waitFor(() => expect(ReleaseMedia).toHaveBeenCalledWith('http://127.0.0.1:1/media/late'));
    });

    test('a relocated file is read at the instant that was asked', async () => {
        const { container, component, rerender } = render(VideoPane, { props: { source: '/v/a.mp4', startMs: 64000 } });
        let video = await vi.waitFor(() => container.querySelector('video') ?? expect.fail('no video'));
        await fireEvent(video, new Event('loadedmetadata'));
        component.seek(64000);
        await rerender({ source: '/v/moved.mp4' });
        video = await vi.waitFor(() => container.querySelector('video') ?? expect.fail('no video'));
        await fireEvent(video, new Event('loadedmetadata'));
        expect(video.currentTime).toBe(64);
    });

    test('a file plays through a <video> on the loopback URL and seeks in milliseconds', async () => {
        const { container, component } = render(VideoPane, { props: { source: '/v/final.mp4' } });
        const video = await vi.waitFor(() => {
            const el = container.querySelector('video');
            expect(el).not.toBeNull();
            return el;
        });
        expect(video.getAttribute('src')).toBe('http://127.0.0.1:1/m/tok');
        expect(component.currentTimeMs()).toBeNull();
        await fireEvent(video, new Event('loadedmetadata'));
        component.seek(12500);
        expect(video.currentTime).toBe(12.5);
        expect(component.currentTimeMs()).toBe(12500);
    });

    test('a file the webview cannot decode is diagnosed by its container', async () => {
        const { container } = render(VideoPane, { props: { source: '/v/final.mkv' } });
        const video = await vi.waitFor(() => container.querySelector('video') ?? expect.fail('no video'));
        video.canPlayType = () => '';
        await fireEvent(video, new Event('error'));
        const note = await vi.waitFor(() => container.querySelector('[data-testid="video-diagnostic"]') ?? expect.fail('no diagnostic'));
        expect(note.textContent).toContain('Matroska');
        expect(note.textContent).toContain('gst-libav');
        // The packages are named per distribution: Debian's and Fedora's names differ from Arch's.
        expect(note.textContent).toContain('gstreamer1.0-libav');
        expect(note.textContent).toContain('gstreamer1-plugin-libav');
    });

    test('a missing file offers to relocate it', async () => {
        gui.mediaError = new Error('no such file');
        const onrelocate = vi.fn();
        const { container } = render(VideoPane, { props: { source: '/gone.mp4', onrelocate } });
        const note = await vi.waitFor(() => container.querySelector('[data-testid="video-missing"]') ?? expect.fail('not missing'));
        await fireEvent.click(note.querySelector('button'));
        await vi.waitFor(() => expect(onrelocate).toHaveBeenCalledWith('/new/place.mp4'));
    });

    test('a YouTube source is an iframe on the hosted player page', async () => {
        gui.kind = 'youtube';
        const { container } = render(VideoPane, { props: { source: 'https://youtu.be/dQw4w9WgXcQ' } });
        const frame = await vi.waitFor(() => container.querySelector('iframe') ?? expect.fail('no iframe'));
        // The page's origin rides along: a webview that sends no referrer leaves the hosted
        // page nothing else to post back to.
        expect(frame.getAttribute('src')).toBe(`http://127.0.0.1:1/yt/abc?origin=${encodeURIComponent(window.location.origin)}`);
    });
});
