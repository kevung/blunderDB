<script>
    import { onMount } from 'svelte';

    export let hideCommandText;
    let inputEl;
    let commandText = '';

    function handleKeyDown(event) {
        if(event.code === 'Backspace' && inputEl.value === '') {
            hideCommandText();
        }
    }

    onMount(() => {
        inputEl.focus();
    });
</script>

<input
    bind:this={inputEl}
    bind:value={commandText}
    placeholder="enter your command"
    style="position: fixed; top: 10px; left: 50%; transform: translateX(-50%); z-index: 1000;"
    on:keydown={handleKeyDown}
/>

<style>
    input {
        width: 90%;
        font-size: 25px;
        padding: 1px;
        border: 1px solid #ccc;
        border-radius: 4px;
        outline: none;
        background-color: white; /* Ensure background is opaque */
    }
</style>
