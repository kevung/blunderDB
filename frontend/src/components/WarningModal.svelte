<script>
    export let message = ''; // Ensure the message prop is declared
    export let visible = false;
    export let onClose = () => {};

    function handleClose() {
        onClose();
    }

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            handleClose();
        }
    }

    function handleClickOutside(event) {
        const modalContent = document.getElementById('modalContent');
        if (modalContent && !modalContent.contains(event.target)) {
            handleClose();
        }
    }

    import { onMount, onDestroy } from 'svelte';

    onMount(() => {
        window.addEventListener('keydown', handleKeyDown);
        window.addEventListener('click', handleClickOutside);
    });

    onDestroy(() => {
        window.removeEventListener('keydown', handleKeyDown);
        window.removeEventListener('click', handleClickOutside);
    });
</script>

<style>
    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background-color: rgba(0, 0, 0, 0.5);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 1000;
    }

    .modal-content {
        background-color: white;
        padding: 0; /* Remove padding */
        border-radius: 4px;
        width: 60%; /* Reduce width */
        height: 50%; /* Reduce height */
        box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
        position: relative;
        display: flex;
        flex-direction: column;
    }

    .close-button {
        position: absolute;
        top: -8px;
        right: 4px;
        font-size: 24px;
        font-weight: bold;
        color: #666;
        cursor: pointer;
        z-index: 10;
        transition: background-color 0.3s ease, opacity 0.3s ease;
    }

    .tab-content {
        flex-grow: 1;
        overflow-y: auto;
        border-top: 1px solid #ddd;
        padding: 0; /* Remove padding */
        box-sizing: border-box;
        height: calc(100% - 50px); /* Adjust height to ensure uniform tab size */
    }

    .tab-content p {
        margin: 20px; /* Add margin for spacing */
        text-align: justify;
        white-space: pre-wrap; /* Preserve whitespace for new lines */
    }

    .highlight {
        font-weight: bold;
        color: red;
    }
</style>

{#if visible}
    <div class="modal-overlay">
        <div class="modal-content" id="modalContent">
            <div class="close-button" on:click={handleClose}>×</div>
            <div class="tab-content">
                <p><span class="highlight">{message.split('\n')[0]}</span></p>
                <p>{message.split('\n').slice(1).join('\n')}</p> <!-- Ensure the message is displayed correctly -->
            </div>
        </div>
    </div>
{/if}
