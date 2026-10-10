<script>
    // The theatre of the transcription (services/transcriptionTheatre.js): a bar with the way
    // out, the floating board, and the video, which the dock stretches over the window where it
    // stands. The transcription panel keeps its player and its keys; the player is never moved.
    import { t } from '../i18n';
    import { theatreStore, theatreTargetStore, theatreRotateStore, exitTheatre } from '../services/transcriptionTheatre.js';
    import TheatreMiniBoard from './TheatreMiniBoard.svelte';

    /** @type {HTMLDivElement | null} */
    let host = $state(null);

    $effect(() => {
        theatreTargetStore.set(host);
        return () => theatreTargetStore.set(null);
    });

    /** @param {MouseEvent} event */
    const keepFocus = (event) => event.preventDefault();
</script>

{#if $theatreStore}
    <div class="theatre" data-testid="transcription-theatre" role="region" aria-label={$t('theatre.region')}>
        <div class="theatre-video" bind:this={host} hidden></div>
        <TheatreMiniBoard />
        <div class="theatre-bar">
            <span class="theatre-hint">{$t('theatre.hint')}</span>
            {#if $theatreRotateStore}
                <button class="theatre-exit" data-testid="video-rotate" onmousedown={keepFocus} onclick={$theatreRotateStore} title={$t('video.rotate')} aria-label={$t('video.rotate')}>⟳</button>
            {/if}
            <button class="theatre-exit" data-testid="theatre-exit" onmousedown={keepFocus} onclick={exitTheatre} title={$t('theatre.exit')} aria-label={$t('theatre.exit')}>
                <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"
                    ><path d="M9 3v4a2 2 0 0 1-2 2H3M21 9h-4a2 2 0 0 1-2-2V3M3 15h4a2 2 0 0 1 2 2v4M15 21v-4a2 2 0 0 1 2-2h4" /></svg
                >
            </button>
        </div>
    </div>
{/if}

<style>
    /* Over the application, under its menus (1000) and dialogs. The video, stretched by its
       dock just below this layer (899), shows through. */
    .theatre {
        position: fixed;
        inset: 0;
        z-index: 900;
        pointer-events: none;
    }
    .theatre-bar {
        position: absolute;
        inset: 0 0 auto 0;
        height: var(--theatre-bar-height, 32px);
        z-index: 3;
        display: flex;
        align-items: center;
        justify-content: flex-end;
        gap: var(--space-2);
        padding: 0 8px;
        background: black;
        pointer-events: auto;
    }
    .theatre-hint {
        color: rgb(255 255 255 / 0.75);
        font-size: var(--font-size-small);
    }
    .theatre-exit {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 26px;
        height: 24px;
        padding: 0;
        border: 1px solid rgb(255 255 255 / 0.35);
        border-radius: 6px;
        background: rgb(0 0 0 / 0.75);
        color: white;
        cursor: pointer;
    }
    .theatre-exit:hover,
    .theatre-exit:focus-visible {
        border-color: white;
    }
</style>
