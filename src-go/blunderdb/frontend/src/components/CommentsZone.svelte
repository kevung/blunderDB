<script>
    export let visible = false;
    export let onClose;
    export let text = '';

    import { onMount } from "svelte";

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            onClose();
        }
    }

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
        <div class="close-icon" on:click={onClose}>×</div>
        <textarea
            id="commentsTextArea"
            rows="5"
            cols="30"
            bind:value={text}
            placeholder="Type your comments here..."
            on:keydown={handleKeyDown}
        ></textarea>
    </div>
{/if}

<style>

    .comments-zone {
        position: absolute;
        bottom: 0; /* Above the status bar */
        left: 0;
        right: 0;
        max-height: 50vh; /* Limit height to half the viewport */
        overflow-y: auto; /* Allow scrolling inside the comment zone */
        overflow-x: hidden; /* Disable horizontal scrolling inside the comment zone */
        background-color: white;
        border-top: 1px solid rgba(0, 0, 0, 0.1);
        padding: 25px;
        box-sizing: border-box; /* Include padding in width */
        z-index: 5;
    }

    .close-icon {
        position: absolute;
        top: -6px;
        right: 4px;
        font-size: 24px;
        font-weight: bold;
        color: #666;
        cursor: pointer;
        transition: background-color 0.3s ease, opacity 0.3s ease;
    }

    textarea {
        position: relative;
        width: 100%;
        box-sizing: border-box; /* Include padding in width */
        height: 150px;
        padding: 8px;
        margin-bottom: 16px;
        border: 1px solid rgba(0, 0, 0, 0);
        border-radius: 0px;
        outline: none;
        resize: none;
        background-color: white; /* Ensure background is opaque */
        font-size: 18px;
    }

</style>

