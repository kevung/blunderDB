<script>
    import { onMount, onDestroy } from 'svelte';
    import { fade } from 'svelte/transition';
    import Board from './components/Board.svelte';
    import Command from './components/Command.svelte';
    import StatusBar from './components/StatusBar.svelte';
    import CommentsZone from './components/CommentsZone.svelte';
    let showCommand = false;
    let showCommentsZone = false;
    let mode = "normal";
    let position = 0;
    let infoMessage = "";

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

        // Update the status bar based on the key pressed
        if (showCommand) {
            mode = "COMMAND"; // Set mode to edit when the TextLine is shown
            position++; // Increment the position for demonstration
            infoMessage = "A command is being entered..."; // Set info message
        } else {
            mode = "NORMAL"; // Reset mode to normal
            infoMessage = ""; // Clear info message
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

<main class="main-container">
    <Board />

    {#if showCommand}
        <Command {hideCommandText} />
    {/if}

    <StatusBar {mode} {position} {infoMessage} />

    {#if showCommentsZone}
        <CommentsZone {hideCommentsZone} />
    {/if}
</main>

<style>
    .main-container {
        display: flex;
        flex-direction: column; /* Stack children vertically */
        align-items: stretch; /* Allow children to stretch to fill the width */
        min-height: 100vh; /* Full height of the viewport */
        padding: 16px; /* Add some padding */
        box-sizing: border-box; /* Include padding in total height */
    }
</style>
