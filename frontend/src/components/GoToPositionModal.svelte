<script>
    import { onMount } from 'svelte';
    import { positionsStore, matchContextStore, positionStore } from '../stores/positionStore'; // Import stores
    import { currentPositionIndexStore, statusBarModeStore, statusBarTextStore, commentTextStore } from '../stores/uiStore'; // Import stores
    import { analysisStore, selectedMoveStore } from '../stores/analysisStore';
    import { LoadAnalysis } from '../../wailsjs/go/main/Database.js';

    export let visible = false;
    export let onClose;

    let positionNumber = 0;
    let inputField;
    let maxPositionNumber = 0;
    let currentIndex = 0;

    // Subscribe to positionsStore and matchContextStore to get the number of positions
    $: if ($statusBarModeStore === 'MATCH' && $matchContextStore.isMatchMode) {
        maxPositionNumber = $matchContextStore.movePositions.length;
        currentIndex = $matchContextStore.currentIndex + 1; // Adjust for 1-based index
    } else {
        maxPositionNumber = $positionsStore.length;
        currentIndex = $currentPositionIndexStore + 1; // Adjust for 1-based index
    }

    async function handleGoToPosition() {
        if (positionNumber < 1) {
            positionNumber = 1;
        } else if (positionNumber > maxPositionNumber) {
            positionNumber = maxPositionNumber;
        }
        
        // Handle MATCH mode differently
        if ($statusBarModeStore === 'MATCH' && $matchContextStore.isMatchMode) {
            const newIndex = positionNumber - 1;
            matchContextStore.update(ctx => ({ ...ctx, currentIndex: newIndex }));
            const movePos = $matchContextStore.movePositions[newIndex];
            positionStore.set(movePos.position);
            
            // Load analysis for the position
            let analysis = null;
            try {
                analysis = await LoadAnalysis(movePos.position.id);
            } catch (error) {
                // No analysis for this position
            }
            
            analysisStore.set({
                positionId: analysis?.positionId || null,
                xgid: analysis?.xgid || '',
                player1: analysis?.player1 || '',
                player2: analysis?.player2 || '',
                analysisType: analysis?.analysisType || '',
                analysisEngineVersion: analysis?.analysisEngineVersion || '',
                checkerAnalysis: analysis?.checkerAnalysis || { moves: [] },
                doublingCubeAnalysis: analysis?.doublingCubeAnalysis || {
                    analysisDepth: '',
                    playerWinChances: 0,
                    playerGammonChances: 0,
                    playerBackgammonChances: 0,
                    opponentWinChances: 0,
                    opponentGammonChances: 0,
                    opponentBackgammonChances: 0,
                    cubelessNoDoubleEquity: 0,
                    cubelessDoubleEquity: 0,
                    cubefulNoDoubleEquity: 0,
                    cubefulNoDoubleError: 0,
                    cubefulDoubleTakeEquity: 0,
                    cubefulDoubleTakeError: 0,
                    cubefulDoublePassEquity: 0,
                    cubefulDoublePassError: 0,
                    bestCubeAction: '',
                    wrongPassPercentage: 0,
                    wrongTakePercentage: 0
                },
                playedMove: analysis?.playedMove || '',
                playedCubeAction: analysis?.playedCubeAction || '',
                creationDate: analysis?.creationDate || '',
                lastModifiedDate: analysis?.lastModifiedDate || ''
            });
            
            commentTextStore.set('');
            selectedMoveStore.set(null);
            
            statusBarTextStore.set(`${$matchContextStore.player1Name} vs ${$matchContextStore.player2Name}`);
        } else {
            currentPositionIndexStore.set(positionNumber - 1); // Set the store value directly
        }
        onClose(); // Close the modal after going to the position
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

    $: if (visible && $statusBarModeStore !== 'NORMAL' && $statusBarModeStore !== 'MATCH') {
        onClose(); // Close the modal if not in normal or match mode
    }
</script>

{#if visible}
<div class="modal-overlay" on:click={onClose}>
    <div class="modal-content" on:click|stopPropagation>
        <div class="close-button" on:click={onClose}>×</div>
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
        padding: 1rem;
        border-radius: 4px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        z-index: 1000;
        width: 90%;
        max-width: 300px; /* Decrease the max-width */
        max-height: 80vh; /* Limit the height of the modal */
        overflow-y: auto; /* Add vertical scrollbar if content exceeds max height */
        padding-left: 1rem; /* Add left padding to make it symmetric */
        position: relative;
        display: flex;
        flex-direction: column;
        text-align: center;
    }

    .close-button {
        position: absolute;
        top: 8px;
        right: 8px;
        font-size: 1.5rem;
        font-weight: bold;
        color: #666;
        cursor: pointer;
        z-index: 10;
        transition: background-color 0.3s ease, opacity 0.3s ease;
    }

    .input-field {
        width: 80%; /* Adjust the width */
        padding: 8px;
        margin: 8px auto; /* Center the input field */
        border: 1px solid #ccc;
        border-radius: 4px;
        box-sizing: border-box;
        font-size: 18px; /* Set font size */
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
        padding: 8px 14px; /* Slightly increase padding */
        border: none;
        border-radius: 4px;
        cursor: pointer;
        font-size: 15px; /* Slightly increase font size */
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