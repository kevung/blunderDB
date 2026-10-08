<script>
    // Stands in for VideoPane.svelte with the same driving surface.
    import { fakeVideo } from './fakeVideo.js';

    import { untrack } from 'svelte';

    /** @type {{ source: string, startMs?: number, onrelocate?: (path: string) => void }} */
    let { source, startMs = 0 } = $props();
    fakeVideo.startMs = untrack(() => startMs);

    export function currentTimeMs() {
        return fakeVideo.now;
    }

    /** @param {number} ms */
    export function seek(ms) {
        fakeVideo.seeks.push(ms);
    }

    /** @param {-1 | 1} direction */
    export function stepRate(direction) {
        fakeVideo.rateSteps.push(direction);
    }

    export function togglePlay() {
        fakeVideo.toggles++;
    }
</script>

<div data-testid="video-pane" data-source={source}></div>
