<script>
    export let onNewDatabase;
    export let onOpenDatabase;
    export let onImportDatabase;
    export let onExportDatabase;
    export let onExit;
    export let onImportPosition;
    export let onCopyPosition;
    export let onPastePosition;
    export let onSavePosition;
    export let onUpdatePosition;
    export let onDeletePosition;
    export let onFirstPosition;
    export let onPreviousPosition;
    export let onNextPosition;
    export let onLastPosition;
    export let onGoToPosition;
    export let onTogglePipcount;
    export let onRandomPosition;
    export let onToggleEditMode;
    export let onToggleCommandMode;
    export let onShowAnalysis;
    export let onShowComment;
    export let onFindPosition;
    export let onToggleHelp;
    export let onLoadAllPositions;
    export let onShowMetadata;
    export let onToggleFilterLibraryPanel; // Add export for the new function
    export let onToggleSearchHistory; // Add export for search history
    

    import { statusBarModeStore } from '../stores/uiStore';
    import { databasePathStore } from '../stores/databaseStore';
    let statusBarMode;
    let databasePath;
    statusBarModeStore.subscribe(value => {
        statusBarMode = value;
    });
    databasePathStore.subscribe(value => {
        databasePath = value;
    });

    function handleFindPosition() {
        if (statusBarMode !== 'EDIT') {
            console.error('Search is only available in edit mode');
            setStatusBarMessage('Search is only available in edit mode');
        } else {
            onFindPosition();
        }
    }

    function handleImportPosition() {
        if (statusBarMode === 'EDIT') {
            console.error('Cannot import position in edit mode');
            setStatusBarMessage('Cannot import position in edit mode');
        } else {
            onImportPosition();
        }
    }

    function handleCopyPosition() {
        if (statusBarMode === 'EDIT') {
            console.error('Cannot copy position in edit mode');
            setStatusBarMessage('Cannot copy position in edit mode');
        } else {
            onCopyPosition();
        }
    }

    function handlePastePosition() {
        if (statusBarMode === 'EDIT') {
            console.error('Cannot paste position in edit mode');
            setStatusBarMessage('Cannot paste position in edit mode');
        } else {
            onPastePosition();
        }
    }

    function handleSavePosition() {
        if (statusBarMode !== 'EDIT') {
            console.error('Save is only available in edit mode');
            setStatusBarMessage('Save is only available in edit mode');
        } else {
            onSavePosition();
        }
    }

    function handleUpdatePosition() {
        if (statusBarMode !== 'EDIT') {
            console.error('Update is only available in edit mode');
            setStatusBarMessage('Update is only available in edit mode');
        } else {
            onUpdatePosition();
        }
    }

    function handleDeletePosition() {
        if (statusBarMode !== 'NORMAL' && statusBarMode !== 'COMMAND') {
            console.error('Cannot delete position in current mode');
            setStatusBarMessage('Cannot delete position in current mode');
        } else {
            onDeletePosition();
        }
    }

    function handleFirstPosition() {
        if (statusBarMode === 'EDIT') {
            console.error('Cannot browse positions in edit mode');
            setStatusBarMessage('Cannot browse positions in edit mode');
        } else {
            onFirstPosition();
        }
    }

    function handlePreviousPosition() {
        if (statusBarMode === 'EDIT') {
            console.error('Cannot browse positions in edit mode');
            setStatusBarMessage('Cannot browse positions in edit mode');
        } else {
            onPreviousPosition();
        }
    }

    function handleNextPosition() {
        if (statusBarMode === 'EDIT') {
            console.error('Cannot browse positions in edit mode');
            setStatusBarMessage('Cannot browse positions in edit mode');
        } else {
            onNextPosition();
        }
    }

    function handleLastPosition() {
        if (statusBarMode === 'EDIT') {
            console.error('Cannot browse positions in edit mode');
            setStatusBarMessage('Cannot browse positions in edit mode');
        } else {
            onLastPosition();
        }
    }

    function handleGoToPosition() {
        if (statusBarMode !== 'NORMAL') {
            console.error('Cannot go to position in current mode');
            setStatusBarMessage('Cannot go to position in current mode');
        } else {
            onGoToPosition();
        }
    }
</script>

<!--// https://heroicons.com/-->
<div class="toolbar">
    <button on:click|stopPropagation={onNewDatabase} aria-label="New Database" title="New Database (Ctrl-N)">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 10.5v6m3-3H9m4.06-7.19-2.12-2.12a1.5 1.5 0 0 0-1.061-.44H4.5A2.25 2.25 0 0 0 2.25 6v12a2.25 2.25 0 0 0 2.25 2.25h15A2.25 2.25 0 0 0 21.75 18V9a2.25 2.25 0 0 0-2.25-2.25h-5.379a1.5 1.5 0 0 1-1.06-.44Z" />
        </svg>
    </button>

    <button on:click|stopPropagation={onOpenDatabase} aria-label="Open Database" title="Open Database (Ctrl-O)">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 9.776c.112-.017.227-.026.344-.026h15.812c.117 0 .232.009.344.026m-16.5 0a2.25 2.25 0 0 0-1.883 2.542l.857 6a2.25 2.25 0 0 0 2.227 1.932H19.05a2.25 2.25 0 0 0 2.227-1.932l.857-6a2.25 2.25 0 0 0-1.883-2.542m-16.5 0V6A2.25 2.25 0 0 1 6 3.75h3.879a1.5 1.5 0 0 1 1.06.44l2.122 2.12a1.5 1.5 0 0 0 1.06.44H18A2.25 2.25 0 0 1 20.25 9v.776" />
        </svg>
    </button>

    <button on:click|stopPropagation={onImportDatabase} aria-label="Import Database" title="Import Database (Ctrl-Shift-I)" disabled={!databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9 8.25H7.5a2.25 2.25 0 0 0-2.25 2.25v9a2.25 2.25 0 0 0 2.25 2.25h9a2.25 2.25 0 0 0 2.25-2.25v-9a2.25 2.25 0 0 0-2.25-2.25H15M9 12l3 3m0 0 3-3m-3 3V2.25" />
        </svg>
    </button>

    <button on:click|stopPropagation={onExportDatabase} aria-label="Export Database" title="Export Database (Ctrl-Shift-S)" disabled={!databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9 8.25H7.5a2.25 2.25 0 0 0-2.25 2.25v9a2.25 2.25 0 0 0 2.25 2.25h9a2.25 2.25 0 0 0 2.25-2.25v-9a2.25 2.25 0 0 0-2.25-2.25H15m0-3-3-3m0 0-3 3m3-3V15" />
        </svg>
    </button>

    <button on:click|stopPropagation={onExit} aria-label="Exit" title="Exit blunderDB (Ctrl-Q)">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M8.25 9V5.25A2.25 2.25 0 0 1 10.5 3h6a2.25 2.25 0 0 1 2.25 2.25v13.5A2.25 2.25 0 0 1 16.5 21h-6a2.25 2.25 0 0 1-2.25-2.25V15M12 9l3 3m0 0-3 3m3-3H2.25" />
        </svg>
    </button>

    <button on:click|stopPropagation={onShowMetadata} aria-label="Database Metadata" title="Database Metadata (Ctrl-M)" disabled={statusBarMode !== 'NORMAL' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="m11.25 11.25.041-.02a.75.75 0 0 1 1.063.852l-.708 2.836a.75.75 0 0 0 1.063.853l.041-.021M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9-3.75h.008v.008H12V8.25Z" />
          </svg>
    </button>

    <div class="separator"></div>

    <button on:click|stopPropagation={handleImportPosition} aria-label="Import Position" title="Import Position (Ctrl-I)" disabled={statusBarMode === 'EDIT' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m3.75 9v6m3-3H9m1.5-12H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />
        </svg>
    </button>

    <button on:click|stopPropagation={handleCopyPosition} aria-label="Copy Position" title="Copy Position (Ctrl-C)" disabled={statusBarMode === 'EDIT' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 0 1-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 0 1 1.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 0 0-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 0 1-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 0 0-3.375-3.375h-1.5a1.125 1.125 0 0 1-1.125-1.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H9.75" />
        </svg>
    </button>

    <button on:click|stopPropagation={handlePastePosition} aria-label="Paste Position" title="Paste Position (Ctrl-V)" disabled={statusBarMode === 'EDIT' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0 0 13.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 0 1-.75.75H9a.75.75 0 0 1-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 0 1-2.25 2.25H6.75A2.25 2.25 0 0 1 4.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 0 1 1.927-.184" />
        </svg>
    </button>

    <button on:click|stopPropagation={handleSavePosition} aria-label="Save Position" title="Save Position (Ctrl-S)" disabled={statusBarMode !== 'EDIT' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v6m3-3H9m12 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
        </svg>
    </button>

    <button on:click|stopPropagation={handleUpdatePosition} aria-label="Update Position" title="Update Position (Ctrl-U)" disabled={statusBarMode !== 'EDIT' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99" />
        </svg>
    </button>

    <button on:click|stopPropagation={handleDeletePosition} aria-label="Delete Position" title="Delete Position (Del)" disabled={statusBarMode !== 'NORMAL' && statusBarMode !== 'COMMAND' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
        </svg>
    </button>

    <div class="separator"></div>

    <button on:click|stopPropagation={onLoadAllPositions} aria-label="Reload All Positions" title="Reload All Positions (Ctrl-R)" disabled={statusBarMode !== 'NORMAL' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5M16.5 12 12 16.5m0 0L7.5 12m4.5 4.5V3" />
        </svg>
    </button>

    <button on:click|stopPropagation={handleFirstPosition} aria-label="First Position" title="First Position (PageUp, h)" disabled={statusBarMode === 'EDIT' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="m18.75 4.5-7.5 7.5 7.5 7.5m-6-15L5.25 12l7.5 7.5" />
        </svg>
    </button>

    <button on:click|stopPropagation={handlePreviousPosition} aria-label="Previous Position" title="Previous Position (Left, k)" disabled={statusBarMode === 'EDIT' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5 8.25 12l7.5-7.5" />
        </svg>
    </button>

    <button on:click|stopPropagation={handleNextPosition} aria-label="Next Position" title="Next Position (Right, j)" disabled={statusBarMode === 'EDIT' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
        </svg>
    </button>

    <button on:click|stopPropagation={handleLastPosition} aria-label="Last Position" title="Last Position (PageDown, l)" disabled={statusBarMode === 'EDIT' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="m5.25 4.5 7.5 7.5-7.5 7.5m6-15 7.5 7.5-7.5 7.5" />
        </svg>
    </button>  

    <button on:click|stopPropagation={handleGoToPosition} aria-label="Go To Position" title="Go To Position (Ctrl-K)" disabled={statusBarMode !== 'NORMAL' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 6.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5ZM12 12.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5ZM12 18.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5Z" />
        </svg>
    </button>

    <div class="separator"></div>

    <button on:click|stopPropagation={onTogglePipcount} aria-label="Toggle Pipcount" title="Toggle Pipcount (p)" disabled={statusBarMode !== 'NORMAL' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5.25 8.25h15m-16.5 7.5h15m-1.8-13.5-3.9 19.5m-2.1-19.5-3.9 19.5" />
          </svg>
    </button>

    <button on:click|stopPropagation={onRandomPosition} aria-label="Random Position" title="Random Position (r)" disabled={statusBarMode !== 'NORMAL' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" aria-hidden="true" data-slot="icon" fill="none" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M14.5 14.5c.12.069.384-.166.592-.525.207-.358.278-.705.158-.774-.12-.07-.384.166-.592.524-.207.36-.278.706-.158.775Zm3.25 1.5c.12.069.384-.166.592-.525.207-.358.278-.705.158-.774-.12-.07-.384.166-.592.524-.207.36-.278.706-.158.775ZM9.5 14.5c-.12.069-.384-.166-.592-.525-.207-.358-.278-.705-.158-.774.12-.07.384.166.592.524.207.36.278.706.158.775Zm0 3.25c-.12.069-.384-.166-.592-.525-.207-.358-.278-.705-.158-.774.12-.07.384.166.592.524.207.36.278.706.158.775ZM6.25 16c-.12.069-.384-.166-.592-.525-.207-.358-.278-.705-.158-.774.12-.07.384.166.592.524.207.36.278.706.158.775Zm0-3.201c-.12.07-.384-.166-.592-.524-.207-.36-.278-.706-.158-.775.12-.069.384.166.592.525.207.358.278.705.158.774Zm6.5-5.549a.75.25 0 0 1-.75.25.75.25 0 0 1-.75-.25A.75.25 0 0 1 12 7a.75.25 0 0 1 .75.25ZM21 7.5l-9-5.25L3 7.5m18 0-9 5.25m9-5.25v9l-9 5.25M3 7.5l9 5.25M3 7.5v9l9 5.25m0-9v9"></path>
          </svg>
    </button>

    <div class="separator"></div>

    <button on:click|stopPropagation={onToggleEditMode} aria-label="Edit Mode" title="Edit Mode (Tab)" disabled={!databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="m16.862 4.487 1.687-1.688a1.875 1.875 0 1 1 2.652 2.652L6.832 19.82a4.5 4.5 0 0 1-1.897 1.13l-2.685.8.8-2.685a4.5 4.5 0 0 1 1.13-1.897L16.863 4.487Zm0 0L19.5 7.125" />
        </svg>
    </button>

    <button on:click|stopPropagation={onToggleCommandMode} aria-label="Command Mode" title="Command Mode (Space)">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="m6.75 7.5 3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0 0 21 18V6a2.25 2.25 0 0 0-2.25-2.25H5.25A2.25 2.25 0 0 0 3 6v12a2.25 2.25 0 0 0 2.25 2.25Z" />
        </svg>
    </button>

    <div class="separator"></div>
    
    <button on:click|stopPropagation={onShowAnalysis} aria-label="Show Analysis" title="Show Analysis (Ctrl-L)" disabled={statusBarMode === 'EDIT' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M8.242 5.992h12m-12 6.003H20.24m-12 5.999h12M4.117 7.495v-3.75H2.99m1.125 3.75H2.99m1.125 0H5.24m-1.92 2.577a1.125 1.125 0 1 1 1.591 1.59l-1.83 1.83h2.16M2.99 15.745h1.125a1.125 1.125 0 0 1 0 2.25H3.74m0-.002h.375a1.125 1.125 0 0 1 0 2.25H2.99" />
        </svg>
    </button>

    <button on:click|stopPropagation={onShowComment} aria-label="Show Comments" title="Write Comments (Ctrl-P)" disabled={statusBarMode === 'EDIT' || !databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.76c0 1.6 1.123 2.994 2.707 3.227 1.087.16 2.185.283 3.293.369V21l4.076-4.076a1.526 1.526 0 0 1 1.037-.443 48.282 48.282 0 0 0 5.68-.494c1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0 0 12 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018Z" />
        </svg>
    </button>

    <button on:click|stopPropagation={onToggleFilterLibraryPanel} aria-label="Filter Library" title="Filter Library (Ctrl-B)" disabled={!databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M17.593 3.322c1.1.128 1.907 1.077 1.907 2.185V21L12 17.25 4.5 21V5.507c0-1.108.806-2.057 1.907-2.185a48.507 48.507 0 0 1 11.186 0Z" />
        </svg>
    </button>

    <button on:click|stopPropagation={handleFindPosition} aria-label="Find Position" title="Find Position (Ctrl-F)" disabled={statusBarMode !== 'EDIT'}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z" />
        </svg>
    </button>

    <button on:click|stopPropagation={onToggleSearchHistory} aria-label="Search History" title="Search History (Ctrl-H)" disabled={!databasePath}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
        </svg>
    </button>

    <button on:click|stopPropagation={onToggleHelp} aria-label="Help" title="Help (?)">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 5.25h.008v.008H12v-.008Z" />
        </svg>
    </button>
</div>

<style>
    .toolbar {
        display: flex;
        align-items: center;
        padding: 4px;
        background-color: #f0f0f0;
        border-bottom: 1px solid #ccc;
        height: 22px; /* Ensure toolbar height is sufficient */
        width: 100%; /* Ensure toolbar takes up full available width */
    }

    .toolbar button {
        background: none;
        border: none;
        padding: 4px;
        margin: 0;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .toolbar button:first-child {
        margin-left: 4px; /* Add space to the left of the first button */
    }

    .toolbar button svg {
        width: 20px;
        height: 20px;
        stroke: currentColor; /* Ensure the stroke color matches the current text color */
    }

    .toolbar button:focus {
        outline: none;
    }

    .toolbar button:hover {
        background-color: #e0e0e0;
    }

    .toolbar button:disabled {
        opacity: 0.5; /* Make disabled buttons paler */
        cursor: not-allowed;
    }

    .separator {
        width: 1px;
        margin: 0 8px; /* Add some space between the icon groups */
        height: 80%; /* Ensure separator height matches toolbar height */
        border-left: 1px solid #d3d3d3; /* Use a lighter shade for the separator */
    }

</style>

