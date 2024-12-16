<script>
    import { onMount } from 'svelte';

    export let visible = false;
    export let onClose;
    export let onGoToPosition;
    export let maxPositionNumber = 0;
    export let currentIndex = 0; // Add currentIndex prop

    let positionNumber = 0;
    let inputField;

    function handleGoToPosition() {
        if (positionNumber > maxPositionNumber) {
            positionNumber = maxPositionNumber;
        }
        onGoToPosition(positionNumber);
    }

    function handleKeyDown(event) {
        if (event.key === 'Enter') {
            handleGoToPosition();
        } else if (event.key === 'Escape') {
            onClose();
        }
    }

    onMount(() => {
        if (visible && inputField) {
            positionNumber = currentIndex; // Set positionNumber to currentIndex initially
            inputField.focus();
            inputField.select(); // Select the text to allow direct replacement
        }
    });

    $: if (visible && inputField) {
        inputField.focus();
        inputField.select(); // Select the text to allow direct replacement
    }
</script>

{#if visible}
<div class="modal-overlay" on:click={onClose}>
    <div class="modal-content" on:click|stopPropagation>
        <h2>Go To Position</h2>
        <input type="number" bind:value={positionNumber} min="1" max={maxPositionNumber} placeholder="Enter position number" class="input-field" bind:this={inputField} on:keydown={handleKeyDown} />
        <div class="modal-buttons">
            <button class="primary-button" on:click={handleGoToPosition}>Go</button>
            <button class="secondary-button" on:click={onClose}>Cancel</button>
        </div>
    </div>
</div>
{/if}

<style>
    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.5);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
    }

    .modal-content {
        background: white;
        padding: 10px;
        border-radius: 8px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        text-align: center;
        width: 250px;
        font-size: 14px; /* Same font size as in AnalysisPanel */
    }

    .input-field {
        width: 40%; /* Make the input field smaller */
        padding: 8px;
        margin: 8px 0;
        border: 1px solid #ccc;
        border-radius: 4px;
        box-sizing: border-box;
    }

    .input-field:focus {
        outline: none;
        border-color: #6c757d; /* Sober grey color */
        box-shadow: 0 0 5px rgba(108, 117, 125, 0.5); /* Slight shadow for focus */
    }

    .modal-buttons {
        margin-top: 10px;
        display: flex;
        justify-content: center;
        gap: 10px; /* Add space between buttons */
    }

    .modal-buttons button {
        padding: 8px 16px;
        border: none;
        border-radius: 4px;
        cursor: pointer;
    }

    .primary-button {
        background-color: #6c757d; /* Sober grey color */
        color: white;
    }

    .secondary-button {
        background-color: #ccc;
    }

    .primary-button:hover {
        background-color: #5a6268; /* Slightly darker grey on hover */
    }

    .secondary-button:hover {
        background-color: #999;
    }
</style>