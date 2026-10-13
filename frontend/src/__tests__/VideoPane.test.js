/**
 * VideoPane.test.js
 *
 * The pane serves a file through the loopback URL, answers the caller's seek, and says why a
 * file does not play (its container, the webview's verdict, the packages to install) instead
 * of leaving a black frame.
 */

import { must } from './helpers/must.js';
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { flushSync } from 'svelte';

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
        expect(must(video).getAttribute('src')).toBe('http://127.0.0.1:1/m/tok');
        expect(component.currentTimeMs()).toBeNull();
        await fireEvent(must(video), new Event('loadedmetadata'));
        component.seek(12500);
        expect(must(video).currentTime).toBe(12.5);
        expect(component.currentTimeMs()).toBe(12500);
    });

    test('seeks asked in a burst wait for the one under way, stay within the file and survive no number', async () => {
        const { container, component } = render(VideoPane, { props: { source: '/v/final.mp4' } });
        const video = /** @type {HTMLVideoElement} */ (
            await vi.waitFor(() => {
                const el = container.querySelector('video');
                expect(el).not.toBeNull();
                return el;
            })
        );
        let seeking = false;
        Object.defineProperty(video, 'seeking', { get: () => seeking });
        Object.defineProperty(video, 'duration', { get: () => 60 });
        /** @type {number[]} */
        const applied = [];
        let current = 0;
        Object.defineProperty(video, 'currentTime', {
            get: () => current,
            set: (/** @type {number} */ v) => {
                applied.push(v);
                current = v;
                seeking = true;
            }
        });
        await fireEvent(video, new Event('loadedmetadata'));
        component.seek(10000);
        // Shift+Right held down: each press reads the instant and asks five seconds more.
        for (let i = 0; i < 12; i++) component.seek(/** @type {number} */ (component.currentTimeMs()) + 5000);
        component.seek(NaN);
        expect(applied).toEqual([10]);
        expect(component.currentTimeMs()).toBe(60000);
        seeking = false;
        await fireEvent(video, new Event('seeked'));
        expect(applied).toEqual([10, 60]);
        seeking = false;
        await fireEvent(video, new Event('seeked'));
        expect(applied).toEqual([10, 60]);
        component.seek(-3000);
        expect(applied).toEqual([10, 60, 0]);
    });

    /**
     * A <video> whose pipeline is driven by hand: seeking, paused, the instant, the speed and
     * the data at hand. A speed that changes fires ratechange, as WebKit does.
     *
     * @param {HTMLVideoElement} video
     */
    function drive(video) {
        const el = { seeking: false, paused: true, muted: false, current: 0, rate: 1, ready: 4, loads: 0, plays: 0, applied: /** @type {number[]} */ ([]) };
        const setRate = (/** @type {number} */ v) => {
            if (v === el.rate) return;
            el.rate = v;
            // WebKitGTK unmutes the element on a speed change.
            if (el.muted) {
                el.muted = false;
                video.dispatchEvent(new Event('volumechange'));
            }
            video.dispatchEvent(new Event('ratechange'));
        };
        Object.defineProperty(video, 'muted', {
            get: () => el.muted,
            set: (/** @type {boolean} */ v) => {
                el.muted = v;
            }
        });
        Object.defineProperty(video, 'seeking', { get: () => el.seeking });
        Object.defineProperty(video, 'paused', { get: () => el.paused });
        Object.defineProperty(video, 'ended', { get: () => false });
        Object.defineProperty(video, 'readyState', { get: () => el.ready });
        Object.defineProperty(video, 'duration', { get: () => 3600 });
        Object.defineProperty(video, 'currentTime', {
            get: () => el.current,
            set: (/** @type {number} */ v) => {
                el.applied.push(v);
                el.current = v;
                el.seeking = true;
            }
        });
        Object.defineProperty(video, 'playbackRate', { get: () => el.rate, set: setRate });
        video.load = () => {
            el.loads++;
            el.seeking = false;
            el.paused = true;
            setRate(1);
        };
        video.play = () => {
            el.plays++;
            el.paused = false;
            return Promise.resolve();
        };
        video.pause = () => {
            el.paused = true;
        };
        return el;
    }

    describe('a stalled file is reloaded', () => {
        beforeEach(() => vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval', 'setTimeout', 'clearTimeout', 'Date', 'performance'] }));
        afterEach(() => vi.useRealTimers());

        /** @returns {Promise<{ container: HTMLElement, component: any, video: HTMLVideoElement }>} */
        async function open() {
            const { container, component } = render(VideoPane, { props: { source: '/v/round2.webm' } });
            const video = /** @type {HTMLVideoElement} */ (await vi.waitFor(() => container.querySelector('video') ?? expect.fail('no video')));
            return { container, component, video };
        }

        test('a seek that never lands reloads the source and goes to the last instant asked, at its speed, playing', async () => {
            const { component, video } = await open();
            const el = drive(video);
            await fireEvent(video, new Event('loadedmetadata'));
            el.seeking = false;
            el.paused = false;
            component.stepRate(1);
            component.seek(600000);
            component.seek(720000);
            expect(el.applied).toEqual([600]);
            vi.advanceTimersByTime(2500);
            expect(el.loads).toBe(0);
            vi.advanceTimersByTime(1500);
            expect(el.loads).toBe(1);
            expect(video.getAttribute('src')).toBe('http://127.0.0.1:1/m/tok');
            // Asked while the source reloads: kept for the moment the metadata is in.
            component.seek(750000);
            expect(component.currentTimeMs()).toBe(750000);
            expect(el.applied).toEqual([600]);
            await fireEvent(video, new Event('loadedmetadata'));
            expect(el.applied).toEqual([600, 750]);
            expect(el.rate).toBe(1.25);
            expect(el.plays).toBe(1);
        });

        test('a paused seek that never lands comes back paused', async () => {
            const { component, video } = await open();
            const el = drive(video);
            await fireEvent(video, new Event('loadedmetadata'));
            el.seeking = false;
            component.seek(90000);
            vi.advanceTimersByTime(4000);
            expect(el.loads).toBe(1);
            await fireEvent(video, new Event('loadedmetadata'));
            expect(el.applied).toEqual([90, 90]);
            expect(el.plays).toBe(0);
        });

        test('playback frozen on one instant reloads there; playback that moves, or a pause, does not', async () => {
            const { video } = await open();
            const el = drive(video);
            await fireEvent(video, new Event('loadedmetadata'));
            el.paused = false;
            el.current = 30;
            for (let i = 0; i < 10; i++) {
                vi.advanceTimersByTime(1000);
                el.current += 1;
            }
            el.paused = true;
            vi.advanceTimersByTime(10000);
            expect(el.loads).toBe(0);
            el.paused = false;
            vi.advanceTimersByTime(4000);
            expect(el.loads).toBe(1);
            await fireEvent(video, new Event('loadedmetadata'));
            expect(el.applied).toEqual([40]);
            expect(el.plays).toBe(1);
        });

        test('playback waiting for data is given longer before it counts as frozen', async () => {
            const { video } = await open();
            const el = drive(video);
            await fireEvent(video, new Event('loadedmetadata'));
            el.paused = false;
            el.current = 30;
            el.ready = 2;
            vi.advanceTimersByTime(12000);
            expect(el.loads).toBe(0);
            vi.advanceTimersByTime(4000);
            expect(el.loads).toBe(1);
        });

        test('a seek that fails at the same instant every time gives up after three reloads, each waiting longer, and offers to retry', async () => {
            const { container, component, video } = await open();
            const el = drive(video);
            await fireEvent(video, new Event('loadedmetadata'));
            el.seeking = false;
            el.paused = false;
            component.seek(1200000);
            const reloadsAfter = (/** @type {number} */ ms) => {
                vi.advanceTimersByTime(ms);
                return el.loads;
            };
            // 3 s, then 6 s, then 9 s: the metadata comes back, the seek never lands.
            expect(reloadsAfter(3500)).toBe(1);
            await fireEvent(video, new Event('loadedmetadata'));
            expect(reloadsAfter(5000)).toBe(1);
            expect(reloadsAfter(1500)).toBe(2);
            await fireEvent(video, new Event('loadedmetadata'));
            expect(reloadsAfter(8000)).toBe(2);
            expect(reloadsAfter(1500)).toBe(3);
            await fireEvent(video, new Event('loadedmetadata'));
            expect(reloadsAfter(30000)).toBe(3);
            flushSync();
            expect(container.querySelector('.video-pane')?.getAttribute('data-status')).toBe('error');
            expect(container.textContent).toContain('frozen');
            // Seeks asked meanwhile are where the retry goes.
            component.seek(1300000);
            await fireEvent.click(must(container.querySelector('[data-testid="video-retry"]')));
            expect(el.loads).toBe(4);
            expect(container.querySelector('[data-testid="video-retry"]')).toBeNull();
            await fireEvent(video, new Event('loadedmetadata'));
            expect(el.applied.at(-1)).toBe(1300);
            expect(el.plays).toBe(4);
        });

        test('a reload that brings no metadata is tried again, then given up', async () => {
            const { container, component, video } = await open();
            const el = drive(video);
            await fireEvent(video, new Event('loadedmetadata'));
            el.seeking = false;
            component.seek(5000);
            vi.advanceTimersByTime(3500);
            expect(el.loads).toBe(1);
            vi.advanceTimersByTime(6500);
            expect(el.loads).toBe(2);
            vi.advanceTimersByTime(9500);
            expect(el.loads).toBe(3);
            vi.advanceTimersByTime(60000);
            expect(el.loads).toBe(3);
            flushSync();
            expect(container.querySelector('[data-testid="video-retry"]')).not.toBeNull();
        });

        test('playback that moves again after a reload starts the count afresh', async () => {
            const { component, video } = await open();
            const el = drive(video);
            await fireEvent(video, new Event('loadedmetadata'));
            el.seeking = false;
            el.paused = false;
            for (let round = 1; round <= 5; round++) {
                component.seek(round * 60000);
                vi.advanceTimersByTime(3500);
                expect(el.loads).toBe(round);
                await fireEvent(video, new Event('loadedmetadata'));
                el.seeking = false;
                for (let i = 0; i < 3; i++) {
                    el.current += 0.5;
                    vi.advanceTimersByTime(500);
                }
            }
        });

        /**
         * The seek a reload asked lands, as WebKit reports it.
         *
         * @param {HTMLVideoElement} video
         * @param {ReturnType<typeof drive>} el
         */
        async function land(video, el) {
            el.seeking = false;
            await fireEvent(video, new Event('seeked'));
        }

        test('playback that lands after each reload but freezes again on the same frame gives up after three', async () => {
            const { container, video } = await open();
            const el = drive(video);
            await fireEvent(video, new Event('loadedmetadata'));
            el.paused = false;
            el.current = 30;
            const waits = [4000, 7000, 10000];
            for (let n = 1; n <= 3; n++) {
                vi.advanceTimersByTime(waits[n - 1]);
                expect(el.loads).toBe(n);
                await fireEvent(video, new Event('loadedmetadata'));
                await land(video, el);
                // One frame drawn past the point is not progress.
                el.current += 0.04;
            }
            vi.advanceTimersByTime(30000);
            expect(el.loads).toBe(3);
            flushSync();
            expect(container.querySelector('[data-testid="video-retry"]')).not.toBeNull();
            // An answer that comes after the pane gave up leaves it as it is.
            await fireEvent(video, new Event('loadedmetadata'));
            flushSync();
            expect(container.querySelector('.video-pane')?.getAttribute('data-status')).toBe('error');
        });

        test('a paused seek that lands after a reload starts the count afresh', async () => {
            const { component, video } = await open();
            const el = drive(video);
            await fireEvent(video, new Event('loadedmetadata'));
            el.seeking = false;
            component.seek(90000);
            vi.advanceTimersByTime(3500);
            expect(el.loads).toBe(1);
            await fireEvent(video, new Event('loadedmetadata'));
            await land(video, el);
            component.seek(120000);
            // Counted afresh: 3 s, not the 6 s a second reload in a row would wait.
            vi.advanceTimersByTime(3500);
            expect(el.loads).toBe(2);
        });

        test('a reload never unmutes a muted video, through the speed it puts back', async () => {
            const { component, video } = await open();
            const el = drive(video);
            await fireEvent(video, new Event('loadedmetadata'));
            el.seeking = false;
            el.paused = false;
            el.muted = true;
            await fireEvent(video, new Event('volumechange'));
            component.stepRate(1);
            expect(el.muted).toBe(true);
            // Past the window a speed gesture opens: what follows is the reload's own doing.
            vi.advanceTimersByTime(1000);
            component.seek(60000);
            vi.advanceTimersByTime(3500);
            expect(el.loads).toBe(1);
            expect(el.muted).toBe(true);
            await fireEvent(video, new Event('loadedmetadata'));
            expect(el.rate).toBe(1.25);
            expect(el.muted).toBe(true);
        });

        test('a speed or a pause asked during a reload is the one kept, and the restored speed shows no indicator', async () => {
            const { container, component, video } = await open();
            const el = drive(video);
            await fireEvent(video, new Event('loadedmetadata'));
            el.seeking = false;
            el.paused = false;
            component.stepRate(1);
            vi.advanceTimersByTime(1500);
            expect(container.querySelector('[data-testid="video-rate-flash"]')).toBeNull();
            component.seek(60000);
            vi.advanceTimersByTime(3500);
            expect(el.loads).toBe(1);
            expect(container.querySelector('[data-testid="video-rate-flash"]')).toBeNull();
            component.stepRate(1);
            component.togglePlay();
            expect(component.playbackRate()).toBe(1.5);
            vi.advanceTimersByTime(1500);
            await fireEvent(video, new Event('loadedmetadata'));
            expect(el.rate).toBe(1.5);
            expect(el.plays).toBe(0);
            await vi.waitFor(() => expect(container.querySelector('[data-testid="video-rate"]')?.textContent).toBe('1.5×'));
            expect(container.querySelector('[data-testid="video-rate-flash"]')).toBeNull();
        });
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
        await fireEvent.click(must(note.querySelector('button')));
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

    test('a speed change shows a large centred indicator that then leaves, the corner badge staying', async () => {
        const { container, component } = render(VideoPane, { props: { source: '/v/final.mp4' } });
        const video = await vi.waitFor(() => container.querySelector('video') ?? expect.fail('no video'));
        await fireEvent(video, new Event('loadedmetadata'));
        expect(container.querySelector('[data-testid="video-rate-flash"]')).toBeNull();
        component.stepRate(1);
        await vi.waitFor(() => expect(container.querySelector('[data-testid="video-rate-flash"]')?.textContent).toBe('1.25×'));
        await vi.waitFor(() => expect(container.querySelector('[data-testid="video-rate-flash"]')).toBeNull(), { timeout: 3000 });
        expect(container.querySelector('[data-testid="video-rate"]')?.textContent).toBe('1.25×');
    });

    test('a muted file stays muted through a speed change that WebKit unmutes', async () => {
        const { container, component } = render(VideoPane, { props: { source: '/v/final.mp4' } });
        const video = await vi.waitFor(() => container.querySelector('video') ?? expect.fail('no video'));
        await fireEvent(video, new Event('loadedmetadata'));
        video.muted = true;
        await fireEvent(video, new Event('volumechange'));
        component.stepRate(1);
        // WebKitGTK drops the mute on the speed change and says so with volumechange.
        video.muted = false;
        await fireEvent(video, new Event('volumechange'));
        await fireEvent(video, new Event('ratechange'));
        expect(video.muted).toBe(true);
    });

    test('an unmute that is not part of a speed change is left alone', async () => {
        const { container } = render(VideoPane, { props: { source: '/v/final.mp4' } });
        const video = await vi.waitFor(() => container.querySelector('video') ?? expect.fail('no video'));
        await fireEvent(video, new Event('loadedmetadata'));
        video.muted = true;
        await fireEvent(video, new Event('volumechange'));
        video.muted = false;
        await fireEvent(video, new Event('volumechange'));
        expect(video.muted).toBe(false);
    });

    test('a file steps its speed by quarters up to 4×, shows it, and holds at the ends', async () => {
        const { container, component } = render(VideoPane, { props: { source: '/v/final.mp4' } });
        const video = await vi.waitFor(() => container.querySelector('video') ?? expect.fail('no video'));
        await fireEvent(video, new Event('loadedmetadata'));
        expect(container.querySelector('[data-testid="video-rate"]')).toBeNull();
        component.stepRate(1);
        await vi.waitFor(() => expect(container.querySelector('[data-testid="video-rate"]')?.textContent).toBe('1.25×'));
        expect(video.playbackRate).toBe(1.25);
        for (let i = 0; i < 20; i++) component.stepRate(1);
        expect(video.playbackRate).toBe(4);
        for (let i = 0; i < 20; i++) component.stepRate(-1);
        expect(video.playbackRate).toBe(0.25);
        expect(component.playbackRate()).toBe(0.25);
    });

    test('a new source starts again at 1×', async () => {
        const { container, component, rerender } = render(VideoPane, { props: { source: '/v/a.mp4' } });
        let video = await vi.waitFor(() => container.querySelector('video') ?? expect.fail('no video'));
        await fireEvent(video, new Event('loadedmetadata'));
        component.stepRate(1);
        await rerender({ source: '/v/b.mp4' });
        video = await vi.waitFor(() => container.querySelector('video') ?? expect.fail('no video'));
        await fireEvent(video, new Event('loadedmetadata'));
        expect(component.playbackRate()).toBe(1);
        expect(video.playbackRate).toBe(1);
    });

    test('a focused <video> hands the focus back to the pane, which owns no key itself', async () => {
        const { container } = render(VideoPane, { props: { source: '/v/final.mp4' } });
        const video = await vi.waitFor(() => container.querySelector('video') ?? expect.fail('no video'));
        video.focus();
        await fireEvent.focus(video);
        expect(document.activeElement).toBe(container.querySelector('[data-testid="video-pane"]'));
    });

    test('YouTube steps only through the speeds its player offers, and shows the one it applied', async () => {
        gui.kind = 'youtube';
        const { container, component } = render(VideoPane, { props: { source: 'https://youtu.be/dQw4w9WgXcQ' } });
        const frame = /** @type {HTMLIFrameElement} */ (await vi.waitFor(() => container.querySelector('iframe') ?? expect.fail('no iframe')));
        const win = /** @type {Window} */ (frame.contentWindow);
        const posted = vi.spyOn(win, 'postMessage').mockImplementation(() => {});
        const say = (/** @type {any} */ data) => window.dispatchEvent(new MessageEvent('message', { data: { source: 'blunderdb-yt', ...data }, origin: 'http://127.0.0.1:1', source: win }));
        say({ type: 'ready', rates: [0.25, 0.5, 0.75, 1, 1.25, 1.5, 1.75, 2] });
        component.stepRate(1);
        expect(posted).toHaveBeenLastCalledWith({ type: 'rate', rate: 1.25 }, 'http://127.0.0.1:1');
        // Nothing shows until the player says what it applied.
        expect(component.playbackRate()).toBe(1);
        say({ type: 'rate', rate: 2 });
        await vi.waitFor(() => expect(container.querySelector('[data-testid="video-rate"]')?.textContent).toBe('2×'));
        posted.mockClear();
        component.stepRate(1);
        expect(posted).not.toHaveBeenCalled();
    });

    test('YouTube: a report sent before a seek reached the player does not pull the instant back', async () => {
        gui.kind = 'youtube';
        const { container, component } = render(VideoPane, { props: { source: 'https://youtu.be/dQw4w9WgXcQ' } });
        const frame = /** @type {HTMLIFrameElement} */ (await vi.waitFor(() => container.querySelector('iframe') ?? expect.fail('no iframe')));
        const win = /** @type {Window} */ (frame.contentWindow);
        const posted = vi.spyOn(win, 'postMessage').mockImplementation(() => {});
        const say = (/** @type {any} */ data) => window.dispatchEvent(new MessageEvent('message', { data: { source: 'blunderdb-yt', ...data }, origin: 'http://127.0.0.1:1', source: win }));
        say({ type: 'ready' });
        say({ type: 'time', time: 5 });
        component.seek(120000);
        expect(posted).toHaveBeenLastCalledWith({ type: 'seek', time: 120 }, 'http://127.0.0.1:1');
        say({ type: 'time', time: 5.25 });
        say({ type: 'state', state: 3, time: 5.25 });
        expect(component.currentTimeMs()).toBe(120000);
        say({ type: 'time', time: 120.25 });
        expect(component.currentTimeMs()).toBe(120250);
        // Past the seek, the player's clock is the truth again, wherever it goes.
        say({ type: 'time', time: 7 });
        expect(component.currentTimeMs()).toBe(7000);
    });

    test('YouTube: the instant asked before the player is ready, or given at opening, is sought on ready', async () => {
        gui.kind = 'youtube';
        const { container, component } = render(VideoPane, { props: { source: 'https://youtu.be/dQw4w9WgXcQ', startMs: 90000 } });
        const frame = /** @type {HTMLIFrameElement} */ (await vi.waitFor(() => container.querySelector('iframe') ?? expect.fail('no iframe')));
        const win = /** @type {Window} */ (frame.contentWindow);
        const posted = vi.spyOn(win, 'postMessage').mockImplementation(() => {});
        const say = (/** @type {any} */ data) => window.dispatchEvent(new MessageEvent('message', { data: { source: 'blunderdb-yt', ...data }, origin: 'http://127.0.0.1:1', source: win }));
        expect(component.currentTimeMs()).toBeNull();
        say({ type: 'ready' });
        expect(posted).toHaveBeenCalledWith({ type: 'seek', time: 90 }, 'http://127.0.0.1:1');
        say({ type: 'time', time: 0 });
        expect(component.currentTimeMs()).toBe(90000);
    });
});
