<script>

    // svelte functions
    import { onMount, onDestroy } from 'svelte';
    import { fade } from 'svelte/transition';

    // import backend functions
    import {
        SaveDatabaseDialog,
        OpenDatabaseDialog,
        OpenImportDatabaseDialog,
        OpenExportDatabaseDialog,
        OpenPositionDialog,
        OpenPositionFilesDialog,
        OpenPositionFolderDialog,
        CollectImportableFiles,
        ReadFileContent,
        DeleteFile,
        ShowQuestionDialog,
        IsDirectory,

        ShowAlert

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
        GetDatabaseVersion, // Import GetDatabaseVersion
        AnalyzeImportDatabase, // Import AnalyzeImportDatabase
        CommitImportDatabase, // Import CommitImportDatabase
        CancelImport, // Import CancelImport
        ExportDatabase, // Import ExportDatabase
        SaveFilter, // Import SaveFilter
        SaveEditPosition, // Import SaveEditPosition
        ImportXGMatch, // Import ImportXGMatch
        ImportGnuBGMatch, // Import ImportGnuBGMatch
        ImportGnuBGMatchFromText, // Import ImportGnuBGMatchFromText
        ImportBGFMatch, // Import ImportBGFMatch
        ImportBGFPosition, // Import ImportBGFPosition
        ImportBGFPositionFromText, // Import ImportBGFPositionFromText
        GetAllMatches, // Import GetAllMatches
        SaveSessionState, // Import SaveSessionState
        LoadSessionState, // Import LoadSessionState
        ClearSessionState, // Import ClearSessionState
        GetAllCollections, // Import GetAllCollections
        GetCollectionPositions, // Import GetCollectionPositions
        GetAllTournaments, // Import GetAllTournaments
        ComputeEPCFromPosition, // Import ComputeEPCFromPosition
        SaveLastVisitedPosition, // Import SaveLastVisitedPosition
        GetLastVisitedMatch, // Import GetLastVisitedMatch
        GetMatchMovePositions // Import GetMatchMovePositions
    } from '../wailsjs/go/main/Database.js';

    import { WindowSetTitle, Quit, ClipboardGetText, WindowGetSize, OnFileDrop, OnFileDropOff } from '../wailsjs/runtime/runtime.js';

    import { SaveWindowDimensions, SaveLastDatabasePath, GetLastDatabasePath } from '../wailsjs/go/main/Config.js';

    // import stores
    import {
        databasePathStore,
        databaseLoadedStore // Import databaseLoadedStore
    } from './stores/databaseStore';

    import {
        pastePositionTextStore,
        positionStore,
        positionsStore, // Import positionsStore
        positionBeforeFilterLibraryStore, // Import position before filter library store
        positionIndexBeforeFilterLibraryStore, // Import position index before filter library store
        matchContextStore, // Import matchContextStore
        lastVisitedMatchStore // Import lastVisitedMatchStore
    } from './stores/positionStore';

    import {
        analysisStore,
        selectedMoveStore,
    } from './stores/analysisStore';

    import {
        currentPositionIndexStore, // Import currentPositionIndexStore
        statusBarTextStore,
        statusBarModeStore,
        commentTextStore,
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
        showSearchHistoryPanelStore, // Import showSearchHistoryPanelStore
        showWarningModalStore, // Import showWarningModalStore
        showMetadataModalStore, // Import showMetadataModalStore
        showTakePoint2ModalStore, // Import showTakePoint2ModalStore
        isAnyModalOrPanelOpenStore, // Import the derived store
        isAnyModalOpenStore, // Import the derived store
        previousModeStore, // Import previousModeStore
        showFilterLibraryPanelStore, // Import showFilterLibraryPanelStore
        showMatchPanelStore, // Import showMatchPanelStore
        matchPanelRefreshTriggerStore, // Import matchPanelRefreshTriggerStore
        positionReloadTriggerStore, // Import positionReloadTriggerStore
        showCollectionPanelStore, // Import showCollectionPanelStore
        showTournamentPanelStore, // Import showTournamentPanelStore
        showPipcountStore,
        showExportDatabaseModalStore // Import showExportDatabaseModalStore
    } from './stores/uiStore';

    import { metaStore } from './stores/metaStore'; // Import metaStore

    import { 
        collectionsStore,
        collectionPositionsStore,
        selectedCollectionStore,
        activeCollectionStore
    } from './stores/collectionStore'; // Import collection stores

    import { tournamentsStore } from './stores/tournamentStore'; // Import tournament store

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
    import SearchHistoryPanel from './components/SearchHistoryPanel.svelte'; // Import SearchHistoryPanel component
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
    import FilterLibraryPanel from './components/FilterLibraryPanel.svelte'; // Update import
    import ImportProgressModal from './components/ImportProgressModal.svelte'; // Import ImportProgressModal component
    import FileImportProgressModal from './components/FileImportProgressModal.svelte'; // Import FileImportProgressModal component
    import ExportDatabaseModal from './components/ExportDatabaseModal.svelte'; // Import ExportDatabaseModal component
    import MatchPanel from './components/MatchPanel.svelte'; // Import MatchPanel component
    import CollectionPanel from './components/CollectionPanel.svelte'; // Import CollectionPanel component
    import TournamentPanel from './components/TournamentPanel.svelte'; // Import TournamentPanel component

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
    let showSearchHistoryPanel = false; // Add search history panel visibility
    let showWarningModal = false;
    let showImportProgressModal = false;
    let importModalMode = 'analyzing'; // 'analyzing', 'preview', 'committing', 'completed'
    let importAnalysis = {
        toAdd: 0,
        toMerge: 0,
        toSkip: 0,
        total: 0,
        importPath: ''
    };
    let importResult = {
        added: 0,
        merged: 0,
        skipped: 0,
        total: 0
    };
    let pendingImportPath = null;

    // File import (multi-file/folder) state
    let showFileImportModal = false;
    let fileImportMode = 'idle'; // 'idle', 'importing', 'completed'
    let fileImportTotalFiles = 0;
    let fileImportCurrentIndex = 0;
    let fileImportCurrentFile = '';
    let fileImportResults = { succeeded: 0, failed: 0, skipped: 0, errors: [] };
    let fileImportCancelled = false;

    let exportModalMode = 'preparing'; // 'preparing', 'metadata', 'exporting', 'completed'
    let exportPositionCount = 0;
    let exportMetadata = {
        user: '',
        description: '',
        dateOfCreation: ''
    };
    let exportOptions = {
        includeAnalysis: true,
        includeComments: true,
        includeFilterLibrary: false,
        includePlayedMoves: true,
        includeMatches: true,
        matchIDs: [],
        includeTournaments: false,
        includeTournamentIDs: [],
        includeCollections: false,
        collectionIDs: []
    };
    let exportMatches = []; // matches loaded for export selection
    let pendingExportPath = null;

    // Drag and drop state
    let showDropOverlay = false;
    let dropOverlayMessage = 'Drop files here';

    let warningMessage = '';
    let databaseVersion = '';
    let applicationVersion = '';
    let showMetadataModal = false;
    let databaseLoaded = false;
    let mode = 'NORMAL';
    let showTakePoint2Modal = false;
    let showTakePoint4Modal = false;
    let isAnyModalOrPanelOpen = false;
    let isAnyModalOpen = false;
    let showFilterLibraryPanel = false; // Update variable
    let showMatchPanel = false; // Add match panel visibility variable
    let showCollectionPanel = false; // Add collection panel visibility variable
    let showTournamentPanel = false; // Add tournament panel visibility variable
    let showPipcount = true; // Update variable

    // EPC mode state
    let savedPositionBeforeEPC = null;
    let savedPositionIndexBeforeEPC = -1;
    let savedPositionsBeforeEPC = null;
    
    // Save position before entering COLLECTION mode (for restore on exit)
    let savedPositionBeforeCollection = null;
    let savedPositionIndexBeforeCollection = -1;
    let savedPositionsBeforeCollection = null;
    
    // Session state tracking
    let lastSearchCommand = '';
    let lastSearchPosition = null;
    let hasActiveSearch = false;
    
    // Subscribe to tournament panel store
    showTournamentPanelStore.subscribe(value => {
        showTournamentPanel = value;
    });

    // Subscribe to collection panel store
    showCollectionPanelStore.subscribe(value => {
        showCollectionPanel = value;
    });

    // Subscrive to pipcount store
    showPipcountStore.subscribe(value => {
        showPipcount = value;
    });

    // Subscribe to the metaStore
    metaStore.subscribe(value => {
        applicationVersion = value.applicationVersion;
    });

    // Subscribe to the derived store
    isAnyModalOrPanelOpenStore.subscribe(value => {
        isAnyModalOrPanelOpen = value;
    });

        // Subscribe to the derived store
        isAnyModalOpenStore.subscribe(value => {
        isAnyModalOpen = value;
    });

    // Subscribe to the databaseLoadedStore
    databaseLoadedStore.subscribe(value => {
        databaseLoaded = value;
    });

    // Subscribe to the showFilterLibraryPanelStore
    showFilterLibraryPanelStore.subscribe(value => {
        showFilterLibraryPanel = value;
    });

    // Subscribe to the showMatchPanelStore
    showMatchPanelStore.subscribe(value => {
        showMatchPanel = value;
    });

    // Subscribe to position store for EPC mode updates
    positionStore.subscribe(value => {
        if ($statusBarModeStore === 'EPC' && value) {
            updateEPC(value);
        }
    });

    // Subscribe to position reload trigger
    positionReloadTriggerStore.subscribe(async () => {
        if ($databasePathStore) {
            await loadAllPositions();
        }
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
                id: 0, // Add a default id
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
            analysisStore.set({
                positionId: null,
                xgid: '',
                player1: '',
                player2: '',
                analysisType: '',
                analysisEngineVersion: '',
                checkerAnalysis: { moves: [] },
                doublingCubeAnalysis: {
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
                allCubeAnalyses: [],
                playedMove: '',
                playedCubeAction: '',
                playedMoves: [],
                playedCubeActions: [],
                creationDate: '',
                lastModifiedDate: ''
            }); // Reset analysisStore when no positions
        }
    });

    // Debounce timer for saving session state
    let saveSessionTimeout = null;

    currentPositionIndexStore.subscribe(async value => {
        currentPositionIndex = value;
        if (positions.length > 0 && currentPositionIndex >= 0 && currentPositionIndex < positions.length) {
            await showPosition(positions[currentPositionIndex]);
            if ($statusBarModeStore !== 'EPC') {
                setStatusBarMessage(''); // Reset status bar message when changing position
            }
            
            // Debounce session state saving to avoid too many writes when scrolling
            if (saveSessionTimeout) {
                clearTimeout(saveSessionTimeout);
            }
            saveSessionTimeout = setTimeout(() => {
                saveSessionState();
            }, 500); // Save after 500ms of no changes
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

    showSearchHistoryPanelStore.subscribe(value => {
        showSearchHistoryPanel = value;
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



    databasePathStore.subscribe(value => {
        databaseLoaded = !!value;
    });

    statusBarModeStore.subscribe(value => {
        mode = value;
    });

    //Global shortcuts
    function handleKeyDown(event) {
        event.stopPropagation();

        // Debug logging for Ctrl-H
        if (event.ctrlKey && event.code === 'KeyH') {
            console.log('DEBUG: Ctrl-H pressed');
            console.log('  - isAnyModalOpenStore:', $isAnyModalOpenStore);
            console.log('  - activeElement:', document.activeElement);
            console.log('  - closest filter-library-panel:', document.activeElement.closest('.filter-library-panel'));
            console.log('  - closest search-history-panel:', document.activeElement.closest('.search-history-panel'));
            console.log('  - showComment:', showComment);
            console.log('  - showSearchHistoryPanel:', $showSearchHistoryPanelStore);
        }

        // Prevent shortcuts if any modal is open
        if ($isAnyModalOpenStore) {
            console.log('DEBUG: Returning because modal is open');
            return;
        }

        // Allow normal typing in input fields and textareas (only process Ctrl shortcuts)
        if (document.activeElement.matches('input, textarea, [contenteditable]') && !event.ctrlKey) {
            return;
        }

        // Special handling for analysis panel
        if (document.activeElement.closest('.analysis-panel')) {
            // Analysis panel has focus
            if (event.ctrlKey) {
                // Don't return, let the shortcut be processed below
            } else {
                // Check if a move is selected in the analysis panel
                const isNavigationKey = (event.key === 'j' || event.key === 'k' || 
                                        event.key === 'ArrowLeft' || event.key === 'ArrowRight' ||
                                        event.key === 'h' || event.key === 'l' ||
                                        event.key === 'PageUp' || event.key === 'PageDown');
                
                if (isNavigationKey && !$selectedMoveStore) {
                    // Navigation key pressed and no move selected - allow position navigation
                    // Don't return, let position navigation happen
                } else {
                    // Either not a navigation key, or a move is selected - let panel handle it
                    return; // Let AnalysisPanel.svelte handle it
                }
            }
        }

        // Prevent command line from opening when editing filter panel fields or comment panel
        if (document.activeElement.closest('.filter-library-panel') || document.activeElement.closest('.search-history-panel') || document.activeElement.closest('.match-panel') || document.activeElement.closest('.collection-panel') || document.activeElement.closest('.tournament-panel') || showComment) {
            // Allow all Ctrl+key shortcuts to work globally
            if (event.ctrlKey) {
                console.log('DEBUG: Inside panel, but Ctrl key pressed - allowing shortcut');
                event.preventDefault();
                // Don't return, let the shortcut be processed below
            } else if (event.code === 'Space') {
                // Allow Space key to open command line even when panels are open
                console.log('DEBUG: Inside panel, but Space key pressed - allowing command line to open');
                // Don't return, let the Space key handler below process it
            } else {
                // Allow j/k/left/right keys for position navigation when no row is selected in panels
                const isNavigationKey = (event.key === 'j' || event.key === 'k' || 
                                        event.key === 'ArrowLeft' || event.key === 'ArrowRight' ||
                                        event.key === 'h' || event.key === 'l' ||
                                        event.key === 'PageUp' || event.key === 'PageDown');
                
                if (isNavigationKey) {
                    // Check if any row is selected in the panels (not analysis - already handled above)
                    const filterLibraryHasSelection = document.querySelector('.filter-library-panel tr.highlight');
                    const searchHistoryHasSelection = document.querySelector('.search-history-panel tr.selected');
                    const matchPanelHasSelection = document.querySelector('.match-panel tr.selected');
                    
                    if (!filterLibraryHasSelection && !searchHistoryHasSelection && !matchPanelHasSelection) {
                        // No selection, allow position navigation - don't return early
                        console.log('DEBUG: Panel open but no selection, allowing navigation key:', event.key);
                    } else {
                        // Has selection, let panel handle it
                        console.log('DEBUG: Inside panel with selection, returning early');
                        return;
                    }
                } else {
                    console.log('DEBUG: Inside panel, returning early (not Ctrl or navigation)');
                    return;
                }
            }
        }

        if (event.key === 'Escape') {
            event.preventDefault();
            event.stopPropagation();
        } else if(event.ctrlKey && event.code == 'KeyN') {
            newDatabase();
        } else if(event.ctrlKey && event.code == 'KeyO') {
            openDatabase();
        } else if (event.ctrlKey && event.code === 'KeyQ') {
            exitApp();
        } else if(event.ctrlKey && event.shiftKey && event.code == 'KeyI') {
            importDatabase();
        } else if(event.ctrlKey && event.shiftKey && event.code == 'KeyF') {
            importFolder();
        } else if(event.ctrlKey && event.code == 'KeyI') {
            importPosition();
        } else if(event.ctrlKey && event.code == 'KeyC') {
            copyPosition();
        } else if(event.ctrlKey && event.code == 'KeyV') {
            pastePosition();
        } else if(event.ctrlKey && event.shiftKey && event.code == 'KeyS') {
            exportDatabase();
        } else if(event.ctrlKey && event.code == 'KeyS') {
            saveCurrentPosition();
        } else if(event.ctrlKey && event.code == 'KeyU') {
            updatePosition();
        } else if (event.code === 'Delete') {
            deletePosition();
        } else if (!event.ctrlKey && event.key === 'PageUp') {
            if (!showComment) {
                event.preventDefault();
                firstPosition();
            }
        } else if (!event.ctrlKey && event.key === 'h') {
            if (!showComment) {
                firstPosition();
            }
        } else if (!event.ctrlKey && event.key === 'ArrowLeft') {
            if (!showComment && !$selectedMoveStore) {
                event.preventDefault();
                previousPosition();
            }
        } else if (!event.ctrlKey && event.key === 'k') {
            if (!showComment && !$selectedMoveStore) {
                previousPosition();
            }
        } else if (!event.ctrlKey && event.key === 'ArrowRight') {
            if (!showComment && !$selectedMoveStore) {
                event.preventDefault();
                nextPosition();
            }
        } else if (!event.ctrlKey && event.key === 'j') {
            if (!showComment && !$selectedMoveStore) {
                nextPosition();
            }
        } else if (!event.ctrlKey && event.key === 'PageDown') {
            if (!showComment) {
                event.preventDefault();
                lastPosition();
            }
        } else if (!event.ctrlKey && event.key === 'l') {
            if (!showComment) {
                lastPosition();
            }
        } else if(event.ctrlKey && event.code == 'KeyK') {
            event.preventDefault();
            toggleCollectionPanelAction();
        } else if(event.ctrlKey && event.code == 'KeyR') {
            loadAllPositions();
        } else if(event.ctrlKey && event.code === 'Tab') {
                event.preventDefault();
                toggleMatchMode();
        } else if(!event.ctrlKey && event.code === 'Tab') {
                toggleEditMode();
        } else if (!event.ctrlKey && event.code === 'Space') {        
                event.preventDefault();
                showCommandStore.set(true); // Show command line
        } else if (event.ctrlKey && event.code === 'KeyL') {
            event.preventDefault();
            if (showComment) {
                toggleCommentPanel(); // Close comment panel if open
            }
            toggleAnalysisPanel();
        } else if(event.ctrlKey && event.code == 'KeyP') {
                event.preventDefault();
                toggleCommentPanel();
        } else if (event.ctrlKey && event.code === 'KeyF') {
            if ($statusBarModeStore === 'EDIT') {
                findPosition();
            } else {
                setStatusBarMessage('Search is only available in edit mode');
            }
        } else if (event.ctrlKey && event.code === 'KeyH') {
            console.log('DEBUG: Calling toggleSearchHistoryPanel');
            toggleSearchHistoryPanel(); // Changed to toggleSearchHistoryPanel
        } else if (!event.ctrlKey && event.key === '?') {
            toggleHelpModal(); // Keep '?' for help modal
        } else if (event.ctrlKey && event.code === 'KeyM') {
            toggleMetadataModal();
        } else if (event.ctrlKey && event.code === 'KeyB') {
            toggleFilterLibraryPanel();
        } else if (event.ctrlKey && event.code === 'KeyT') {
            toggleMatchPanel();
        } else if (event.ctrlKey && event.code === 'KeyY') {
            event.preventDefault();
            toggleTournamentPanel();
        } else if (!event.ctrlKey && event.code === 'KeyP') {
            togglePipcount();
        } else if (!event.ctrlKey && event.code === 'KeyR') {
            loadRandomPosition();
        }
    }

    function getFilenameFromPath(filePath) {
        return filePath.split('/').pop();
    }

    // Helper function to reset analysis and comment stores
    function resetAnalysisAndCommentStores() {
        analysisStore.set({
            positionId: null,
            xgid: '',
            player1: '',
            player2: '',
            analysisType: '',
            analysisEngineVersion: '',
            checkerAnalysis: { moves: [] },
            doublingCubeAnalysis: {
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
            allCubeAnalyses: [],
            playedMove: '',
            playedCubeAction: '',
            playedMoves: [],
            playedCubeActions: [],
            creationDate: '',
            lastModifiedDate: ''
        });
        commentTextStore.set('');
        selectedMoveStore.set(null);
    }

    export function setStatusBarMessage(message) {
        statusBarTextStore.set(message);
    }

    async function newDatabase() {
        console.log('newDatabase');
        try {
            const filePath = await SaveDatabaseDialog();
            if (filePath) {
                // Reset analysis and comment stores before creating new database
                resetAnalysisAndCommentStores();

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
        } finally {
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
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

            await openDatabaseByPath(filePath);
        } catch (error) {
            console.error('Error opening file dialog:', error);
            setStatusBarMessage('Error opening database');
        }
    }

    // Open a database given its file path (used by both dialog and auto-reopen)
    async function openDatabaseByPath(filePath) {
        try {
            // Reset analysis and comment stores before opening new database
            resetAnalysisAndCommentStores();

            databasePathStore.set(filePath);
            console.log('databasePathStore:', $databasePathStore);

            // Save the database path to config for auto-reopen on next launch
            await SaveLastDatabasePath(filePath);

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

            // Try to restore session state
            await restoreSessionState();
        } catch (error) {
            console.error('Error opening database:', error);
            setStatusBarMessage('Error opening database');
        } finally {
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
        }
    }
    
    // Restore the last session state when opening a database
    async function restoreSessionState() {
        try {
            const sessionState = await LoadSessionState();
            console.log('Loaded session state:', sessionState);
            
            if (sessionState && sessionState.hasActiveSearch && sessionState.lastPositionIds && sessionState.lastPositionIds.length > 0) {
                // Restore the session state
                lastSearchCommand = sessionState.lastSearchCommand || '';
                lastSearchPosition = sessionState.lastSearchPosition ? JSON.parse(sessionState.lastSearchPosition) : null;
                hasActiveSearch = true;
                
                // Load positions by IDs (we need to fetch them from DB)
                const allPositions = await LoadAllPositions();
                const positionIdSet = new Set(sessionState.lastPositionIds);
                const restoredPositions = allPositions.filter(pos => positionIdSet.has(pos.id));
                
                // Preserve original order from lastPositionIds
                const positionMap = new Map(restoredPositions.map(pos => [pos.id, pos]));
                const orderedPositions = sessionState.lastPositionIds
                    .map(id => positionMap.get(id))
                    .filter(pos => pos !== undefined);
                
                if (orderedPositions.length > 0) {
                    positionsStore.set(orderedPositions);
                    
                    // Restore position index, ensuring it's within valid bounds
                    let indexToRestore = sessionState.lastPositionIndex || 0;
                    if (indexToRestore < 0) indexToRestore = 0;
                    if (indexToRestore >= orderedPositions.length) indexToRestore = orderedPositions.length - 1;
                    
                    currentPositionIndexStore.set(-1); // Force redraw
                    currentPositionIndexStore.set(indexToRestore);
                    
                    setStatusBarMessage(`Session restored: ${orderedPositions.length} positions, showing #${indexToRestore + 1}`);
                    console.log(`Session restored with ${orderedPositions.length} positions at index ${indexToRestore}`);
                    return;
                }
            }
            
            // No session to restore, or session had no positions - load all positions
            hasActiveSearch = false;
            lastSearchCommand = '';
            lastSearchPosition = null;
            await loadAllPositions();
        } catch (error) {
            console.error('Error restoring session state:', error);
            // Fall back to loading all positions
            await loadAllPositions();
        }
    }
    
    // Save the current session state to the database
    async function saveSessionState() {
        if (!$databasePathStore) return;
        
        try {
            const positionIds = positions.map(pos => pos.id);
            const sessionState = {
                lastSearchCommand: lastSearchCommand,
                lastSearchPosition: lastSearchPosition ? JSON.stringify(lastSearchPosition) : '',
                lastPositionIndex: currentPositionIndex,
                lastPositionIds: positionIds,
                hasActiveSearch: hasActiveSearch
            };
            
            await SaveSessionState(sessionState);
            console.log('Session state saved');
        } catch (error) {
            console.error('Error saving session state:', error);
        }
    }

    function getMajorVersion(version) {
        return version.split('.')[0];
    }

    async function importDatabase() {
        console.log('importDatabase');
        
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened. Please open a database first.');
            return;
        }

        try {
            const importFilePath = await OpenImportDatabaseDialog();
            if (!importFilePath) {
                console.log('No import database selected');
                return;
            }

            console.log('Analyzing import from:', importFilePath);
            
            // Show modal in analyzing mode
            showImportProgressModal = true;
            importModalMode = 'analyzing';
            pendingImportPath = importFilePath;

            try {
                // First, analyze what would be imported (ACID: no changes yet)
                const analysis = await AnalyzeImportDatabase(importFilePath);
                
                console.log('Import analysis:', analysis);
                
                // Update to preview mode with analysis results
                importAnalysis = {
                    toAdd: analysis.toAdd,
                    toMerge: analysis.toMerge,
                    toSkip: analysis.toSkip,
                    total: analysis.total,
                    importPath: importFilePath
                };
                importModalMode = 'preview';
                
            } catch (analyzeError) {
                showImportProgressModal = false;
                throw analyzeError;
            }
            
        } catch (error) {
            console.error('Error analyzing import:', error);
            setStatusBarMessage(`Error analyzing import: ${error}`);
            await ShowAlert(`Error analyzing import: ${error}`);
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
        }
    }

    async function handleImportCommit() {
        if (!pendingImportPath) {
            console.error('No pending import path');
            return;
        }

        console.log('Committing import from:', pendingImportPath);
        
        // Change to committing mode
        importModalMode = 'committing';

        try {
            // Commit the import (ACID transaction)
            const result = await CommitImportDatabase(pendingImportPath);
            
            console.log('Import result:', result);
            
            // Update to completed mode with results
            importResult = {
                added: result.added,
                merged: result.merged,
                skipped: result.skipped,
                total: result.total
            };
            importModalMode = 'completed';
            
            setStatusBarMessage(`Import completed: ${result.added} added, ${result.merged} merged, ${result.skipped} skipped`);
            
            // Reload all positions to refresh the view
            await loadAllPositions();
            
        } catch (error) {
            console.error('Error committing import:', error);
            showImportProgressModal = false;
            setStatusBarMessage(`Error committing import: ${error}`);
            await ShowAlert(`Error committing import: ${error}`);
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
        } finally {
            pendingImportPath = null;
        }
    }

    function handleImportCancel() {
        console.log('Import cancelled by user');
        
        // If we're in the committing phase, call the backend to cancel
        if (importModalMode === 'committing') {
            console.log('Aborting ongoing commit transaction');
            CancelImport().catch(err => {
                console.error('Error calling CancelImport:', err);
            });
        }
        
        showImportProgressModal = false;
        pendingImportPath = null;
        importModalMode = 'analyzing';
        setStatusBarMessage('Import cancelled');
        previousModeStore.set('NORMAL');
        statusBarModeStore.set('NORMAL');
    }

    function handleImportClose() {
        console.log('Import completed and closed');
        showImportProgressModal = false;
        pendingImportPath = null;
        importModalMode = 'analyzing';
        previousModeStore.set('NORMAL');
        statusBarModeStore.set('NORMAL');
    }

    async function exportDatabase() {
        console.log('exportDatabase');
        
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened. Please open a database first.');
            return;
        }

        if (positions.length === 0) {
            setStatusBarMessage('No positions to export.');
            await ShowAlert('No positions to export. Please load positions first.');
            return;
        }

        try {
            const exportFilePath = await OpenExportDatabaseDialog();
            if (!exportFilePath) {
                console.log('No export path selected');
                return;
            }

            console.log('Exporting to:', exportFilePath);
            pendingExportPath = exportFilePath;
            
            // Get matches for export selection
            try {
                const matches = await GetAllMatches();
                exportMatches = matches || [];
            } catch (e) {
                console.log('Could not get matches:', e);
                exportMatches = [];
            }

            // Load collections for export selection
            try {
                const colls = await GetAllCollections();
                collectionsStore.set(colls || []);
            } catch (e) {
                console.log('Could not get collections:', e);
            }

            // Load tournaments for export selection
            try {
                const tourns = await GetAllTournaments();
                tournamentsStore.set(tourns || []);
            } catch (e) {
                console.log('Could not get tournaments:', e);
            }
            
            // Show modal in metadata mode
            exportPositionCount = positions.length;
            exportModalMode = 'metadata';
            showExportDatabaseModalStore.set(true);
            
        } catch (error) {
            console.error('Error during export preparation:', error);
            setStatusBarMessage(`Error preparing export: ${error}`);
            await ShowAlert(`Error preparing export: ${error}`);
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
        }
    }

    async function handleExportCommit() {
        if (!pendingExportPath) {
            console.error('No pending export path');
            return;
        }

        console.log('Committing export to:', pendingExportPath);
        
        // Change to exporting mode
        exportModalMode = 'exporting';

        try {
            // Prepare metadata
            const metadata = {
                user: exportMetadata.user || '',
                description: exportMetadata.description || '',
                dateOfCreation: exportMetadata.dateOfCreation || ''
            };

            // Call the backend to export
            await ExportDatabase(
                pendingExportPath, 
                positions, 
                metadata,
                exportOptions.includeAnalysis,
                exportOptions.includeComments,
                exportOptions.includeFilterLibrary,
                exportOptions.includePlayedMoves,
                exportOptions.includeMatches,
                exportOptions.includeCollections,
                exportOptions.collectionIDs || [],
                exportOptions.matchIDs || [],
                exportOptions.includeTournamentIDs || []
            );
            
            console.log('Export completed successfully');
            
            // Update to completed mode
            exportModalMode = 'completed';
            
            setStatusBarMessage(`Export completed: ${exportPositionCount} position(s) exported`);
            
        } catch (error) {
            console.error('Error committing export:', error);
            showExportDatabaseModalStore.set(false);
            setStatusBarMessage(`Error committing export: ${error}`);
            await ShowAlert(`Error committing export: ${error}`);
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
        } finally {
            // Reset metadata for next export
            exportMetadata = {
                user: '',
                description: '',
                dateOfCreation: ''
            };
            exportOptions = {
                includeAnalysis: true,
                includeComments: true,
                includeFilterLibrary: false,
                includePlayedMoves: true,
                includeMatches: true,
                matchIDs: [],
                includeTournaments: false,
                includeTournamentIDs: [],
                includeCollections: false,
                collectionIDs: []
            };
            exportMatches = [];
        }
    }

    function handleExportCancel() {
        console.log('Export cancelled by user');
        
        showExportDatabaseModalStore.set(false);
        pendingExportPath = null;
        exportModalMode = 'preparing';
        exportMetadata = {
            user: '',
            description: '',
            dateOfCreation: ''
        };
        exportOptions = {
            includeAnalysis: true,
            includeComments: true,
            includeFilterLibrary: false,
            includePlayedMoves: true,
            includeMatches: true,
            matchIDs: [],
            includeTournaments: false,
            includeTournamentIDs: [],
            includeCollections: false,
            collectionIDs: []
        };
        exportMatches = [];
        setStatusBarMessage('Export cancelled');
        previousModeStore.set('NORMAL');
        statusBarModeStore.set('NORMAL');
    }

    function handleExportClose() {
        console.log('Export completed and closed');
        showExportDatabaseModalStore.set(false);
        pendingExportPath = null;
        exportModalMode = 'preparing';
        exportMetadata = {
            user: '',
            description: '',
            dateOfCreation: ''
        };
        exportOptions = {
            includeAnalysis: true,
            includeComments: true,
            includeFilterLibrary: false,
            includePlayedMoves: true,
            includeMatches: true,
            matchIDs: [],
            includeTournaments: false,
            includeTournamentIDs: [],
            includeCollections: false,
            collectionIDs: []
        };
        exportMatches = [];
        previousModeStore.set('NORMAL');
        statusBarModeStore.set('NORMAL');
    }

    function closeWarningModal() {
        showWarningModalStore.set(false); // Use store to close warning modal
    }

    async function exitApp() {
        // Save session state before quitting so the user can resume
        await saveSessionState();
        Quit();
    }

    async function savePositionAndAnalysis(positionData, parsedAnalysis, successMessage) {
        // Ensure checkerAnalysis is correctly structured
        if (Array.isArray(parsedAnalysis.checkerAnalysis)) {
            parsedAnalysis.checkerAnalysis = { moves: parsedAnalysis.checkerAnalysis };
        }

        // Remove creationDate and lastModifiedDate from parsedAnalysis since they are dealt by the backend
        delete parsedAnalysis.creationDate;
        delete parsedAnalysis.lastModifiedDate;

        // Check if the position already exists
        const positionExistsResult = await PositionExists(positionData);
        if (positionExistsResult.exists) {
            console.log('Position already exists with ID:', positionExistsResult.id);
            try {
                parsedAnalysis.positionId = positionExistsResult.id; // Ensure the position ID is set in the analysis
                await SaveAnalysis(positionExistsResult.id, parsedAnalysis);

                // Merge comments intelligently
                let existingComment = await LoadComment(positionExistsResult.id);
                const newComment = parsedAnalysis.comment || '';
                
                // Trim both comments for comparison
                const trimmedExisting = (existingComment || '').trim();
                const trimmedNew = newComment.trim();
                
                let mergedComment = trimmedExisting;
                
                // Only merge if the new comment has content and is not already present
                if (trimmedNew && !trimmedExisting.includes(trimmedNew)) {
                    // If existing comment has content, add separator
                    if (trimmedExisting) {
                        mergedComment = `${trimmedExisting}\n\n${trimmedNew}`;
                    } else {
                        // No existing comment, just use the new one
                        mergedComment = trimmedNew;
                    }
                }
                
                await SaveComment(positionExistsResult.id, mergedComment);

                console.log('Analysis and comment updated for position ID:', positionExistsResult.id);
                setStatusBarMessage('Position already exists, analysis and comment merged');
                currentPositionIndexStore.set(-1); //force change to trigger re-render
                currentPositionIndexStore.set(positions.findIndex(pos => pos.id === positionExistsResult.id)); // Set current position index to display the existing position
                
                // Update the comment store to reflect the merged comment
                commentTextStore.set(mergedComment);
            } catch (error) {
                console.error('Error updating analysis and comment:', error);
                setStatusBarMessage('Error updating analysis and comment');
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
            await SaveComment(positionID, parsedAnalysis.comment); // Save the comment
            console.log('Analysis and comment saved for position ID:', positionID);

            // Reload all positions and show the last one
            await loadAllPositions();
            setStatusBarMessage(successMessage);
        } catch (error) {
            console.error('Error saving position, analysis, and comment:', error);
            setStatusBarMessage('Error saving position, analysis, and comment');
        }
    }

    export async function importPosition() {
        const isNormalMode = $statusBarModeStore === 'NORMAL' || ($statusBarModeStore === 'COMMAND' && $previousModeStore === 'NORMAL');
        const isMatchMode = $statusBarModeStore === 'MATCH' || ($statusBarModeStore === 'COMMAND' && $previousModeStore === 'MATCH');
        if (!isNormalMode && !isMatchMode) {
            setStatusBarMessage('Cannot import position in current mode');
            return;
        }
        const wasMatchMode = isMatchMode;
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        try {
            // Show a choice: files or folder
            const files = await OpenPositionFilesDialog();

            if (!files || files.length === 0) {
                return;
            }

            if (files.length === 1) {
                // Single file: use the original inline import logic
                const filePath = files[0];
                await importSingleFile(filePath);
            } else {
                // Multiple files: batch import with progress
                await importMultipleFiles(files);
            }
        } catch (error) {
            console.error("Error importing position:", error);
        } finally {
            if (wasMatchMode) {
                matchContextStore.set({
                    isMatchMode: false,
                    matchID: null,
                    movePositions: [],
                    currentIndex: 0,
                    player1Name: '',
                    player2Name: ''
                });
                loadAllPositions();
            }
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
        }
    }

    export async function importFolder() {
        const isNormalMode = $statusBarModeStore === 'NORMAL' || ($statusBarModeStore === 'COMMAND' && $previousModeStore === 'NORMAL');
        const isMatchMode = $statusBarModeStore === 'MATCH' || ($statusBarModeStore === 'COMMAND' && $previousModeStore === 'MATCH');
        if (!isNormalMode && !isMatchMode) {
            setStatusBarMessage('Cannot import in current mode');
            return;
        }
        const wasMatchMode = isMatchMode;
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        try {
            const dirPath = await OpenPositionFolderDialog();
            if (!dirPath) {
                return;
            }

            const files = await CollectImportableFiles(dirPath);
            if (!files || files.length === 0) {
                setStatusBarMessage('No importable files found in folder');
                return;
            }

            await importMultipleFiles(files);
        } catch (error) {
            console.error("Error importing folder:", error);
        } finally {
            if (wasMatchMode) {
                matchContextStore.set({
                    isMatchMode: false,
                    matchID: null,
                    movePositions: [],
                    currentIndex: 0,
                    player1Name: '',
                    player2Name: ''
                });
                loadAllPositions();
            }
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
        }
    }

    // Import a single file using the original logic (inline, with status bar messages and alerts)
    async function importSingleFile(filePath) {
        const lowerPath = filePath.toLowerCase();
        const isXGFile = lowerPath.endsWith('.xg');
        const isBGFFile = lowerPath.endsWith('.bgf');
        const isSGFFile = lowerPath.endsWith('.sgf');
        const isMATFile = lowerPath.endsWith('.mat');
        const isTXTFile = lowerPath.endsWith('.txt');

        if (isXGFile) {
            console.log("Importing XG match file:", filePath);
            try {
                const matchID = await ImportXGMatch(filePath);
                console.log('XG match imported with ID:', matchID);
                setStatusBarMessage(`XG match imported successfully (ID: ${matchID})`);
                matchPanelRefreshTriggerStore.update(n => n + 1);
                showMatchPanelStore.set(true);
            } catch (error) {
                console.error("Error importing XG match:", error);
                const errorStr = String(error);
                if (errorStr.includes('duplicate match') || errorStr.includes('already been imported')) {
                    setStatusBarMessage('This match has already been imported');
                } else {
                    setStatusBarMessage('Error importing XG match: ' + error);
                    await ShowAlert('Error importing XG match: ' + error);
                }
            }
        } else if (isBGFFile) {
            console.log("Importing BGF match file:", filePath);
            try {
                const matchID = await ImportBGFMatch(filePath);
                console.log('BGF match imported with ID:', matchID);
                setStatusBarMessage(`BGBlitz match imported successfully (ID: ${matchID})`);
                matchPanelRefreshTriggerStore.update(n => n + 1);
                showMatchPanelStore.set(true);
            } catch (error) {
                console.error("Error importing BGF match:", error);
                const errorStr = String(error);
                if (errorStr.includes('duplicate match') || errorStr.includes('already been imported')) {
                    setStatusBarMessage('This match has already been imported');
                } else {
                    setStatusBarMessage('Error importing BGBlitz match: ' + error);
                    await ShowAlert('Error importing BGBlitz match: ' + error);
                }
            }
        } else if (isSGFFile || isMATFile) {
            const formatName = isSGFFile ? 'GnuBG SGF' : 'Jellyfish MAT';
            console.log(`Importing ${formatName} match file:`, filePath);
            try {
                const matchID = await ImportGnuBGMatch(filePath);
                console.log(`${formatName} match imported with ID:`, matchID);
                setStatusBarMessage(`${formatName} match imported successfully (ID: ${matchID})`);
                matchPanelRefreshTriggerStore.update(n => n + 1);
                showMatchPanelStore.set(true);
            } catch (error) {
                console.error(`Error importing ${formatName} match:`, error);
                const errorStr = String(error);
                if (errorStr.includes('duplicate match') || errorStr.includes('already been imported')) {
                    setStatusBarMessage('This match has already been imported');
                } else {
                    setStatusBarMessage(`Error importing ${formatName} match: ` + error);
                    await ShowAlert(`Error importing ${formatName} match: ` + error);
                }
            }
        } else if (isTXTFile) {
            await importTxtFile(filePath);
        }
    }

    // Import a .txt file by reading its content and detecting the type
    async function importTxtFile(filePath) {
        const response = await ReadFileContent(filePath);
        if (response.error) {
            console.error("Error reading file:", response.error);
            setStatusBarMessage('Error reading file: ' + response.error);
            return;
        }
        const content = response.content;

        const isJellyfishTXT = content && /^\s*\d+\s+point\s+match\s*$/m.test(content);
        const isBGBlitzTXT = content && content.includes('Position-ID:');

        if (isJellyfishTXT) {
            console.log("Importing Jellyfish TXT match file:", filePath);
            try {
                const matchID = await ImportGnuBGMatch(filePath);
                console.log('Jellyfish TXT match imported with ID:', matchID);
                setStatusBarMessage(`Jellyfish TXT match imported successfully (ID: ${matchID})`);
                matchPanelRefreshTriggerStore.update(n => n + 1);
                showMatchPanelStore.set(true);
            } catch (error) {
                console.error("Error importing Jellyfish TXT match:", error);
                const errorStr = String(error);
                if (errorStr.includes('duplicate match') || errorStr.includes('already been imported')) {
                    setStatusBarMessage('This match has already been imported');
                } else {
                    setStatusBarMessage('Error importing Jellyfish TXT match: ' + error);
                    await ShowAlert('Error importing Jellyfish TXT match: ' + error);
                }
            }
        } else if (isBGBlitzTXT) {
            console.log("Importing BGBlitz TXT position:", filePath);
            try {
                const posID = await ImportBGFPosition(filePath);
                console.log('BGBlitz position imported with ID:', posID);
                setStatusBarMessage(`BGBlitz position imported successfully (ID: ${posID})`);
                await loadAllPositions();
            } catch (error) {
                console.error("Error importing BGBlitz position:", error);
                const errorStr = String(error);
                if (errorStr.includes('duplicate') || errorStr.includes('already exists')) {
                    setStatusBarMessage('This position already exists');
                } else {
                    setStatusBarMessage('Error importing BGBlitz position: ' + error);
                    await ShowAlert('Error importing BGBlitz position: ' + error);
                }
            }
        } else {
            // XG text position file via JS parser
            console.log("File content:", content);
            const { positionData, parsedAnalysis } = parsePosition(content);
            positionStore.set({ ...positionData, id: 0, board: { ...positionData.board, bearoff: [15, 15] } });
            analysisStore.set({
                positionId: null,
                xgid: parsedAnalysis.xgid,
                player1: '',
                player2: '',
                analysisType: parsedAnalysis.analysisType,
                analysisEngineVersion: parsedAnalysis.analysisEngineVersion,
                checkerAnalysis: { moves: parsedAnalysis.checkerAnalysis },
                doublingCubeAnalysis: {
                    analysisDepth: parsedAnalysis.doublingCubeAnalysis.analysisDepth || '',
                    playerWinChances: parsedAnalysis.doublingCubeAnalysis.playerWinChances || 0,
                    playerGammonChances: parsedAnalysis.doublingCubeAnalysis.playerGammonChances || 0,
                    playerBackgammonChances: parsedAnalysis.doublingCubeAnalysis.playerBackgammonChances || 0,
                    opponentWinChances: parsedAnalysis.doublingCubeAnalysis.opponentWinChances || 0,
                    opponentGammonChances: parsedAnalysis.doublingCubeAnalysis.opponentGammonChances || 0,
                    opponentBackgammonChances: parsedAnalysis.doublingCubeAnalysis.opponentBackgammonChances || 0,
                    cubelessNoDoubleEquity: parsedAnalysis.doublingCubeAnalysis.cubelessNoDoubleEquity || 0,
                    cubelessDoubleEquity: parsedAnalysis.doublingCubeAnalysis.cubelessDoubleEquity || 0,
                    cubefulNoDoubleEquity: parsedAnalysis.doublingCubeAnalysis.cubefulNoDoubleEquity || 0,
                    cubefulNoDoubleError: parsedAnalysis.doublingCubeAnalysis.cubefulNoDoubleError || 0,
                    cubefulDoubleTakeEquity: parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeEquity || 0,
                    cubefulDoubleTakeError: parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeError || 0,
                    cubefulDoublePassEquity: parsedAnalysis.doublingCubeAnalysis.cubefulDoublePassEquity || 0,
                    cubefulDoublePassError: parsedAnalysis.doublingCubeAnalysis.cubefulDoublePassError || 0,
                    bestCubeAction: parsedAnalysis.doublingCubeAnalysis.bestCubeAction || '',
                    wrongPassPercentage: parsedAnalysis.doublingCubeAnalysis.wrongPassPercentage || 0,
                    wrongTakePercentage: parsedAnalysis.doublingCubeAnalysis.wrongTakePercentage || 0
                },
                allCubeAnalyses: [],
                playedMove: '',
                playedCubeAction: '',
                playedMoves: [],
                playedCubeActions: [],
                creationDate: '',
                lastModifiedDate: ''
            });
            console.log('positionStore:', $positionStore);
            console.log('analysisStore:', $analysisStore);
            await savePositionAndAnalysis(positionData, parsedAnalysis, 'Imported position and analysis saved successfully');
        }
    }

    // Import a single file in batch mode (no alerts, returns result string)
    async function importSingleFileBatch(filePath) {
        const lowerPath = filePath.toLowerCase();
        const isXGFile = lowerPath.endsWith('.xg');
        const isBGFFile = lowerPath.endsWith('.bgf');
        const isSGFFile = lowerPath.endsWith('.sgf');
        const isMATFile = lowerPath.endsWith('.mat');
        const isTXTFile = lowerPath.endsWith('.txt');

        if (isXGFile) {
            const matchID = await ImportXGMatch(filePath);
            return { type: 'match', id: matchID };
        } else if (isBGFFile) {
            const matchID = await ImportBGFMatch(filePath);
            return { type: 'match', id: matchID };
        } else if (isSGFFile || isMATFile) {
            const matchID = await ImportGnuBGMatch(filePath);
            return { type: 'match', id: matchID };
        } else if (isTXTFile) {
            return await importTxtFileBatch(filePath);
        }
        throw new Error('Unsupported file type');
    }

    // Import a .txt file in batch mode (no alerts, returns result)
    async function importTxtFileBatch(filePath) {
        const response = await ReadFileContent(filePath);
        if (response.error) {
            throw new Error(response.error);
        }
        const content = response.content;

        const isJellyfishTXT = content && /^\s*\d+\s+point\s+match\s*$/m.test(content);
        const isBGBlitzTXT = content && content.includes('Position-ID:');

        if (isJellyfishTXT) {
            const matchID = await ImportGnuBGMatch(filePath);
            return { type: 'match', id: matchID };
        } else if (isBGBlitzTXT) {
            const posID = await ImportBGFPosition(filePath);
            return { type: 'position', id: posID };
        } else {
            // XG text position via JS parser
            const { positionData, parsedAnalysis } = parsePosition(content);
            positionStore.set({ ...positionData, id: 0, board: { ...positionData.board, bearoff: [15, 15] } });
            await savePositionAndAnalysis(positionData, parsedAnalysis, '');
            return { type: 'position', id: 0 };
        }
    }

    // Batch import multiple files with progress modal
    async function importMultipleFiles(files) {
        fileImportCancelled = false;
        fileImportTotalFiles = files.length;
        fileImportCurrentIndex = 0;
        fileImportCurrentFile = '';
        fileImportResults = { succeeded: 0, failed: 0, skipped: 0, errors: [] };
        fileImportMode = 'importing';
        showFileImportModal = true;

        let hadMatches = false;

        for (let i = 0; i < files.length; i++) {
            if (fileImportCancelled) {
                break;
            }
            const filePath = files[i];
            fileImportCurrentIndex = i + 1;
            fileImportCurrentFile = filePath;

            try {
                const result = await importSingleFileBatch(filePath);
                fileImportResults.succeeded++;
                if (result && result.type === 'match') {
                    hadMatches = true;
                }
            } catch (error) {
                const errorStr = String(error);
                if (errorStr.includes('duplicate match') || errorStr.includes('already been imported') ||
                    errorStr.includes('duplicate') || errorStr.includes('already exists')) {
                    fileImportResults.skipped++;
                } else {
                    fileImportResults.failed++;
                    fileImportResults.errors.push({
                        file: filePath,
                        message: errorStr.replace(/^Error:\s*/, '')
                    });
                }
            }
            // Force Svelte reactivity update
            fileImportResults = { ...fileImportResults };
        }

        fileImportMode = 'completed';

        // Refresh data after batch import
        if (hadMatches) {
            matchPanelRefreshTriggerStore.update(n => n + 1);
        }
        await loadAllPositions();

        const msg = `Import done: ${fileImportResults.succeeded} imported, ${fileImportResults.skipped} skipped, ${fileImportResults.failed} failed`;
        setStatusBarMessage(msg);
    }

    function handleFileImportCancel() {
        fileImportCancelled = true;
        showFileImportModal = false;
        fileImportMode = 'idle';
        setStatusBarMessage('Import cancelled');
    }

    function handleFileImportClose() {
        showFileImportModal = false;
        fileImportMode = 'idle';
    }

    async function pastePosition() {
        if ($statusBarModeStore !== 'NORMAL' && !($statusBarModeStore === 'COMMAND' && $previousModeStore === 'NORMAL')) {
            setStatusBarMessage('Cannot paste position in current mode');
            return;
        }
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        console.log('pastePosition');
        let promise = ClipboardGetText();
        promise.then(
            async (result) => {
                pastePositionTextStore.set(result);
                console.log('pastePositionTextStore:', $pastePositionTextStore);

                // Check if this is a GnuBG/Jellyfish MAT/TXT match
                // MAT/TXT files contain "N point match" header and "Game N" lines
                const isGnuBGMatch = result && /^\s*\d+\s+point\s+match\s*$/m.test(result) && /^\s*Game\s+\d+\s*$/m.test(result);

                // Check if this is a BGBlitz TXT position
                // BGBlitz files contain "Position-ID:" which is unique to BGBlitz
                // XG text exports use the same board art but never have "Position-ID:"
                const isBGBlitzTXT = result && result.includes('Position-ID:');

                if (isGnuBGMatch) {
                    // Import GnuBG/Jellyfish match from clipboard text
                    try {
                        const matchID = await ImportGnuBGMatchFromText(result);
                        console.log('GnuBG match pasted with ID:', matchID);
                        setStatusBarMessage(`Match imported from clipboard successfully (ID: ${matchID})`);
                        matchPanelRefreshTriggerStore.update(n => n + 1);
                        showMatchPanelStore.set(true);
                    } catch (error) {
                        console.error('Error pasting GnuBG match:', error);
                        const errorStr = String(error);
                        if (errorStr.includes('duplicate match') || errorStr.includes('already been imported')) {
                            setStatusBarMessage('This match has already been imported');
                        } else {
                            setStatusBarMessage('Error importing match from clipboard: ' + error);
                        }
                    }
                } else if (isBGBlitzTXT) {
                    // Import BGBlitz TXT position via Go backend
                    try {
                        const posID = await ImportBGFPositionFromText(result);
                        console.log('BGBlitz position pasted with ID:', posID);
                        setStatusBarMessage(`BGBlitz position pasted successfully (ID: ${posID})`);
                        await loadAllPositions();
                    } catch (error) {
                        console.error('Error pasting BGBlitz position:', error);
                        setStatusBarMessage('Error pasting BGBlitz position: ' + error);
                    }
                } else {
                    const { positionData, parsedAnalysis } = parsePosition(result);
                    await savePositionAndAnalysis(positionData, parsedAnalysis, 'Pasted position and analysis saved successfully');
                }
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
        if ($statusBarModeStore !== 'EDIT' && !($statusBarModeStore === 'COMMAND' && $previousModeStore === 'EDIT')) {
            setStatusBarMessage('Save is only possible in edit mode');
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
        analysis.checkerAnalysis = { moves: [] };
        analysis.doublingCubeAnalysis = {
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
        };
        analysis.analysisEngineVersion = "";

        await savePositionAndAnalysis(position, analysis, 'Position and analysis saved successfully');
        statusBarModeStore.set('NORMAL');
        previousModeStore.set('NORMAL');
    }

    function parsePosition(fileContent) {
        if (!fileContent || fileContent.trim().length === 0) {
            throw new Error("File is empty or invalid.");
        }

        let normalizedContent = fileContent.replace(/\r\n|\r/g, '\n').trim();
        const lines = normalizedContent.split('\n').map(line => line.trim());

        const isFrench = normalizedContent.includes("Joueur") || normalizedContent.includes("Adversaire") || normalizedContent.includes("Videau");
        const isJapanese = normalizedContent.includes("プレーヤー") || normalizedContent.includes("対戦相手") || normalizedContent.includes("キューブ");
        const isInternalCheckerAnalysisFormat = normalizedContent.includes("Analysis:\nChecker Move Analysis:");
        const isInternalDoublingAnalysisFormat = normalizedContent.includes("Analysis:\nDoubling Cube Analysis:");
        const isGerman = normalizedContent.includes("Spieler") || normalizedContent.includes("Gegner") || normalizedContent.includes("Dopplerwürfel");

        // Replace commas with dots. Enable to treat decimal numbers with commas as well.
        normalizedContent = normalizedContent.replace(/,/g, '.');

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

        const decisionLine = lines.find(line => line.includes(isFrench ? "jouer" : isJapanese ? "to play" : isGerman ? "spielen" : "to play"));
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

        const engineVersionMatch = normalizedContent.match(/eXtreme Gammon Version: (.+?)(?:\. MET: (.+))?$/m); // remember comma transformed in dot
        if (engineVersionMatch) {
            parsedAnalysis.analysisEngineVersion = `eXtreme Gammon Version: ${engineVersionMatch[1]}`;
            if (engineVersionMatch[2]) {
                parsedAnalysis.analysisEngineVersion += `, MET: ${engineVersionMatch[2]}`;
            }
        }

        // Determine short engine name for per-move/per-cube tagging
        const engineName = engineVersionMatch ? "XG" : "";

        if (isInternalDoublingAnalysisFormat) {
            parsedAnalysis.analysisType = "DoublingCube";
            const doublingCubeAnalysisRegex = /Doubling Cube Analysis:\nAnalysis Depth: "(.+)"\nPlayer Win Chances: ([-.\d]+)%\nPlayer Gammon Chances: ([-.\d]+)%\nPlayer Backgammon Chances: ([-.\d]+)%\nOpponent Win Chances: ([-.\d]+)%\nOpponent Gammon Chances: ([-.\d]+)%\nOpponent Backgammon Chances: ([-.\d]+)%\nCubeless No Double Equity: ([-.\d]+)\nCubeless Double Equity: ([-.\d]+)\nCubeful No Double Equity: ([-.\d]+)\nCubeful No Double Error: ([-.\d]+)\nCubeful Double Take Equity: ([-.\d]+)\nCubeful Double Take Error: ([-.\d]+)\nCubeful Double Pass Equity: ([-.\d]+)\nCubeful Double Pass Error: ([-.\d]+)\nBest Cube Action: (.+)\nWrong Pass Percentage: ([-.\d]+)%\nWrong Take Percentage: ([-.\d]+)%/;
            const doublingCubeMatch = doublingCubeAnalysisRegex.exec(normalizedContent);
            if (doublingCubeMatch) {
                parsedAnalysis.doublingCubeAnalysis = {
                    analysisDepth: doublingCubeMatch[1].trim(),
                    analysisEngine: engineName,
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
            const moveRegex = /^Move (\d+): (.+)\nAnalysis Depth: "(.+)"\nEquity: ([-.\d]+)\nEquity Error: ([-.\d]+)\nPlayer Win Chance: ([-.\d]+)%\nPlayer Gammon Chance: ([-.\d]+)%\nPlayer Backgammon Chance: ([-.\d]+)%\nOpponent Win Chance: ([-.\d]+)%\nOpponent Gammon Chance: ([-.\d]+)%\nOpponent Backgammon Chance: ([-.\d]+)%/gm;
            let moveMatch;
            while ((moveMatch = moveRegex.exec(normalizedContent)) !== null) {
                const moveDetails = {
                    index: parseInt(moveMatch[1], 10),
                    move: moveMatch[2].trim(),
                    analysisDepth: moveMatch[3].trim(),
                    analysisEngine: engineName,
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
                ? /^ {4}(\d+)\.\s(.{11})\s(.{28})\séq:(.{5,7})\s(?:\((-?[-.\d]{5,7})\))?/
                : isJapanese
                ? /^ {4}(\d+)\.\s(.{11})\s(.{28})\seq:(.{5,7})\s(?:\((-?[-.\d]{5,7})\))?/
                : isGerman
                ? /^ {4}(\d+)\.\s(.{11})\s(.{28})\seq:(.{5,7})\s(?:\((-?[-.\d]{5,7})\))?/
                : /^ {4}(\d+)\.\s(.{11})\s(.{28})\seq:(.{5,7})\s(?:\((-?[-.\d]{5,7})\))?/,
                'gm'
            );
            let moveMatch;
            while ((moveMatch = moveRegex.exec(normalizedContent)) !== null) {
                const moveDetails = {
                    index: parseInt(moveMatch[1], 10),
                    analysisDepth: moveMatch[2].trim(),
                    analysisEngine: engineName,
                    move: moveMatch[3].trim(),
                    equity: parseFloat(moveMatch[4]),
                    equityError: moveMatch[5] ? parseFloat(moveMatch[5]) : 0,
                };
                const lineStart = moveMatch.index + moveMatch[0].length;
                const remainingContent = normalizedContent.slice(lineStart);
                const playerRegex = isFrench
                    ? /Joueur:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                    : isJapanese
                    ? /プレーヤー:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                    : isGerman
                    ? /Spieler:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                    : /Player:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/;
                const opponentRegex = isFrench
                    ? /Adversaire:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                    : isJapanese
                    ? /対戦相手:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                    : isGerman
                    ? /Gegner:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
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
            if (playerOnRoll === 1) {
                // Swap win, gammon, and backgammon chances between player and opponent for each move
                parsedAnalysis.checkerAnalysis.forEach(move => {
                    const tempWinChance = move.playerWinChance;
                    const tempGammonChance = move.playerGammonChance;
                    const tempBackgammonChance = move.playerBackgammonChance;

                    move.playerWinChance = move.opponentWinChance;
                    move.playerGammonChance = move.opponentGammonChance;
                    move.playerBackgammonChance = move.opponentBackgammonChance;

                    move.opponentWinChance = tempWinChance;
                    move.opponentGammonChance = tempGammonChance;
                    move.opponentBackgammonChance = tempBackgammonChance;
                });
            }
        } else if (
            (isFrench && (normalizedContent.includes("Equités sans videau") || normalizedContent.includes("Equités avec videau"))) ||
            (isJapanese && (normalizedContent.includes("Cubeless Equities") || normalizedContent.includes("Cubeful Equities"))) ||
            (isGerman && (normalizedContent.includes("Equities ohne Dopplerwürfel") || normalizedContent.includes("Equities mit Dopplerwürfel"))) ||
            (!isFrench && !isJapanese && !isGerman && (normalizedContent.includes("Cubeless Equities") || normalizedContent.includes("Cubeful Equities")))
        ) {
            parsedAnalysis.analysisType = "DoublingCube";

            const analysisDepthMatch = normalizedContent.match(new RegExp(isFrench ? /Analysé avec\s+([^\n]*)/ : isJapanese ? /Analyzed in\s+([^\n]*)/ : isGerman ? /Analysiert in\s+([^\n]*)/ : /Analyzed in\s+([^\n]*)/));

            const playerWinMatch = normalizedContent.match(new RegExp(isFrench ? /Chance de gain du joueur:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/ : isJapanese ? /Player Winning Chances:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/ : isGerman ? /Spieler Gewinnchancen:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/ : /Player Winning Chances:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/));

            const opponentWinMatch = normalizedContent.match(new RegExp(isFrench ? /Chance de gain de l'adversaire:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/ : isJapanese ? /Opponent Winning Chances:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/ : isGerman ? /Gewinnchancen des Gegners:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/ : /Opponent Winning Chances:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/));

            const cubelessMatch = normalizedContent.match(new RegExp(isFrench ? /Equités sans videau\s*:\s*Pas de double=([\+\-\d.]+).\s*Double=([\+\-\d.]+)/ : isJapanese ? /Cubeless Equities:\s*No Double=([\+\-\d.]+).\s*Double=([\+\-\d.]+)./ : isGerman ? /Equities ohne Dopplerwürfel\s*:\s*Nicht Doppeln=([\+\-\d.]+).\s*Doppeln=([\+\-\d.]+)/ : /Cubeless Equities:\s*No Double=([\+\-\d.]+).\s*Double=([\+\-\d.]+)/));

            const cubefulNoDoubleMatch = normalizedContent.match(new RegExp(isFrench ? /Pas de double\s*:\s*([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : isJapanese ? /ノーダブル\s*:\s*([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : isGerman ? /Nicht Doppeln\s*:\s*([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /No double\s*:\s*([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));
            const cubefulDoubleTakeMatch = normalizedContent.match(new RegExp(isFrench ? /Double\/Prend:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : isJapanese ? /ダブル\/テイク:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : isGerman ? /Doppeln\/Annehmen:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /Double\/Take:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));
            const cubefulDoublePassMatch = normalizedContent.match(new RegExp(isFrench ? /Double\/Passe:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : isJapanese ? /ダブル\/パス:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : isGerman ? /Doppeln\/Ablehnen:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /Double\/Pass:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));
            const redoubleNoMatch = normalizedContent.match(new RegExp(isFrench ? /Pas de redouble\s*:\s*([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : isJapanese ? /ノーリダブル\s*:\s*([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : isGerman ? /Nicht Redoppeln\s*:\s*([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /No redouble\s*:\s*([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));
            const redoubleTakeMatch = normalizedContent.match(new RegExp(isFrench ? /Redouble\/Prend:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : isJapanese ? /リダブル\/テイク:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : isGerman ? /Redoppeln\/Annehmen:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /Redouble\/Take:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));
            const redoublePassMatch = normalizedContent.match(new RegExp(isFrench ? /Redouble\/Passe:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : isJapanese ? /リダブル\/パス:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : isGerman ? /Redoppeln\/Ablehnen:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /Redouble\/Pass:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));

            const cubefulDoubleBeaverMatch = normalizedContent.match(new RegExp(isFrench ? /Double\/Beaver:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : isJapanese ? /ダブル\/ビーバー:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : isGerman ? /Doppeln\/Beaver:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/ : /Double\/Beaver:\s+([\+\-\d.]+)(?: \(([\+\-\d.]+)\))?/));

            const bestCubeActionMatch = normalizedContent.match(new RegExp(isFrench ? /Meilleur action du videau:\s*(.*)/ : isJapanese ? /ベストキューブアクション：\s*(.*)/ : isGerman ? /Beste Dopplerwürfel Aktion\s*(.*)/ : /Best Cube action:\s*(.*)/));

            const wrongPassPercentageMatch = normalizedContent.match(new RegExp(isFrench ? /Pourcentage de passes incorrectes pour rendre la décision de double correcte:\s*(\d+\.\d+)%/ : isJapanese ? /ダブルを正当化するのに必要な相手がパスする確率:\s*(\d+\.\d+)%/ : isGerman ? /Prozent von falschen Ablehnen gebraucht damit Doppelentscheidung richtig wäre.:\s*(\d+\.\d+)%/ : /Percentage of wrong pass needed to make the double decision right:\s*(\d+\.\d+)%/));
            const wrongTakePercentageMatch = normalizedContent.match(new RegExp(isFrench ? /Pourcentage de prises incorrectes pour rendre la décision de double correcte:\s*(\d+\.\d+)%/ : isJapanese ? /ダブルを正当化するのに必要な相手がテイクする確率:\s*(\d+\.\d+)%/ : isGerman ? /Prozent von falschen Annehmen gebraucht damit Doppelentscheidung richtig wäre.:\s*(\d+\.\d+)%/ : /Percentage of wrong take needed to make the double decision right:\s*(\d+\.\d+)%/));

            parsedAnalysis.doublingCubeAnalysis.analysisEngine = engineName;
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
            if (cubefulDoubleBeaverMatch) {
                parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeEquity = parseFloat(cubefulDoubleBeaverMatch[1]);
                parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeError = cubefulDoubleBeaverMatch[2] ? parseFloat(cubefulDoubleBeaverMatch[2]) : 0;
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

            if (playerOnRoll === 1) {
                // Swap win, gammon, and backgammon chances between player and opponent
                const tempWinChances = parsedAnalysis.doublingCubeAnalysis.playerWinChances;
                const tempGammonChances = parsedAnalysis.doublingCubeAnalysis.playerGammonChances;
                const tempBackgammonChances = parsedAnalysis.doublingCubeAnalysis.playerBackgammonChances;

                parsedAnalysis.doublingCubeAnalysis.playerWinChances = parsedAnalysis.doublingCubeAnalysis.opponentWinChances;
                parsedAnalysis.doublingCubeAnalysis.playerGammonChances = parsedAnalysis.doublingCubeAnalysis.opponentGammonChances;
                parsedAnalysis.doublingCubeAnalysis.playerBackgammonChances = parsedAnalysis.doublingCubeAnalysis.opponentBackgammonChances;

                parsedAnalysis.doublingCubeAnalysis.opponentWinChances = tempWinChances;
                parsedAnalysis.doublingCubeAnalysis.opponentGammonChances = tempGammonChances;
                parsedAnalysis.doublingCubeAnalysis.opponentBackgammonChances = tempBackgammonChances;
            }
        }

        // Extract comment section
        const commentSection = extractCommentSection(normalizedContent, parsedAnalysis.analysisType === "DoublingCube");
        parsedAnalysis.comment = commentSection;

        return { positionData, parsedAnalysis };
    }

    function extractCommentSection(content, isDoublingCube) {
        if (isDoublingCube) {
            const commentRegex = /(?:Best Cube action: .+|Meilleur action du videau: .+|Percentage of wrong .+|Pourcentage de passes incorrectes .+%)\n\n([\s\S]+?)\n\neXtreme Gammon Version:/;

            let match = commentRegex.exec(content);
            return match ? match[1].trim() : '';
        } else {
            const lines = content.split('\n');
            let lastOpponentIndex = -1;

            // Find the last line where "Opponent" appears
            for (let i = lines.length - 1; i >= 0; i--) {
                if (lines[i].includes('Opponent') || lines[i].includes('Adversaire')) {
                    lastOpponentIndex = i;
                    break;
                }
            }

            if (lastOpponentIndex === -1) {
                return '';
            }

            // Count 2 blank lines after the last "Opponent" line
            let blankLineCount = 0;
            let commentStartIndex = -1;
            for (let i = lastOpponentIndex + 1; i < lines.length; i++) {
                if (lines[i].trim() === '') {
                    blankLineCount++;
                } else {
                    blankLineCount = 0;
                }

                if (blankLineCount === 2) {
                    commentStartIndex = i + 1;
                    break;
                }
            }

            if (commentStartIndex === -1) {
                return '';
            }

            // Extract the comment section until the next blank line before the engine version
            let commentEndIndex = -1;
            for (let i = commentStartIndex; i < lines.length; i++) {
                if (lines[i].trim() === '' && lines[i + 1] && lines[i + 1].startsWith('eXtreme Gammon Version:')) {
                    commentEndIndex = i;
                    break;
                }
            }

            if (commentEndIndex === -1) {
                return '';
            }

            const commentLines = lines.slice(commentStartIndex, commentEndIndex);
            return commentLines.join('\n').trim();
        }
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
        if ($statusBarModeStore !== 'NORMAL' && !($statusBarModeStore === 'COMMAND' && $previousModeStore === 'NORMAL')) {
            setStatusBarMessage('Cannot copy position in current mode');
            return;
        }
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        // @ts-ignore
        if ($statusBarModeStore === 'EDIT') {
            setStatusBarMessage('Cannot copy position in edit mode');
            return;
        }
        console.log('copyPosition');
        const position = $positionStore;
        const analysis = $analysisStore;
        const comment = $commentTextStore;

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
            clipboardContent += `Analysis Depth: "${analysis.doublingCubeAnalysis.analysisDepth}"\n`;
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
            
            // Add comment section if present (with proper formatting for doubling cube parsing)
            if (comment && comment.trim() !== '') {
                clipboardContent += `\n${comment}\n\n`;
            }
        } else if (analysis.analysisType === "CheckerMove") {
            clipboardContent += `Checker Move Analysis:\n`;
            analysis.checkerAnalysis.moves.forEach(move => {
                clipboardContent += `Move ${move.index}: ${move.move}\n`;
                clipboardContent += `Analysis Depth: "${move.analysisDepth}"\n`;
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
            
            // Add comment section if present (with proper formatting for checker move parsing)
            if (comment && comment.trim() !== '') {
                // Add two blank lines before comment to match expected format
                clipboardContent += `\n${comment}\n\n`;
            }
        }

        // Add engine version
        if (analysis.analysisEngineVersion) {
            clipboardContent += `eXtreme Gammon Version: ${analysis.analysisEngineVersion}\n`;
        }

        // Copy to clipboard
        navigator.clipboard.writeText(clipboardContent).then(() => {
            console.log('Position, analysis, and comment copied to clipboard');
            setStatusBarMessage('Position, analysis, and comment copied to clipboard');
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
        if ($statusBarModeStore !== 'NORMAL' && !($statusBarModeStore === 'COMMAND' && $previousModeStore === 'NORMAL')) {
            setStatusBarMessage('Cannot delete position in current mode');
            return;
        }
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
        } finally {
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
        }
    }

    async function updatePosition() {
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        if ($statusBarModeStore !== 'EDIT' && !($statusBarModeStore === 'COMMAND' && $previousModeStore === 'EDIT')) {
            setStatusBarMessage('Update is only possible in edit mode');
            return;
        }
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        console.log('updatePosition');

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
            analysis.checkerAnalysis = { moves: [] };
            analysis.doublingCubeAnalysis = {
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
            };
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
            // @ts-ignore
            await UpdatePosition(position);
            console.log('Position updated with ID:', positionID);

            // Update the analysis in the database
            // @ts-ignore
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
        } finally {
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
        }
    }

    async function firstPosition() {
        if ($statusBarModeStore === 'EDIT') {
            setStatusBarMessage('Cannot browse positions in edit mode');
            return;
        }
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        
        // In MATCH mode, go to previous game (or first position of first game)
        if ($statusBarModeStore === 'MATCH' && $matchContextStore.isMatchMode) {
            const matchCtx = $matchContextStore;
            const currentGameNumber = matchCtx.movePositions[matchCtx.currentIndex].game_number;
            
            // Find the first position of the previous game
            let targetIndex = -1;
            for (let i = matchCtx.currentIndex - 1; i >= 0; i--) {
                if (matchCtx.movePositions[i].game_number < currentGameNumber) {
                    targetIndex = i;
                    break;
                }
            }
            
            // If no previous game found, go to the first position of the current/first game
            if (targetIndex === -1) {
                targetIndex = 0;
            } else {
                // Find the first position of that previous game
                const targetGameNumber = matchCtx.movePositions[targetIndex].game_number;
                for (let i = 0; i < matchCtx.movePositions.length; i++) {
                    if (matchCtx.movePositions[i].game_number === targetGameNumber) {
                        targetIndex = i;
                        break;
                    }
                }
            }
            
            matchContextStore.update(ctx => ({ ...ctx, currentIndex: targetIndex }));
            const movePos = matchCtx.movePositions[targetIndex];
            // Load position and analysis
            await showPosition(movePos.position);
            statusBarTextStore.set(`${matchCtx.player1Name} vs ${matchCtx.player2Name}`);
            saveCurrentMatchPosition(); // Save for later restoration
        } else if (positions && positions.length > 0) {
            currentPositionIndexStore.set(0);
        }
    }

    // Helper function to save current match position for later restoration
    function saveCurrentMatchPosition() {
        if ($statusBarModeStore === 'MATCH' && $matchContextStore.isMatchMode) {
            const matchCtx = $matchContextStore;
            const currentMovePos = matchCtx.movePositions[matchCtx.currentIndex];
            if (currentMovePos) {
                lastVisitedMatchStore.set({
                    matchID: matchCtx.matchID,
                    currentIndex: matchCtx.currentIndex,
                    gameNumber: currentMovePos.game_number
                });
                // Persist to database
                SaveLastVisitedPosition(matchCtx.matchID, matchCtx.currentIndex).catch(e => {
                    console.error('Error persisting last visited position:', e);
                });
            }
        }
    }

    async function previousPosition() {
        if ($statusBarModeStore === 'EDIT') {
            setStatusBarMessage('Cannot browse positions in edit mode');
            return;
        }
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        
        // Check if we're in MATCH mode
        if ($statusBarModeStore === 'MATCH' && $matchContextStore.isMatchMode) {
            const matchCtx = $matchContextStore;
            if (matchCtx.currentIndex > 0) {
                // Navigate to previous position, stopping at checker and cube moves
                let newIndex = matchCtx.currentIndex - 1;
                while (newIndex >= 0) {
                    const movePos = matchCtx.movePositions[newIndex];
                    // Stop at checker moves and cube decisions (Double, Take)
                    if (movePos.move_type === 'checker' || movePos.move_type === 'cube') {
                        break;
                    }
                    newIndex--;
                }
                
                if (newIndex >= 0) {
                    matchContextStore.update(ctx => ({ ...ctx, currentIndex: newIndex }));
                    const movePos = matchCtx.movePositions[newIndex];
                    // Load position and analysis
                    await showPosition(movePos.position);
                    statusBarTextStore.set(`${matchCtx.player1Name} vs ${matchCtx.player2Name}`);
                    saveCurrentMatchPosition(); // Save for later restoration
                }
            }
        } else if (positions && $currentPositionIndexStore > 0) {
            currentPositionIndexStore.set($currentPositionIndexStore - 1);
        }
    }

    async function nextPosition() {
        if ($statusBarModeStore === 'EDIT') {
            setStatusBarMessage('Cannot browse positions in edit mode');
            return;
        }
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        
        // Check if we're in MATCH mode
        if ($statusBarModeStore === 'MATCH' && $matchContextStore.isMatchMode) {
            const matchCtx = $matchContextStore;
            if (matchCtx.currentIndex < matchCtx.movePositions.length - 1) {
                // Navigate to next position, stopping at checker and cube moves
                let newIndex = matchCtx.currentIndex + 1;
                while (newIndex < matchCtx.movePositions.length) {
                    const movePos = matchCtx.movePositions[newIndex];
                    // Stop at checker moves and cube decisions (Double, Take)
                    if (movePos.move_type === 'checker' || movePos.move_type === 'cube') {
                        break;
                    }
                    newIndex++;
                }
                
                if (newIndex < matchCtx.movePositions.length) {
                    matchContextStore.update(ctx => ({ ...ctx, currentIndex: newIndex }));
                    const movePos = matchCtx.movePositions[newIndex];
                    // Load position and analysis
                    await showPosition(movePos.position);
                    statusBarTextStore.set(`${matchCtx.player1Name} vs ${matchCtx.player2Name}`);
                    saveCurrentMatchPosition(); // Save for later restoration
                }
            }
        } else if (positions && $currentPositionIndexStore < positions.length - 1) {
            currentPositionIndexStore.set($currentPositionIndexStore + 1);
        }
    }

    async function lastPosition() {
        if ($statusBarModeStore === 'EDIT') {
            setStatusBarMessage('Cannot browse positions in edit mode');
            return;
        }
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        
        // In MATCH mode, go to next game (or first position of last game)
        if ($statusBarModeStore === 'MATCH' && $matchContextStore.isMatchMode) {
            const matchCtx = $matchContextStore;
            const currentGameNumber = matchCtx.movePositions[matchCtx.currentIndex].game_number;
            
            // Find the first position of the next game
            let targetIndex = -1;
            for (let i = matchCtx.currentIndex + 1; i < matchCtx.movePositions.length; i++) {
                if (matchCtx.movePositions[i].game_number > currentGameNumber) {
                    targetIndex = i;
                    break;
                }
            }
            
            // If no next game found, find the first position of the last game
            if (targetIndex === -1) {
                const maxGameNumber = Math.max(...matchCtx.movePositions.map(p => p.game_number));
                for (let i = 0; i < matchCtx.movePositions.length; i++) {
                    if (matchCtx.movePositions[i].game_number === maxGameNumber) {
                        targetIndex = i;
                        break;
                    }
                }
            }
            
            if (targetIndex !== -1) {
                matchContextStore.update(ctx => ({ ...ctx, currentIndex: targetIndex }));
                const movePos = matchCtx.movePositions[targetIndex];
                // Load position and analysis
                await showPosition(movePos.position);
                statusBarTextStore.set(`${matchCtx.player1Name} vs ${matchCtx.player2Name}`);
                saveCurrentMatchPosition(); // Save for later restoration
            }
        } else if (positions && positions.length > 0) {
            currentPositionIndexStore.set(positions.length - 1);
        }
    }

    function gotoPosition() {
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        if ($statusBarModeStore !== 'NORMAL' && $statusBarModeStore !== 'MATCH') {
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

    async function toggleEditMode() {
        console.log('toggleEditMode');
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            statusBarModeStore.set('NORMAL');
            return;
        }
        
        // Special handling for MATCH mode: exit to NORMAL and reload all positions
        if ($statusBarModeStore === "MATCH") {
            console.log('Exiting MATCH mode to NORMAL mode');
            // Save current position to DB before leaving
            if ($matchContextStore.isMatchMode && $matchContextStore.matchID) {
                try {
                    await SaveLastVisitedPosition($matchContextStore.matchID, $matchContextStore.currentIndex);
                } catch (e) {
                    console.error('Error saving last visited position:', e);
                }
            }
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
            // Reset match context
            matchContextStore.set({
                isMatchMode: false,
                matchID: null,
                movePositions: [],
                currentIndex: 0,
                player1Name: '',
                player2Name: ''
            });
            // Reload all positions from database
            loadAllPositions();
            return;
        }

        // Special handling for COLLECTION mode: exit to NORMAL and reload all positions
        if ($statusBarModeStore === "COLLECTION") {
            await exitCollectionMode();
            return;
        }

        // Special handling for EPC mode: exit to NORMAL and restore saved state
        if ($statusBarModeStore === "EPC") {
            toggleEPCMode(); // toggleEPCMode handles exit when already in EPC mode
            return;
        }
        
        if ($statusBarModeStore !== "EDIT") {
            previousModeStore.set($statusBarModeStore);
            statusBarModeStore.set('EDIT');
            showCommentStore.set(false);
            showAnalysisStore.set(false);
        } else {
            previousModeStore.set($statusBarModeStore);
            statusBarModeStore.set('NORMAL');
            // Refresh board and display position associated with currentPositionIndexStore
            const currentIndex = $currentPositionIndexStore;
            currentPositionIndexStore.set(-1); // Temporarily set to a different value to force redraw
            currentPositionIndexStore.set(currentIndex); // Set back to the original value
        }
    }

    function toggleEPCMode() {
        console.log('toggleEPCMode');

        if ($statusBarModeStore === 'EPC') {
            // Exit EPC mode: restore previous state
            statusBarModeStore.set('NORMAL');
            previousModeStore.set('NORMAL');
            statusBarTextStore.set('');
            if (savedPositionsBeforeEPC) {
                positionsStore.set(savedPositionsBeforeEPC);
                if (savedPositionBeforeEPC) {
                    positionStore.set(savedPositionBeforeEPC);
                    currentPositionIndexStore.set(savedPositionIndexBeforeEPC);
                }
                savedPositionsBeforeEPC = null;
                savedPositionBeforeEPC = null;
                savedPositionIndexBeforeEPC = -1;
            } else {
                // Reload all positions
                loadAllPositions();
            }
            return;
        }

        // Save current state before entering EPC mode
        savedPositionBeforeEPC = $positionStore ? { ...$positionStore } : null;
        savedPositionIndexBeforeEPC = $currentPositionIndexStore;
        savedPositionsBeforeEPC = $positionsStore ? [...$positionsStore] : null;

        // Create EPC initial position: 6-point closed jan with 3 extra checkers on 4, 5, 6
        // Points 1-6 for bottom (White): 2 on each + 1 extra on 4, 5, 6 = [2, 2, 2, 3, 3, 3]
        const epcPoints = Array(26).fill({ checkers: 0, color: -1 });
        epcPoints[1] = { checkers: 2, color: 0 }; // point 1: 2 black
        epcPoints[2] = { checkers: 2, color: 0 }; // point 2: 2 black
        epcPoints[3] = { checkers: 2, color: 0 }; // point 3: 2 black
        epcPoints[4] = { checkers: 3, color: 0 }; // point 4: 3 black
        epcPoints[5] = { checkers: 3, color: 0 }; // point 5: 3 black
        epcPoints[6] = { checkers: 3, color: 0 }; // point 6: 3 black

        const epcPosition = {
            id: 0,
            board: {
                points: epcPoints,
                bearoff: [0, 15], // 0 black off, 15 white off (only bottom side matters)
            },
            cube: { owner: -1, value: 0 },
            dice: [0, 0],
            score: [-1, -1],
            player_on_roll: 0,
            decision_type: 0,
            has_jacoby: 0,
            has_beaver: 0,
        };

        previousModeStore.set($statusBarModeStore);
        statusBarModeStore.set('EPC');
        showCommentStore.set(false);
        showAnalysisStore.set(false);

        positionsStore.set([epcPosition]);
        positionStore.set(epcPosition);
        currentPositionIndexStore.set(0);
    }

    async function updateEPC(position) {
        try {
            const result = await ComputeEPCFromPosition(position);
            if (result && result.bottomEPC) {
                const epc = result.bottomEPC;
                statusBarTextStore.set(`EPC: ${epc.epc.toFixed(2)} | Pips: ${epc.pipCount} | Wastage: ${epc.wastage.toFixed(2)} | Avg rolls: ${epc.meanRolls.toFixed(3)}`);
            } else {
                statusBarTextStore.set('EPC: N/A (checkers not all in home board)');
            }
        } catch (error) {
            console.error('Error computing EPC:', error);
            statusBarTextStore.set('EPC: Error computing');
        }
    }

    // Toggle MATCH mode: enter match mode showing the last visited match/position
    // or exit match mode back to NORMAL.
    async function toggleMatchMode() {
        console.log('toggleMatchMode');
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }

        // If already in MATCH mode, exit to NORMAL
        if ($statusBarModeStore === 'MATCH') {
            console.log('Exiting MATCH mode to NORMAL mode via toggleMatchMode');
            // Save current position to DB before leaving
            if ($matchContextStore.isMatchMode && $matchContextStore.matchID) {
                try {
                    await SaveLastVisitedPosition($matchContextStore.matchID, $matchContextStore.currentIndex);
                } catch (e) {
                    console.error('Error saving last visited position:', e);
                }
            }
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
            matchContextStore.set({
                isMatchMode: false,
                matchID: null,
                movePositions: [],
                currentIndex: 0,
                player1Name: '',
                player2Name: ''
            });
            loadAllPositions();
            return;
        }

        // Exit other modes first
        if ($statusBarModeStore === 'EDIT' || $statusBarModeStore === 'EPC' || $statusBarModeStore === 'COLLECTION') {
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
        }

        // Enter MATCH mode: get the last visited match from DB
        try {
            const match = await GetLastVisitedMatch();
            if (!match) {
                setStatusBarMessage('No matches in database');
                return;
            }

            const movePositions = await GetMatchMovePositions(match.id);
            if (!movePositions || movePositions.length === 0) {
                setStatusBarMessage('No moves found in this match');
                return;
            }

            // Determine start index: use last_visited_position from DB if valid, else 0
            let startIndex = 0;
            if (match.last_visited_position >= 0 && match.last_visited_position < movePositions.length) {
                startIndex = match.last_visited_position;
            }

            matchContextStore.set({
                isMatchMode: true,
                matchID: match.id,
                movePositions: movePositions,
                currentIndex: startIndex,
                player1Name: match.player1_name,
                player2Name: match.player2_name,
            });

            const startMovePos = movePositions[startIndex];
            positionStore.set(startMovePos.position);

            let analysis = null;
            try {
                analysis = await LoadAnalysis(startMovePos.position.id);
            } catch (error) {}

            const currentPlayedMove = startMovePos.checker_move || '';
            const currentPlayedCubeAction = startMovePos.cube_action || '';

            analysisStore.set({
                positionId: analysis?.positionId || null,
                xgid: analysis?.xgid || '',
                player1: analysis?.player1 || '',
                player2: analysis?.player2 || '',
                analysisType: analysis?.analysisType || '',
                analysisEngineVersion: analysis?.analysisEngineVersion || '',
                checkerAnalysis: analysis?.checkerAnalysis || { moves: [] },
                doublingCubeAnalysis: analysis?.doublingCubeAnalysis || {
                    analysisDepth: '', playerWinChances: 0, playerGammonChances: 0,
                    playerBackgammonChances: 0, opponentWinChances: 0, opponentGammonChances: 0,
                    opponentBackgammonChances: 0, cubelessNoDoubleEquity: 0, cubelessDoubleEquity: 0,
                    cubefulNoDoubleEquity: 0, cubefulNoDoubleError: 0, cubefulDoubleTakeEquity: 0,
                    cubefulDoubleTakeError: 0, cubefulDoublePassEquity: 0, cubefulDoublePassError: 0,
                    bestCubeAction: '', wrongPassPercentage: 0, wrongTakePercentage: 0
                },
                allCubeAnalyses: analysis?.allCubeAnalyses || [],
                playedMove: currentPlayedMove,
                playedCubeAction: currentPlayedCubeAction,
                playedMoves: analysis?.playedMoves || [],
                playedCubeActions: analysis?.playedCubeActions || [],
                creationDate: analysis?.creationDate || '',
                lastModifiedDate: analysis?.lastModifiedDate || ''
            });

            commentTextStore.set('');
            selectedMoveStore.set(null);
            statusBarModeStore.set('MATCH');
            statusBarTextStore.set(`${match.player1_name} vs ${match.player2_name}`);

            lastVisitedMatchStore.set({
                matchID: match.id,
                currentIndex: startIndex,
                gameNumber: startMovePos.game_number
            });

        } catch (error) {
            console.error('Error entering match mode:', error);
            const errMsg = error?.toString() || '';
            if (errMsg.includes('no matches')) {
                setStatusBarMessage('No matches in database');
            } else {
                setStatusBarMessage('Error entering match mode');
            }
        }
    }

    function toggleAnalysisPanel() {
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        console.log('toggleAnalysisPanel');
        const wasOpen = showAnalysis;
        const inCollectionOrMatch = $statusBarModeStore === 'MATCH' || $statusBarModeStore === 'COLLECTION';
        showAnalysisStore.set(!wasOpen);
        
        if (!wasOpen) {
            // Panel is now opening - close other panels
            if (!inCollectionOrMatch) {
                statusBarModeStore.set('NORMAL');
            }
            // Don't close collection panel when in COLLECTION mode
            if ($statusBarModeStore !== 'COLLECTION') {
                showCollectionPanelStore.set(false);
            }
            showFilterLibraryPanelStore.set(false);
            showCommentStore.set(false);
            showSearchHistoryPanelStore.set(false);
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

        // Don't change mode to NORMAL if we're in MATCH or COLLECTION mode
        if (!inCollectionOrMatch) {
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
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
        const wasOpen = showComment;
        const inCollectionOrMatch = $statusBarModeStore === 'MATCH' || $statusBarModeStore === 'COLLECTION';
        showCommentStore.set(!wasOpen);

        if (!wasOpen) {
            // Panel is now opening - close other panels
            // Don't close collection panel when in COLLECTION mode
            if ($statusBarModeStore !== 'COLLECTION') {
                showCollectionPanelStore.set(false);
            }
            if (!inCollectionOrMatch) {
                statusBarModeStore.set('NORMAL');
            }
            showAnalysisStore.set(false);
            showFilterLibraryPanelStore.set(false);
            showSearchHistoryPanelStore.set(false);
            showCommandStore.set(false);
            const currentIndex = $currentPositionIndexStore;
            currentPositionIndexStore.set(-1); // Temporarily set to a different value to force redraw
            currentPositionIndexStore.set(currentIndex); // Set back to the original value
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
            SaveComment(parseInt(positions[currentPositionIndex].id), $commentTextStore);
            mainArea.scrollIntoView({
                behavior: 'smooth'
            });
        }

        // Don't change mode to NORMAL if we're in MATCH or COLLECTION mode
        if (!inCollectionOrMatch) {
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
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

    function toggleFilterLibraryPanel() {
        console.log('toggleFilterLibraryPanel');
        if (!databaseLoaded) {
            statusBarTextStore.set('No database loaded');
            return;
        }
        const wasOpen = showFilterLibraryPanel;
        showFilterLibraryPanelStore.set(!wasOpen);
        if (!wasOpen) {
            // Opening filter library exits COLLECTION mode
            if ($statusBarModeStore === 'COLLECTION') {
                exitCollectionMode();
            }
            // Panel is now opening - close other panels
            statusBarModeStore.set('NORMAL');
            showCommentStore.set(false);
            showAnalysisStore.set(false);
            showSearchHistoryPanelStore.set(false);
            showMatchPanelStore.set(false);
            showCollectionPanelStore.set(false);
            // Refresh board and display position associated with currentPositionIndexStore
            const currentIndex = $currentPositionIndexStore;
            currentPositionIndexStore.set(-1); // Temporarily set to a different value to force redraw
            currentPositionIndexStore.set(currentIndex); // Set back to the original value
        }

        previousModeStore.set('NORMAL');
        statusBarModeStore.set('NORMAL');
    }

    function toggleMatchPanel() {
        console.log('toggleMatchPanel');
        if (!databaseLoaded) {
            statusBarTextStore.set('No database loaded');
            return;
        }
        const wasOpen = showMatchPanel;
        showMatchPanelStore.set(!wasOpen);
        if (!wasOpen) {
            // Opening match panel exits COLLECTION mode
            if ($statusBarModeStore === 'COLLECTION') {
                exitCollectionMode();
            }
            // Panel is now opening - close other panels
            // Don't change mode if we're in MATCH mode
            if ($statusBarModeStore !== 'MATCH') {
                statusBarModeStore.set('NORMAL');
                previousModeStore.set('NORMAL');
            }
            showCommentStore.set(false);
            showAnalysisStore.set(false);
            showSearchHistoryPanelStore.set(false);
            showFilterLibraryPanelStore.set(false);
            showCollectionPanelStore.set(false);
            showTournamentPanelStore.set(false);
            // Refresh board and display position associated with currentPositionIndexStore
            const currentIndex = $currentPositionIndexStore;
            currentPositionIndexStore.set(-1); // Temporarily set to a different value to force redraw
            currentPositionIndexStore.set(currentIndex); // Set back to the original value
        }
    }

    function toggleCollectionPanelAction() {
        console.log('toggleCollectionPanelAction');
        if (!databaseLoaded) {
            statusBarTextStore.set('No database loaded');
            return;
        }
        const wasOpen = showCollectionPanel;
        showCollectionPanelStore.set(!wasOpen);
        if (!wasOpen) {
            // Panel is now opening - close other panels
            if ($statusBarModeStore !== 'COLLECTION') {
                statusBarModeStore.set('NORMAL');
                previousModeStore.set('NORMAL');
            }
            showCommentStore.set(false);
            showAnalysisStore.set(false);
            showSearchHistoryPanelStore.set(false);
            showFilterLibraryPanelStore.set(false);
            showMatchPanelStore.set(false);
            showTournamentPanelStore.set(false);
            // Refresh board and display position associated with currentPositionIndexStore
            const currentIndex = $currentPositionIndexStore;
            currentPositionIndexStore.set(-1);
            currentPositionIndexStore.set(currentIndex);
        }
    }

    function toggleTournamentPanel() {
        console.log('toggleTournamentPanel');
        if (!databaseLoaded) {
            statusBarTextStore.set('No database loaded');
            return;
        }
        const wasOpen = showTournamentPanel;
        showTournamentPanelStore.set(!wasOpen);
        if (!wasOpen) {
            // Opening tournament panel exits COLLECTION mode
            if ($statusBarModeStore === 'COLLECTION') {
                exitCollectionMode();
            }
            // Panel is now opening - close other panels
            statusBarModeStore.set('NORMAL');
            previousModeStore.set('NORMAL');
            showCommentStore.set(false);
            showAnalysisStore.set(false);
            showSearchHistoryPanelStore.set(false);
            showFilterLibraryPanelStore.set(false);
            showMatchPanelStore.set(false);
            showCollectionPanelStore.set(false);
            // Refresh board and display position associated with currentPositionIndexStore
            const currentIndex = $currentPositionIndexStore;
            currentPositionIndexStore.set(-1);
            currentPositionIndexStore.set(currentIndex);
        }
    }

    // Exit COLLECTION mode: reload all positions and navigate to last viewed position
    async function exitCollectionMode() {
        console.log('Exiting COLLECTION mode to NORMAL mode');
        // Capture the last viewed position in collection mode before clearing stores
        const lastViewedPosition = $positionStore;
        previousModeStore.set('NORMAL');
        statusBarModeStore.set('NORMAL');
        activeCollectionStore.set(null);
        selectedCollectionStore.set(null);
        collectionPositionsStore.set([]);
        showCollectionPanelStore.set(false);
        // Reload all positions and navigate to the last viewed position from collection mode
        savedPositionBeforeCollection = null;
        savedPositionIndexBeforeCollection = -1;
        savedPositionsBeforeCollection = null;
        try {
            const allPositions = await LoadAllPositions();
            positionsStore.set(Array.isArray(allPositions) ? allPositions : []);
            if (allPositions && allPositions.length > 0) {
                // Find the last viewed position by ID in the full list
                let targetIdx = allPositions.length - 1; // default to last
                if (lastViewedPosition && lastViewedPosition.id) {
                    const foundIdx = allPositions.findIndex(p => p.id === lastViewedPosition.id);
                    if (foundIdx >= 0) {
                        targetIdx = foundIdx;
                    }
                }
                currentPositionIndexStore.set(-1);
                currentPositionIndexStore.set(targetIdx);
                loadAnalysisForPosition(allPositions[targetIdx]);
                hasActiveSearch = false;
                lastSearchCommand = '';
                lastSearchPosition = null;
            }
        } catch (error) {
            console.error('Error reloading positions after collection exit:', error);
            loadAllPositions();
        }
    }

    // Called when double-clicking a collection - switch to COLLECTION mode
    function handleOpenCollection(collection, collectionPositions) {
        if (!collectionPositions || collectionPositions.length === 0) {
            statusBarTextStore.set('Collection is empty');
            return;
        }

        // Save current position/index before entering COLLECTION mode
        savedPositionBeforeCollection = $positionStore;
        savedPositionIndexBeforeCollection = $currentPositionIndexStore;
        savedPositionsBeforeCollection = positions;

        // Exit match mode if active
        if ($matchContextStore.isMatchMode) {
            matchContextStore.update(ctx => ({
                ...ctx,
                isMatchMode: false,
                matchID: null,
                movePositions: [],
                currentIndex: 0
            }));
        }

        // Switch to COLLECTION mode
        previousModeStore.set('NORMAL');
        statusBarModeStore.set('COLLECTION');

        // Load positions in main view
        positionsStore.set(collectionPositions);
        positionStore.set(collectionPositions[0]);
        currentPositionIndexStore.set(0);
        
        // Load analysis for the first position
        loadAnalysisForPosition(collectionPositions[0]);
        
        statusBarTextStore.set(`Collection "${collection.name}" — ${collectionPositions.length} position(s)`);
    }

function togglePipcount() {
        console.log('togglePipcount');
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            statusBarModeStore.set('NORMAL');
            return;
        }
        if ($statusBarModeStore == "NORMAL") {
            showPipcountStore.set(!showPipcount);
            // Refresh board and display position associated with currentPositionIndexStore
            const currentIndex = $currentPositionIndexStore;
            currentPositionIndexStore.set(-1); // Temporarily set to a different value to force redraw
            currentPositionIndexStore.set(currentIndex); // Set back to the original value
        } else if ($statusBarModeStore == "MATCH") {
            showPipcountStore.set(!showPipcount);
            // Refresh board by triggering a positionStore update
            const currentPosition = $positionStore;
            positionStore.set({ ...currentPosition });
        }
    }

    function loadRandomPosition() {
        console.log('loadRandomPosition');
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            statusBarModeStore.set('NORMAL');
            return;
        }
        if ($statusBarModeStore == "NORMAL") {
            // Load a random position from the position store
            let randomIndex = Math.floor(Math.random() * positions.length);
            // ensure that the random index is not the same as the current index
            while (randomIndex === $currentPositionIndexStore) {
                randomIndex = Math.floor(Math.random() * positions.length);
            }
            // Set the current position index to the random index
            console.log('Random position index:', randomIndex);
            // Set the current position index to the random index
            currentPositionIndexStore.set(randomIndex);
            
        }
    }

    // Mirror a position (swap players) for search purposes.
    // When the user edits with player 2 on roll, the position must be
    // normalized (player_on_roll = 0) before querying the database.
    function mirrorPositionForSearch(pos) {
        const mirrored = JSON.parse(JSON.stringify(pos)); // Deep copy

        // Mirror the board points
        const tempPoints = [...mirrored.board.points];
        for (let i = 0; i < 26; i++) {
            mirrored.board.points[25 - i] = {
                color: tempPoints[i].color === -1 ? -1 : 1 - tempPoints[i].color,
                checkers: tempPoints[i].checkers
            };
        }

        // Swap bearoff
        [mirrored.board.bearoff[0], mirrored.board.bearoff[1]] =
            [mirrored.board.bearoff[1], mirrored.board.bearoff[0]];

        // Swap player on roll
        mirrored.player_on_roll = 1 - mirrored.player_on_roll;

        // Swap scores
        [mirrored.score[0], mirrored.score[1]] = [mirrored.score[1], mirrored.score[0]];

        // Swap cube owner if owned
        if (mirrored.cube.owner !== -1) {
            mirrored.cube.owner = 1 - mirrored.cube.owner;
        }

        return mirrored;
    }


    async function loadPositionsByFilters(
        filters,
        includeCube,
        includeScore,
        pipCountFilter,
        winRateFilter,
        gammonRateFilter,
        backgammonRateFilter,
        player2WinRateFilter,
        player2GammonRateFilter,
        player2BackgammonRateFilter,
        player1CheckerOffFilter,
        player2CheckerOffFilter,
        player1BackCheckerFilter,
        player2BackCheckerFilter,
        player1CheckerInZoneFilter,
        player2CheckerInZoneFilter,
        searchText,
        player1AbsolutePipCountFilter,
        equityFilter,
        decisionTypeFilter,
        diceRollFilter,
        movePatternFilter,
        dateFilter,
        player1OutfieldBlotFilter,
        player2OutfieldBlotFilter,
        player1JanBlotFilter,
        player2JanBlotFilter,
        noContactFilter,
        mirrorPositionFilter,
        moveErrorFilter,
        searchCommand = '',  // Optional: the original search command for session tracking
    ) {
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened');
            return;
        }
        console.log('loadPositionsByFilters',
        filters,
        includeCube,
        includeScore,
        pipCountFilter,
        winRateFilter,
        gammonRateFilter,
        backgammonRateFilter,
        player2WinRateFilter,
        player2GammonRateFilter,
        player2BackgammonRateFilter,
        player1CheckerOffFilter,
        player2CheckerOffFilter,
        player1BackCheckerFilter,
        player2BackCheckerFilter,
        player1CheckerInZoneFilter,
        player2CheckerInZoneFilter,
        searchText,
        player1AbsolutePipCountFilter,
        equityFilter,
        decisionTypeFilter,
        diceRollFilter,
        movePatternFilter,
        dateFilter,
        player1OutfieldBlotFilter,
        player2OutfieldBlotFilter,
        player1JanBlotFilter,
        player2JanBlotFilter,
        noContactFilter,
        mirrorPositionFilter,
        moveErrorFilter);
        try {
            let currentPosition = $positionStore;
            
            // If player 2 is on roll during editing, mirror the position before searching
            // since all stored positions are normalized with player_on_roll = 0
            if (currentPosition.player_on_roll === 1) {
                currentPosition = mirrorPositionForSearch(currentPosition);
            }

            // @ts-ignore
            const loadedPositions = await LoadPositionsByFilters(
                // @ts-ignore
                currentPosition,
                includeCube,
                includeScore,
                pipCountFilter,
                winRateFilter,
                gammonRateFilter,
                backgammonRateFilter,
                player2WinRateFilter,
                player2GammonRateFilter,
                player2BackgammonRateFilter,
                player1CheckerOffFilter,
                player2CheckerOffFilter,
                player1BackCheckerFilter,
                player2BackCheckerFilter,
                player1CheckerInZoneFilter,
                player2CheckerInZoneFilter,
                searchText,
                player1AbsolutePipCountFilter,
                equityFilter,
                decisionTypeFilter,
                diceRollFilter,
                movePatternFilter,
                dateFilter,
                player1OutfieldBlotFilter,
                player2OutfieldBlotFilter,
                player1JanBlotFilter,
                player2JanBlotFilter,
                noContactFilter,
                mirrorPositionFilter,
                moveErrorFilter);
                
            // Set mode to NORMAL and reset match context BEFORE triggering position display
            // so that showPosition sees the correct mode and doesn't use stale match context
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
            matchContextStore.set({
                isMatchMode: false,
                matchID: null,
                movePositions: [],
                currentIndex: 0,
                player1Name: '',
                player2Name: ''
            });

            positionsStore.set(Array.isArray(loadedPositions) ? loadedPositions : []);

            if (loadedPositions && loadedPositions.length > 0) {
                if ($currentPositionIndexStore === 0) {
                    currentPositionIndexStore.set(1); // Temporarily set to a different value to force redraw board
                }
                currentPositionIndexStore.set(0); // Ensure the first matching position is displayed
                
                // Track session state for search results
                hasActiveSearch = true;
                lastSearchCommand = searchCommand || '';
                lastSearchPosition = JSON.parse(JSON.stringify($positionStore));
                
                // Save session state after successful search
                saveSessionState();
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

            // Set mode to NORMAL and reset match context BEFORE triggering position display
            // Save last visited position to DB if exiting MATCH mode
            if ($statusBarModeStore === 'MATCH' && $matchContextStore.isMatchMode && $matchContextStore.matchID) {
                SaveLastVisitedPosition($matchContextStore.matchID, $matchContextStore.currentIndex).catch(e => {
                    console.error('Error persisting last visited position:', e);
                });
            }
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
            matchContextStore.set({
                isMatchMode: false,
                matchID: null,
                movePositions: [],
                currentIndex: 0,
                player1Name: '',
                player2Name: ''
            });
            // Clear EPC mode state if active
            savedPositionsBeforeEPC = null;
            savedPositionBeforeEPC = null;
            savedPositionIndexBeforeEPC = -1;

            positionsStore.set(Array.isArray(positions) ? positions : []);
            if (positions && positions.length > 0) {
                currentPositionIndexStore.set(-1); // Temporarily set to a different value to force redraw board
                currentPositionIndexStore.set(positions.length - 1);
                
                // When loading all positions, clear the active search state
                hasActiveSearch = false;
                lastSearchCommand = '';
                lastSearchPosition = null;
                saveSessionState();
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

    // Helper function to load analysis for a position
    async function loadAnalysisForPosition(position) {
        if (!position || !position.id) return;
        
        try {
            const analysis = await LoadAnalysis(position.id);
            if (analysis) {
                analysisStore.set(analysis);
            } else {
                analysisStore.set({
                    positionId: position.id,
                    xgid: '',
                    player1: '',
                    player2: '',
                    analysisType: '',
                    analysisEngineVersion: '',
                    checkerAnalysis: { moves: [] },
                    doublingCubeAnalysis: null,
                    allCubeAnalyses: [],
                    playedMove: '',
                    playedCubeAction: '',
                    playedMoves: [],
                    playedCubeActions: [],
                    creationDate: '',
                    lastModifiedDate: ''
                });
            }
        } catch (error) {
            console.error('Error loading analysis:', error);
        }
    }

    // ── Drag & Drop ────────────────────────────────────────────────

    /**
     * Classifies dropped file paths by extension.
     * Returns { dbFiles: string[], importFiles: string[], folders: string[], unsupported: string[] }
     */
    async function classifyDroppedFiles(paths) {
        const dbFiles = [];
        const importFiles = [];
        const folders = [];
        const unsupported = [];
        for (const p of paths) {
            const isDir = await IsDirectory(p);
            if (isDir) {
                folders.push(p);
            } else {
                const ext = p.toLowerCase().split('.').pop();
                if (ext === 'db') {
                    dbFiles.push(p);
                } else if (['txt', 'xg', 'sgf', 'mat', 'bgf'].includes(ext)) {
                    importFiles.push(p);
                } else {
                    unsupported.push(p);
                }
            }
        }
        return { dbFiles, importFiles, folders, unsupported };
    }

    /**
     * Handle a .db file drop.
     * - If no database is open: open it directly.
     * - If a database is already open: ask user whether to open or merge.
     */
    async function handleDbFileDrop(dbPath) {
        if (!$databasePathStore) {
            // No database open — open it
            await openDatabaseByPath(dbPath);
        } else {
            // Database already open — ask the user
            const filename = dbPath.split('/').pop().split('\\').pop();
            try {
                const answer = await ShowQuestionDialog(
                    'Database already open',
                    `A database is already open.\n\nWhat would you like to do with "${filename}"?`,
                    ['Open', 'Merge', 'Cancel'],
                    'Merge'
                );
                if (answer === 'Open') {
                    await openDatabaseByPath(dbPath);
                } else if (answer === 'Merge') {
                    await importDatabaseByPath(dbPath);
                }
                // 'Cancel' or dialog closed — do nothing
            } catch (error) {
                console.error('Error in DB drop dialog:', error);
                setStatusBarMessage('Error handling dropped database');
            }
        }
    }

    /**
     * Import a database by path (same logic as importDatabase but without the file dialog).
     */
    async function importDatabaseByPath(importFilePath) {
        if (!$databasePathStore) {
            setStatusBarMessage('No database opened. Please open a database first.');
            return;
        }

        try {
            console.log('Analyzing import from:', importFilePath);

            // Show modal in analyzing mode
            showImportProgressModal = true;
            importModalMode = 'analyzing';
            pendingImportPath = importFilePath;

            try {
                const analysis = await AnalyzeImportDatabase(importFilePath);
                console.log('Import analysis:', analysis);

                importAnalysis = {
                    toAdd: analysis.toAdd,
                    toMerge: analysis.toMerge,
                    toSkip: analysis.toSkip,
                    total: analysis.total,
                    importPath: importFilePath
                };
                importModalMode = 'preview';
            } catch (analyzeError) {
                showImportProgressModal = false;
                throw analyzeError;
            }
        } catch (error) {
            console.error('Error analyzing import:', error);
            setStatusBarMessage(`Error analyzing import: ${error}`);
            await ShowAlert(`Error analyzing import: ${error}`);
            previousModeStore.set('NORMAL');
            statusBarModeStore.set('NORMAL');
        }
    }

    /**
     * Main handler called by Wails when files are dropped onto the window.
     */
    async function handleFileDrop(x, y, paths) {
        console.log('Files dropped:', paths);
        showDropOverlay = false;

        if (!paths || paths.length === 0) return;

        const { dbFiles, importFiles, folders, unsupported } = await classifyDroppedFiles(paths);

        if (unsupported.length > 0) {
            const exts = [...new Set(unsupported.map(p => '.' + p.split('.').pop()))].join(', ');
            console.warn('Unsupported file extensions dropped:', exts);
        }

        // Handle .db file(s) — use only the first one
        if (dbFiles.length > 0) {
            await handleDbFileDrop(dbFiles[0]);
            if (importFiles.length === 0 && folders.length === 0) {
                return;
            }
        }

        // Expand folders into importable files
        let allImportFiles = [...importFiles];
        for (const folder of folders) {
            try {
                const folderFiles = await CollectImportableFiles(folder);
                if (folderFiles && folderFiles.length > 0) {
                    allImportFiles = allImportFiles.concat(folderFiles);
                }
            } catch (error) {
                console.error('Error collecting files from folder:', folder, error);
            }
        }

        // Handle importable files
        if (allImportFiles.length > 0) {
            if (!$databasePathStore) {
                setStatusBarMessage('No database opened. Please open a database before importing files.');
                await ShowAlert('No database opened. Please open or drop a database file first.');
                return;
            }

            if (allImportFiles.length === 1) {
                await importSingleFile(allImportFiles[0]);
            } else {
                await importMultipleFiles(allImportFiles);
            }
        }

        // If only unsupported files were dropped (no folders, no db, no import files)
        if (dbFiles.length === 0 && allImportFiles.length === 0 && unsupported.length > 0) {
            const exts = [...new Set(unsupported.map(p => '.' + p.split('.').pop()))].join(', ');
            setStatusBarMessage(`Unsupported file type(s): ${exts}`);
        } else if (folders.length > 0 && allImportFiles.length === 0 && dbFiles.length === 0) {
            setStatusBarMessage('No importable files found in dropped folder(s)');
        }
    }

    // Visual drag-over feedback handlers
    let dragCounter = 0;

    function handleDragOver(e) {
        e.preventDefault();
        if (!showDropOverlay) {
            dragCounter++;
            showDropOverlay = true;
        }
    }

    function handleDragLeave(e) {
        dragCounter--;
        if (dragCounter <= 0) {
            dragCounter = 0;
            showDropOverlay = false;
        }
    }

    function handleDragEnd(e) {
        dragCounter = 0;
        showDropOverlay = false;
    }

    onMount(async () => {
        // @ts-ignore
        console.log('Wails runtime:', runtime);
        window.addEventListener("keydown", handleKeyDown);
        mainArea.addEventListener("wheel", handleWheel); // Add wheel event listener to main container
        window.addEventListener("resize", handleResize);

        // Register drag and drop handler
        OnFileDrop((x, y, paths) => {
            handleFileDrop(x, y, paths);
        }, false);

        // Visual feedback: show/hide drop overlay on native dragover/dragleave
        window.addEventListener('dragover', handleDragOver);
        window.addEventListener('dragleave', handleDragLeave);
        window.addEventListener('drop', handleDragEnd);

        // Auto-reopen last database on startup
        try {
            const lastDbPath = await GetLastDatabasePath();
            if (lastDbPath) {
                console.log('Auto-reopening last database:', lastDbPath);
                await openDatabaseByPath(lastDbPath);
            }
        } catch (error) {
            console.error('Error auto-reopening last database:', error);
            // Clear the stored path so we don't retry a broken path on every launch
            try { await SaveLastDatabasePath(''); } catch (e) { /* ignore */ }
        }
    });

    onDestroy(() => {
        window.removeEventListener("keydown", handleKeyDown);
        mainArea.removeEventListener("wheel", handleWheel); // Remove wheel event listener from main container
        window.removeEventListener("resize", handleResize);
        window.removeEventListener('dragover', handleDragOver);
        window.removeEventListener('dragleave', handleDragLeave);
        window.removeEventListener('drop', handleDragEnd);
        OnFileDropOff();
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
                        // @ts-ignore
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

    function toggleSearchHistoryPanel() {
        console.log('toggleSearchHistoryPanel');
        if (!$databasePathStore) {
            setStatusBarMessage('Search history requires an open database');
            return;
        }
        const wasOpen = $showSearchHistoryPanelStore;
        showSearchHistoryPanelStore.set(!wasOpen);
        if (!wasOpen) {
            // Panel is now opening - close other panels
            statusBarModeStore.set('NORMAL');
            showCommentStore.set(false);
            showAnalysisStore.set(false);
            showFilterLibraryPanelStore.set(false);
        }

        previousModeStore.set('NORMAL');
        statusBarModeStore.set('NORMAL');
    }

    async function addSearchToFilterLibrary(filterName, filterCommand, positionJson) {
        try {
            await SaveFilter(filterName, filterCommand);
            if (positionJson) {
                await SaveEditPosition(filterName, positionJson);
            }
            statusBarTextStore.set('Filter saved successfully');
        } catch (error) {
            console.error('Error saving filter:', error);
            statusBarTextStore.set('Error saving filter');
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
        let analysis = null;
        try {
            analysis = await LoadAnalysis(position.id);
        } catch (error) {
            // No analysis for this position
        }
        
        // Check if we're in match mode and on first position of a game
        // First position can be move_number 0 or 1
        // Use statusBarModeStore as the authoritative mode check to avoid stale matchContextStore
        const matchCtx = $matchContextStore;
        const inMatchMode = $statusBarModeStore === 'MATCH' && matchCtx.isMatchMode;
        const isFirstPositionOfGame = inMatchMode && 
                                      matchCtx.movePositions.length > 0 &&
                                      (matchCtx.movePositions[matchCtx.currentIndex]?.move_number === 0 ||
                                       matchCtx.movePositions[matchCtx.currentIndex]?.move_number === 1);
        
        // In match mode, use the specific move from the current match context
        // instead of the aggregated played moves from all matches
        let currentPlayedMove = '';
        let currentPlayedCubeAction = '';
        let allPlayedMoves = analysis?.playedMoves || [];
        let allPlayedCubeActions = analysis?.playedCubeActions || [];
        
        if (inMatchMode && matchCtx.movePositions.length > 0) {
            const currentMovePos = matchCtx.movePositions[matchCtx.currentIndex];
            if (currentMovePos) {
                // Use the specific move from this match
                currentPlayedMove = currentMovePos.checker_move || '';
                currentPlayedCubeAction = currentMovePos.cube_action || '';
            }
        } else {
            // Not in match mode, use the first played move (backward compatibility)
            currentPlayedMove = analysis?.playedMove || '';
            currentPlayedCubeAction = analysis?.playedCubeAction || '';
        }
        
        analysisStore.set({
            positionId: analysis?.positionId || null,
            xgid: analysis?.xgid || '',
            player1: analysis?.player1 || '',
            player2: analysis?.player2 || '',
            analysisType: analysis?.analysisType || '',
            analysisEngineVersion: analysis?.analysisEngineVersion || '',
            checkerAnalysis: analysis?.checkerAnalysis || { moves: [] },
            // Clear cube analysis if first position of game, otherwise use loaded analysis
            doublingCubeAnalysis: isFirstPositionOfGame ? null : (analysis?.doublingCubeAnalysis || {
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
            }),
            allCubeAnalyses: isFirstPositionOfGame ? [] : (analysis?.allCubeAnalyses || []),
            playedMove: currentPlayedMove,
            playedCubeAction: isFirstPositionOfGame ? '' : currentPlayedCubeAction,
            playedMoves: allPlayedMoves,
            playedCubeActions: isFirstPositionOfGame ? [] : allPlayedCubeActions,
            creationDate: analysis?.creationDate || '',
            lastModifiedDate: analysis?.lastModifiedDate || ''
        });

        // Load the comment for the current position
        let comment = '';
        try {
            comment = await LoadComment(position.id);
        } catch (error) {
            // No comment for this position
        }
        commentTextStore.set(comment || '');
    }

    // Function to handle mouse wheel events
    function handleWheel(event) {
        if ($isAnyModalOpenStore || $statusBarModeStore === 'EDIT') {
            return; // Prevent changing position when any modal is open or in edit mode
        }

        // Prevent changing position when scrolling in the analysis panel, comment panel, filter panel, match panel, collection panel, or tournament panel
        const analysisPanel = document.querySelector('.analysis-panel');
        const commentPanel = document.querySelector('.comment-panel');
        const filterPanel = document.querySelector('.filter-library-panel'); // Ensure correct class name
        const matchPanel = document.querySelector('.match-panel');
        const collectionPanel = document.querySelector('.collection-panel');
        const tournamentPanel = document.querySelector('.tournament-panel');
        if ((analysisPanel && analysisPanel.contains(event.target)) || 
            (commentPanel && commentPanel.contains(event.target)) || 
            (filterPanel && filterPanel.contains(event.target)) ||
            (matchPanel && matchPanel.contains(event.target)) ||
            (collectionPanel && collectionPanel.contains(event.target)) ||
            (tournamentPanel && tournamentPanel.contains(event.target))) { // Check all panels
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

    async function handleResize() {
        try {
            const size = await WindowGetSize();
            if (size) {
                const { w, h } = size;
                console.log('Window dimensions:', w, h);
                await SaveWindowDimensions(w, h);
            } else {
                console.error('Error: WindowGetSize returned undefined size');
            }
        } catch (err) {
            console.error('Error getting window dimensions:', err);
        }
    }

</script>

<main class="main-container" bind:this={mainArea}>

    {#if showDropOverlay}
        <div class="drop-overlay" transition:fade={{ duration: 150 }}>
            <div class="drop-overlay-content">
                <svg class="drop-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="7 10 12 15 17 10" />
                    <line x1="12" y1="15" x2="12" y2="3" />
                </svg>
                <span>Drop files to import</span>
                <span class="drop-hint">.db &middot; .xg &middot; .sgf &middot; .mat &middot; .bgf &middot; .txt</span>
            </div>
        </div>
    {/if}

    <Toolbar
    on:click={() => {}}
        onNewDatabase={newDatabase}
        onOpenDatabase={openDatabase}
        onImportDatabase={importDatabase}
        onExportDatabase={exportDatabase}
        onExit={exitApp}
        onImportPosition={importPosition}
        onImportFolder={importFolder}
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
        onTogglePipcount={togglePipcount}
        onRandomPosition={loadRandomPosition}
        onToggleEditMode={toggleEditMode}
        onToggleCommandMode={() => showCommandStore.set(true)}
        onShowAnalysis={toggleAnalysisPanel}
        onShowComment={toggleCommentPanel}
        onFindPosition={findPosition}
        onToggleHelp={toggleHelpModal}
        onLoadAllPositions={loadAllPositions}
        onShowMetadata={toggleMetadataModal}
        onToggleFilterLibraryPanel={toggleFilterLibraryPanel}
        onToggleSearchHistory={toggleSearchHistoryPanel}
        onToggleMatchPanel={toggleMatchPanel}
        onToggleCollectionPanel={toggleCollectionPanelAction}
        onToggleTournamentPanel={toggleTournamentPanel}
        onToggleEPCMode={toggleEPCMode}
        onToggleMatchMode={toggleMatchMode}
    />

    <div class="scrollable-content">

        <Board />

        <CommandLine
            onToggleHelp={toggleHelpModal}
            bind:this={commandInput}
            onNewDatabase={newDatabase}
            onOpenDatabase={openDatabase}
            onImportDatabase={importDatabase}
            onExportDatabase={exportDatabase}
            importPosition={importPosition}
            onSavePosition={saveCurrentPosition}
            onUpdatePosition={updatePosition}
            onDeletePosition={deletePosition}
            onToggleAnalysis={toggleAnalysisPanel}
            onToggleComment={toggleCommentPanel}
            exitApp={exitApp}
            onLoadPositionsByFilters={loadPositionsByFilters}
            onLoadAllPositions={loadAllPositions}
            toggleFilterLibraryPanel={toggleFilterLibraryPanel}
            toggleSearchHistoryPanel={toggleSearchHistoryPanel}
            toggleMatchPanel={toggleMatchPanel}
            toggleCollectionPanel={toggleCollectionPanelAction}
            toggleEPCMode={toggleEPCMode}
            toggleMatchMode={toggleMatchMode}
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

    <TakePoint2Modal/>

    <TakePoint4Modal/>

    <ImportProgressModal
        visible={showImportProgressModal}
        mode={importModalMode}
        analysis={importAnalysis}
        result={importResult}
        onCancel={handleImportCancel}
        onCommit={handleImportCommit}
        onClose={handleImportClose}
    />

    <FileImportProgressModal
        visible={showFileImportModal}
        mode={fileImportMode}
        totalFiles={fileImportTotalFiles}
        currentIndex={fileImportCurrentIndex}
        currentFile={fileImportCurrentFile}
        results={fileImportResults}
        onCancel={handleFileImportCancel}
        onClose={handleFileImportClose}
    />

    <ExportDatabaseModal
        visible={$showExportDatabaseModalStore}
        mode={exportModalMode}
        positionCount={exportPositionCount}
        matches={exportMatches}
        bind:metadata={exportMetadata}
        bind:exportOptions={exportOptions}
        onCancel={handleExportCancel}
        onExport={handleExportCommit}
        onClose={handleExportClose}
    />

    <FilterLibraryPanel onLoadPositionsByFilters={loadPositionsByFilters} />

    <MatchPanel />

    <CollectionPanel onOpenCollection={handleOpenCollection} />

    <TournamentPanel />

    <HelpModal
        visible={showHelp}
        onClose={toggleHelpModal}
        handleGlobalKeydown={handleKeyDown}
    />

    <SearchHistoryPanel
        onLoadPositionsByFilters={loadPositionsByFilters}
        onAddToFilterLibrary={addSearchToFilterLibrary}
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
        width: 100vw; /* Ensure it takes the full viewport width */
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


    .panel-container {
        display: flex;
        flex-direction: column; /* Or row, depending on layout */
        height: 100%;
    }

    /* Drag & Drop overlay */
    .drop-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        z-index: 10000;
        background: rgba(30, 60, 114, 0.85);
        display: flex;
        align-items: center;
        justify-content: center;
        pointer-events: none;
    }

    .drop-overlay-content {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 12px;
        color: #ffffff;
        font-size: 1.3rem;
        font-weight: 600;
        text-shadow: 0 1px 4px rgba(0, 0, 0, 0.3);
        border: 3px dashed rgba(255, 255, 255, 0.6);
        border-radius: 16px;
        padding: 40px 60px;
    }

    .drop-icon {
        width: 48px;
        height: 48px;
        color: #ffffff;
        opacity: 0.9;
    }

    .drop-hint {
        font-size: 0.8rem;
        font-weight: 400;
        opacity: 0.7;
    }
    
</style>
