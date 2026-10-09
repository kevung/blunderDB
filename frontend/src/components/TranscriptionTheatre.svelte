<script>
    // The theatre of the transcription (services/transcriptionTheatre.js): the video over the
    // whole window, the floating board, and a way out. The transcription panel keeps its player
    // and its keys; this layer only lends the player a place, as the board side does.
    import { t } from '../i18n';
    import { theatreStore, theatreTargetStore, exitTheatre } from '../services/transcriptionTheatre.js';
    import TheatreMiniBoard from './TheatreMiniBoard.svelte';

    /** @type {HTMLDivElement | null} */
    let host = $state(null);

    $effect(() => {
        theatreTargetStore.set(host);
        return () => theatreTargetStore.set(null);
    });

    // The controls fade while the pointer rests, and come back as soon as it moves.
    const IDLE_MS = 2500;
    let idle = $state(false);
    /** @type {ReturnType<typeof setTimeout> | undefined} */
    let idleTimer;

    function wake() {
        idle = false;
        clearTimeout(idleTimer);
        idleTimer = setTimeout(() => (idle = true), IDLE_MS);
    }

    $effect(() => {
        if (!$theatreStore) return;
        wake();
        return () => clearTimeout(idleTimer);
    });

    /** @param {MouseEvent} event */
    const keepFocus = (event) => event.preventDefault();
</script>

{#if $theatreStore}
    <div class="theatre" class:idle data-testid="transcription-theatre" role="region" aria-label={$t('theatre.region')} onpointermove={wake}>
        <div class="theatre-video" bind:this={host}></div>
        <TheatreMiniBoard />
        <div class="theatre-controls">
            <span class="theatre-hint">{$t('theatre.hint')}</span>
            <button class="theatre-exit" data-testid="theatre-exit" onmousedown={keepFocus} onclick={exitTheatre} title={$t('theatre.exit')} aria-label={$t('theatre.exit')}>
                <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"
                    ><path d="M9 3v4a2 2 0 0 1-2 2H3M21 9h-4a2 2 0 0 1-2-2V3M3 15h4a2 2 0 0 1 2 2v4M15 21v-4a2 2 0 0 1 2-2h4" /></svg
                >
            </button>
        </div>
    </div>
{/if}

<style>
    /* Over the application, under its menus (1000) and dialogs. */
    .theatre {
        position: fixed;
        inset: 0;
        z-index: 900;
        background: black;
    }
    .theatre-video {
        position: absolute;
        inset: 0;
    }
    .theatre-controls {
        position: absolute;
        top: 12px;
        right: 12px;
        z-index: 3;
        display: flex;
        align-items: center;
        gap: var(--space-2);
        transition: opacity 0.4s;
    }
    .theatre.idle .theatre-controls {
        opacity: 0;
    }
    .theatre-controls:hover,
    .theatre-controls:focus-within {
        opacity: 1;
    }
    .theatre-hint {
        padding: 3px 8px;
        border-radius: var(--radius);
        background: rgb(0 0 0 / 0.75);
        color: white;
        font-size: var(--font-size-small);
    }
    .theatre-exit {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 36px;
        height: 36px;
        padding: 0;
        border: 1px solid rgb(255 255 255 / 0.35);
        border-radius: 6px;
        background: rgb(0 0 0 / 0.75);
        color: white;
        cursor: pointer;
    }
    .theatre-exit:hover,
    .theatre-exit:focus-visible {
        background: rgb(0 0 0 / 0.85);
        border-color: white;
    }
</style>
