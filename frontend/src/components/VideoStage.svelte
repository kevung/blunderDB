<script>
    // The main area, split while a panel's video sits beside the board: video on the left,
    // a vertical separator, the board on the right. The panel keeps its player; this side
    // only offers it a place (stores/videoStageStore.js). Without such a video the area holds
    // the board alone, as before.
    import { tick } from 'svelte';
    import { t } from '../i18n';
    import { videoSideStore, setVideoSide, videoStageOwnerStore, videoStageTargetStore } from '../stores/videoStageStore.js';

    /** @type {{ children: import('svelte').Snippet }} */
    let { children } = $props();

    const WIDTH_KEY = 'blunderdb.video.stageWidth';
    const WIDTH_MIN = 160;
    // The board keeps at least this much beside the video.
    const BOARD_MIN = 200;

    let width = $state(readWidth());
    /** @type {HTMLDivElement | null} */
    let stage = $state(null);
    /** @type {HTMLDivElement | null} */
    let split = $state(null);

    const side = $derived($videoSideStore);
    /** @type {'left' | 'right' | null} */
    let dragSide = $state(null);
    let dragging = $state(false);

    const shown = $derived($videoStageOwnerStore !== null);

    $effect(() => {
        videoStageTargetStore.set(stage);
        return () => videoStageTargetStore.set(null);
    });

    // The board measures its container on 'resize' only.
    function refitBoard() {
        tick().then(() => window.dispatchEvent(new Event('resize')));
    }

    $effect(() => {
        void shown;
        refitBoard();
    });

    function readWidth() {
        try {
            const v = Number(localStorage.getItem(WIDTH_KEY));
            if (Number.isFinite(v) && v >= WIDTH_MIN) return v;
        } catch (_e) {
            /* storage unavailable: the default width */
        }
        return 560;
    }

    /** @param {number} w */
    function clamp(w) {
        // Unmeasured (0) leaves the width alone; the CSS max-width still guards the board.
        const room = (split?.clientWidth || Infinity) - BOARD_MIN;
        return Math.round(Math.max(WIDTH_MIN, Math.min(w, room)));
    }

    // The separator takes no focus (preventDefault on pointerdown and mousedown alike): the
    // panel's keys stay live during and after a drag.
    /** @param {PointerEvent} event */
    function startResize(event) {
        event.preventDefault();
        const handle = /** @type {HTMLElement} */ (event.currentTarget);
        // Captured, or a player frame under the pointer would swallow the moves.
        handle.setPointerCapture?.(event.pointerId);
        const startX = event.clientX;
        const startWidth = stage?.clientWidth || width;
        /** @param {PointerEvent} e */
        const move = (e) => {
            width = clamp(startWidth + (side === 'right' ? startX - e.clientX : e.clientX - startX));
            refitBoard();
        };
        const up = () => {
            handle.removeEventListener('pointermove', move);
            handle.removeEventListener('pointerup', up);
            handle.removeEventListener('pointercancel', up);
            try {
                localStorage.setItem(WIDTH_KEY, String(width));
            } catch (_e) {
                /* storage unavailable: the width lasts the session */
            }
        };
        handle.addEventListener('pointermove', move);
        handle.addEventListener('pointerup', up);
        handle.addEventListener('pointercancel', up);
    }

    // The grip moves the video to the other side of the board. Pointer events, not HTML5 drag
    // (unreliable under WebKitGTK, tangled with the file drop). A veil over the whole area
    // keeps the player frame from taking the pointer; releasing outside the area, or Escape,
    // cancels.
    /** @param {number} x @param {number} y */
    function sideAt(x, y) {
        const r = split?.getBoundingClientRect();
        if (!r || x < r.left || x > r.right || y < r.top || y > r.bottom) return null;
        return x < r.left + r.width / 2 ? 'left' : 'right';
    }

    /** @param {PointerEvent} event */
    function startMove(event) {
        event.preventDefault();
        const grip = /** @type {HTMLElement} */ (event.currentTarget);
        grip.setPointerCapture?.(event.pointerId);
        dragging = true;
        dragSide = sideAt(event.clientX, event.clientY);
        /** @param {PointerEvent} e */
        const move = (e) => {
            dragSide = sideAt(e.clientX, e.clientY);
        };
        /** @param {KeyboardEvent} e */
        const key = (e) => {
            if (e.key === 'Escape') {
                e.stopPropagation();
                end(false);
            }
        };
        /** @param {PointerEvent} e */
        const up = (e) => {
            end(true, sideAt(e.clientX, e.clientY));
        };
        const cancel = () => end(false);
        /** @param {boolean} commit @param {'left' | 'right' | null} [chosen] */
        function end(commit, chosen = null) {
            grip.removeEventListener('pointermove', move);
            grip.removeEventListener('pointerup', up);
            grip.removeEventListener('pointercancel', cancel);
            window.removeEventListener('keydown', key, true);
            dragging = false;
            dragSide = null;
            if (commit && chosen) {
                setVideoSide(chosen);
                refitBoard();
            }
        }
        grip.addEventListener('pointermove', move);
        grip.addEventListener('pointerup', up);
        grip.addEventListener('pointercancel', cancel);
        window.addEventListener('keydown', key, true);
    }
</script>

<div class="video-stage-split" class:right={side === 'right'} bind:this={split}>
    {#if shown}
        <div class="video-stage" data-testid="video-stage" style="width: {width}px" bind:this={stage}>
            <div
                class="video-stage-grip"
                data-testid="video-stage-grip"
                role="presentation"
                title={$t('video.stageMove')}
                onmousedown={(event) => event.preventDefault()}
                onpointerdown={startMove}
            ></div>
        </div>
        <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
        <div
            class="video-stage-resize"
            data-testid="video-stage-resize"
            role="separator"
            aria-orientation="vertical"
            aria-label={$t('video.stageResize')}
            onmousedown={(event) => event.preventDefault()}
            onpointerdown={startResize}
        ></div>
    {/if}
    <div class="board-side">
        {@render children()}
    </div>
    {#if dragging}
        <div class="video-stage-veil" data-testid="video-stage-veil">
            <div class="veil-half" class:lit={dragSide === 'left'} data-testid="veil-left"></div>
            <div class="veil-half" class:lit={dragSide === 'right'} data-testid="veil-right"></div>
        </div>
    {/if}
</div>

<style>
    .video-stage-split {
        display: flex;
        width: 100%;
        height: 100%;
        min-width: 0;
        min-height: 0;
    }
    .video-stage-split {
        position: relative;
    }
    .video-stage-split.right {
        flex-direction: row-reverse;
    }
    .video-stage-grip {
        position: absolute;
        top: 0;
        left: 50%;
        transform: translateX(-50%);
        width: 56px;
        height: 10px;
        border-radius: 0 0 6px 6px;
        background: rgb(255 255 255 / 0.35);
        cursor: grab;
        z-index: 2;
        touch-action: none;
    }
    .video-stage-grip:hover {
        background: rgb(255 255 255 / 0.7);
    }
    .video-stage-veil {
        position: absolute;
        inset: 0;
        z-index: 10;
        display: flex;
        cursor: grabbing;
    }
    .veil-half {
        flex: 1;
        border: 2px dashed transparent;
        background: rgb(0 0 0 / 0.15);
    }
    .veil-half.lit {
        background: rgb(80 140 255 / 0.35);
        border-color: rgb(80 140 255);
    }
    .video-stage {
        position: relative;
        flex: 0 1 auto;
        min-width: 160px;
        max-width: calc(100% - 206px);
        height: 100%;
        background: var(--color-text);
        overflow: hidden;
    }
    .video-stage-resize {
        flex: 0 0 auto;
        width: 6px;
        cursor: col-resize;
        background: var(--color-border);
        touch-action: none;
    }
    .board-side {
        flex: 1;
        min-width: 0;
        min-height: 0;
        height: 100%;
        display: flex;
        justify-content: center;
        align-items: center;
        position: relative;
    }
</style>
