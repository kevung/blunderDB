<script>
    import { onMount } from 'svelte';
    import Modal from './Modal.svelte';
    import { positionsStore, matchContextStore, positionStore } from '../stores/positionStore'; // Import stores
    import { currentPositionIndexStore, statusBarModeStore, statusBarTextStore, commentTextStore } from '../stores/uiStore'; // Import stores
    import { analysisStore, selectedMoveStore } from '../stores/analysisStore';
    import { LoadAnalysis } from '../../wailsjs/go/database/Database.js';
    import { t } from '../i18n';

    let { visible = false, onClose } = $props();

    let positionNumber = $state(0);
    let inputField = $state();
    let maxPositionNumber = $state(0);
    let currentIndex = $state(0);

    // Subscribe to positionsStore and matchContextStore to get the number of positions
    $effect(() => {
        if ($statusBarModeStore === 'MATCH' && $matchContextStore.isMatchMode) {
            maxPositionNumber = $matchContextStore.movePositions.length;
            currentIndex = $matchContextStore.currentIndex + 1; // Adjust for 1-based index
        } else {
            maxPositionNumber = $positionsStore.length;
            currentIndex = $currentPositionIndexStore + 1; // Adjust for 1-based index
        }
    });
    async function handleGoToPosition() {
        if (positionNumber < 1) {
            positionNumber = 1;
        } else if (positionNumber > maxPositionNumber) {
            positionNumber = maxPositionNumber;
        }

        // Handle MATCH mode differently
        if ($statusBarModeStore === 'MATCH' && $matchContextStore.isMatchMode) {
            const newIndex = positionNumber - 1;
            matchContextStore.update((ctx) => ({ ...ctx, currentIndex: newIndex }));
            const movePos = $matchContextStore.movePositions[newIndex];
            positionStore.set(movePos.position);

            // Load analysis for the position
            let analysis = null;
            try {
                analysis = await LoadAnalysis(movePos.position.id);
            } catch (_error) {
                // No analysis for this position
            }

            // Use the specific move from this match context, not all played moves
            const currentPlayedMove = movePos.checker_move || '';
            const currentPlayedCubeAction = movePos.cube_action || '';

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
                playedMove: currentPlayedMove,
                playedCubeAction: currentPlayedCubeAction,
                playedMoves: analysis?.playedMoves || [],
                playedCubeActions: analysis?.playedCubeActions || [],
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
        }
    }

    onMount(() => {
        if (visible && inputField) {
            positionNumber = currentIndex; // Set positionNumber to currentIndex initially
            inputField.focus();
            inputField.select(); // Select the text to allow direct replacement
        }
    });

    $effect(() => {
        if (visible && inputField) {
            inputField.focus();
            inputField.select(); // Select the text to allow direct replacement
        }
    });
    $effect(() => {
        if (visible && $statusBarModeStore === 'EDIT') {
            onClose(); // Close the modal if in edit mode
        }
    });
</script>

<Modal open={visible} onclose={onClose} size="small" align="center" closeOnOverlay>
    {#snippet title()}{$t('goToPosition.title')}{/snippet}
    <input
        type="number"
        bind:value={positionNumber}
        min="1"
        max={maxPositionNumber}
        placeholder={$t('goToPosition.placeholder')}
        class="input-field"
        bind:this={inputField}
        onkeydown={handleKeyDown}
    />
    {#snippet footer()}
        <button class="primary" onclick={handleGoToPosition}>{$t('common.go')}</button>
        <button onclick={onClose}>{$t('common.cancel')}</button>
    {/snippet}
</Modal>

<style>
    .input-field {
        width: 80%; /* Adjust the width */
        padding: 8px;
        margin: 8px auto; /* Center the input field */
        border: 1px solid #ccc;
        border-radius: 4px;
        box-sizing: border-box;
        /* Larger than body text on purpose: this is the number entry the whole
           dialog exists for. Reuses the dialog-title token rather than a bespoke
           size — see docs/adr/0008. */
        font-size: var(--font-size-dialog-title);
    }

    .input-field:focus {
        outline: none;
        border-color: #6c757d; /* Sober grey color */
        box-shadow: 0 0 5px rgba(108, 117, 125, 0.5); /* Slight shadow for focus */
    }
</style>
