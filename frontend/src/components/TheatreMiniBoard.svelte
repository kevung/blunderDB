<script>
    // The floating board of the theatre: the transcription's position, the move being played on
    // the board and the arrows of the selected candidate (services/theatreBoard.js), drawn by
    // the board's own scene functions (services/diagramService.js). It is dragged anywhere, takes
    // three sizes and folds into a tab; where it sits and its size are remembered. The fold is
    // not: a tab folded in an earlier theatre is easily missed over a video, and the board is what
    // the theatre is for, so each theatre opens with it unfolded. It never takes the focus:
    // the transcription keys stay live while it is handled.
    import { t } from '../i18n';
    import { positionStore } from '../stores/positionStore.js';
    import { quizPlayStore } from '../stores/quizPlayStore.js';
    import { selectedMoveStore } from '../stores/analysisStore.js';
    import { transcriptionBoardSwapStore } from '../stores/transcriptionStore.js';
    import { boardColorsStore } from '../stores/boardColorsStore.js';
    import { renderPositionSVG, DIAGRAM_WIDTH, DIAGRAM_HEIGHT } from '../services/diagramService.js';
    import { theatreScene } from '../services/theatreBoard.js';
    import { logger } from '../utils/logger.js';

    const STORE_KEY = 'blunderdb.theatre.board';
    const MARGIN = 16;
    // Above the bottom edge, where a player draws its controls.
    const BOTTOM = 64;
    // The folded tab's box, enough to keep it whole inside the window.
    const TAB_W = 110;
    const TAB_H = 34;
    /** Widths of the three sizes; the height follows the board's own ratio. */
    const SIZES = /** @type {const} */ ({ s: 240, m: 340, l: 460 });
    // The nominal diagram is centred on its board, whose point numbers hang below it and
    // overflow the nominal height; this taller drawing keeps them inside.
    const DRAW_HEIGHT = DIAGRAM_HEIGHT + 24;
    /** @typedef {keyof typeof SIZES} Size */

    /** @type {{ x: number | null, y: number | null, size: Size }} */
    const saved = readSaved();
    // Fractions of the window, so the board keeps its corner when the window changes size;
    // null until dragged, the bottom-right corner.
    let fx = $state(saved.x);
    let fy = $state(saved.y);
    let size = $state(saved.size);
    let hidden = $state(false);
    let dragging = $state(false);
    // The layer's own size, in its CSS pixels: the theatre sits inside the interface's `zoom`,
    // so a window pixel is `scale` of them and a place computed in window pixels lands off screen.
    let viewW = $state(window.innerWidth);
    let viewH = $state(window.innerHeight);
    let scale = 1;
    /** @type {HTMLElement | null} */
    let card = $state(null);

    $effect(() => {
        if (card) measure();
    });

    const width = $derived(SIZES[size]);
    const height = $derived(Math.round((width * DRAW_HEIGHT) / DIAGRAM_WIDTH));

    const scene = $derived(theatreScene({ position: $positionStore, play: $quizPlayStore, swap: $transcriptionBoardSwapStore, selectedMove: $selectedMoveStore }));

    const svg = $derived.by(() => {
        void $boardColorsStore;
        if (!scene) return '';
        try {
            return renderPositionSVG(scene.position, { height: DRAW_HEIGHT, showPipcount: true, moves: scene.moves, flip: scene.flip });
        } catch (error) {
            logger.error('theatre board:', error);
            return '';
        }
    });

    const style = $derived.by(() => {
        if (fx === null || fy === null) return `right: ${MARGIN}px; bottom: ${BOTTOM}px; width: ${width}px;`;
        const left = clamp(fx * viewW, viewW - width - 2);
        const top = clamp(fy * viewH, viewH - height - 28);
        return `left: ${left}px; top: ${top}px; width: ${width}px;`;
    });

    // Folded, the tab stands where the board stood, in the window: that is where it is looked for.
    const tabStyle = $derived.by(() => {
        if (fx === null || fy === null) return `right: ${MARGIN}px; bottom: ${BOTTOM}px;`;
        return `left: ${clamp(fx * viewW, viewW - TAB_W)}px; top: ${clamp(fy * viewH, viewH - TAB_H)}px;`;
    });

    /**
     * @param {number} v
     * @param {number} max
     */
    function clamp(v, max) {
        return Math.round(Math.max(0, Math.min(v, Math.max(0, max))));
    }

    function readSaved() {
        try {
            const v = JSON.parse(localStorage.getItem(STORE_KEY) || 'null');
            if (v && typeof v === 'object') {
                const ok = (/** @type {unknown} */ n) => typeof n === 'number' && n >= 0 && n <= 1;
                return { x: ok(v.x) ? v.x : null, y: ok(v.y) ? v.y : null, size: v.size in SIZES ? v.size : 'm' };
            }
        } catch (_e) {
            /* storage unavailable or unreadable: the defaults */
        }
        return { x: null, y: null, size: /** @type {Size} */ ('m') };
    }

    function save() {
        try {
            localStorage.setItem(STORE_KEY, JSON.stringify({ x: fx, y: fy, size }));
        } catch (_e) {
            /* storage unavailable: the place lasts the session */
        }
    }

    /** @param {PointerEvent} event */
    function startDrag(event) {
        if (event.button !== 0 || (event.target instanceof Element && event.target.closest('button'))) return;
        // No focus taken: the transcription keys stay where they were.
        event.preventDefault();
        const el = /** @type {HTMLElement} */ (event.currentTarget);
        el.setPointerCapture?.(event.pointerId);
        measure();
        const origin = el.parentElement?.getBoundingClientRect() ?? { left: 0, top: 0 };
        const box = el.getBoundingClientRect();
        const dx = event.clientX - box.left;
        const dy = event.clientY - box.top;
        const boxW = box.width / scale;
        const boxH = box.height / scale;
        dragging = true;
        /** @param {PointerEvent} e */
        const move = (e) => {
            fx = clamp((e.clientX - dx - origin.left) / scale, viewW - boxW) / Math.max(1, viewW);
            fy = clamp((e.clientY - dy - origin.top) / scale, viewH - boxH) / Math.max(1, viewH);
        };
        const up = () => {
            el.removeEventListener('pointermove', move);
            el.removeEventListener('pointerup', up);
            el.removeEventListener('pointercancel', up);
            dragging = false;
            save();
        };
        el.addEventListener('pointermove', move);
        el.addEventListener('pointerup', up);
        el.addEventListener('pointercancel', up);
    }

    function cycleSize() {
        size = size === 's' ? 'm' : size === 'm' ? 'l' : 's';
        save();
    }

    function toggleHidden() {
        hidden = !hidden;
    }

    function measure() {
        const layer = card?.parentElement;
        if (!layer || !layer.offsetWidth || !layer.offsetHeight) return;
        viewW = layer.offsetWidth;
        viewH = layer.offsetHeight;
        scale = layer.getBoundingClientRect().width / layer.offsetWidth || 1;
    }

    /** @param {MouseEvent} event */
    const keepFocus = (event) => event.preventDefault();
</script>

<svelte:window onresize={measure} />

{#if hidden}
    <button
        class="theatre-board-tab"
        bind:this={card}
        style={tabStyle}
        data-testid="theatre-board-show"
        onmousedown={keepFocus}
        onclick={toggleHidden}
        title={$t('theatre.boardShow')}
        aria-label={$t('theatre.boardShow')}
    >
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"
            ><rect x="3" y="5" width="18" height="14" rx="1.5" /><path d="M12 5v14" /></svg
        >
        <span>{$t('theatre.board')}</span>
    </button>
{:else}
    <div class="theatre-board" class:dragging bind:this={card} data-testid="theatre-board" data-size={size} {style} role="group" aria-label={$t('theatre.board')} onpointerdown={startDrag}>
        <div class="theatre-board-head" title={$t('theatre.boardMove')}>
            <span class="grip" aria-hidden="true">⠿</span>
            <span class="move" data-testid="theatre-board-move">{$selectedMoveStore ?? ''}</span>
            <button class="head-btn" data-testid="theatre-board-size" onmousedown={keepFocus} onclick={cycleSize} title={$t('theatre.boardSize')} aria-label={$t('theatre.boardSize')}>
                <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2.2" aria-hidden="true"><path d="M14 4h6v6M10 20H4v-6M20 4l-7 7M4 20l7-7" /></svg>
            </button>
            <button class="head-btn" data-testid="theatre-board-hide" onmousedown={keepFocus} onclick={toggleHidden} title={$t('theatre.boardHide')} aria-label={$t('theatre.boardHide')}>
                <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2.4" aria-hidden="true"><path d="M5 12h14" /></svg>
            </button>
        </div>
        <div class="theatre-board-svg" style="height: {height}px">
            <!-- eslint-disable-next-line svelte/no-at-html-tags -- drawn here by the scene functions, from the position alone -->
            {@html svg}
        </div>
    </div>
{/if}

<style>
    .theatre-board,
    .theatre-board-tab {
        position: absolute;
        z-index: 2;
        pointer-events: auto;
        color: var(--color-text);
        background: var(--color-surface);
        border: 1px solid var(--color-border);
        border-radius: 8px;
        box-shadow: 0 6px 24px rgb(0 0 0 / 0.45);
    }
    .theatre-board {
        overflow: hidden;
        cursor: grab;
        touch-action: none;
        user-select: none;
        opacity: 0.96;
        transition: opacity 0.15s;
    }
    .theatre-board:hover,
    .theatre-board.dragging {
        opacity: 1;
    }
    .theatre-board.dragging {
        cursor: grabbing;
    }
    .theatre-board-head {
        display: flex;
        align-items: center;
        gap: var(--space-1);
        height: 26px;
        padding: 0 4px 0 8px;
        font-size: var(--font-size-small);
        border-bottom: 1px solid var(--color-border);
        background: var(--color-surface-alt);
    }
    .grip {
        color: var(--color-text-muted);
    }
    .move {
        flex: 1;
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-family: var(--font-family-mono);
        font-weight: 600;
    }
    .head-btn {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 22px;
        height: 22px;
        padding: 0;
        border: 0;
        border-radius: var(--radius);
        background: transparent;
        color: var(--color-text-muted);
        cursor: pointer;
    }
    .head-btn:hover,
    .head-btn:focus-visible {
        background: var(--color-surface);
        color: var(--color-text);
    }
    .theatre-board-svg :global(svg) {
        display: block;
        width: 100%;
        height: 100%;
    }
    /* Over a video, a discreet tab passes for part of the picture: it stands out in the
       application's own colour. */
    .theatre-board-tab {
        display: flex;
        align-items: center;
        gap: var(--space-1);
        padding: 6px 12px;
        font-weight: 600;
        color: white;
        background: var(--color-primary);
        border: 2px solid white;
        cursor: pointer;
    }
    .theatre-board-tab:hover,
    .theatre-board-tab:focus-visible {
        filter: brightness(1.15);
    }
</style>
