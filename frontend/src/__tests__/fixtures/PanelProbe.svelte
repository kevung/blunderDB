<!-- A stand-in tab panel that counts its mounts, for TabbedPanel tests. -->
<script module>
    export const probeLog = { mounts: 0, destroys: 0 };
</script>

<script>
    import { onMount, onDestroy } from 'svelte';

    onMount(() => probeLog.mounts++);
    onDestroy(() => probeLog.destroys++);
</script>

<div data-testid="panel-probe"><input data-testid="probe-input" /></div>
