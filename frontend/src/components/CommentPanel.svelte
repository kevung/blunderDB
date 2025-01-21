<script>
    export let visible = false;
    export let onClose;

    import { onMount } from "svelte";
    import { commentTextStore, currentPositionIndexStore } from '../stores/uiStore';
    import { SaveComment } from '../../wailsjs/go/main/Database.js';

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            onClose();
        } else if (event.ctrlKey && event.code === 'KeyL') {
            event.preventDefault();
            onClose();
            const analysisPanel = document.querySelector('.analysis-panel');
            if (analysisPanel) {
                analysisPanel.focus();
            }
        }
    }

    function handleClose() {
        SaveComment(parseInt($currentPositionIndexStore), $commentTextStore); // Ensure position ID is an int64
        onClose();
    }

    $: if (visible) {
       setTimeout(() => {
          const textAreaEl = document.getElementById('commentTextArea');
          if (textAreaEl) {
             textAreaEl.focus();
          }
       }, 0);
    }

    $: console.log('CommentPanel visibility:', visible); // Debugging log
</script>

{#if visible}
    <div class="comment-panel">
        <button class="close-icon" on:click={handleClose} aria-label="Close" on:keydown={(event) => { if (event.key === 'Enter' || event.key === ' ') handleClose(); }}>×</button>
        <textarea
            id="commentTextArea"
            rows="5"
            cols="30"
            bind:value={$commentTextStore}
            placeholder="Type your comments here..."
            on:keydown={handleKeyDown}
        ></textarea>
    </div>
{/if}

<style>
    .comment-panel {
        position: absolute;
        bottom: 0; /* Above the status bar */
        left: 0;
        right: 0;
        max-height: 50vh; /* Limit height to half the viewport */
        min-height: 200px;
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
        background: none;
        border: none;
        padding: 0;
        transition: color 0.3s ease;
    }

    .close-icon:hover {
        color: #333;
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

