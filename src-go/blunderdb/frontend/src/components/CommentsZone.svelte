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
        <div class="close-icon" on:click={onClose}>x</div>
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
        bottom: 22px; /* Above the status bar */
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

    .close-icon {
        position: absolute;
        top: 8px;
        right: 12px;
        font-size: 24px;
        font-weight: bold;
        color: #666;
        cursor: pointer;
        background-color: #d0d0d0; /* Same background as the comments zone */
        border: 1px solid rgba(0, 0, 0, 0.1); /* Border to match */
        width: 10px;
        height: 16px;
        display: flex;
        justify-content: center;
        align-items: center;
        box-shadow: 0 2px 5px rgba(0, 0, 0, 0.2); /* Add shadow for visibility */
        border-radius: 2px; /* Make it circular */
        padding: 4px 8px;
        z-index: 10; /* Ensure it's on top of other content */
        transition: background-color 0.3s ease, opacity 0.3s ease;
    }

    .close-icon:hover {
        color: #000;
        background-color: #f0f0f0; /* Slightly darker on hover */
    }

    textarea {
        position: relative;
        width: 100%;
        box-sizing: border-box; /* Include padding in width */
        height: 150px;
        padding: 8px;
        margin-bottom: 16px;
        border: 1px solid rgba(0, 0, 0, 0.3);
        border-radius: 0px;
        outline: none;
        resize: none;
        background-color: white; /* Ensure background is opaque */
        font-size: 18px;
    }

</style>

