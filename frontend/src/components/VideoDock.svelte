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
     *   onrelocate?: (path: string) => void,
     *   onended?: () => void,
     *   onBoard?: boolean,
     *   theatreTarget?: HTMLElement | null,
     *   ontheatre?: () => void
     * }}
     */
    // `theatreTarget`: a whole-window place the owner opens for its video, over any other;
    // `ontheatre` offers the button that opens it, for the owners that have one.
    let { owner, source, startMs = 0, onrelocate = undefined, onended = undefined, onBoard = $bindable(false), theatreTarget = null, ontheatre = undefined } = $props();

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
    const target = $derived(theatreTarget ?? boardTarget);

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

<div class="video-dock" class:on-board={onBoard} data-testid="video-dock" data-placement={inTheatre ? 'theatre' : onBoard ? 'board' : 'panel'} bind:this={dock} use:portal={target}>
    <VideoPane bind:this={pane} {source} {startMs} {onrelocate} {onended} />
    {#if !inTheatre}
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
        <button
            class="video-dock-place"
            data-testid="video-placement"
            onmousedown={(event) => event.preventDefault()}
            onclick={togglePlacement}
            aria-pressed={wantsBoard}
            title={wantsBoard ? $t('video.placePanel') : $t('video.placeBoard')}
            aria-label={wantsBoard ? $t('video.placePanel') : $t('video.placeBoard')}>{wantsBoard ? '⇲' : '⇱'}</button
        >
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
    {/if}
</div>

<style>
    .video-dock {
        position: relative;
        width: 100%;
        height: 100%;
    }
    .video-dock-place {
        position: absolute;
        top: 6px;
        right: 6px;
        padding: 0 6px;
        line-height: 1.6;
        border: 0;
        border-radius: 3px;
        background: rgb(0 0 0 / 0.55);
        color: white;
        cursor: pointer;
        opacity: 0.6;
    }
    .video-dock-swap {
        right: 38px;
    }
    /* Left of the placement button, one slot further when the swap button shows. */
    .video-dock-theatre {
        right: 38px;
        display: flex;
        align-items: center;
        height: 1.6em;
    }
    .video-dock.on-board .video-dock-theatre {
        right: 70px;
    }
    .video-dock:hover .video-dock-place,
    .video-dock-place:focus-visible {
        opacity: 1;
    }
</style>
