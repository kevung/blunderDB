<script>
    export let visible = false;
    export let onClose;
    export let text = '';

    import { onMount } from "svelte";

    $: if (visible) {
       setTimeout(() => {
          const textAreaEl = document.getElementById('commentsTextArea');
          if (textAreaEl) {
             textAreaEl.focus();
          }
       }, 0);
    }

</script>

{#if visible}
    <div class="comments-zone">
        <textarea
            id="commentsTextArea"
            rows="5"
            cols="30"
            bind:value={text}
            placeholder="Type your comments here..."
        ></textarea>
        <button on:click={onClose}>Close</button>
    </div>
{/if}

<style>

    textarea {
        position: relative;
        width: 100%; /* Full width */
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        box-sizing: border-box; /* Include padding and border in total width */
        padding: 8px;
        margin-top: 16px; /* Space between main content and comments zone */
        border: 1px solid rgba(0, 0, 0, 0.1); /* Subtle border */
        border-radius: 4px;
        outline: none;
        resize: none;
        background-color: white; /* Ensure background is opaque */
        font-size: 20px;
    }

    button {
        border: none;
        border-radius: 4px;
        cursor: pointer;
        transition: background-color 0.3s ease, opacity 0.3s ease;
    }

    button:hover {
        opacity: 0.7;
    }

</style>

