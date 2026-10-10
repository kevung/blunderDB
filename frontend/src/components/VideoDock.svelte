<script>
    // A panel's video, beside the board or inside the panel (stores/videoStageStore.js). The
    // panel renders it and drives it by bind:this as it would a VideoPane; with the board side
    // chosen, the dock's node moves there and the same player keeps playing.
    import { tick } from 'svelte';
    import VideoPane from './VideoPane.svelte';
    import { t } from '../i18n';
    import { portal } from '../utils/portal.js';
    import { videoPlacementStore, videoStageOwnerStore, videoStageTargetStore, claimVideoStage, releaseVideoStage, setVideoPlacement, swapVideoSide } from '../stores/videoStageStore.js';

    /**
     * @type {{
     *   owner: string,
     *   source: string,
     *   startMs?: number,
     *   rotation?: number,
     *   onrotate?: () => void,
     *   onrelocate?: (path: string) => void,
     *   onended?: () => void,
     *   onBoard?: boolean,
     *   theatreTarget?: HTMLElement | null,
     *   ontheatre?: () => void
     * }}
     */
    // `theatreTarget`: set while the owner's theatre is open; the dock then covers the window by
    // CSS where it stands, never moved in the DOM (a moved <iframe> reloads, a moved player
    // loses its instant); `ontheatre` offers the button that opens it, for the owners that have one.
    let {
        owner,
        source,
        startMs = 0,
        rotation = 0,
        onrotate = undefined,
        onrelocate = undefined,
        onended = undefined,
        onBoard = $bindable(false),
        theatreTarget = null,
        ontheatre = undefined
    } = $props();

    /** @type {any} */
    let pane = $state(null);
    /** @type {HTMLDivElement | null} */
    let dock = $state(null);

    const wantsBoard = $derived($videoPlacementStore === 'board');

    $effect(() => {
        if (!wantsBoard) return;
        const me = owner;
        claimVideoStage(me);
        return () => releaseVideoStage(me);
    });

    const boardTarget = $derived(wantsBoard && $videoStageOwnerStore === owner ? $videoStageTargetStore : null);
    const inTheatre = $derived(theatreTarget !== null);
    const target = $derived(boardTarget);

    $effect(() => {
        onBoard = boardTarget !== null;
    });

    // A focused node moved in the DOM loses the focus; the video keeps it when it had it.
    async function togglePlacement() {
        const had = holds(document.activeElement);
        setVideoPlacement(wantsBoard ? 'panel' : 'board');
        if (!had) return;
        await tick();
        /** @type {HTMLElement | null | undefined} */ (dock?.querySelector('.video-pane'))?.focus({ preventScroll: true });
    }

    /** @returns {number | null} */
    export function currentTimeMs() {
        return pane?.currentTimeMs?.() ?? null;
    }

    /** @param {number} ms */
    export function seek(ms) {
        pane?.seek?.(ms);
    }

    export function togglePlay() {
        pane?.togglePlay?.();
    }

    /** @param {-1 | 1} direction */
    export function stepRate(direction) {
        pane?.stepRate?.(direction);
    }

    /**
     * Whether the keyboard focus sits in the video, wherever it shows: the panel owns its keys
     * there as it does inside itself.
     *
     * @param {Element | null} element
     */
    export function holds(element) {
        return !!element && !!dock?.contains(element);
    }
</script>

<div
    class="video-dock"
    class:on-board={onBoard}
    class:in-theatre={inTheatre}
    data-testid="video-dock"
    data-placement={inTheatre ? 'theatre' : onBoard ? 'board' : 'panel'}
    bind:this={dock}
    use:portal={target}
>
    {#if !inTheatre}
        <div class="video-dock-bar">
            {#if ontheatre}
                <button
                    class="video-dock-place video-dock-theatre"
                    data-testid="video-theatre"
                    onmousedown={(event) => event.preventDefault()}
                    onclick={ontheatre}
                    title={$t('theatre.enter')}
                    aria-label={$t('theatre.enter')}
                    ><svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2.4" aria-hidden="true"
                        ><path d="M3 9V5a2 2 0 0 1 2-2h4M15 3h4a2 2 0 0 1 2 2v4M21 15v4a2 2 0 0 1-2 2h-4M9 21H5a2 2 0 0 1-2-2v-4" /></svg
                    ></button
                >
            {/if}
            {#if onrotate}
                <button
                    class="video-dock-place video-dock-rotate"
                    data-testid="video-rotate"
                    onmousedown={(event) => event.preventDefault()}
                    onclick={onrotate}
                    title={$t('video.rotate')}
                    aria-label={$t('video.rotate')}>⟳</button
                >
            {/if}
            {#if onBoard}
                <button
                    class="video-dock-place video-dock-swap"
                    data-testid="video-swap-side"
                    onmousedown={(event) => event.preventDefault()}
                    onclick={swapVideoSide}
                    title={$t('video.swapSide')}
                    aria-label={$t('video.swapSide')}>⇄</button
                >
            {/if}
            <button
                class="video-dock-place"
                data-testid="video-placement"
                onmousedown={(event) => event.preventDefault()}
                onclick={togglePlacement}
                aria-pressed={wantsBoard}
                title={wantsBoard ? $t('video.placePanel') : $t('video.placeBoard')}
                aria-label={wantsBoard ? $t('video.placePanel') : $t('video.placeBoard')}>{wantsBoard ? '⇲' : '⇱'}</button
            >
        </div>
    {/if}
    <div class="video-dock-stage"><VideoPane bind:this={pane} {source} {startMs} {rotation} {onrelocate} {onended} /></div>
</div>

<style>
    .video-dock {
        display: flex;
        flex-direction: column;
        width: 100%;
        height: 100%;
    }
    /* Above the window's own layers, below the theatre's bar and board (900). */
    .video-dock.in-theatre {
        position: fixed;
        inset: var(--theatre-bar-height, 32px) 0 0 0;
        z-index: 899;
        width: auto;
        height: auto;
        background: black;
    }
    .video-dock-stage {
        position: relative;
        flex: 1;
        min-height: 0;
    }
    /* A bar of its own above the image: the player draws its controls in the corners and along
       the bottom of the picture. */
    .video-dock-bar {
        display: flex;
        flex: none;
        align-items: center;
        justify-content: flex-end;
        gap: 4px;
        padding: 2px 4px;
        background: var(--color-surface-alt);
    }
    .video-dock-place {
        padding: 0 6px;
        line-height: 1.6;
        border: 0;
        border-radius: 3px;
        background: rgb(0 0 0 / 0.55);
        color: white;
        cursor: pointer;
        opacity: 0.7;
    }
    .video-dock-theatre {
        display: flex;
        align-items: center;
        height: 1.6em;
    }
    .video-dock-place:hover,
    .video-dock-place:focus-visible {
        opacity: 1;
    }
</style>
