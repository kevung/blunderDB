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
import { videoPlacementStore, setVideoPlacement, videoStageOwnerStore, videoSideStore, setVideoSide } from '../stores/videoStageStore.js';

async function settle() {
    for (let i = 0; i < 5; i++) await tick();
}

beforeEach(() => {
    localStorage.clear();
    fakeVideo.reset();
    setVideoPlacement('board');
    setVideoSide('left');
    localStorage.clear();
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

describe('the side of the video', () => {
    /** Stubs the area at 0..1000 x 0..600 so the halves are measurable. */
    function measure() {
        const split = /** @type {HTMLElement} */ (screen.getByTestId('video-stage').parentElement);
        split.getBoundingClientRect = () => /** @type {any} */ ({ left: 0, right: 1000, top: 0, bottom: 600, width: 1000, height: 600 });
    }
    async function drag(/** @type {number[]} */ points) {
        const grip = screen.getByTestId('video-stage-grip');
        await fireEvent.pointerDown(grip, { clientX: 100, clientY: 5, pointerId: 1 });
        for (const x of points) await fireEvent.pointerMove(grip, { clientX: x, clientY: 100, pointerId: 1 });
        return grip;
    }

    test('left by default, the preference is remembered and the width stays the video width', async () => {
        render(Harness);
        await settle();
        expect(get(videoSideStore)).toBe('left');
        setVideoSide('right');
        await settle();
        expect(localStorage.getItem('blunderdb.video.side')).toBe('right');
        const split = /** @type {HTMLElement} */ (screen.getByTestId('video-stage').parentElement);
        expect(split.classList.contains('right')).toBe(true);
        // Dragging the separator grows the video the other way on the right.
        const handle = screen.getByTestId('video-stage-resize');
        await fireEvent.pointerDown(handle, { clientX: 500, pointerId: 1 });
        await fireEvent.pointerMove(handle, { clientX: 420, pointerId: 1 });
        await fireEvent.pointerUp(handle, { clientX: 420, pointerId: 1 });
        await settle();
        expect(screen.getByTestId('video-stage').style.width).toBe('640px');
    });

    test('the swap button flips the side, remembered, without taking focus', async () => {
        render(Harness);
        await settle();
        const before = document.activeElement;
        const button = screen.getByTestId('video-swap-side');
        expect(await fireEvent.mouseDown(button)).toBe(false);
        await fireEvent.click(button);
        expect(get(videoSideStore)).toBe('right');
        await fireEvent.click(button);
        expect(get(videoSideStore)).toBe('left');
        expect(localStorage.getItem('blunderdb.video.side')).toBe('left');
        expect(document.activeElement).toBe(before);
    });

    test('no swap button while the video is in the panel', async () => {
        render(Harness);
        await settle();
        await fireEvent.click(screen.getByTestId('video-placement'));
        await settle();
        expect(screen.queryByTestId('video-swap-side')).toBeNull();
    });

    test('dragging the grip over a half lights it and releasing there chooses the side', async () => {
        render(Harness);
        await settle();
        measure();
        const grip = await drag([800]);
        expect(screen.getByTestId('video-stage-veil')).toBeTruthy();
        expect(screen.getByTestId('veil-right').classList.contains('lit')).toBe(true);
        expect(screen.getByTestId('veil-left').classList.contains('lit')).toBe(false);
        await fireEvent.pointerUp(grip, { clientX: 800, clientY: 100, pointerId: 1 });
        await settle();
        expect(get(videoSideStore)).toBe('right');
        expect(screen.queryByTestId('video-stage-veil')).toBeNull();
        // And back to the left.
        const g2 = await drag([200]);
        expect(screen.getByTestId('veil-left').classList.contains('lit')).toBe(true);
        await fireEvent.pointerUp(g2, { clientX: 200, clientY: 100, pointerId: 1 });
        expect(get(videoSideStore)).toBe('left');
    });

    test('Escape cancels the drag', async () => {
        render(Harness);
        await settle();
        measure();
        const grip = await drag([800]);
        await fireEvent.keyDown(window, { key: 'Escape' });
        await settle();
        expect(screen.queryByTestId('video-stage-veil')).toBeNull();
        await fireEvent.pointerUp(grip, { clientX: 800, clientY: 100, pointerId: 1 });
        expect(get(videoSideStore)).toBe('left');
    });

    test('releasing outside the area cancels', async () => {
        render(Harness);
        await settle();
        measure();
        const grip = await drag([800]);
        await fireEvent.pointerUp(grip, { clientX: 1500, clientY: 100, pointerId: 1 });
        expect(get(videoSideStore)).toBe('left');
        expect(screen.queryByTestId('video-stage-veil')).toBeNull();
    });

    test('the grip takes no focus, and the board is not draggable', async () => {
        render(Harness);
        await settle();
        measure();
        const before = document.activeElement;
        const grip = screen.getByTestId('video-stage-grip');
        expect(await fireEvent.pointerDown(grip, { clientX: 100, clientY: 5, pointerId: 1 })).toBe(false);
        expect(await fireEvent.mouseDown(grip)).toBe(false);
        expect(document.activeElement).toBe(before);
        await fireEvent.pointerUp(grip, { clientX: 100, clientY: 100, pointerId: 1 });
        expect(screen.getByTestId('board-stub').closest('.board-side')?.querySelector('[data-testid="video-stage-grip"]')).toBeNull();
    });
});
