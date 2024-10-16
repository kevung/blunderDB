<script>
    import { onMount, onDestroy } from 'svelte';
    import { fade } from 'svelte/transition';
    import Toolbar from './components/Toolbar.svelte';
    import Board from './components/Board.svelte';
    import Command from './components/Command.svelte';
    import StatusBar from './components/StatusBar.svelte';
    import CommentsZone from './components/CommentsZone.svelte';

    let showCommand = false;
    let showCommentsZone = false;

    let mainArea;
    let commentArea;

    let mode = "NORMAL";
    let position = 0;
    let infoMessage = "";
    let commandText = '';
    let commentText = '';

    function handleKeyDown(event) {
        if (event.code === 'Space') { // to open command line
            if(!showCommand && !showCommentsZone) {
                event.preventDefault();
                showCommand = true;
            }
        } else if ( // to close command line
            (showCommand && (event.code === 'Escape' || event.code === 'Enter')) 
            || (showCommand && event.ctrlKey && event.code === 'KeyC')
        ) {
            closeCommandText();
        } else if(event.ctrlKey && event.code == 'KeyP') { // to toggle comment zone
            event.preventDefault();
            toggleCommentZone();
            showCommand = false;
        }

        // Update the status bar based on the key pressed
        if (showCommand) {
            mode = "COMMAND"; // Set mode to edit when the TextLine is shown
            position++; // Increment the position for demonstration
            infoMessage = "A command is being entered..."; // Set info message
        } else {
            mode = "NORMAL"; // Reset mode to normal
            infoMessage = ""; // Clear info message
        }
    }

    function closeCommandText() {
        showCommand = false;
    }

    function handleNewDatabase() {
        console.log('New Database');
    }

    function handleOpenDatabase() {
        console.log('Open Database');
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

    function handlePreviousPosition() {
        console.log('Previous Position');
    }

    function handleNextPosition() {
        console.log('Next Position');
    }

    function handleSearch() {
        console.log('Search');
    }

    function handleShowAnalysis() {
        console.log('Show Analysis');
    }

    function handleAbout() {
        console.log('About');
    }


    function toggleCommentZone() {
        showCommentsZone = !showCommentsZone;
        if (showCommentsZone) {
            setTimeout(() => {
                commentArea.scrollIntoView({ behavior: 'smooth', block: 'start' });
            }, 0);
        } else {
            mainArea.scrollIntoView({ behavior: 'smooth' });
        }
    }

    onMount(() => {
        window.addEventListener("keydown", handleKeyDown);
    });

    onDestroy(() => {
        window.removeEventListener("keydown", handleKeyDown);
    });
</script>

<main class="main-container" bind:this={mainArea}>

    <Toolbar 
        onNewDatabase={handleNewDatabase}
        onOpenDatabase={handleOpenDatabase}
        onImportPosition={handleImportPosition}
        onCopyPosition={handleCopyPosition}
        onPastePosition={handlePastePosition}
        onPreviousPosition={handlePreviousPosition}
        onNextPosition={handleNextPosition}
        onShowAnalysis={handleShowAnalysis}
        onShowComment={toggleCommentZone}
        onSearch={handleSearch}
        onAbout={handleAbout}
    />

    <div class="scrollable-content">
        <Board />
        <Command visible={showCommand} onClose={closeCommandText} text={commandText} />
    </div>

    <StatusBar mode={mode} infoMessage={infoMessage} position={position}  />

    <CommentsZone bind:this={commentArea} text={commentText} visible={showCommentsZone} onClose={toggleCommentZone} />

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

</style>
