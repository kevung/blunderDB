<script>
   import { onMount } from 'svelte';

   export let visible = false;
   export let onClose;
   export let onToggleHelp;
   export let text = '';
   let inputEl;

   let initialized = false;

   $: if (visible && !initialized) {
      text = '';
      initialized = true;

      setTimeout(() => {
         inputEl?.focus();
      }, 0);

   }

    // Focus the input when the component is visible
    $: if (!visible) {
        initialized = false;
    }

    function handleKeyDown(event) {
        event.stopPropagation();

        if(visible) {
            if(event.code === 'Backspace' && inputEl.value === '') {
                onClose();
            } else if (event.code === 'Escape') {
                onClose();
            } else if (event.code === 'Enter') {
                onClose();
            } else if (event.ctrlKey && event.code === 'KeyC') {
                onClose();
            } else if (event.ctrlKey && event.code === 'KeyH') {
                onToggleHelp();
            }
        }
    }

</script>

{#if visible}
   <input
         type="text"
         bind:this={inputEl}
         bind:value={text}
         class="command-input"
         placeholder=" Type your command here. "
         on:keydown={handleKeyDown}
         />
{/if}

<style>
    input {
        position: fixed;
        top: 350px;
        left: 50%;
        transform: translateX(-50%);
        z-index: 1000;
        width: 70%;
        padding: 8px;
        border: 1px solid rgba(0, 0, 0, 0.3); /* Subtle border */
        border-radius: 1px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0);
        outline: none;
        background-color: white; /* Ensure background is opaque */
        font-size: 18px;
    }
</style>
