<script>
    import { onMount, onDestroy } from 'svelte';
    import { fade } from 'svelte/transition';
    import {SaveDatabaseDialog, OpenDatabaseDialog, OpenPositionDialog} from '../wailsjs/go/main/App.js';
    import Toolbar from './components/Toolbar.svelte';
    import Board from './components/Board.svelte';
    import Command from './components/Command.svelte';
    import StatusBar from './components/StatusBar.svelte';
    import AnalysisPanel from './components/AnalysisPanel.svelte';
    import CommentsZone from './components/CommentsZone.svelte';
    import HelpModal from './components/HelpModal.svelte';

    let showCommand = false;
    let showAnalysis = false;
    let showHelp = false;  // Add state for help modal
    let showCommentsZone = false;


    let mainArea;
    let commentArea;
    let commandInput; // Reference for the command input element

    let filePath;
    let mode = "NORMAL";
    let position = 0;
    let infoMessage = "";
    let commandText = '';
    let commentText = '';
    let analysisData = 'This is where your analysis data will be displayed.';

    function handleKeyDown(event) {
        if(event.ctrlKey && event.code == 'KeyN') {
            newDatabase();
        } else if(event.ctrlKey && event.code == 'KeyO') {
            openDatabase();
        } else if (event.ctrlKey && event.code === 'KeyQ') {
            handleExit();
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
            previousPosition();
        } else if (!event.ctrlKey && event.key === 'k') {
            previousPosition();
        } else if (!event.ctrlKey && event.key === 'ArrowRight') {
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
            if(!showCommand && !showCommentsZone && !showHelp) {
                event.preventDefault();
                toggleCommandMode();
            }
        } else if (event.ctrlKey && event.code === 'KeyL') {
            event.preventDefault();
            toggleAnalysisPanel();
        } else if(event.ctrlKey && event.code == 'KeyP') {
            if(!showHelp && !showCommand) {
                event.preventDefault();
                toggleCommentZone();
            }
        } else if (event.ctrlKey && event.code === 'KeyF') {
            findPosition();
        } else if (event.ctrlKey && event.code === 'KeyH') {
            event.preventDefault();
            toggleHelpModal();
        }
    }

    async function newDatabase() {
        console.log('newDatabase');
        try {
            filePath = await SaveDatabaseDialog();
            if (filePath) {
                console.log('Database selected:', filePath);
                // Add your logic to handle the selected file
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
            filePath = await OpenDatabaseDialog();
            if (filePath) {
                console.log('Database selected:', filePath);
                // add your logic to handle the selected file
            } else {
                console.log('No Database selected');
            }
        } catch (error) {
            console.error('Error opening file dialog:', error);
        }
    }

    function handleExit() {
        window.runtime.Quit();
    }

    async function importPosition() {
        console.log('importPosition');
        try {
            filePath = await OpenPositionDialog();
            if (filePath) {
                console.log('Position selected:', filePath);
                // Add your logic to handle the selected file
            } else {
                console.log('No file selected');
            }
        } catch (error) {
            console.error('Error opening file dialog:', error);
        }
    }

    function copyPosition() {
        console.log('copyPosition');
    }

    function pastePosition() {
        console.log('pastePosition');
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
        if(mode !== "EDIT") {
            if(showCommentsZone){
                toggleCommentZone();
            }
            if(showAnalysis){
                toggleAnalysisPanel();
            }
            mode = "EDIT";
        } else {
            mode = "NORMAL";
        }
    }

    function toggleCommandMode(){
        console.log('toggleCommandMode');
        if(!showCommand) {
            mode = "COMMAND";
        } else {
            mode = "NORMAL";
        }
        showCommand = !showCommand;
    }

    function toggleAnalysisPanel() {
        console.log('toggleAnalysisPanel');

        if(mode === 'NORMAL') {
            showAnalysis = !showAnalysis;
        }

        if (showAnalysis) {
            showCommentsZone = false;
            setTimeout(() => {
            //event.preventDefault();
                document.querySelector('.analysis-panel').scrollIntoView({ behavior: 'smooth', block: 'start' });
            }, 0);
        }
        else {
            setTimeout(() => {
                mainArea.scrollIntoView({ behavior: 'smooth', block: 'start' });
            }, 0);
        }
    }

    function toggleCommentZone() {
        console.log('toggleCommentZone');

        if(mode === 'NORMAL'){
            showCommentsZone = !showCommentsZone;
        }

        if (showCommentsZone) {
            showAnalysis = false;
            showCommand = false;
            setTimeout(() => {
                commentArea.scrollIntoView({ behavior: 'smooth', block: 'start' });
            }, 0);
        } else {
            mainArea.scrollIntoView({ behavior: 'smooth' });
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

                } else if(showCommentsZone) {
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
        onExit={handleExit}
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
        onShowComment={toggleCommentZone}
        onFindPosition={findPosition}
        onToggleHelp={toggleHelpModal}
    />

    <div class="scrollable-content">
        <Board />

        <Command visible={showCommand}
            onClose={toggleCommandMode}
            text={commandText}
            bind:this={commandInput}
        />

    </div>

    <div class="panel-container">
        <CommentsZone bind:this={commentArea} text={commentText}
            visible={showCommentsZone} onClose={toggleCommentZone} />

        <AnalysisPanel visible={showAnalysis} analysisData={analysisData}
            onClose={toggleAnalysisPanel} /> 
    </div>

    <HelpModal visible={showHelp} onClose={toggleHelpModal} />

    <StatusBar mode={mode} infoMessage={infoMessage} position={position}  />

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
