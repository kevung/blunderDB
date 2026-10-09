/**
 * youtubePlayerPage.test.js — the page the desktop serves around YouTube's IFrame Player
 * (internal/gui/youtube_player.html), run against a fake player: a seek reaches the player
 * whatever its state, without starting one that has not started, and the instant it reports is
 * the one asked.
 */

import { describe, test, expect, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

const html = readFileSync(resolve(process.cwd(), '../internal/gui/youtube_player.html'), 'utf8');
const script = html.match(/<script>([\s\S]*?)<\/script>/)?.[1] ?? '';

/**
 * Loads the page with a fake YT player in `state`, a parent window whose posts are recorded.
 *
 * @param {{ state?: number, search?: string, referrer?: string }} [opts]
 */
function load({ state = 5, search = '', referrer = '' } = {}) {
    /** @type {any[]} */
    const posted = [];
    const parent = { postMessage: (/** @type {any} */ m, /** @type {string} */ target) => posted.push({ m: { ...m }, target }) };
    /** @type {((e: any) => void)[]} */
    const listeners = [];
    /** @type {(() => void)[]} */
    const ticks = [];
    const player = {
        state,
        time: 0,
        /** @type {any} */
        events: null,
        /** @type {any} */
        cfg: null,
        seekTo: vi.fn(function (/** @type {number} */ t) {
            // An unstarted player starts playing on seekTo and keeps reporting 0 for a while.
            if (player.state === -1 || player.state === 5) return;
            player.time = t;
        }),
        pauseVideo: vi.fn(),
        playVideo: vi.fn(() => {
            player.state = 1;
            player.events.onStateChange({ data: 1 });
        }),
        getPlayerState: () => player.state,
        getCurrentTime: () => player.time,
        getDuration: () => 600,
        getAvailablePlaybackRates: () => [1],
        setPlaybackRate: vi.fn(),
        muted: false,
        volume: 100,
        isMuted: () => player.muted,
        getVolume: () => player.volume,
        mute: vi.fn(() => (player.muted = true)),
        unMute: vi.fn(() => (player.muted = false)),
        setVolume: vi.fn((/** @type {number} */ v) => (player.volume = v)),
        getPlaybackRate: () => 1
    };
    const YT = {
        Player: function (/** @type {string} */ _id, /** @type {any} */ cfg) {
            player.events = cfg.events;
            player.cfg = cfg;
            return player;
        }
    };
    const run = new Function('YT', 'parent', 'location', 'document', 'addEventListener', 'setInterval', `${script}\nreturn onYouTubeIframeAPIReady;`);
    const apiReady = run(
        YT,
        parent,
        { search },
        { referrer },
        (/** @type {string} */ _t, /** @type {any} */ fn) => listeners.push(fn),
        (/** @type {any} */ fn) => ticks.push(fn)
    );
    apiReady();
    const say = (/** @type {any} */ data, origin = 'http://wails.localhost') => listeners.forEach((fn) => fn({ data, origin, source: parent }));
    const tick = () => ticks.forEach((fn) => fn());
    const last = (/** @type {string} */ type) => [...posted].reverse().find((p) => p.m.type === type);
    return { player, posted, say, tick, last, ready: () => player.events.onReady() };
}

describe('the hosted YouTube page', () => {
    test('the player has no full screen of its own: the theatre owns the window', () => {
        expect(load().player.cfg.playerVars.fs).toBe(0);
    });

    test.each([
        [-1, 'unstarted'],
        [5, 'cued']
    ])('a seek on a player not started yet (%i, %s) leaves it untouched and reports the instant asked', (state) => {
        const page = load({ state, search: '?origin=http%3A%2F%2Fwails.localhost' });
        page.ready();
        page.say({ type: 'seek', time: 120 });
        // seekTo there autoplays, WebKitGTK refuses it without a gesture, and pausing what
        // never started leaves the player on a black frame that a click no longer starts.
        expect(page.player.seekTo).not.toHaveBeenCalled();
        expect(page.player.pauseVideo).not.toHaveBeenCalled();
        expect(page.player.playVideo).not.toHaveBeenCalled();
        page.tick();
        expect(page.last('time')?.m.time).toBe(120);
        // Play starts from the instant asked, not from 0.
        page.say({ type: 'play' });
        expect(page.player.seekTo).toHaveBeenLastCalledWith(120, true);
    });

    test('a seek asked before the player is ready waits for it', () => {
        const page = load({ state: 2, search: '?origin=http%3A%2F%2Fwails.localhost' });
        page.say({ type: 'seek', time: 42 });
        expect(page.player.seekTo).not.toHaveBeenCalled();
        page.ready();
        expect(page.player.seekTo).toHaveBeenCalledWith(42, true);
        page.tick();
        expect(page.last('time')?.m.time).toBe(42);
    });

    test('a paused player seeks and reports its own clock', () => {
        const page = load({ state: 2, search: '?origin=http%3A%2F%2Fwails.localhost' });
        page.ready();
        page.say({ type: 'seek', time: 30 });
        expect(page.player.pauseVideo).not.toHaveBeenCalled();
        page.player.time = 31;
        page.tick();
        expect(page.last('time')?.m.time).toBe(31);
    });

    test('a parent on a custom scheme is reached and heard, without an origin check that would drop it', () => {
        const page = load({ state: 2, search: '?origin=wails%3A%2F%2Fwails', referrer: 'wails://wails/' });
        page.ready();
        expect(page.last('ready')?.target).toBe('*');
        page.say({ type: 'seek', time: 9 }, 'wails://wails');
        expect(page.player.seekTo).toHaveBeenCalledWith(9, true);
    });

    test('an http parent is targeted by its origin, and another origin is not heard', () => {
        const page = load({ state: 2, search: '?origin=http%3A%2F%2Fwails.localhost' });
        page.ready();
        expect(page.last('ready')?.target).toBe('http://wails.localhost');
        page.say({ type: 'seek', time: 9 }, 'http://evil.example');
        expect(page.player.seekTo).not.toHaveBeenCalled();
    });

    test('a speed change keeps the sound muted and the volume the viewer chose', () => {
        const page = load({ state: 2, search: '?origin=http%3A%2F%2Fwails.localhost' });
        page.ready();
        page.player.muted = true;
        page.player.volume = 40;
        page.tick();
        // The player drops its mute when the speed changes.
        page.player.setPlaybackRate.mockImplementation(() => {
            page.player.muted = false;
            page.player.volume = 100;
        });
        page.say({ type: 'rate', rate: 1.5 });
        expect(page.player.muted).toBe(true);
        expect(page.player.volume).toBe(40);
        page.tick();
        expect(page.player.muted).toBe(true);
    });

    test('the sound the parent restores after a reload is applied once the player is ready', () => {
        const page = load({ state: 2, search: '?origin=http%3A%2F%2Fwails.localhost' });
        page.ready();
        page.say({ type: 'volume', muted: true, volume: 30 });
        expect(page.player.muted).toBe(true);
        expect(page.player.volume).toBe(30);
    });
});
