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
        analysisStore,
    } from './stores/analysisStore';

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
            const {positionData, parsedAnalysis} = parsePosition(response.content);
            console.log('positionData:', positionData);
            console.log('parsedAnalysis:', parsedAnalysis);
            positionStore.set(positionData);
            analysisStore.set(parsedAnalysis);
        } catch (error) {
            console.error("Error importing position:", error);
        }
    }

    function parsePosition(fileContent) {
        if (!fileContent || fileContent.trim().length === 0) {
            throw new Error("File is empty or invalid.");
        }

        // Normalize line endings for both Windows (\r\n) and old Mac (\r) to Unix (\n)
        //const normalizedContent = fileContent.replace(/\r\n|\r/g, '\n');
        const normalizedContent = fileContent
            .replace(/\r\n|\r/g, '\n') // Normalize line endings
            .trim(); // Remove leading and trailing spaces


        // Split the file content into lines and trim each line to remove excess whitespace
        const lines = normalizedContent.split('\n').map(line => line.trim());

        // Log each line for debugging (optional)
        lines.forEach((line, index) => console.log(`Cleaned Line ${index}: "${line}"`));

        // Detect if the file is in French or English by checking for French words
        const isFrench = normalizedContent.includes("Joueur") || normalizedContent.includes("Adversaire");


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
            playerDownOnDiagram, 
            dicePart, 
            score1, 
            score2, 
            isCrawford, 
            matchLength, 
            dummy
        ] = xgid.split(":");

        // Decode board positions from the XGID
        const board = { points: Array(26).fill({ checkers: 0, color: -1 }), bearoff: [0, 0] };

        const pointChars = positionPart.split('');

        // Parse player on roll
        let pointIndex = 0;  // Start from the last point (24th)
        pointChars.forEach(char => {
            if (char >= 'A' && char <= 'Z') {
                const numCheckers = char.charCodeAt(0) - 'A'.charCodeAt(0) + 1;
                board.points[pointIndex] = { checkers: numCheckers, color: 0 };
            } else if (char >= 'a' && char <= 'z') {
                const numCheckers = char.charCodeAt(0) - 'a'.charCodeAt(0) + 1;
                board.points[pointIndex] = { checkers: numCheckers, color: 1 };
            }
            pointIndex++;
        });

        // Parse dice
        const diceValues = dicePart.split("").map(num => parseInt(num));
        const dice = [diceValues[0], diceValues[1]];
        console.log('diceValues', diceValues);
        console.log('dice', dice);

        const player1Score = parseInt(score1);
        const player2Score = parseInt(score2);
        const matchLengthValue = parseInt(matchLength);
        const playerOnRoll = parseInt(playerDownOnDiagram) === 1 ? 0 : 1;  // 0 for player1, 1 for player2
        


        // if unlimited mode
        let hasJacoby = 0, hasBeaver = 0, awayScores = [matchLengthValue - player1Score, matchLengthValue - player2Score];
        if (parseInt(isCrawford) === 0) {
            awayScores = awayScores.map(score => score === 1 ? 0 : score);
        }
        if (matchLengthValue === 0) {
            awayScores = [-1, -1];
            console.log('isCrawford', parseInt(isCrawford));
            switch (parseInt(isCrawford)) {
                case 1: hasJacoby = 1; break;
                case 2: hasBeaver = 1; break;
                case 3: hasJacoby = 1; hasBeaver = 1; break;
            }
        }

        const player1Bearoff = 15 - board.points.reduce((sum, point) => sum + (point.color === 0 ? point.checkers : 0), 0);
        const player2Bearoff = 15 - board.points.reduce((sum, point) => sum + (point.color === 1 ? point.checkers : 0), 0);
        board.bearoff = [player1Bearoff, player2Bearoff];

        const decisionLine = lines.find(line => line.includes(isFrench ? "jouer" : "to play"));
        const decisionType = decisionLine ? 0 : 1; // 0 for checker decision, 1 for cube action

        const positionData = {
            board: board,
            cube: {
                owner: parseInt(cubeOwner) === 1 ? 0 : parseInt(cubeOwner) === -1 ? 1 : -1,
                value: parseInt(cubeValue)
            },
            dice: dice,
            score: awayScores,
            player_on_roll: playerOnRoll,
            decision_type: decisionType,
            has_jacoby: hasJacoby,
            has_beaver: hasBeaver,
        };

        // Analysis Parsing
        const parsedAnalysis = { xgid, analysisType: "", checkerAnalysis: [], doublingCubeAnalysis: {} };

        // Doubling Cube Analysis Parsing
        if (normalizedContent.includes(isFrench ? "Equités sans videau" : "Cubeless Equities") || normalizedContent.includes(isFrench ? "Equités avec videau" : "Cubeful Equities")) {
            parsedAnalysis.analysisType = "DoublingCube";

            // Player Winning Chances: 79.25% (G:57.02% B:8.79%)
            const playerWinMatch = normalizedContent.match(new RegExp(isFrench ? /Joueur gagnant chances:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/ : /Player Winning Chances:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/));

            // Opponent Winning Chances: 20.75% (G:0.00% B:0.00%)
            const opponentWinMatch = normalizedContent.match(new RegExp(isFrench ? /Adversaire gagnant chances:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/ : /Opponent Winning Chances:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/));

            // Cubeless Equities: No Double=+1.286, Double=+1.405
            const cubelessMatch = normalizedContent.match(new RegExp(isFrench ? /Equité sans double\s*:\s*Pas de double=([\+\-\d.]+),\s*Double=([\+\-\d.]+)/ : /Cubeless Equities:\s*No Double=([\+\-\d.]+),\s*Double=([\+\-\d.]+)/));

            // Cubeful Equities: No double: +1.065
            const cubefulNoDoubleMatch = normalizedContent.match(new RegExp(isFrench ? /Pas de double\s*:\s*([\+\-\d.]+)/ : /No double\s*:\s*([\+\-\d.]+)/));

            const cubefulDoubleTakeMatch = normalizedContent.match(new RegExp(isFrench ? /Double\/Prend:\s+([\+\-\d.]+) \(([\+\-\d.]+)\)/ : /Double\/Take:\s+([\+\-\d.]+) \(([\+\-\d.]+)\)/));
            const cubefulDoublePassMatch = normalizedContent.match(new RegExp(isFrench ? /Double\/Passe:\s+([\+\-\d.]+) \(([\+\-\d.]+)\)/ : /Double\/Pass:\s+([\+\-\d.]+) \(([\+\-\d.]+)\)/));

            // Best Cube action parsing
            const bestCubeActionMatch = normalizedContent.match(/Best Cube action:\s*(.*)/);

            if (playerWinMatch) {
                parsedAnalysis.doublingCubeAnalysis.playerWinChances = parseFloat(playerWinMatch[1]);
                parsedAnalysis.doublingCubeAnalysis.playerGammonChances = parseFloat(playerWinMatch[2]);
                parsedAnalysis.doublingCubeAnalysis.playerBackgammonChances = parseFloat(playerWinMatch[3]);
            }
            if (opponentWinMatch) {
                parsedAnalysis.doublingCubeAnalysis.opponentWinChances = parseFloat(opponentWinMatch[1]);
                parsedAnalysis.doublingCubeAnalysis.opponentGammonChances = parseFloat(opponentWinMatch[2]);
                parsedAnalysis.doublingCubeAnalysis.opponentBackgammonChances = parseFloat(opponentWinMatch[3]);
            }
            if (cubelessMatch) {
                parsedAnalysis.doublingCubeAnalysis.cubelessNoDoubleEquity = parseFloat(cubelessMatch[1]);
                parsedAnalysis.doublingCubeAnalysis.cubelessDoubleEquity = parseFloat(cubelessMatch[2]);
            }
            if (cubefulNoDoubleMatch) {
                parsedAnalysis.doublingCubeAnalysis.cubefulNoDoubleEquity = parseFloat(cubefulNoDoubleMatch[1]);
                parsedAnalysis.doublingCubeAnalysis.cubefulNoDoubleError = cubefulNoDoubleMatch[2] ? parseFloat(cubefulNoDoubleMatch[2]) : 0;
            }
            if (cubefulDoubleTakeMatch) {
                parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeEquity = parseFloat(cubefulDoubleTakeMatch[1]);
                parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeError = cubefulDoubleTakeMatch[2] ? parseFloat(cubefulDoubleTakeMatch[2]) : 0;
            }
            if (cubefulDoublePassMatch) {
                parsedAnalysis.doublingCubeAnalysis.cubefulDoublePassEquity = parseFloat(cubefulDoublePassMatch[1]);
                parsedAnalysis.doublingCubeAnalysis.cubefulDoublePassError = cubefulDoublePassMatch[2] ? parseFloat(cubefulDoublePassMatch[2]) : 0;
            }
            if (bestCubeActionMatch) {
                parsedAnalysis.doublingCubeAnalysis.bestCubeAction = bestCubeActionMatch[1].trim();
            }



        } else if (/^\s*\d+\.\s+[A-Za-z0-9\+\-]+(?:\s+[A-Za-z0-9\+\-]+)*\s+[A-Za-z0-9\/\- ]+\s+/gm.test(normalizedContent)) {
            parsedAnalysis.analysisType = "CheckerMove";

            // Checker Move Analysis Parsing for both English and French
            const moveRegex = new RegExp(isFrench ? /^ {4}(\d+)\.\s+([A-Za-z0-9\+\-]+(?:\s+[A-Za-z0-9\+\-]+)*)\s+([A-Za-z0-9\/\- ]+)\s+éq:([-.\d]+)\s*(?:\((-?[-.\d]+)\))?/ : 
                /^ {4}(\d+)\.\s+([A-Za-z0-9\+\-]+(?:\s+[A-Za-z0-9\+\-]+)*)\s+([A-Za-z0-9\/\- ]+)\s+eq:([-.\d]+)\s*(?:\((-?[-.\d]+)\))?/,
                'gm');

            let moveMatch;
            let playerMatch;
            let opponentMatch;

            // Loop through all move lines in the normalized content
            while ((moveMatch = moveRegex.exec(normalizedContent)) !== null) {
                // Extract the move details from moveRegex match
                const moveDetails = {
                    index: parseInt(moveMatch[1], 10), // Move number (e.g., 1)
                    analysisDepth: moveMatch[2].trim(), // XG Roller++ or ply value
                    move: moveMatch[3].trim(), // The move description (e.g., 20/15 10/8)
                    equity: parseFloat(moveMatch[4]), // Equity value (e.g., -1.448)
                    equityError: moveMatch[5] ? parseFloat(moveMatch[5]) : 0, // Optional equity change value (e.g., -0.038)
                };

                // Find the next lines for Player and Opponent information
                const lineStart = moveMatch.index + moveMatch[0].length; // Position after the main move line
                const remainingContent = normalizedContent.slice(lineStart);

                // Check for Player and Opponent percentage details in the following lines
                // This regex pattern ensures it matches Player and Opponent details on the next lines
                const playerRegex = isFrench
                    ? /Joueur:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)%\s+B:(\d+\.\d+)%\)/  // French version
                    : /Player:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)%\s+B:(\d+\.\d+)%\)/;  // English version

                const opponentRegex = isFrench
                    ? /Adversaire:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)%\s+B:(\d+\.\d+)%\)/  // French version
                    : /Opponent:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)%\s+B:(\d+\.\d+)%\)/;  // English version

                // Capture Player details
                if ((playerMatch = playerRegex.exec(remainingContent)) !== null) {
                    moveDetails.playerWinChance = parseFloat(playerMatch[1]);
                    moveDetails.playerGammonChance = parseFloat(playerMatch[2]);
                    moveDetails.playerBackgammonChance = parseFloat(playerMatch[3]);
                }

                // Capture Opponent details
                if ((opponentMatch = opponentRegex.exec(remainingContent)) !== null) {
                    moveDetails.opponentWinChance = parseFloat(opponentMatch[1]);
                    moveDetails.opponentGammonChance = parseFloat(opponentMatch[2]);
                    moveDetails.opponentBackgammonChance = parseFloat(opponentMatch[3]);
                }

                // Push the current move details into checkerAnalysis array
                parsedAnalysis.checkerAnalysis.push(moveDetails);
            }

        }

        return { positionData, parsedAnalysis };

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
                const importedPosition = parsePosition(result);
                console.log('importedPosition:', importedPosition);
                positionStore.set(importedPosition);
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
