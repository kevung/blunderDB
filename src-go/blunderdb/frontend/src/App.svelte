<script>

    // svelte functions
    import { onMount, onDestroy } from 'svelte';
    import { fade } from 'svelte/transition';

    // import backend functions
    import {
        SaveDatabaseDialog,
        OpenDatabaseDialog,
        OpenPositionDialog,
        SaveImportedPosition,
        ShowAlert,
    } from '../wailsjs/go/main/App.js';

    // import stores
    import {
        newDatabasePathStore,
        openDatabasePathStore,
    } from './stores/databaseStore';

    import {
        importPositionPathStore,
        pastePositionTextStore,
        currentPositionStore,
        listPositionStore,
        positionStore,
    } from './stores/positionStore';

    import {
        statusBarTextStore,
        statusBarModeStore,
        commandTextStore,
        commentTextStore,
        analysisDataStore,
    } from './stores/uiStore';

    // import components
    import Toolbar from './components/Toolbar.svelte';
    import Board from './components/Board.svelte';
    import CommandLine from './components/CommandLine.svelte';
    import StatusBar from './components/StatusBar.svelte';
    import AnalysisPanel from './components/AnalysisPanel.svelte';
    import CommentPanel from './components/CommentPanel.svelte';
    import HelpModal from './components/HelpModal.svelte';

    // Visibility variables
    let showCommand = false;
    let showAnalysis = false;
    let showHelp = false;
    let showComment = false;

    // Reference for various elements.
    let mainArea;
    let commandInput;

    //Global shortcuts
    function handleKeyDown(event) {
        event.stopPropagation();
        if(event.ctrlKey && event.code == 'KeyN') {
            newDatabase();
        } else if(event.ctrlKey && event.code == 'KeyO') {
            openDatabase();
        } else if (event.ctrlKey && event.code === 'KeyQ') {
            exitApp();
        } else if(event.ctrlKey && event.code == 'KeyI') {
            importPosition();
        } else if(event.ctrlKey && event.code == 'KeyC') {
            copyPosition();
        } else if(event.ctrlKey && event.code == 'KeyV') {
            pastePosition();
        } else if (!event.ctrlKey && event.key === 'PageUp') {
            event.preventDefault();
            firstPosition();
        } else if (!event.ctrlKey && event.key === 'h') {
            firstPosition();
        } else if (!event.ctrlKey && event.key === 'ArrowLeft') {
            event.preventDefault();
            previousPosition();
        } else if (!event.ctrlKey && event.key === 'k') {
            previousPosition();
        } else if (!event.ctrlKey && event.key === 'ArrowRight') {
            event.preventDefault();
            nextPosition();
        } else if (!event.ctrlKey && event.key === 'j') {
            nextPosition();
        } else if (!event.ctrlKey && event.key === 'PageDown') {
            event.preventDefault();
            lastPosition();
        } else if (!event.ctrlKey && event.key === 'l') {
            lastPosition();
        } else if(event.ctrlKey && event.code == 'KeyK') {
            gotoPosition();
        } else if(!event.ctrlKey && event.code === 'Tab') {
            if(!showHelp) {
                toggleEditMode();
            }
        } else if (!event.ctrlKey && event.code === 'Space') {
            if(!showCommand && !showComment && !showHelp) {
                event.preventDefault();
                toggleCommandMode();
            }
        } else if (event.ctrlKey && event.code === 'KeyL') {
            event.preventDefault();
            toggleAnalysisPanel();
        } else if(event.ctrlKey && event.code == 'KeyP') {
            if(!showHelp && !showCommand) {
                event.preventDefault();
                toggleCommentPanel();
            }
        } else if (event.ctrlKey && event.code === 'KeyF') {
            findPosition();
        } else if (event.ctrlKey && event.code === 'KeyH') {
            toggleHelpModal();
        } else if (!event.ctrlKey && event.key === '?') {
            toggleHelpModal();
        }
    }

    async function newDatabase() {
        console.log('newDatabase');
        try {
            const filePath = await SaveDatabaseDialog();
            if (filePath) {
                newDatabasePathStore.set(filePath);
                console.log('newDatabasePathStore:', $newDatabasePathStore);
            } else {
                console.log('No file selected');
            }
        } catch (error) {
            console.error('Error opening file dialog:', error);
        }
    }

    async function openDatabase() {
        console.log('openDatabase');
        try {
            const filePath = await OpenDatabaseDialog();
            if (filePath) {
                openDatabasePathStore.set(filePath);
                console.log('openDatabasePathStore:', $openDatabasePathStore);
            } else {
                console.log('No Database selected');
            }
        } catch (error) {
            console.error('Error opening file dialog:', error);
        }
    }

    function exitApp() {
        window.runtime.Quit();
    }

    export async function importPosition() {
        try {
            const response = await OpenPositionDialog();

            if (response.error) {
                console.error("Error:", response.error);
                return;
            }

            console.log("File path:", response.file_path);
            console.log("File content:", response.content);

            // Now you can parse and use the file content
            const importedPosition = parsePosition(response.content);
            console.log('importedPosition:', importedPosition);
            positionStore.set(importedPosition);
        } catch (error) {
            console.error("Error importing position:", error);
        }
    }

    function parsePosition(fileContent) {
        if (!fileContent || fileContent.trim().length === 0) {
            throw new Error("File is empty or invalid.");
        }

        // Normalize line endings for both Windows (\r\n) and old Mac (\r) to Unix (\n)
        const normalizedContent = fileContent.replace(/\r\n|\r/g, '\n');

        // Split the file content into lines and trim each line to remove excess whitespace
        const lines = normalizedContent.split('\n').map(line => line.trim());

        // Log each line for debugging (optional)
        lines.forEach((line, index) => console.log(`Cleaned Line ${index}: "${line}"`));

        // Parse the XGID
        const xgidLine = lines.find(line => line.startsWith("XGID="));
        const xgid = xgidLine ? xgidLine.split('=')[1] : null;

        console.log("xgidLine:", xgidLine);
        console.log("xgid:", xgid);

        if (!xgid) {
            throw new Error("XGID not found in the file content.");
        }

        // XGID components: "-A-CCD-----a---a------dfc-:4:1:-1:33:1:0:0:9:10"
        const [
            positionPart, 
            cubeValue, 
            cubeOwner, 
            playerOnRoll, 
            dicePart, 
            score1, 
            score2, 
            isCrawford, 
            matchLength, 
            dummy
        ] = xgid.split(":");

        // Decode board positions from the XGID
        const board = { points: Array(26).fill({ checkers: 0, color: -1 }), bearoff: [0, 0] };

        const pointEncoding = positionPart.slice(1);  // Remove initial '-'
        const pointChars = pointEncoding.split('');

        // Parse player on roll
        const playerOnRollValue = parseInt(playerOnRoll) === 1 ? 0 : 1;  // 0 for player1, 1 for player2
        let pointIndex = 24;  // Start from the last point (24th)
        pointChars.forEach(char => {
            if (char >= 'A' && char <= 'Z') {
                // O's checkers
                const numCheckers = char.charCodeAt(0) - 'A'.charCodeAt(0) + 1;
                board.points[pointIndex] = { checkers: numCheckers,
                    color: playerOnRollValue === 0 ? 1 : 0 };  // O is player on top
            } else if (char >= 'a' && char <= 'z') {
                // X's checkers
                const numCheckers = char.charCodeAt(0) - 'a'.charCodeAt(0) + 1;
                board.points[pointIndex] = { checkers: numCheckers,
                    color: playerOnRollValue === 0? 0 : 1 };  // X is player on bottom
            }
            pointIndex--;
        });

        // Parse dice
        const diceValues = dicePart.split("").map(num => parseInt(num));
        const dice = [diceValues[0], diceValues[1]];
        console.log('diceValues', diceValues);
        console.log('dice', dice);

        // Parse cube information
        const cube = {
            owner: parseInt(cubeOwner),
            value: parseInt(cubeValue)
        };

        // Parse scores
        const xScore = playerOnRollValue === 0 ? parseInt(score1) : parseInt(score2);
        const oScore = playerOnRollValue === 0 ? parseInt(score2) : parseInt(score1);

        // Parse match length
        const matchLengthValue = parseInt(matchLength);

        
        // Determine away scores based on isCrawford
        let awayScores;
        if (parseInt(isCrawford) === 0) {
            // If post-Crawford
            awayScores = playerOnRollValue === 0 ? [
                xScore === 1 ? 0 : matchLengthValue - xScore, // X's away score
                oScore === 1 ? 0 : matchLengthValue - oScore  // O's away score
            ] : [
                oScore === 1 ? 0 : matchLengthValue - oScore, // O's away score
                xScore === 1 ? 0 : matchLengthValue - xScore  // X's away score
            ];
        } else {
            // Calculate away scores normally
            awayScores = playerOnRollValue === 0 ? 
                [matchLengthValue - xScore, matchLengthValue - oScore] : 
                [matchLengthValue - oScore, matchLengthValue - xScore];
        }

        // Calculate bearoff counts
        const xCheckersOnBoard = board.points.reduce((sum, point) => sum + (point.color === 0 ? point.checkers : 0), 0);
        const oCheckersOnBoard = board.points.reduce((sum, point) => sum + (point.color === 1 ? point.checkers : 0), 0);

        const xBearoff = 15 - xCheckersOnBoard;
        const oBearoff = 15 - oCheckersOnBoard;

        // Store bearoff based on playerOnRollValue
        board.bearoff = playerOnRollValue === 0 ? [xBearoff, oBearoff] : [oBearoff, xBearoff];

        // Determine decision type
        const decisionLine = lines.find(line => line.includes("to play") || line.includes("jouer"));
        const decisionType = decisionLine ? 0 : 1; // 0 for checker decision, 1 for cube action


        // Return the structured Position object
        return {
            board: board,
            cube: cube,
            dice: dice,
            score: awayScores,
            player_on_roll: playerOnRollValue,
            decision_type: decisionType,  // Assuming checker action by default
        };
    }

    function copyPosition() {
        console.log('copyPosition');
    }

    function pastePosition() {
        console.log('pastePosition');
        let promise = window.runtime.ClipboardGetText();
        promise.then(
            (result) => {
                pastePositionTextStore.set(result);
                console.log('pastePositionTextStore:', $pastePositionTextStore);
            })
            .catch((error) => {
                console.error('Error pasting from clipboard:', error);
            });
    }

    function addPosition() {
        console.log('addPosition');
    }

    function updatePosition() {
        console.log('updatePosition');
    }

    function deletePosition() {
        console.log('deletePosition');
    }

    function firstPosition() {
        console.log('firstPosition');
    }

    function previousPosition() {
        console.log('previousPosition');
    }

    function nextPosition() {
        console.log('nextPosition');
    }

    function lastPosition() {
        console.log('lastPosition');
    }

    function gotoPosition() {
        console.log('gotoPosition');
    }

    function findPosition() {
        console.log('findPosition');
    }

    function toggleEditMode(){
        console.log('toggleEditMode');
        if($statusBarModeStore !== "EDIT") {
            if(showComment){
                toggleCommentPanel();
            }
            if(showAnalysis){
                toggleAnalysisPanel();
            }
            statusBarModeStore.set('EDIT');
        } else {
            statusBarModeStore.set('NORMAL');
        }
    }

    function toggleCommandMode(){
        console.log('toggleCommandMode');
        if(!showCommand) {
            statusBarModeStore.set('COMMAND');
        } else {
            statusBarModeStore.set('NORMAL');
        }
        showCommand = !showCommand;
    }

    function toggleAnalysisPanel() {
        console.log('toggleAnalysisPanel');

        if($statusBarModeStore === 'NORMAL') {
            showAnalysis = !showAnalysis;
        }

        if (showAnalysis) {
            showComment = false;
            setTimeout(() => {
            //event.preventDefault();
                document.querySelector('.analysis-panel').scrollIntoView({
                    behavior: 'smooth',
                    block: 'start' });
            }, 0);
        }
        else {
            setTimeout(() => {
                mainArea.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start' });
            }, 0);
        }
    }

    function toggleCommentPanel() {
        console.log('toggleCommentPanel');

        if($statusBarModeStore === 'NORMAL'){
            showComment = !showComment;
        }

        if (showComment) {
            showAnalysis = false;
            showCommand = false;
            setTimeout(() => {
                document.querySelector('.comment-panel').scrollIntoView({
                    behavior: 'smooth',
                    block: 'start' });
            }, 0);
        } else {
            mainArea.scrollIntoView({
                behavior: 'smooth'
            });
        }
    }

    onMount(() => {
        console.log('Wails runtime:', window.runtime);
        window.addEventListener("keydown", handleKeyDown);
    });

    onDestroy(() => {
        window.removeEventListener("keydown", handleKeyDown);
    });


    function toggleHelpModal() {
        console.log('Help button clicked!');
        showHelp = !showHelp;

        // Focus the command input when closing the Help modal
        if (!showHelp) {
            setTimeout(() => {

                if(showCommand) {
                    const commandInput = document.querySelector('.command-input');
                    if (commandInput) {
                        commandInput.focus();
                    }

                } else if(showComment) {
                    const textAreaEl = document.getElementById('commentsTextArea');
                    if (textAreaEl) {
                        textAreaEl.focus();
                    }
                }

            }, 0);
        }
    }

</script>

<main class="main-container" bind:this={mainArea}>

    <Toolbar 
        onNewDatabase={newDatabase}
        onOpenDatabase={openDatabase}
        onExit={exitApp}
        onImportPosition={importPosition}
        onCopyPosition={copyPosition}
        onPastePosition={pastePosition}
        onAddPosition={addPosition}
        onUpdatePosition={updatePosition}
        onDeletePosition={deletePosition}
        onFirstPosition={firstPosition}
        onPreviousPosition={previousPosition}
        onNextPosition={nextPosition}
        onLastPosition={lastPosition}
        onGoToPosition={gotoPosition}
        onToggleEditMode={toggleEditMode}
        onToggleCommandMode={toggleCommandMode}
        onShowAnalysis={toggleAnalysisPanel}
        onShowComment={toggleCommentPanel}
        onFindPosition={findPosition}
        onToggleHelp={toggleHelpModal}
    />

    <div class="scrollable-content">

        <Board />

        <CommandLine
            visible={showCommand}
            onClose={toggleCommandMode}
            onToggleHelp={toggleHelpModal}
            text={$commandTextStore}
            bind:this={commandInput}
        />

    </div>

    <div class="panel-container">

        <CommentPanel
            text={$commentTextStore}
            visible={showComment}
            onClose={toggleCommentPanel}
        />

        <AnalysisPanel
            visible={showAnalysis}
            analysisData={$analysisDataStore}
            onClose={toggleAnalysisPanel}
        /> 

    </div>

    <HelpModal
        visible={showHelp}
        onClose={toggleHelpModal}
        handleGlobalKeydown={handleKeyDown}
    />

    <StatusBar
        mode={$statusBarModeStore}
        text={$statusBarTextStore}
        positionIndex={$currentPositionStore}
        positionTotal={$listPositionStore}
    />

</main>

<style>
    .main-container {
        display: flex;
        flex-direction: column;
        min-height: 100vh;
        padding: 0; /* No padding so content fills entire viewport */
        box-sizing: border-box;
        position: relative;
        overflow: hidden; /* Hide overflow initially */
    }

    .scrollable-content {
        flex-grow: 1;
        overflow-y: auto; /* Allow vertical scrolling */
        overflow-x: hidden; /* Disable horizontal scrolling */
        padding: 16px; /* Add padding for content */
        width: 100%;
        box-sizing: border-box;
    }

    .comments-zone {
        position: absolute;
        bottom: 50px; /* Adjust based on height of StatusBar */
        left: 0;
        right: 0;
        max-height: 50vh; /* Limit height of comment zone */
        overflow-y: auto; /* Scroll inside the comment zone if content exceeds max height */
        overflow-x: hidden; /* Disable horizontal scrolling */
        background: white;
        border-top: 1px solid rgba(0, 0, 0, 0.1);
        padding: 16px;
        box-sizing: border-box;
        border-top: 1px solid rgba(0, 0, 0, 0.1);
    }

    .panel-container {
        display: flex;
        flex-direction: column; /* Or row, depending on layout */
        height: 100%;
    }

</style>
