<script>

    // svelte functions
    import { onMount, onDestroy } from 'svelte';
    import { fade } from 'svelte/transition';

    // import backend functions
    import {
        SaveDatabaseDialog,
        OpenDatabaseDialog,
        OpenPositionDialog,
        ShowAlert,
        DeleteFile
    } from '../wailsjs/go/main/App.js';
    import {
        SetupDatabase,
        SavePosition,
        SaveAnalysis,
        PositionExists,
        LoadAllPositions,
        LoadAllAnalyses,
        DeletePosition,
        DeleteAnalysis,
        UpdatePosition,
        LoadComment,
        SaveComment
    } from '../wailsjs/go/main/Database.js';

    import { WindowSetTitle } from '../wailsjs/runtime/runtime.js';

    // import stores
    import {
        databasePathStore,
    } from './stores/databaseStore';

    import {
        importPositionPathStore,
        pastePositionTextStore,
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
    import GoToPositionModal from './components/GoToPositionModal.svelte';

    // Visibility variables
    let showCommand = false;
    let showAnalysis = false;
    let showHelp = false;
    let showComment = false;
    let showGoToPositionModal = false;

    // Reference for various elements.
    let mainArea;
    let commandInput;

    let currentPositionIndex = 0;
    let totalPositions = 0;
    let positions = [];
    let analyses = [];
    let db;

    //Global shortcuts
    function handleKeyDown(event) {
        event.stopPropagation();

        // Prevent all shortcuts except toggleCommentPanel when comment panel is visible and focused
        if (showComment && document.activeElement.id === 'commentTextArea') {
            if (event.ctrlKey && event.code === 'KeyP') {
                event.preventDefault();
                toggleCommentPanel();
            }
            return;
        }

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
        } else if(event.ctrlKey && event.code == 'KeyS') {
            saveCurrentPosition();
        } else if(event.ctrlKey && event.code == 'KeyU') {
            updatePosition();
        } else if (event.code === 'Delete') {
            deletePosition();
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
        } else if (event.ctrlKey && event.key === 'ArrowLeft') {
            setBoardOrientation("left");
        } else if (event.ctrlKey && event.key === 'ArrowRight') {
            setBoardOrientation("right");
        }
    }

    // Function to update the status bar message temporarily
    function updateStatusBarMessage(message, duration = 5000) {
        statusBarTextStore.set(message);
        setTimeout(() => {
            statusBarTextStore.set('');
        }, duration);
    }

    function getFilenameFromPath(filePath) {
        return filePath.split('/').pop();
    }

    async function newDatabase() {
        console.log('newDatabase');
        try {
            const filePath = await SaveDatabaseDialog();
            if (filePath) {
                // Check if the file exists and delete it
                try {
                    await DeleteFile(filePath);
                    console.log('Existing file deleted:', filePath);
                } catch (error) {
                    console.log('No existing file to delete or error deleting file:', error);
                }

                databasePathStore.set(filePath);
                console.log('databasePathStore:', $databasePathStore);
                await SetupDatabase(filePath);
                updateStatusBarMessage('New database created successfully');
                const filename = getFilenameFromPath(filePath);
                WindowSetTitle(`blunderDB - ${filename}`);
                console.log(`New database created at ${filePath}`);
            } else {
                console.log('No file selected');
            }
        } catch (error) {
            console.error('Error opening file dialog:', error);
            updateStatusBarMessage('Error creating new database');
        }
    }

    async function openDatabase() {
        console.log('openDatabase');
        try {
            const filePath = await OpenDatabaseDialog();
            if (filePath) {
                databasePathStore.set(filePath);
                console.log('databasePathStore:', $databasePathStore);
                await SetupDatabase(filePath);
                updateStatusBarMessage('Database opened successfully');
                const filename = getFilenameFromPath(filePath);
                WindowSetTitle(`blunderDB - ${filename}`);

                // Load positions and analyses
                positions = await LoadAllPositions();
                analyses = await LoadAllAnalyses();

                // Update status bar and show the last position by default
                if (positions.length > 0) {
                    currentPositionIndex = positions.length - 1;
                    updateStatusBar(currentPositionIndex, positions.length);
                    showPosition(positions[currentPositionIndex], analyses[currentPositionIndex]);
                }
            } else {
                console.log('No Database selected');
            }
        } catch (error) {
            console.error('Error opening file dialog:', error);
            updateStatusBarMessage('Error opening database');
        }
    }

    function exitApp() {
        window.runtime.Quit();
    }

    async function savePositionAndAnalysis(positionData, parsedAnalysis, successMessage) {
        // Ensure checkerAnalysis is correctly structured
        if (Array.isArray(parsedAnalysis.checkerAnalysis)) {
            parsedAnalysis.checkerAnalysis = { moves: parsedAnalysis.checkerAnalysis };
        }

        // Check if the position already exists
        const positionExistsResult = await PositionExists(positionData);
        if (positionExistsResult.exists) {
            console.log('Position already exists with ID:', positionExistsResult.id);
            try {
                await SaveAnalysis(positionExistsResult.id, parsedAnalysis);
                console.log('Analysis updated for position ID:', positionExistsResult.id);
                updateStatusBarMessage('Position already exists, analysis updated');
            } catch (error) {
                console.error('Error updating analysis:', error);
                updateStatusBarMessage('Error updating analysis');
            }
            return;
        }

        // Save the position and analysis to the database
        try {
            const positionID = await SavePosition(positionData);
            console.log('Position saved with ID:', positionID);

            await SaveAnalysis(positionID, parsedAnalysis);
            console.log('Analysis saved for position ID:', positionID);

            // Reload all positions and show the last one
            positions = await LoadAllPositions();
            analyses = await LoadAllAnalyses();

            if (positions.length > 0) {
                currentPositionIndex = positions.length - 1;
                updateStatusBar(currentPositionIndex, positions.length);
                showPosition(positions[currentPositionIndex], analyses[currentPositionIndex]);
            }

            updateStatusBarMessage(successMessage);
        } catch (error) {
            console.error('Error saving position and analysis:', error);
            updateStatusBarMessage('Error saving position and analysis');
        }
    }

    export async function importPosition() {
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            return;
        }
        try {
            const response = await OpenPositionDialog();

            if (response.error) {
                console.error("Error:", response.error);
                return;
            }

            console.log("File path:", response.file_path);
            console.log("File content:", response.content);

            // Now you can parse and use the file content
            const { positionData, parsedAnalysis } = parsePosition(response.content);
            positionStore.set(positionData);
            analysisStore.set(parsedAnalysis);
            console.log('positionStore:', $positionStore);
            console.log('analysisStore:', $analysisStore);

            await savePositionAndAnalysis(positionData, parsedAnalysis, 'Imported position and analysis saved successfully');
        } catch (error) {
            console.error("Error importing position:", error);
        }
    }

    async function pastePosition() {
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            return;
        }
        console.log('pastePosition');
        let promise = window.runtime.ClipboardGetText();
        promise.then(
            async (result) => {
                pastePositionTextStore.set(result);
                console.log('pastePositionTextStore:', $pastePositionTextStore);
                const { positionData, parsedAnalysis } = parsePosition(result);
                positionStore.set(positionData);
                analysisStore.set(parsedAnalysis);
                console.log('positionStore:', $positionStore);
                console.log('analysisStore:', $analysisStore);

                await savePositionAndAnalysis(positionData, parsedAnalysis, 'Pasted position and analysis saved successfully');
                statusBarModeStore.set('NORMAL'); // Set to normal mode after pasting
            })
            .catch((error) => {
                console.error('Error pasting from clipboard:', error);
            });
    }

    async function saveCurrentPosition() {
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            return;
        }
        console.log('saveCurrentPosition');
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            return;
        }

        const position = $positionStore;
        const analysis = $analysisStore;

        if (!isValidPosition(position)) {
            return;
        }

        console.log('Position to save:', position);
        console.log('Analysis to save:', analysis);

        // Reset all fields of analysis to initialized values
        analysis.xgid = "";
        analysis.analysisType = "";
        analysis.checkerAnalysis = [];
        analysis.doublingCubeAnalysis = {};
        analysis.analysisEngineVersion = "";

        // Generate XGID in analysis
        analysis.xgid = generateXGID(position);

        await savePositionAndAnalysis(position, analysis, 'Position and analysis saved successfully');
        statusBarModeStore.set('NORMAL');
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
        const isFrench = normalizedContent.includes("Joueur")
            || normalizedContent.includes("Adversaire")
            || normalizedContent.includes("Videau");


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
        const parsedAnalysis = { xgid, analysisType: "", checkerAnalysis: [], doublingCubeAnalysis: {}, analysisEngineVersion: "" };

        const engineVersionMatch = normalizedContent.match(new RegExp(/eXtreme Gammon Version: .*/));  // For English version, match the full engine version line

        if (engineVersionMatch) {
            parsedAnalysis.analysisEngineVersion = engineVersionMatch[0]; // Store the whole line as-is
        }

        // Doubling Cube Analysis Parsing
        if (
            (isFrench && (normalizedContent.includes("Equités sans videau") || normalizedContent.includes("Equités avec videau"))) ||
            (!isFrench && (normalizedContent.includes("Cubeless Equities") || normalizedContent.includes("Cubeful Equities")))
        ) {
            parsedAnalysis.analysisType = "DoublingCube";

            const analysisDepthMatch = normalizedContent.match(new RegExp(isFrench ? /Analysé avec\s+([^\n]*)/ : /Analyzed in\s+([^\n]*)/));

            const playerWinMatch = normalizedContent.match(new RegExp(isFrench ? /Chance de gain du joueur:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/ : /Player Winning Chances:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/));

            const opponentWinMatch = normalizedContent.match(new RegExp(isFrench ? /Chance de gain de l'adversaire:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/ : /Opponent Winning Chances:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/));

            const cubelessMatch = normalizedContent.match(new RegExp(isFrench ? /Equités sans videau\s*:\s*Pas de double=([\+\-\d.]+),\s*Double=([\+\-\d.]+)/ : /Cubeless Equities:\s*No Double=([\+\-\d.]+),\s*Double=([\+\-\d.]+)/));

            // Cubeful Equities with optional error margin for No Double, Double/Take, and Double/Pass
            const cubefulNoDoubleMatch = normalizedContent.match(new RegExp(isFrench ? /Pas de double\s*:\s*([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /No double\s*:\s*([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));
            const cubefulDoubleTakeMatch = normalizedContent.match(new RegExp(isFrench ? /Double\/Prend:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /Double\/Take:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));
            const cubefulDoublePassMatch = normalizedContent.match(new RegExp(isFrench ? /Double\/Passe:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /Double\/Pass:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));
                // Redouble handling
            const redoubleNoMatch = normalizedContent.match(new RegExp(isFrench ? /Pas de redouble\s*:\s*([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /No redouble\s*:\s*([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));
            const redoubleTakeMatch = normalizedContent.match(new RegExp(isFrench ? /Redouble\/Prend:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /Redouble\/Take:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));
            const redoublePassMatch = normalizedContent.match(new RegExp(isFrench ? /Redouble\/Passe:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /Redouble\/Pass:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));


            // Best Cube action parsing
            const bestCubeActionMatch = normalizedContent.match(new RegExp(isFrench ? /Meilleur action du videau:\s*(.*)/ : /Best Cube action:\s*(.*)/));

            const wrongPassPercentageMatch = normalizedContent.match(new RegExp(isFrench ? /Pourcentage de passes incorrectes pour rendre la décision de double correcte:\s*(\d+\.\d+)%/ : /Percentage of wrong pass needed to make the double decision right:\s*(\d+\.\d+)%/));
            const wrongTakePercentageMatch = normalizedContent.match(new RegExp(isFrench ? /Pourcentage de prises incorrectes pour rendre la décision de double correcte:\s*(\d+\.\d+)%/ : /Percentage of wrong take needed to make the double decision right:\s*(\d+\.\d+)%/));



            if (analysisDepthMatch) {
                parsedAnalysis.doublingCubeAnalysis.analysisDepth = analysisDepthMatch[1].trim();
            }
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
            // Redouble Handling
            if (redoubleNoMatch) {
                parsedAnalysis.doublingCubeAnalysis.cubefulNoDoubleEquity = parseFloat(redoubleNoMatch[1]);
                parsedAnalysis.doublingCubeAnalysis.cubefulNoDoubleError = redoubleNoMatch[2] ? parseFloat(redoubleNoMatch[2]) : 0;
            }
            if (redoubleTakeMatch) {
                parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeEquity = parseFloat(redoubleTakeMatch[1]);
                parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeError = parseFloat(redoubleTakeMatch[2]) ? parseFloat(redoubleTakeMatch[2]) : 0;
            }
            if (redoublePassMatch) {
                parsedAnalysis.doublingCubeAnalysis.cubefulDoublePassEquity = parseFloat(redoublePassMatch[1]);
                parsedAnalysis.doublingCubeAnalysis.cubefulDoublePassError = parseFloat(redoublePassMatch[2]) ? parseFloat(redoublePassMatch[2]) : 0;
            }
            if (bestCubeActionMatch) {
                parsedAnalysis.doublingCubeAnalysis.bestCubeAction = bestCubeActionMatch[1].trim();
            }
            if (wrongPassPercentageMatch) {
                parsedAnalysis.doublingCubeAnalysis.wrongPassPercentage = parseFloat(wrongPassPercentageMatch[1]);
            }
            if (wrongTakePercentageMatch) {
                parsedAnalysis.doublingCubeAnalysis.wrongTakePercentage = parseFloat(wrongTakePercentageMatch[1]);
            }



        } else if (/^ {4}(\d+)\./gm.test(normalizedContent)) {
            parsedAnalysis.analysisType = "CheckerMove";

            // Checker Move Analysis Parsing for both English and French

            const moveRegex = new RegExp(
                isFrench
                ? /^ {4}(\d+)\.\s(.{11})\s(.{28})\séq:(.{6})\s(?:\((-?[-.\d]+)\))?/
                : /^ {4}(\d+)\.\s(.{11})\s(.{28})\seq:(.{6})\s(?:\((-?[-.\d.]+)\))?/,
                'gm'
            );

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

    function generateXGID(position) {
        const { board, cube, dice, score, player_on_roll, decision_type } = position;

        // Encode board positions
        let positionPart = '';
        for (let i = 0; i < 26; i++) {
            const point = board.points[i];
            if (point.checkers > 0) {
                const charCode = point.color === 0 ? 'A'.charCodeAt(0) : 'a'.charCodeAt(0);
                positionPart += String.fromCharCode(charCode + point.checkers - 1);
            } else {
                positionPart += '-';
            }
        }

        // Encode cube value and owner
        const cubeValue = cube.value;
        const cubeOwner = cube.owner === 0 ? 1 : cube.owner === 1 ? -1 : 0;

        // Encode dice
        const dicePart = decision_type === 1 ? '00' : dice.join('');

        // Compute minimal match length
        const matchLength = score[0] === -1 || score[1] === -1 ? 0 : Math.max(score[0], score[1]);

        // Deduce actual scores
        const actualScore1 = score[0] === -1 ? 0 : matchLength - score[0];
        const actualScore2 = score[1] === -1 ? 0 : matchLength - score[1];

        // Encode Crawford status
        const isCrawford = score[0] === 1 || score[1] === 1 ? 1 : 0;

        // Encode dummy value (not used)
        const dummy = 0;

        // Correctly encode player on roll
        const playerOnRoll = player_on_roll === 0 ? 1 : -1;

        // Combine all parts to form the XGID
        const xgid = `${positionPart}:${cubeValue}:${cubeOwner}:${playerOnRoll}:${dicePart}:${actualScore1}:${actualScore2}:${isCrawford}:${matchLength}:${dummy}`;
        return xgid;
    }

    function copyPosition() {
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            return;
        }
        console.log('copyPosition');
        const position = $positionStore;
        const analysis = $analysisStore;

        // Generate XGID if not present in the analysis
        const xgid = analysis.xgid || generateXGID(position);

        // Construct the clipboard content
        let clipboardContent = `XGID=${xgid}\n\n`;

        // Add position details
        clipboardContent += `Position:\n`;
        clipboardContent += `Board: ${JSON.stringify(position.board)}\n`;
        clipboardContent += `Cube: ${JSON.stringify(position.cube)}\n`;
        clipboardContent += `Dice: ${position.dice.join(', ')}\n`;
        clipboardContent += `Score: ${position.score.join(', ')}\n`;
        clipboardContent += `Player on roll: ${position.player_on_roll}\n`;
        clipboardContent += `Decision type: ${position.decision_type}\n\n`;

        // Add analysis details
        clipboardContent += `Analysis:\n`;
        if (analysis.analysisType === "DoublingCube") {
            clipboardContent += `Doubling Cube Analysis:\n`;
            clipboardContent += `Player Win Chances: ${analysis.doublingCubeAnalysis.playerWinChances}%\n`;
            clipboardContent += `Player Gammon Chances: ${analysis.doublingCubeAnalysis.playerGammonChances}%\n`;
            clipboardContent += `Player Backgammon Chances: ${analysis.doublingCubeAnalysis.playerBackgammonChances}%\n`;
            clipboardContent += `Opponent Win Chances: ${analysis.doublingCubeAnalysis.opponentWinChances}%\n`;
            clipboardContent += `Opponent Gammon Chances: ${analysis.doublingCubeAnalysis.opponentGammonChances}%\n`;
            clipboardContent += `Opponent Backgammon Chances: ${analysis.doublingCubeAnalysis.opponentBackgammonChances}%\n`;
            clipboardContent += `Cubeless No Double Equity: ${analysis.doublingCubeAnalysis.cubelessNoDoubleEquity}\n`;
            clipboardContent += `Cubeless Double Equity: ${analysis.doublingCubeAnalysis.cubelessDoubleEquity}\n`;
            clipboardContent += `Cubeful No Double Equity: ${analysis.doublingCubeAnalysis.cubefulNoDoubleEquity}\n`;
            clipboardContent += `Cubeful Double Take Equity: ${analysis.doublingCubeAnalysis.cubefulDoubleTakeEquity}\n`;
            clipboardContent += `Cubeful Double Pass Equity: ${analysis.doublingCubeAnalysis.cubefulDoublePassEquity}\n`;
        } else if (analysis.analysisType === "CheckerMove") {
            clipboardContent += `Checker Move Analysis:\n`;
            analysis.checkerAnalysis.moves.forEach(move => {
                clipboardContent += `Move ${move.index}: ${move.move}\n`;
                clipboardContent += `Equity: ${move.equity}\n`;
                clipboardContent += `Player Win Chance: ${move.playerWinChance}%\n`;
                clipboardContent += `Player Gammon Chance: ${move.playerGammonChance}%\n`;
                clipboardContent += `Player Backgammon Chance: ${move.playerBackgammonChance}%\n`;
                clipboardContent += `Opponent Win Chance: ${move.opponentWinChance}%\n`;
                clipboardContent += `Opponent Gammon Chance: ${move.opponentGammonChance}%\n`;
                clipboardContent += `Opponent Backgammon Chance: ${move.opponentBackgammonChance}%\n\n`;
            });
        }

        // Copy to clipboard
        navigator.clipboard.writeText(clipboardContent).then(() => {
            console.log('Position and analysis copied to clipboard');
            updateStatusBarMessage('Position and analysis copied to clipboard');
        }).catch(err => {
            console.error('Error copying to clipboard:', err);
            updateStatusBarMessage('Error copying to clipboard');
        });
    }

    function isValidPosition(position) {
        const player1Checkers = position.board.points.reduce((acc, point) => acc + (point.color === 0 ? point.checkers : 0), 0);
        const player2Checkers = position.board.points.reduce((acc, point) => acc + (point.color === 1 ? point.checkers : 0), 0);

        if (player1Checkers > 15) {
            updateStatusBarMessage('Invalid position: Player 1 has more than 15 checkers');
            return false;
        }

        if (player2Checkers > 15) {
            updateStatusBarMessage('Invalid position: Player 2 has more than 15 checkers');
            return false;
        }

        if (position.decision_type === 1) {
            if (position.cube.owner !== position.player_on_roll && position.cube.owner !== -1) {
                updateStatusBarMessage('Invalid position: Cube is not available for doubling');
                return false;
            }
            if (position.score[position.player_on_roll] === 1) {
                updateStatusBarMessage('Invalid position: Crawford rule prevents doubling');
                return false;
            }
        }

        return true;
    }

    async function deletePosition() {
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            return;
        }
        console.log('deletePosition');
        if ($statusBarModeStore !== 'NORMAL' && $statusBarModeStore !== 'COMMAND') {
            updateStatusBarMessage('Cannot delete position in current mode');
            return;
        }

        if (!positions || positions.length === 0) {
            updateStatusBarMessage('No positions to delete');
            return;
        }

        try {
            const positionID = positions[currentPositionIndex].id;
            await DeletePosition(positionID);
            console.log('Position and associated analysis deleted with ID:', positionID);

            // Load all positions from the database
            positions = await LoadAllPositions();

            if (!positions || positions.length === 0) {
                // If no positions left, initialize positionStore to its initial value
                positionStore.set({
                    board: {
                        points: Array(26).fill({ checkers: 0, color: -1 }), // 24 points + 2 bars
                        bearoff: [15, 15],
                    },
                    cube: {
                        owner: -1,
                        value: 0,
                    },
                    dice: [3, 1],
                    score: [-1, -1],
                    player_on_roll: 0,
                    decision_type: 0,
                    has_jacoby: 0,
                    has_beaver: 0,
                });
                console.log('Database is empty. Showing index position 0 on total number position equal to 0.');
                updateStatusBar(0, 0); // Reset status bar to 0 / 0
            } else {
                // Go to the last position
                const lastPosition = positions[positions.length - 1];
                positionStore.set(lastPosition);
                console.log(`Showing index position ${positions.length - 1} on total number position equal to ${positions.length}.`);
                updateStatusBar(positions.length - 1, positions.length);
            }

            updateStatusBarMessage('Position and associated analysis deleted successfully');
        } catch (error) {
            console.error('Error deleting position and associated analysis:', error);
            updateStatusBarMessage('Error deleting position and associated analysis');
        }
    }

    async function updatePosition() {
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            return;
        }
        console.log('updatePosition');
        if ($statusBarModeStore !== 'EDIT') {
            updateStatusBarMessage('Update is only possible in edit mode');
            return;
        }
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            return;
        }

        if (positions.length === 0) {
            updateStatusBarMessage('No positions to update');
            return;
        }

        const position = $positionStore;
        const analysis = $analysisStore;

        if (!isValidPosition(position)) {
            return;
        }

        try {
            const originalPosition = positions[currentPositionIndex];

            console.log('Position to update:', position);
            console.log('Analysis to update:', analysis);

            // Reset all fields of analysis to initialized values
            analysis.xgid = "";
            analysis.analysisType = "";
            analysis.checkerAnalysis = [];
            analysis.doublingCubeAnalysis = {};
            analysis.analysisEngineVersion = "";

            // Ensure checkerAnalysis is correctly structured
            if (Array.isArray(analysis.checkerAnalysis)) {
                analysis.checkerAnalysis = { moves: analysis.checkerAnalysis };
            }

            // Set dice to [0, 0] if decision type is doubling
            if (position.decision_type === 1) {
                position.dice = [0, 0];
            }

            const positionID = originalPosition.id;

            // Check if the edited position is different from the original position
            const positionJSON = JSON.stringify(position);
            const originalPositionJSON = JSON.stringify(originalPosition);

            if (positionJSON !== originalPositionJSON) {
                // Delete the existing analysis if the position has changed
                await DeleteAnalysis(positionID);
                console.log('Analysis deleted for position ID:', positionID);
            }

            // Update the xgid in the analysis
            analysis.xgid = generateXGID(position);

            // Update the position in the database
            await UpdatePosition(position);
            console.log('Position updated with ID:', positionID);

            // Update the analysis in the database
            await SaveAnalysis(positionID, analysis);
            console.log('Analysis updated for position ID:', positionID);

            // Retrieve all positions and show the last one
            positions = await LoadAllPositions();
            analyses = await LoadAllAnalyses();

            if (positions.length > 0) {
                currentPositionIndex = positions.length - 1;
                updateStatusBar(currentPositionIndex, positions.length);
                showPosition(positions[currentPositionIndex], analyses[currentPositionIndex]);
            }

            updateStatusBarMessage('Position and analysis updated successfully');
            statusBarModeStore.set('NORMAL');
        } catch (error) {
            console.error('Error updating position and analysis:', error);
            updateStatusBarMessage('Error updating position and analysis');
        }
    }

    function firstPosition() {
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            return;
        }
        if (positions && positions.length > 0) {
            currentPositionIndex = 0;
            showPosition(positions[currentPositionIndex], analyses[currentPositionIndex]);
            updateStatusBar(currentPositionIndex, positions.length);
        }
    }

    function previousPosition() {
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            return;
        }
        if (positions && currentPositionIndex > 0) {
            currentPositionIndex--;
            showPosition(positions[currentPositionIndex], analyses[currentPositionIndex]);
            updateStatusBar(currentPositionIndex, positions.length);
        }
    }

    function nextPosition() {
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            return;
        }
        if (positions && currentPositionIndex < positions.length - 1) {
            currentPositionIndex++;
            showPosition(positions[currentPositionIndex], analyses[currentPositionIndex]);
            updateStatusBar(currentPositionIndex, positions.length);
        }
    }

    function lastPosition() {
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            return;
        }
        if (positions && positions.length > 0) {
            currentPositionIndex = positions.length - 1;
            showPosition(positions[currentPositionIndex], analyses[currentPositionIndex]);
            updateStatusBar(currentPositionIndex, positions.length);
        }
    }

    function gotoPosition() {
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            return;
        }
        showGoToPositionModal = true;
    }

    function handleGoToPosition(positionNumber) {
        if (positions && positionNumber > 0 && positionNumber <= positions.length) {
            currentPositionIndex = positionNumber - 1;
            showPosition(positions[currentPositionIndex], analyses[currentPositionIndex]);
            updateStatusBar(currentPositionIndex, positions.length);
        } else {
            updateStatusBarMessage(`Invalid position number: ${positionNumber}`);
        }
        showGoToPositionModal = false;
    }

    function findPosition() {
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            return;
        }
        console.log('findPosition');
    }

    function toggleEditMode(){
        console.log('toggleEditMode');
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            statusBarModeStore.set('NORMAL');
            return;
        }
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

    function toggleCommandMode() {
        console.log('toggleCommandMode');
        return new Promise((resolve) => {
            if (!showCommand) {
                statusBarModeStore.set('COMMAND');
            } else {
                statusBarModeStore.set('NORMAL');
            }
            showCommand = !showCommand;
            resolve();
        });
    }

    function toggleAnalysisPanel() {
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            return;
        }
        if ($statusBarModeStore === 'EDIT') {
            updateStatusBarMessage('Cannot toggle analysis panel in edit mode');
            return;
        }
        if (JSON.stringify($positionStore) !== JSON.stringify(positions[currentPositionIndex])) {
            updateStatusBarMessage('Cannot toggle analysis panel with unsaved changes');
            return;
        }
        console.log('toggleAnalysisPanel'); // Debugging log

        statusBarModeStore.set('NORMAL'); // Ensure normal mode

        if ($statusBarModeStore === 'NORMAL') {
            showAnalysis = !showAnalysis;
            console.log('showAnalysis:', showAnalysis); // Debugging log
        }

        if (showAnalysis) {
            showComment = false;
            setTimeout(() => {
                const analysisPanel = document.querySelector('.analysis-panel');
                if (analysisPanel) {
                    analysisPanel.scrollIntoView({
                        behavior: 'smooth',
                        block: 'start'
                    });
                }
            }, 0);
        } else {
            setTimeout(() => {
                mainArea.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }, 0);
        }
    }

    function toggleCommentPanel() {
        if (!$databasePathStore) {
            updateStatusBarMessage('No database opened');
            return;
        }
        if (!positions[currentPositionIndex]) {
            updateStatusBarMessage('No current position to comment on');
            return;
        }
        console.log('toggleCommentPanel called');

        if ($statusBarModeStore === 'NORMAL' || $statusBarModeStore === 'COMMAND') {
            if (showComment) {
                SaveComment(parseInt(positions[currentPositionIndex].id), $commentTextStore); // Ensure position ID is an int64
            }
            showComment = !showComment;
            console.log('showComment state:', showComment); // Debugging log
        }

        if (showComment) {
            showAnalysis = false;
            showCommand = false;
            setTimeout(() => {
                const commentPanel = document.querySelector('.comment-panel');
                if (commentPanel) {
                    commentPanel.scrollIntoView({
                        behavior: 'smooth',
                        block: 'start'
                    });
                }
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
        mainArea.addEventListener("wheel", handleWheel); // Add wheel event listener to main container
    });

    onDestroy(() => {
        window.removeEventListener("keydown", handleKeyDown);
        mainArea.removeEventListener("wheel", handleWheel); // Remove wheel event listener from main container
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

    // Function to update the status bar
    function updateStatusBar(currentIndex, total) {
        currentPositionIndex = currentIndex;
        totalPositions = total;
    }

    // Function to show a specific position and analysis
    async function showPosition(position, analysis) {
        if (!position || !analysis) {
            console.error('Invalid position or analysis:', position, analysis);
            return;
        }

        // Create a deep copy of the position data
        const positionCopy = JSON.parse(JSON.stringify(position));
        const analysisCopy = JSON.parse(JSON.stringify(analysis));
        
        positionStore.set(positionCopy);
        analysisStore.set(analysisCopy);

        console.log('Analysis Data:', analysisCopy); // Debugging log

        // Load the comment for the current position
        const comment = await LoadComment(position.id);
        commentTextStore.set(comment || '');
    }

    // Function to handle mouse wheel events
    function handleWheel(event) {
        if (showGoToPositionModal || $statusBarModeStore === 'EDIT') {
            return; // Prevent changing position when GoToPositionModal is open or in edit mode
        }

        // Prevent changing position when scrolling in the analysis panel
        const analysisPanel = document.querySelector('.analysis-panel');
        if (analysisPanel && analysisPanel.contains(event.target)) {
            return;
        }

        if (positions && positions.length > 0) {
            if (event.deltaY < 0) {
                previousPosition();
            } else if (event.deltaY > 0) {
                nextPosition();
            }
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
        onSavePosition={saveCurrentPosition}
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

        <Board
            mode={$statusBarModeStore}
            class="full-size-board"
        />

        <CommandLine
            visible={showCommand}
            onClose={toggleCommandMode}
            onToggleHelp={toggleHelpModal}
            text={$commandTextStore}
            bind:this={commandInput}
            onNewDatabase={newDatabase}
            onOpenDatabase={openDatabase}
            importPosition={importPosition}
            onSavePosition={saveCurrentPosition}
            onUpdatePosition={() => {
                statusBarModeStore.set('EDIT'); // Set mode to EDIT before updating
                updatePosition();
            }}
            onDeletePosition={deletePosition}
            onGoToPosition={handleGoToPosition}
            onToggleAnalysis={toggleAnalysisPanel}
            onToggleComment={toggleCommentPanel}
            exitApp={exitApp}
            currentPositionId={positions.length > 0 ? positions[currentPositionIndex].id : null}
        />

    </div> <!-- Close the scrollable-content div properly -->

    <div class="panel-container">

        <CommentPanel
            text={$commentTextStore}
            visible={showComment}
            onClose={toggleCommentPanel}
            currentPositionId={positions.length > 0 ? positions[currentPositionIndex].id : null}
        />

        <AnalysisPanel
            visible={showAnalysis}
            analysisData={$analysisStore}
            onClose={toggleAnalysisPanel}
        /> 

    </div>

    <GoToPositionModal
        visible={showGoToPositionModal}
        onClose={() => showGoToPositionModal = false}
        onGoToPosition={handleGoToPosition}
        maxPositionNumber={positions.length}
    />

    <HelpModal
        visible={showHelp}
        onClose={toggleHelpModal}
        handleGlobalKeydown={handleKeyDown}
    />

    <StatusBar
        mode={$statusBarModeStore}
        text={$statusBarTextStore}
        currentPosition={positions.length > 0 ? currentPositionIndex : 0}
        totalPositions={positions.length}
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
        padding: 0; /* Remove padding */
        width: 100%;
        box-sizing: border-box;
        display: flex;
        justify-content: center; /* Center the board initially */
        align-items: flex-start; /* Align items to the start to remove space */
        margin-top: 0; /* Remove any margin on top */
    }

    .full-size-board {
        width: 100%;
        height: auto; /* Maintain aspect ratio */
        max-height: 100%; /* Ensure the board fits within the available height */
        margin: 0; /* Remove margin */
        padding: 0; /* Remove padding */
        border: 1px solid black; /* Add border for debugging */
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