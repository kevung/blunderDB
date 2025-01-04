<script>

    // svelte functions
    import { onMount, onDestroy } from 'svelte';
    import { fade } from 'svelte/transition';

    // import backend functions
    import {
        SaveDatabaseDialog,
        OpenDatabaseDialog,
        OpenPositionDialog,
        DeleteFile
    } from '../wailsjs/go/main/App.js';
    import {
        SetupDatabase,
        SavePosition,
        SaveAnalysis,
        PositionExists,
        LoadAllPositions,
        DeletePosition,
        DeleteAnalysis,
        UpdatePosition,
        LoadComment,
        SaveComment,
        LoadAnalysis,
        LoadPositionsByFilters, // Update import
        CheckDatabaseVersion, // Import CheckDatabaseVersion
        OpenDatabase, // Import OpenDatabase
        GetDatabaseVersion // Import GetDatabaseVersion
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
        positionsStore // Import positionsStore
    } from './stores/positionStore';

    import {
        analysisStore,
    } from './stores/analysisStore';

    import {
        currentPositionIndexStore, // Import currentPositionIndexStore
        statusBarTextStore,
        statusBarModeStore,
        commandTextStore,
        commentTextStore,
        analysisDataStore,
        showSearchModalStore, // Import showSearchModalStore
        showMetModalStore, // Import showMetModalStore
        showTakePoint2LastModalStore, // Import showTakePoint2LastModalStore
        showTakePoint2LiveModalStore, // Import showTakePoint2LiveModalStore
        showTakePoint4LastModalStore, // Import showTakePoint4LastModalStore
        showTakePoint4LiveModalStore, // Import showTakePoint4LiveModalStore
        showGammonValue1ModalStore, // Import showGammonValue1ModalStore
        showGammonValue2ModalStore, // Import showGammonValue2ModalStore
        showGammonValue4ModalStore, // Import showGammonValue4ModalStore
        showCommandStore,
        showAnalysisStore,
        showHelpStore,
        showCommentStore,
        showGoToPositionModalStore,
        showWarningModalStore, // Import showWarningModalStore
        showMetadataModalStore, // Import showMetadataModalStore
        showTakePoint2ModalStore, // Import showTakePoint2ModalStore
        showTakePoint4ModalStore // Import showTakePoint4ModalStore
    } from './stores/uiStore';

    import { metaStore } from './stores/metaStore'; // Import metaStore

    // import components
    import Toolbar from './components/Toolbar.svelte';
    import Board from './components/Board.svelte';
    import CommandLine from './components/CommandLine.svelte';
    import StatusBar from './components/StatusBar.svelte';
    import AnalysisPanel from './components/AnalysisPanel.svelte';
    import CommentPanel from './components/CommentPanel.svelte';
    import HelpModal from './components/HelpModal.svelte';
    import GoToPositionModal from './components/GoToPositionModal.svelte';
    import SearchModal from './components/SearchModal.svelte'; // Import SearchModal component
    import MetModal from './components/MetModal.svelte'; // Import MetModal component
    import TakePoint2LastModal from './components/TakePoint2LastModal.svelte'; // Import TakePoint2LastModal component
    import TakePoint2LiveModal from './components/TakePoint2LiveModal.svelte'; // Import TakePoint2LiveModal component
    import TakePoint4LastModal from './components/TakePoint4LastModal.svelte'; // Import TakePoint4LastModal component
    import TakePoint4LiveModal from './components/TakePoint4LiveModal.svelte'; // Import TakePoint4LiveModal component
    import GammonValue1Modal from './components/GammonValue1Modal.svelte'; // Import GammonValue1Modal component
    import GammonValue2Modal from './components/GammonValue2Modal.svelte'; // Import GammonValue2Modal component
    import GammonValue4Modal from './components/GammonValue4Modal.svelte'; // Import GammonValue4Modal component
    import WarningModal from './components/WarningModal.svelte'; // Import WarningModal component
    import MetadataModal from './components/MetadataModal.svelte'; // Import MetadataModal component
    import TakePoint2Modal from './components/TakePoint2Modal.svelte'; // Import TakePoint2Modal component
    import TakePoint4Modal from './components/TakePoint4Modal.svelte'; // Import TakePoint4Modal component

    // Visibility variables
    let showSearchModal = false;
    let showMetModal = false;
    let showTakePoint2LastModal = false;
    let showTakePoint2LiveModal = false;
    let showTakePoint4LastModal = false;
    let showTakePoint4LiveModal = false;
    let showGammonValue1Modal = false;
    let showGammonValue2Modal = false;
    let showGammonValue4Modal = false;
    let showCommand = false;
    let showAnalysis = false;
    let showHelp = false;
    let showComment = false;
    let showGoToPositionModal = false;
    let showWarningModal = false;
    let warningMessage = '';
    let databaseVersion = '';
    let applicationVersion = '';
    let showMetadataModal = false;
    let databaseLoaded = false;
    let mode = 'NORMAL';
    let showTakePoint2Modal = false;
    let showTakePoint4Modal = false;

    // Subscribe to the metaStore
    metaStore.subscribe(value => {
        databaseVersion = value.databaseVersion;
        applicationVersion = value.applicationVersion;
    });

    // Reference for various elements.
    let mainArea;
    let commandInput;

    let currentPositionIndex = 0;

    // Subscribe to the stores
    let positions = [];
    positionsStore.subscribe(value => {
        positions = Array.isArray(value) ? value : [];
        if (positions.length === 0) {
            positionStore.set({
                board: {
                    points: Array(26).fill({ checkers: 0, color: -1 }),
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
            analysisStore.set({}); // Reset analysisStore when no positions
        }
    });

    currentPositionIndexStore.subscribe(async value => {
        currentPositionIndex = value;
        if (positions.length > 0 && currentPositionIndex >= 0 && currentPositionIndex < positions.length) {
            await showPosition(positions[currentPositionIndex]);
            setStatusBarMessage(''); // Reset status bar message when changing position
        }
    });

    showSearchModalStore.subscribe(value => {
        showSearchModal = value;
    });

    showMetModalStore.subscribe(value => {
        showMetModal = value;
    });

    showTakePoint2LastModalStore.subscribe(value => {
        showTakePoint2LastModal = value;
    });

    showTakePoint2LiveModalStore.subscribe(value => {
        showTakePoint2LiveModal = value;
    });

    showTakePoint4LastModalStore.subscribe(value => {
        showTakePoint4LastModal = value;
    });

    showTakePoint4LiveModalStore.subscribe(value => {
        showTakePoint4LiveModal = value;
    });

    showGammonValue1ModalStore.subscribe(value => {
        showGammonValue1Modal = value;
    });

    showGammonValue2ModalStore.subscribe(value => {
        showGammonValue2Modal = value;
    });

    showGammonValue4ModalStore.subscribe(value => {
        showGammonValue4Modal = value;
    });

    showCommandStore.subscribe(value => {
        showCommand = value;
    });

    showAnalysisStore.subscribe(value => {
        showAnalysis = value;
    });

    showHelpStore.subscribe(value => {
        showHelp = value;
    });

    showCommentStore.subscribe(value => {
        showComment = value;
    });

    showGoToPositionModalStore.subscribe(value => {
        showGoToPositionModal = value;
    });

    showWarningModalStore.subscribe(value => {
        showWarningModal = value;
    });

    showMetadataModalStore.subscribe(value => {
        showMetadataModal = value;
    });

    showTakePoint2ModalStore.subscribe(value => {
        showTakePoint2Modal = value;
    });

    showTakePoint4ModalStore.subscribe(value => {
        showTakePoint4Modal = value;
    });

    databasePathStore.subscribe(value => {
        databaseLoaded = !!value;
    });

    statusBarModeStore.subscribe(value => {
        mode = value;
    });

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

        // Prevent shortcuts when any modal is visible
        if (showSearchModal || showMetModal || showTakePoint2LastModal || showTakePoint2LiveModal || showTakePoint4LastModal || showTakePoint4LiveModal || showGammonValue1Modal || showGammonValue2Modal || showGammonValue4Modal || showMetadataModal || showTakePoint2Modal || showTakePoint4Modal) {
            if (event.key === 'Escape') {
                showSearchModalStore.set(false);
                showMetModalStore.set(false);
                showTakePoint2LastModalStore.set(false);
                showTakePoint2LiveModalStore.set(false);
                showTakePoint4LastModalStore.set(false);
                showTakePoint4LiveModalStore.set(false);
                showGammonValue1ModalStore.set(false);
                showGammonValue2ModalStore.set(false);
                showGammonValue4ModalStore.set(false);
                showMetadataModalStore.set(false); // Close Metadata modal
                showTakePoint2ModalStore.set(false); // Close TakePoint2 modal
                showTakePoint4ModalStore.set(false); // Close TakePoint4 modal
            }
            // Prevent browsing position shortcuts unless focused on input or textarea
            if (['PageUp', 'PageDown', 'ArrowLeft', 'ArrowRight', 'h', 'k', 'j', 'l'].includes(event.key) && document.activeElement.tagName !== 'INPUT' && document.activeElement.tagName !== 'TEXTAREA') {
                event.preventDefault();
            }
            // Allow backspace to work inside input fields
            if (event.key === 'Backspace' && document.activeElement.tagName !== 'INPUT' && document.activeElement.tagName !== 'TEXTAREA') {
                event.preventDefault(); // Prevent backspace from erasing position
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
        } else if(event.ctrlKey && event.code == 'KeyR') {
            loadAllPositions();
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
            if (showComment) {
                toggleCommentPanel(); // Close comment panel if open
            }
            toggleAnalysisPanel();
        } else if(event.ctrlKey && event.code == 'KeyP') {
            if(!showHelp && !showCommand) {
                event.preventDefault();
                toggleCommentPanel();
            }
        } else if (event.ctrlKey && event.code === 'KeyF') {
            if ($statusBarModeStore === 'EDIT') {
                findPosition();
            } else {
                setStatusBarMessage('Search is only available in edit mode');
            }
        } else if (event.ctrlKey && event.code === 'KeyH') {
            toggleHelpModal();
        } else if (!event.ctrlKey && event.key === '?') {
            toggleHelpModal();
        } else if (event.ctrlKey && event.key === 'ArrowLeft') {
            if (!showComment) {
                setBoardOrientation("left");
            }
        } else if (event.ctrlKey && event.key === 'ArrowRight') {
            if (!showComment) {
                setBoardOrientation("right");
            }
        } else if (event.ctrlKey && event.code === 'KeyM') {
            toggleMetadataModal();
        }
    }

    function getFilenameFromPath(filePath) {
        return filePath.split('/').pop();
    }

    export function setStatusBarMessage(message) {
        statusBarTextStore.set(message);
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
                setStatusBarMessage('New database created successfully');
                const filename = getFilenameFromPath(filePath);
                WindowSetTitle(`blunderDB - ${filename}`);
                console.log(`New database created at ${filePath}`);
                // Reset the display
                await loadAllPositions();
            } else {
                console.log('No file selected');
            }
        } catch (error) {
            console.error('Error opening file dialog:', error);
            setStatusBarMessage('Error creating new database');
        }
    }

    async function openDatabase() {
        console.log('openDatabase');
        try {
            const filePath = await OpenDatabaseDialog();
            if (!filePath) {
                console.log('No Database selected');
                return;
            }

            databasePathStore.set(filePath);
            console.log('databasePathStore:', $databasePathStore);

            // Open the database and check for required tables and metadata keys
            await OpenDatabase(filePath);

            // Check database version after opening the database
            const dbVersion = await CheckDatabaseVersion();
            const modelVersion = await GetDatabaseVersion();
            console.log(`Database version: ${dbVersion}`);
            console.log(`Model version: ${modelVersion}`);
            setStatusBarMessage(`Database version: ${dbVersion}`);

            if (getMajorVersion(dbVersion) !== getMajorVersion(modelVersion)) {
                warningMessage = `Major database version mismatch. The database schema might be incompatible with the current version of blunderDB. Continuing to edit the database is done at your own risk. Backup your file before proceeding any further.\nDatabase version: ${dbVersion}\nExpected version: ${modelVersion}`;
                showWarningModalStore.set(true); // Use store to show warning modal
            }

            setStatusBarMessage('Database opened successfully');
            const filename = getFilenameFromPath(filePath);
            WindowSetTitle(`blunderDB - ${filename}`);

            // Load positions
            await loadAllPositions();
        } catch (error) {
            console.error('Error opening file dialog:', error);
            setStatusBarMessage('Error opening database');
        }
    }

    function getMajorVersion(version) {
        return version.split('.')[0];
    }

    function closeWarningModal() {
        showWarningModalStore.set(false); // Use store to close warning modal
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
                parsedAnalysis.positionId = positionExistsResult.id; // Ensure the position ID is set in the analysis
                await SaveAnalysis(positionExistsResult.id, parsedAnalysis);
                console.log('Analysis updated for position ID:', positionExistsResult.id);
                setStatusBarMessage('Position already exists, analysis updated');
                currentPositionIndexStore.set(-1); //force change to trigger re-render
                currentPositionIndexStore.set(positions.findIndex(pos => pos.id === positionExistsResult.id)); // Set current position index to display the existing position
            } catch (error) {
                console.error('Error updating analysis:', error);
                setStatusBarMessage('Error updating analysis');
            }
            return;
        }

        // Save the position and analysis to the database
        try {
            const positionID = await SavePosition(positionData); // Remove databaseVersion
            console.log('Position saved with ID:', positionID);

            positionData.id = positionID; // Ensure the position ID is set in the position data
            parsedAnalysis.positionId = positionID; // Ensure the position ID is set in the analysis
            await SaveAnalysis(positionID, parsedAnalysis);
            console.log('Analysis saved for position ID:', positionID);

            // Reload all positions and show the last one
            await loadAllPositions();
            setStatusBarMessage(successMessage);
        } catch (error) {
            console.error('Error saving position and analysis:', error);
            setStatusBarMessage('Error saving position and analysis');
        }
    }

    export async function importPosition() {
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
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
            setStatusBarMessage('No database opened');
            return;
        }
        if ($statusBarModeStore === 'EDIT') {
            setStatusBarMessage('Cannot paste position in edit mode');
            return;
        }
        console.log('pastePosition');
        let promise = window.runtime.ClipboardGetText();
        promise.then(
            async (result) => {
                pastePositionTextStore.set(result);
                console.log('pastePositionTextStore:', $pastePositionTextStore);
                const { positionData, parsedAnalysis } = parsePosition(result);
                await savePositionAndAnalysis(positionData, parsedAnalysis, 'Pasted position and analysis saved successfully');
                statusBarModeStore.set('NORMAL');
            })
            .catch((error) => {
                console.error('Error pasting from clipboard:', error);
            });
    }

    async function saveCurrentPosition() {
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        console.log('saveCurrentPosition');

        const position = $positionStore;
        const analysis = $analysisStore;

        if (!isValidPosition(position)) {
            return;
        }

        console.log('Position to save:', position);
        console.log('Analysis to save:', analysis);

        // Reset all fields of analysis to initialized values
        analysis.xgid = generateXGID(position);
        analysis.analysisType = "";
        analysis.checkerAnalysis = [];
        analysis.doublingCubeAnalysis = {};
        analysis.analysisEngineVersion = "";

        await savePositionAndAnalysis(position, analysis, 'Position and analysis saved successfully');
        statusBarModeStore.set('NORMAL');
    }

    function parsePosition(fileContent) {
        if (!fileContent || fileContent.trim().length === 0) {
            throw new Error("File is empty or invalid.");
        }

        const normalizedContent = fileContent.replace(/\r\n|\r/g, '\n').trim();
        const lines = normalizedContent.split('\n').map(line => line.trim());

        const isFrench = normalizedContent.includes("Joueur") || normalizedContent.includes("Adversaire") || normalizedContent.includes("Videau");
        const isInternalCheckerAnalysisFormat = normalizedContent.includes("Analysis:\nChecker Move Analysis:");
        const isInternalDoublingAnalysisFormat = normalizedContent.includes("Analysis:\nDoubling Cube Analysis:");

        const xgidLine = lines.find(line => line.startsWith("XGID="));
        const xgid = xgidLine ? xgidLine.split('=')[1] : null;

        if (!xgid) {
            throw new Error("XGID not found in the file content.");
        }

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

        const board = { points: Array(26).fill({ checkers: 0, color: -1 }) };

        if (positionPart) {
            const pointChars = positionPart.split('');
            let pointIndex = 0;
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
        }

        const diceValues = dicePart ? dicePart.split("").map(num => parseInt(num)) : [0, 0];
        const dice = [diceValues[0], diceValues[1]];

        const player1Score = parseInt(score1);
        const player2Score = parseInt(score2);
        const matchLengthValue = parseInt(matchLength);
        const playerOnRoll = parseInt(playerDownOnDiagram) === 1 ? 0 : 1;

        let hasJacoby = 0, hasBeaver = 0, awayScores = [matchLengthValue - player1Score, matchLengthValue - player2Score];
        if (parseInt(isCrawford) === 0) {
            awayScores = awayScores.map(score => score === 1 ? 0 : score);
        }
        if (matchLengthValue === 0) {
            awayScores = [-1, -1];
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
        const decisionType = decisionLine || isInternalCheckerAnalysisFormat ? 0 : 1;

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

        const parsedAnalysis = { xgid, analysisType: "", checkerAnalysis: [], doublingCubeAnalysis: {}, analysisEngineVersion: "" };

        const engineVersionMatch = normalizedContent.match(/eXtreme Gammon Version: ([\d.]+)(?:, MET: (.+))?/);
        if (engineVersionMatch) {
            parsedAnalysis.analysisEngineVersion = `eXtreme Gammon Version: ${engineVersionMatch[1]}`;
            if (engineVersionMatch[2]) {
                parsedAnalysis.analysisEngineVersion += `, MET: ${engineVersionMatch[2]}`;
            }
        }

        if (isInternalDoublingAnalysisFormat) {
            parsedAnalysis.analysisType = "DoublingCube";
            const doublingCubeAnalysisRegex = /Doubling Cube Analysis:\nAnalysis Depth: (.+)\nPlayer Win Chances: ([-.\d]+)%\nPlayer Gammon Chances: ([-.\d]+)%\nPlayer Backgammon Chances: ([-.\d]+)%\nOpponent Win Chances: ([-.\d]+)%\nOpponent Gammon Chances: ([-.\d]+)%\nOpponent Backgammon Chances: ([-.\d]+)%\nCubeless No Double Equity: ([-.\d]+)\nCubeless Double Equity: ([-.\d]+)\nCubeful No Double Equity: ([-.\d]+)\nCubeful No Double Error: ([-.\d]+)\nCubeful Double Take Equity: ([-.\d]+)\nCubeful Double Take Error: ([-.\d]+)\nCubeful Double Pass Equity: ([-.\d]+)\nCubeful Double Pass Error: ([-.\d]+)\nBest Cube Action: (.+)\nWrong Pass Percentage: ([-.\d]+)%\nWrong Take Percentage: ([-.\d]+)%/;
            const doublingCubeMatch = doublingCubeAnalysisRegex.exec(normalizedContent);
            if (doublingCubeMatch) {
                parsedAnalysis.doublingCubeAnalysis = {
                    analysisDepth: doublingCubeMatch[1].trim(),
                    playerWinChances: parseFloat(doublingCubeMatch[2]),
                    playerGammonChances: parseFloat(doublingCubeMatch[3]),
                    playerBackgammonChances: parseFloat(doublingCubeMatch[4]),
                    opponentWinChances: parseFloat(doublingCubeMatch[5]),
                    opponentGammonChances: parseFloat(doublingCubeMatch[6]),
                    opponentBackgammonChances: parseFloat(doublingCubeMatch[7]),
                    cubelessNoDoubleEquity: parseFloat(doublingCubeMatch[8]),
                    cubelessDoubleEquity: parseFloat(doublingCubeMatch[9]),
                    cubefulNoDoubleEquity: parseFloat(doublingCubeMatch[10]),
                    cubefulNoDoubleError: parseFloat(doublingCubeMatch[11]),
                    cubefulDoubleTakeEquity: parseFloat(doublingCubeMatch[12]),
                    cubefulDoubleTakeError: parseFloat(doublingCubeMatch[13]),
                    cubefulDoublePassEquity: parseFloat(doublingCubeMatch[14]),
                    cubefulDoublePassError: parseFloat(doublingCubeMatch[15]),
                    bestCubeAction: doublingCubeMatch[16].trim(),
                    wrongPassPercentage: parseFloat(doublingCubeMatch[17]),
                    wrongTakePercentage: parseFloat(doublingCubeMatch[18])
                };
            }
        } else if (isInternalCheckerAnalysisFormat) {
            parsedAnalysis.analysisType = "CheckerMove";
            const moveRegex = /^Move (\d+): (.+)\nAnalysis Depth: (.+)\nEquity: ([-.\d]+)\nEquity Error: ([-.\d]+)\nPlayer Win Chance: ([-.\d]+)%\nPlayer Gammon Chance: ([-.\d]+)%\nPlayer Backgammon Chance: ([-.\d]+)%\nOpponent Win Chance: ([-.\d]+)%\nOpponent Gammon Chance: ([-.\d]+)%\nOpponent Backgammon Chance: ([-.\d]+)%/gm;
            let moveMatch;
            while ((moveMatch = moveRegex.exec(normalizedContent)) !== null) {
                const moveDetails = {
                    index: parseInt(moveMatch[1], 10),
                    move: moveMatch[2].trim(),
                    analysisDepth: moveMatch[3].trim(),
                    equity: parseFloat(moveMatch[4]),
                    equityError: parseFloat(moveMatch[5]),
                    playerWinChance: parseFloat(moveMatch[6]),
                    playerGammonChance: parseFloat(moveMatch[7]),
                    playerBackgammonChance: parseFloat(moveMatch[8]),
                    opponentWinChance: parseFloat(moveMatch[9]),
                    opponentGammonChance: parseFloat(moveMatch[10]),
                    opponentBackgammonChance: parseFloat(moveMatch[11]),
                };
                parsedAnalysis.checkerAnalysis.push(moveDetails);
            }
        } else if (/^ {4}(\d+)\./gm.test(normalizedContent)) {
            parsedAnalysis.analysisType = "CheckerMove";
            const moveRegex = new RegExp(
                isFrench
                ? /^ {4}(\d+)\.\s(.{11})\s(.{28})\séq:(.{6})\s(?:\((-?[-.\d]+)\))?/
                : /^ {4}(\d+)\.\s(.{11})\s(.{28})\seq:(.{6})\s(?:\((-?[-.\d.]+)\))?/,
                'gm'
            );
            let moveMatch;
            while ((moveMatch = moveRegex.exec(normalizedContent)) !== null) {
                const moveDetails = {
                    index: parseInt(moveMatch[1], 10),
                    analysisDepth: moveMatch[2].trim(),
                    move: moveMatch[3].trim(),
                    equity: parseFloat(moveMatch[4]),
                    equityError: moveMatch[5] ? parseFloat(moveMatch[5]) : 0,
                };
                const lineStart = moveMatch.index + moveMatch[0].length;
                const remainingContent = normalizedContent.slice(lineStart);
                const playerRegex = isFrench
                    ? /Joueur:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                    : /Player:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/;
                const opponentRegex = isFrench
                    ? /Adversaire:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                    : /Opponent:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/;
                const playerMatch = playerRegex.exec(remainingContent);
                const opponentMatch = opponentRegex.exec(remainingContent);
                if (playerMatch) {
                    moveDetails.playerWinChance = parseFloat(playerMatch[1]);
                    moveDetails.playerGammonChance = parseFloat(playerMatch[2]);
                    moveDetails.playerBackgammonChance = parseFloat(playerMatch[3]);
                }
                if (opponentMatch) {
                    moveDetails.opponentWinChance = parseFloat(opponentMatch[1]);
                    moveDetails.opponentGammonChance = parseFloat(opponentMatch[2]);
                    moveDetails.opponentBackgammonChance = parseFloat(opponentMatch[3]);
                }
                parsedAnalysis.checkerAnalysis.push(moveDetails);
            }
        } else if (
            (isFrench && (normalizedContent.includes("Equités sans videau") || normalizedContent.includes("Equités avec videau"))) ||
            (!isFrench && (normalizedContent.includes("Cubeless Equities") || normalizedContent.includes("Cubeful Equities")))
        ) {
            parsedAnalysis.analysisType = "DoublingCube";

            const analysisDepthMatch = normalizedContent.match(new RegExp(isFrench ? /Analysé avec\s+([^\n]*)/ : /Analyzed in\s+([^\n]*)/));

            const playerWinMatch = normalizedContent.match(new RegExp(isFrench ? /Chance de gain du joueur:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/ : /Player Winning Chances:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/));

            const opponentWinMatch = normalizedContent.match(new RegExp(isFrench ? /Chance de gain de l'adversaire:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/ : /Opponent Winning Chances:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/));

            const cubelessMatch = normalizedContent.match(new RegExp(isFrench ? /Equités sans videau\s*:\s*Pas de double=([\+\-\d.]+),\s*Double=([\+\-\d.]+)/ : /Cubeless Equities:\s*No Double=([\+\-\d.]+),\s*Double=([\+\-\d.]+)/));

            const cubefulNoDoubleMatch = normalizedContent.match(new RegExp(isFrench ? /Pas de double\s*:\s*([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /No double\s*:\s*([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));
            const cubefulDoubleTakeMatch = normalizedContent.match(new RegExp(isFrench ? /Double\/Prend:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /Double\/Take:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));
            const cubefulDoublePassMatch = normalizedContent.match(new RegExp(isFrench ? /Double\/Passe:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /Double\/Pass:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));
            const redoubleNoMatch = normalizedContent.match(new RegExp(isFrench ? /Pas de redouble\s*:\s*([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /No redouble\s*:\s*([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));
            const redoubleTakeMatch = normalizedContent.match(new RegExp(isFrench ? /Redouble\/Prend:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /Redouble\/Take:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));
            const redoublePassMatch = normalizedContent.match(new RegExp(isFrench ? /Redouble\/Passe:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /Redouble\/Pass:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));

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
            if (redoubleNoMatch) {
                parsedAnalysis.doublingCubeAnalysis.cubefulNoDoubleEquity = parseFloat(redoubleNoMatch[1]);
                parsedAnalysis.doublingCubeAnalysis.cubefulNoDoubleError = redoubleNoMatch[2] ? parseFloat(redoubleNoMatch[2]) : 0;
            }
            if (redoubleTakeMatch) {
                parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeEquity = parseFloat(redoubleTakeMatch[1]);
                parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeError = redoubleTakeMatch[2] ? parseFloat(redoubleTakeMatch[2]) : 0;
            }
            if (redoublePassMatch) {
                parsedAnalysis.doublingCubeAnalysis.cubefulDoublePassEquity = parseFloat(redoublePassMatch[1]);
                parsedAnalysis.doublingCubeAnalysis.cubefulDoublePassError = redoublePassMatch[2] ? parseFloat(redoublePassMatch[2]) : 0;
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
            setStatusBarMessage('No database opened');
            return;
        }
        if ($statusBarModeStore === 'EDIT') {
            setStatusBarMessage('Cannot copy position in edit mode');
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
            clipboardContent += `Analysis Depth: ${analysis.doublingCubeAnalysis.analysisDepth}\n`;
            clipboardContent += `Player Win Chances: ${analysis.doublingCubeAnalysis.playerWinChances}%\n`;
            clipboardContent += `Player Gammon Chances: ${analysis.doublingCubeAnalysis.playerGammonChances}%\n`;
            clipboardContent += `Player Backgammon Chances: ${analysis.doublingCubeAnalysis.playerBackgammonChances}%\n`;
            clipboardContent += `Opponent Win Chances: ${analysis.doublingCubeAnalysis.opponentWinChances}%\n`;
            clipboardContent += `Opponent Gammon Chances: ${analysis.doublingCubeAnalysis.opponentGammonChances}%\n`;
            clipboardContent += `Opponent Backgammon Chances: ${analysis.doublingCubeAnalysis.opponentBackgammonChances}%\n`;
            clipboardContent += `Cubeless No Double Equity: ${analysis.doublingCubeAnalysis.cubelessNoDoubleEquity}\n`;
            clipboardContent += `Cubeless Double Equity: ${analysis.doublingCubeAnalysis.cubelessDoubleEquity}\n`;
            clipboardContent += `Cubeful No Double Equity: ${analysis.doublingCubeAnalysis.cubefulNoDoubleEquity}\n`;
            clipboardContent += `Cubeful No Double Error: ${analysis.doublingCubeAnalysis.cubefulNoDoubleError}\n`;
            clipboardContent += `Cubeful Double Take Equity: ${analysis.doublingCubeAnalysis.cubefulDoubleTakeEquity}\n`;
            clipboardContent += `Cubeful Double Take Error: ${analysis.doublingCubeAnalysis.cubefulDoubleTakeError}\n`;
            clipboardContent += `Cubeful Double Pass Equity: ${analysis.doublingCubeAnalysis.cubefulDoublePassEquity}\n`;
            clipboardContent += `Cubeful Double Pass Error: ${analysis.doublingCubeAnalysis.cubefulDoublePassError}\n`;
            clipboardContent += `Best Cube Action: ${analysis.doublingCubeAnalysis.bestCubeAction}\n`;
            clipboardContent += `Wrong Pass Percentage: ${analysis.doublingCubeAnalysis.wrongPassPercentage}%\n`;
            clipboardContent += `Wrong Take Percentage: ${analysis.doublingCubeAnalysis.wrongTakePercentage}%\n`;
        } else if (analysis.analysisType === "CheckerMove") {
            clipboardContent += `Checker Move Analysis:\n`;
            analysis.checkerAnalysis.moves.forEach(move => {
                clipboardContent += `Move ${move.index}: ${move.move}\n`;
                clipboardContent += `Analysis Depth: ${move.analysisDepth}\n`;
                clipboardContent += `Equity: ${move.equity}\n`;
                if (move.equityError !== undefined) {
                    clipboardContent += `Equity Error: ${move.equityError}\n`;
                }
                clipboardContent += `Player Win Chance: ${move.playerWinChance}%\n`;
                clipboardContent += `Player Gammon Chance: ${move.playerGammonChance}%\n`;
                clipboardContent += `Player Backgammon Chance: ${move.playerBackgammonChance}%\n`;
                clipboardContent += `Opponent Win Chance: ${move.opponentWinChance}%\n`;
                clipboardContent += `Opponent Gammon Chance: ${move.opponentGammonChance}%\n`;
                clipboardContent += `Opponent Backgammon Chance: ${move.opponentBackgammonChance}%\n\n`;
            });
        }

        // Add engine version
        if (analysis.analysisEngineVersion) {
            clipboardContent += `Engine Version: ${analysis.analysisEngineVersion}\n`;
        }

        // Copy to clipboard
        navigator.clipboard.writeText(clipboardContent).then(() => {
            console.log('Position and analysis copied to clipboard');
            setStatusBarMessage('Position and analysis copied to clipboard');
        }).catch(err => {
            console.error('Error copying to clipboard:', err);
            setStatusBarMessage('Error copying to clipboard');
        });
    }

    function isValidPosition(position) {
        const player1Checkers = position.board.points.reduce((acc, point) => acc + (point.color === 0 ? point.checkers : 0), 0);
        const player2Checkers = position.board.points.reduce((acc, point) => acc + (point.color === 1 ? point.checkers : 0), 0);

        if (player1Checkers > 15) {
            setStatusBarMessage('Invalid position: Player 1 has more than 15 checkers');
            return false;
        }

        if (player2Checkers > 15) {
            setStatusBarMessage('Invalid position: Player 2 has more than 15 checkers');
            return false;
        }

        if (player1Checkers === 0) {
            setStatusBarMessage('Invalid position: Player 1 has already borne off all checkers');
            return false;
        }

        if (player2Checkers === 0) {
            setStatusBarMessage('Invalid position: Player 2 has already borne off all checkers');
            return false;
        }

        if (position.decision_type === 1) {
            if (position.cube.owner !== position.player_on_roll && position.cube.owner !== -1) {
                setStatusBarMessage('Invalid position: Cube is not available for doubling');
                return false;
            }
            if (position.score[position.player_on_roll] === 1) {
                setStatusBarMessage('Invalid position: Crawford rule prevents doubling');
                return false;
            }
        }

        if ((position.score[0] === -1 && position.score[1] !== -1) || (position.score[1] === -1 && position.score[0] !== -1)) {
            setStatusBarMessage('Invalid position: Both players must have unlimited score or neither');
            return false;
        }

        return true;
    }

    async function deletePosition() {
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        console.log('deletePosition');
        if ($statusBarModeStore !== 'NORMAL' && $statusBarModeStore !== 'COMMAND') {
            setStatusBarMessage('Cannot delete position in current mode');
            return;
        }

        if (!positions || positions.length === 0) {
            setStatusBarMessage('No positions to delete');
            return;
        }

        try {
            const positionID = positions[currentPositionIndex].id;
            await DeletePosition(positionID); // Remove databaseVersion
            console.log('Position and associated analysis deleted with ID:', positionID);

            // Load all positions from the database
            await loadAllPositions();
            setStatusBarMessage('Position and associated analysis deleted successfully');
        } catch (error) {
            console.error('Error deleting position and associated analysis:', error);
            setStatusBarMessage('Error deleting position and associated analysis');
        }
    }

    async function updatePosition() {
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        console.log('updatePosition');
        if ($statusBarModeStore !== 'EDIT') {
            setStatusBarMessage('Update is only possible in edit mode');
            return;
        }
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }

        if (positions.length === 0) {
            setStatusBarMessage('No positions to update');
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
            await UpdatePosition(position); // Remove databaseVersion
            console.log('Position updated with ID:', positionID);

            // Update the analysis in the database
            await SaveAnalysis(positionID, analysis);
            console.log('Analysis updated for position ID:', positionID);

            // Store the current index before loading all positions
            const currentIndex = currentPositionIndex;

            // Retrieve all positions and update the store
            await loadAllPositions();
            currentPositionIndexStore.set(currentIndex); // Ensure the current index remains the same
            setStatusBarMessage('Position and analysis updated successfully');
            statusBarModeStore.set('NORMAL');
        } catch (error) {
            console.error('Error updating position and analysis:', error);
            setStatusBarMessage('Error updating position and analysis');
        }
    }

    function firstPosition() {
        if ($statusBarModeStore === 'EDIT') {
            setStatusBarMessage('Cannot browse positions in edit mode');
            return;
        }
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        if (positions && positions.length > 0) {
            currentPositionIndexStore.set(0);
        }
    }

    function previousPosition() {
        if ($statusBarModeStore === 'EDIT') {
            setStatusBarMessage('Cannot browse positions in edit mode');
            return;
        }
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        if (positions && $currentPositionIndexStore > 0) {
            currentPositionIndexStore.set($currentPositionIndexStore - 1);
        }
    }

    function nextPosition() {
        if ($statusBarModeStore === 'EDIT') {
            setStatusBarMessage('Cannot browse positions in edit mode');
            return;
        }
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        if (positions && $currentPositionIndexStore < positions.length - 1) {
            currentPositionIndexStore.set($currentPositionIndexStore + 1);
        }
    }

    function lastPosition() {
        if ($statusBarModeStore === 'EDIT') {
            setStatusBarMessage('Cannot browse positions in edit mode');
            return;
        }
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        if (positions && positions.length > 0) {
            currentPositionIndexStore.set(positions.length - 1);
        }
    }

    function gotoPosition() {
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        if ($statusBarModeStore !== 'NORMAL') {
            setStatusBarMessage('Cannot go to position in current mode');
            return;
        }
        showGoToPositionModalStore.set(true);
    }

    function findPosition() {
        console.log('findPosition');
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        showSearchModalStore.set(true); // Show the search modal
    }

    function toggleEditMode(){
        console.log('toggleEditMode');
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
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
            // Refresh board and display position associated with currentPositionIndexStore
            const currentIndex = $currentPositionIndexStore;
            currentPositionIndexStore.set(-1); // Temporarily set to a different value to force redraw
            currentPositionIndexStore.set(currentIndex); // Set back to the original value
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
            showCommandStore.set(!showCommand);
            resolve();
        });
    }

    function toggleAnalysisPanel() {
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        if ($statusBarModeStore === 'EDIT') {
            setStatusBarMessage('Cannot toggle analysis panel in edit mode');
            return;
        }

        console.log('toggleAnalysisPanel'); // Debugging log

        statusBarModeStore.set('NORMAL'); // Ensure normal mode

        if ($statusBarModeStore === 'NORMAL') {
            showAnalysisStore.set(!showAnalysis);
            console.log('showAnalysis:', showAnalysis); // Debugging log
        }

        if (showAnalysis) {
            showCommentStore.set(false);
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
            setStatusBarMessage('No database opened');
            return;
        }
        if (!positions[currentPositionIndex]) {
            setStatusBarMessage('No current position to comment on');
            return;
        }
        console.log('toggleCommentPanel called');

        if ($statusBarModeStore === 'NORMAL' || $statusBarModeStore === 'COMMAND') {
            if (showComment) {
                SaveComment(parseInt(positions[currentPositionIndex].id), $commentTextStore); // Ensure position ID is an int64
            }
            showCommentStore.set(!showComment);
            console.log('showComment state:', showComment); // Debugging log
        }

        if (showComment) {
            showAnalysisStore.set(false);
            showCommandStore.set(false);
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

    async function loadPositionsByFilters(filters, includeCube, includeScore, pipCountFilter, winRateFilter, gammonRateFilter, backgammonRateFilter, player2WinRateFilter, player2GammonRateFilter, player2BackgammonRateFilter, player1CheckerOffFilter, player2CheckerOffFilter, player1BackCheckerFilter, player2BackCheckerFilter, player1CheckerInZoneFilter, player2CheckerInZoneFilter, searchTextArray, player1AbsolutePipCountFilter, equityFilter, decisionTypeFilter, diceRollFilter) {
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        console.log('loadPositionsByFilters', filters, includeCube, includeScore, pipCountFilter, winRateFilter, gammonRateFilter, backgammonRateFilter, player2WinRateFilter, player2GammonRateFilter, player2BackgammonRateFilter, player1CheckerOffFilter, player2CheckerOffFilter, player1BackCheckerFilter, player2BackCheckerFilter, player1CheckerInZoneFilter, player2CheckerInZoneFilter, searchTextArray, player1AbsolutePipCountFilter, equityFilter, decisionTypeFilter, diceRollFilter);
        try {
            const currentPosition = $positionStore;

            const loadedPositions = await LoadPositionsByFilters(currentPosition, includeCube, includeScore, pipCountFilter, winRateFilter, gammonRateFilter, backgammonRateFilter, player2WinRateFilter, player2GammonRateFilter, player2BackgammonRateFilter, player1CheckerOffFilter, player2CheckerOffFilter, player1BackCheckerFilter, player2BackCheckerFilter, player1CheckerInZoneFilter, player2CheckerInZoneFilter, searchTextArray, player1AbsolutePipCountFilter, equityFilter, decisionTypeFilter, diceRollFilter); // Remove databaseVersion
            positionsStore.set(Array.isArray(loadedPositions) ? loadedPositions : []);

            if (loadedPositions && loadedPositions.length > 0) {
                if ($currentPositionIndexStore === 0) {
                    currentPositionIndexStore.set(1); // Temporarily set to a different value to force redraw board
                }
                currentPositionIndexStore.set(0); // Ensure the first matching position is displayed
            } else {
                setStatusBarMessage('No matching positions found');
            }
        } catch (error) {
            console.error('Error loading positions by filters:', error);
            setStatusBarMessage('Error loading positions by filters');
        }
    }

    async function loadAllPositions() {
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        try {
            const positions = await LoadAllPositions(); // Remove databaseVersion
            positionsStore.set(Array.isArray(positions) ? positions : []);
            if (positions && positions.length > 0) {
                currentPositionIndexStore.set(-1); // Temporarily set to a different value to force redraw board
                currentPositionIndexStore.set(positions.length - 1);
            } else {
                currentPositionIndexStore.set(-1);
                setStatusBarMessage('No positions found');
                console.log('No positions found.');
            }
        } catch (error) {
            console.error('Error loading all positions:', error);
            setStatusBarMessage('Error loading all positions');
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
        showHelpStore.set(!showHelp);

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

    // Function to show a specific position and analysis
    async function showPosition(position) {
        if (!position) {
            console.error('Invalid position:', position);
            return;
        }

        // Create a deep copy of the position data
        const positionCopy = JSON.parse(JSON.stringify(position));
        
        positionStore.set(positionCopy);

        // Load the analysis for the current position
        const analysis = await LoadAnalysis(position.id);
        analysisStore.set(analysis || {});

        console.log('Analysis Data:', analysis); // Debugging log

        // Load the comment for the current position
        const comment = await LoadComment(position.id);
        commentTextStore.set(comment || '');
    }

    // Function to handle mouse wheel events
    function handleWheel(event) {
        if (showGoToPositionModal || showSearchModal || showMetModal || showTakePoint2LastModal || showTakePoint2LiveModal || showTakePoint4LastModal || showTakePoint4LiveModal || showGammonValue1Modal || showGammonValue2Modal || showGammonValue4Modal || showMetadataModal || showTakePoint2Modal || showTakePoint4Modal || $statusBarModeStore === 'EDIT') {
            return; // Prevent changing position when any modal is open or in edit mode
        }

        // Prevent changing position when scrolling in the analysis panel or comment panel
        const analysisPanel = document.querySelector('.analysis-panel');
        const commentPanel = document.querySelector('.comment-panel');
        if ((analysisPanel && analysisPanel.contains(event.target)) || (commentPanel && commentPanel.contains(event.target))) {
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

    function toggleMetadataModal() {
        if (databaseLoaded) {
            if (mode === 'EDIT') {
                setStatusBarMessage('Cannot show metadata modal in edit mode');
            } else {
                showMetadataModalStore.set(!showMetadataModal);
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
        onLoadAllPositions={loadAllPositions}
        onShowMetadata={toggleMetadataModal}
    />

    <div class="scrollable-content">

        <Board
            class="full-size-board"
        />

        <CommandLine
            visible={showCommand}
            onClose={toggleCommandMode}
            onToggleHelp={toggleHelpModal}
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
            onToggleAnalysis={toggleAnalysisPanel}
            onToggleComment={toggleCommentPanel}
            exitApp={exitApp}
            onLoadPositionsByFilters={loadPositionsByFilters}
            onLoadAllPositions={loadAllPositions}
        />

    </div>

    <div class="panel-container">

        <CommentPanel
            visible={showComment}
            onClose={toggleCommentPanel}
        />

        <AnalysisPanel
            visible={showAnalysis}
            onClose={toggleAnalysisPanel}
        /> 

    </div>

    <GoToPositionModal
        visible={showGoToPositionModal}
        onClose={() => showGoToPositionModalStore.set(false)}
    />

    <SearchModal
        visible={showSearchModal}
        onClose={() => showSearchModalStore.set(false)}
        onLoadPositionsByFilters={loadPositionsByFilters}
    />

    <MetModal
        visible={showMetModal}
        onClose={() => showMetModalStore.set(false)}
    />

    <TakePoint2LastModal
        visible={showTakePoint2LastModal}
        onClose={() => showTakePoint2LastModalStore.set(false)}
    />

    <TakePoint2LiveModal
        visible={showTakePoint2LiveModal}
        onClose={() => showTakePoint2LiveModalStore.set(false)}
    />

    <TakePoint4LastModal
        visible={showTakePoint4LastModal}
        onClose={() => showTakePoint4LastModalStore.set(false)}
    />

    <TakePoint4LiveModal
        visible={showTakePoint4LiveModal}
        onClose={() => showTakePoint4LiveModalStore.set(false)}
    />

    <GammonValue1Modal
        visible={showGammonValue1Modal}
        onClose={() => showGammonValue1ModalStore.set(false)}
    />

    <GammonValue2Modal
        visible={showGammonValue2Modal}
        onClose={() => showGammonValue2ModalStore.set(false)}
    />

    <GammonValue4Modal
        visible={showGammonValue4Modal}
        onClose={() => showGammonValue4ModalStore.set(false)}
    />

    <WarningModal
        message={warningMessage}
        visible={$showWarningModalStore}
        onClose={closeWarningModal}
    />

    <MetadataModal
        visible={showMetadataModal}
        onClose={() => showMetadataModalStore.set(false)}
    />

    <TakePoint2Modal
        visible={showTakePoint2Modal}
        onClose={() => showTakePoint2ModalStore.set(false)}
    />

    <TakePoint4Modal
        visible={showTakePoint4Modal}
        onClose={() => showTakePoint4ModalStore.set(false)}
    />

    <HelpModal
        visible={showHelp}
        onClose={toggleHelpModal}
        handleGlobalKeydown={handleKeyDown}
    />

    <StatusBar />

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
