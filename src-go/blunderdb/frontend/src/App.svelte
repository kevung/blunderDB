<script>
    import { onMount, onDestroy } from 'svelte';
    import { fade } from 'svelte/transition';
    import Board from './components/Board.svelte';
    import Command from './components/Command.svelte';
    import CommentsZone from './components/CommentsZone.svelte';
    let showCommand = false;
    let showCommentsZone = false;

    function handleKeyDown(event) {
        if (event.code === 'Space') {
            if(!showCommand && !showCommentsZone) {
                event.preventDefault();
                showCommand = true;
            }
        } else if (event.code === 'Escape' || event.code === 'Enter') {
            showCommand = false;
        } else if(showCommand && event.ctrlKey && event.code === 'KeyC') {
            showCommand = false;
        } else if(event.ctrlKey && event.code == 'KeyP') {
            event.preventDefault();
            showCommentsZone = !showCommentsZone;
            showCommand = false;
        }
    }

    function hideCommandText() {
        showCommand = false;
    }

    function hideCommentsZone() {
        showCommentsZone = false;
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

    {#if showCommentsZone}
        <CommentsZone {hideCommentsZone} />
    {/if}
</main>

<style>
</style>
