/**
 * VideoStage.test.js
 *
 * A panel's video sits beside the board or inside the panel, one player either way: the dock
 * moves, the panel keeps driving it, and the choice and the width are remembered.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../components/VideoPane.svelte', async () => await import('./fixtures/FakeVideoPane.svelte'));

import Harness from './fixtures/VideoStageHarness.svelte';
import { fakeVideo } from './fixtures/fakeVideo.js';
import { videoPlacementStore, setVideoPlacement, videoStageOwnerStore } from '../stores/videoStageStore.js';

async function settle() {
    for (let i = 0; i < 5; i++) await tick();
}

beforeEach(() => {
    localStorage.clear();
    fakeVideo.reset();
    setVideoPlacement('board');
});
afterEach(cleanup);

describe('the video beside the board', () => {
    test('by default the video splits the main area, left of the board', async () => {
        render(Harness);
        await settle();
        const stage = screen.getByTestId('video-stage');
        expect(stage.contains(screen.getByTestId('video-pane'))).toBe(true);
        expect(screen.getByTestId('panel').contains(screen.getByTestId('video-pane'))).toBe(false);
        expect(screen.getByTestId('panel').dataset.onBoard).toBe('true');
        // Video, separator, then the board.
        const split = /** @type {HTMLElement} */ (stage.parentElement);
        expect([...split.children].map((el) => el.getAttribute('data-testid') ?? el.className.split(' ')[0])).toEqual(['video-stage', 'video-stage-resize', 'board-side']);
    });

    test('without a video the board has the area alone', async () => {
        render(Harness, { props: { withVideo: false } });
        await settle();
        expect(screen.queryByTestId('video-stage')).toBeNull();
        expect(screen.getByTestId('board-stub')).toBeTruthy();
    });

    test('the button puts it back in the panel, the same player, and the choice is remembered', async () => {
        const { component } = render(Harness);
        await settle();
        const pane = screen.getByTestId('video-pane');
        await fireEvent.click(screen.getByTestId('video-placement'));
        await settle();
        expect(screen.queryByTestId('video-stage')).toBeNull();
        expect(screen.getByTestId('panel').contains(pane)).toBe(true);
        expect(screen.getByTestId('video-pane')).toBe(pane);
        expect(get(videoPlacementStore)).toBe('panel');
        expect(localStorage.getItem('blunderdb.video.placement')).toBe('panel');
        // The panel still drives it.
        component.player().togglePlay();
        component.player().seek(5000);
        component.player().stepRate(1);
        expect(fakeVideo.toggles).toBe(1);
        expect(fakeVideo.seeks).toEqual([5000]);
        expect(fakeVideo.rateSteps).toEqual([1]);
        await fireEvent.click(screen.getByTestId('video-placement'));
        await settle();
        expect(screen.getByTestId('video-stage').contains(pane)).toBe(true);
        expect(localStorage.getItem('blunderdb.video.placement')).toBe('board');
    });

    test('closing the video gives the board its area back', async () => {
        const { rerender } = render(Harness);
        await settle();
        await rerender({ withVideo: false });
        await settle();
        expect(get(videoStageOwnerStore)).toBeNull();
        expect(screen.queryByTestId('video-stage')).toBeNull();
        expect(screen.queryByTestId('video-pane')).toBeNull();
    });

    test('the separator sets the width, remembered, and takes no focus', async () => {
        render(Harness);
        await settle();
        const before = document.activeElement;
        const handle = screen.getByTestId('video-stage-resize');
        const down = await fireEvent.pointerDown(handle, { clientX: 100, pointerId: 1 });
        expect(down).toBe(false);
        await fireEvent.pointerMove(handle, { clientX: 220, pointerId: 1 });
        await fireEvent.pointerUp(handle, { clientX: 220, pointerId: 1 });
        await settle();
        expect(screen.getByTestId('video-stage').style.width).toBe('680px');
        expect(localStorage.getItem('blunderdb.video.stageWidth')).toBe('680');
        expect(document.activeElement).toBe(before);
    });
});
