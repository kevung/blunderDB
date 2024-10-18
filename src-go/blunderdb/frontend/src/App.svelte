<script>
    import { onMount, onDestroy } from 'svelte';
    import { fade } from 'svelte/transition';
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

    let mode = "NORMAL";
    let position = 0;
    let infoMessage = "";
    let commandText = '';
    let commentText = '';
    let analysisData = 'This is where your analysis data will be displayed.';

    function handleKeyDown(event) {
        if (event.code === 'Space') { // to open command line
            if(!showCommand && !showCommentsZone) {
                event.preventDefault();
                toggleCommandMode();
            }
        } else if(event.ctrlKey && event.code == 'KeyP') { // to toggle comment zone
            event.preventDefault();
            toggleCommentZone();
        } else if (event.ctrlKey && event.code === 'KeyL') { // Toggle analysis panel (Ctrl+L)
            event.preventDefault();
            toggleAnalysisPanel();
        } else if (event.ctrlKey && event.code === 'KeyH') { // Ctrl+H for help
            event.preventDefault();
            toggleHelpModal();
        } else if (event.ctrlKey && event.code === 'KeyQ') { // Ctrl+Q for Exit
            handleExit();
        }
    }

    function handleNewDatabase() {
        console.log('New Database');
    }

    function handleOpenDatabase() {
        console.log('Open Database');
    }

    function handleExit() {
        window.runtime.Quit();
    }

    function handleImportPosition() {
        console.log('Import Position');
    }

    function handleCopyPosition() {
        console.log('Copy Position');
    }

    function handlePastePosition() {
        console.log('Paste Position');
    }

    function handleAddPosition() {
        console.log('Add Position');
    }

    function handleUpdatePosition() {
        console.log('Update Position');
    }

    function handleDeletePosition() {
        console.log('Delete Position');
    }

    function handleFirstPosition() {
        console.log('First Position');
    }

    function handlePreviousPosition() {
        console.log('Previous Position');
    }

    function handleNextPosition() {
        console.log('Next Position');
    }

    function handleLastPosition() {
        console.log('Last Position');
    }

    function handleGoToPosition() {
        console.log('Go To Position');
    }

    function handleFindPosition() {
        console.log('Find Position');
    }

    function handleShowAnalysis() {
        console.log('Show Analysis');
    }

    function handleHelp() {
        console.log('Help');
    }


    function toggleEditMode(){
        if(mode !== "EDIT") {
            mode = "EDIT";
        } else {
            mode = "NORMAL";
        }
    }

    function toggleCommandMode(){
        if(!showCommand) {
            mode = "COMMAND";
        } else {
            mode = "NORMAL";
        }
        showCommand = !showCommand;
    }

    function toggleAnalysisPanel() {
        showAnalysis = !showAnalysis;
        if (showAnalysis) {
            showCommentsZone = false;
            setTimeout(() => {
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
        showCommentsZone = !showCommentsZone;
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
    }

</script>

<main class="main-container" bind:this={mainArea}>

    <Toolbar 
        onNewDatabase={handleNewDatabase}
        onOpenDatabase={handleOpenDatabase}
        onExit={handleExit}
        onImportPosition={handleImportPosition}
        onCopyPosition={handleCopyPosition}
        onPastePosition={handlePastePosition}
        onAddPosition={handleAddPosition}
        onUpdatePosition={handleUpdatePosition}
        onDeletePosition={handleDeletePosition}
        onFirstPosition={handleFirstPosition}
        onPreviousPosition={handlePreviousPosition}
        onNextPosition={handleNextPosition}
        onLastPosition={handleLastPosition}
        onGoToPosition={handleGoToPosition}
        onToggleEditMode={toggleEditMode}
        onToggleCommandMode={toggleCommandMode}
        onShowAnalysis={toggleAnalysisPanel}
        onShowComment={toggleCommentZone}
        onFindPosition={handleFindPosition}
        onToggleHelp={toggleHelpModal}
    />

    <div class="scrollable-content">
        <Board />

        <Command visible={showCommand} onClose={toggleCommandMode}
            text={commandText} />

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
