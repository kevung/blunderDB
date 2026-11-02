<script>
    // Stands in for VideoPane.svelte with the same driving surface.
    import { fakeVideo } from './fakeVideo.js';

    import { untrack } from 'svelte';

    /** @type {{ source: string, startMs?: number, viewLost?: boolean, onrelocate?: (path: string) => void, oncalibration?: (c: any) => void }} */
    let { source, startMs = 0, viewLost = false, oncalibration = undefined } = $props();
    fakeVideo.startMs = untrack(() => startMs);
    fakeVideo.oncalibration = untrack(() => oncalibration) ?? null;

    export function videoElement() {
        return fakeVideo;
    }

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

    export function durationMs() {
        return fakeVideo.duration;
    }

    export function togglePlay() {
        fakeVideo.toggles++;
    }
</script>

<div data-testid="video-pane" data-source={source} data-view-lost={viewLost}></div>
