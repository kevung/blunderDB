<script>
    // Stands in for VideoPane.svelte with the same driving surface.
    import { fakeVideo } from './fakeVideo.js';

    /** @type {{ source: string, onrelocate?: (path: string) => void }} */
    let { source } = $props();

    export function currentTimeMs() {
        return fakeVideo.now;
    }

    /** @param {number} ms */
    export function seek(ms) {
        fakeVideo.seeks.push(ms);
    }

    export function togglePlay() {
        fakeVideo.toggles++;
    }
</script>

<div data-testid="video-pane" data-source={source}></div>
