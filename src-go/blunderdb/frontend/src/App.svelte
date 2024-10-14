<script>
    import { onMount, onDestroy } from 'svelte';
    import { fade } from 'svelte/transition';
    import Board from './components/Board.svelte';
    import Command from './components/Command.svelte';
    let showCommand = false;

    function handleKeyDown(event) {
        if (event.code === 'Space') {
            showCommand = true;
        } else if (event.code === 'Escape' || event.code === 'Enter') {
            showCommand = false;
        } else if(showCommand && event.ctrlKey && event.code === 'KeyC') {
            showCommand = false;
        }
    }

    function hideCommandText() {
        showCommand = false;
    }

    onMount(() => {
        window.addEventListener("keydown", handleKeyDown);
    });

    onDestroy(() => {
        window.removeEventListener("keydown", handleKeyDown);
    });
</script>

<main>
    <Board />
    {#if showCommand}
        <Command {hideCommandText} />
    {/if}
</main>

<style>
</style>
