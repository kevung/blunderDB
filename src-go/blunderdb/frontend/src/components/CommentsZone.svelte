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

    .comments-zone {
        position: absolute;
        bottom: 50px; /* Above the status bar */
        left: 0;
        right: 0;
        max-height: 50vh; /* Limit height to half the viewport */
        overflow-y: auto; /* Allow scrolling inside the comment zone */
        overflow-x: hidden; /* Disable horizontal scrolling inside the comment zone */
        background-color: rgba(0, 0, 0, 0);
        padding: 16px;
        box-sizing: border-box; /* Include padding in width */
        box-shadow: 0 -2px 10px rgba(0, 0, 0, 0);
    }

    textarea {
        position: relative;
        width: 100%;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        box-sizing: border-box; /* Include padding in width */
        height: 150px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.3);
        padding: 8px;
        margin-bottom: 16px;
        border: 1px solid rgba(0, 0, 0, 0.1);
        border-radius: 4px;
        outline: none;
        resize: none;
        background-color: white; /* Ensure background is opaque */
        font-size: 16px;
    }

    button {
        margin-top: 8px;
        padding: 8px 12px;
        border: 1px solid rgba(0, 0, 0, 0.1);
        border-radius: 4px;
        cursor: pointer;
        transition: background-color 0.3s ease, opacity 0.3s ease;
    }

    button:hover {
        opacity: 0.85;
    }



</style>

