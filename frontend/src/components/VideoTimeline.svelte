<script>
    // The events the vision core found, as markers along the video's length: a click seeks to
    // one, play or pause kept. The shape says the state (circle sure, diamond to check,
    // triangle uncertain), the colour only doubles it. A marker writes nothing.
    import { onMount } from 'svelte';
    import { t } from '../i18n';

    /**
     * @type {{
     *   markers: { type: string, mediaTime: number, state: string, shape: string }[],
     *   durationMs: () => number | null,
     *   nowMs: () => number | null,
     *   onjump: (mediaTime: number) => void
     * }}
     */
    let { markers, durationMs, nowMs, onjump } = $props();

    let duration = $state(0);
    let now = $state(0);
    onMount(() => {
        const read = () => {
            duration = durationMs() ?? 0;
            now = nowMs() ?? 0;
        };
        read();
        const timer = setInterval(read, 500);
        return () => clearInterval(timer);
    });

    /** @param {number} sec */
    const left = (sec) => (duration > 0 ? Math.min(100, Math.max(0, ((sec * 1000) / duration) * 100)) : 0);
    /** @param {string} state */
    const label = (state) => (state === 'sure' ? $t('video.markerSure') : state === 'check' ? $t('video.markerCheck') : $t('video.markerUncertain'));
</script>

{#if duration > 0}
    <div class="video-timeline" data-testid="video-timeline" role="group" aria-label={$t('video.timeline')}>
        <span class="playhead" style:left="{left(now / 1000)}%"></span>
        {#each markers as m (m.mediaTime + m.type)}
            <button
                class="marker {m.shape}"
                data-testid="video-marker"
                data-type={m.type}
                data-state={m.state}
                data-shape={m.shape}
                style:left="{left(m.mediaTime)}%"
                title="{m.type} · {label(m.state)}"
                aria-label="{m.type} · {label(m.state)}"
                onmousedown={(event) => event.preventDefault()}
                onclick={() => onjump(m.mediaTime)}
            ></button>
        {/each}
    </div>
{/if}

<style>
    .video-timeline {
        position: relative;
        flex: none;
        height: 16px;
        background: var(--color-surface-alt);
    }
    .playhead {
        position: absolute;
        top: 0;
        bottom: 0;
        width: 1px;
        background: var(--color-text);
        opacity: 0.6;
    }
    .marker {
        position: absolute;
        top: 3px;
        width: 10px;
        height: 10px;
        padding: 0;
        margin-left: -5px;
        border: 0;
        background: var(--color-primary);
        cursor: pointer;
    }
    .marker.circle {
        border-radius: 50%;
    }
    .marker.diamond {
        transform: rotate(45deg) scale(0.8);
    }
    .marker.triangle {
        clip-path: polygon(50% 0, 100% 100%, 0 100%);
        background: var(--color-warning);
    }
    .marker:hover,
    .marker:focus-visible {
        outline: 1px solid currentColor;
    }
</style>
