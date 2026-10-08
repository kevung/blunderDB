<script>
    // The main area and a panel, side by side as App.svelte lays them out.
    import VideoStage from '../../components/VideoStage.svelte';
    import VideoDock from '../../components/VideoDock.svelte';

    /** @type {{ withVideo?: boolean }} */
    let { withVideo = true } = $props();
    /** @type {any} */
    let dock = $state(null);
    let onBoard = $state(false);

    export function player() {
        return dock;
    }
</script>

<div data-testid="main-area">
    <VideoStage><div data-testid="board-stub"></div></VideoStage>
</div>
<div data-testid="panel" data-on-board={onBoard}>
    {#if withVideo}
        <VideoDock bind:this={dock} bind:onBoard owner="test" source="/v/match.mp4" />
    {/if}
</div>
