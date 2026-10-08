<script>
    // The main area, split while a panel's video sits beside the board: video on the left,
    // a vertical separator, the board on the right. The panel keeps its player; this side
    // only offers it a place (stores/videoStageStore.js). Without such a video the area holds
    // the board alone, as before.
    import { tick } from 'svelte';
    import { t } from '../i18n';
    import { videoStageOwnerStore, videoStageTargetStore } from '../stores/videoStageStore.js';

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
            width = clamp(startWidth + e.clientX - startX);
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
</script>

<div class="video-stage-split" bind:this={split}>
    {#if shown}
        <div class="video-stage" data-testid="video-stage" style="width: {width}px" bind:this={stage}></div>
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
</div>

<style>
    .video-stage-split {
        display: flex;
        width: 100%;
        height: 100%;
        min-width: 0;
        min-height: 0;
    }
    .video-stage {
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
