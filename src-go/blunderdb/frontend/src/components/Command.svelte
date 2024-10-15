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
        commandText = '';
    });
</script>

<input
    type="text"
    bind:this={inputEl}
    bind:value={commandText}
    placeholder=" Type your command here. "
    style="position: fixed; top: 10px; left: 50%; transform: translateX(-50%); z-index: 1000;"
    on:keydown={handleKeyDown}
/>

<style>
    input {
        width: 90%;
        padding: 8px;
        border: 1px solid rgba(0, 0, 0, 0.1); /* Subtle border */
        border-radius: 4px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        outline: none;
        background-color: white; /* Ensure background is opaque */
        font-size: 21px;
    }
</style>
