<script>
    let {
        onNewDatabase,
        onOpenDatabase,
        onImportDatabase,
        onExportDatabase,
        onExit,
        onImportPosition,
        onImportFolder,
        onCopyPosition,
        onPastePosition,
        onSavePosition,
        onUpdatePosition,
        onDeletePosition,
        onFirstPosition,
        onPreviousPosition,
        onNextPosition,
        onLastPosition,
        onGoToPosition,
        onTogglePipcount,
        onRandomPosition,
        onCopyBoardImage,
        onToggleCommandMode: _onToggleCommandMode,
        onToggleHelp,
        onToggleConfig,
        onToggleTour,
        onLoadAllPositions,
        onToggleEPCMode: _onToggleEPCMode
    } = $props();

    import { t } from '../i18n';
    import { activeTabStore } from '../stores/uiStore';
    import { databasePathStore } from '../stores/databaseStore';
    let databasePath = $derived($databasePathStore);
    let isSearchTab = $derived($activeTabStore === 'search');
</script>

<!--// https://heroicons.com/-->
<div class="toolbar" data-tour="toolbar">
    <button
        onclick={(e) => {
            e.stopPropagation();
            onNewDatabase(e);
        }}
        aria-label={$t('toolbar.newDatabase')}
        title="{$t('toolbar.newDatabase')} (Ctrl-N)"
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M12 10.5v6m3-3H9m4.06-7.19-2.12-2.12a1.5 1.5 0 0 0-1.061-.44H4.5A2.25 2.25 0 0 0 2.25 6v12a2.25 2.25 0 0 0 2.25 2.25h15A2.25 2.25 0 0 0 21.75 18V9a2.25 2.25 0 0 0-2.25-2.25h-5.379a1.5 1.5 0 0 1-1.06-.44Z"
            />
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onOpenDatabase(e);
        }}
        aria-label={$t('toolbar.openDatabase')}
        title="{$t('toolbar.openDatabase')} (Ctrl-O)"
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M3.75 9.776c.112-.017.227-.026.344-.026h15.812c.117 0 .232.009.344.026m-16.5 0a2.25 2.25 0 0 0-1.883 2.542l.857 6a2.25 2.25 0 0 0 2.227 1.932H19.05a2.25 2.25 0 0 0 2.227-1.932l.857-6a2.25 2.25 0 0 0-1.883-2.542m-16.5 0V6A2.25 2.25 0 0 1 6 3.75h3.879a1.5 1.5 0 0 1 1.06.44l2.122 2.12a1.5 1.5 0 0 0 1.06.44H18A2.25 2.25 0 0 1 20.25 9v.776"
            />
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onImportDatabase(e);
        }}
        aria-label={$t('toolbar.importDatabase')}
        title="{$t('toolbar.importDatabase')} (Ctrl-Shift-I)"
        disabled={!databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M9 8.25H7.5a2.25 2.25 0 0 0-2.25 2.25v9a2.25 2.25 0 0 0 2.25 2.25h9a2.25 2.25 0 0 0 2.25-2.25v-9a2.25 2.25 0 0 0-2.25-2.25H15M9 12l3 3m0 0 3-3m-3 3V2.25"
            />
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onExportDatabase(e);
        }}
        aria-label={$t('toolbar.exportDatabase')}
        title="{$t('toolbar.exportDatabase')} (Ctrl-Shift-S)"
        disabled={!databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M9 8.25H7.5a2.25 2.25 0 0 0-2.25 2.25v9a2.25 2.25 0 0 0 2.25 2.25h9a2.25 2.25 0 0 0 2.25-2.25v-9a2.25 2.25 0 0 0-2.25-2.25H15m0-3-3-3m0 0-3 3m3-3V15"
            />
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onExit(e);
        }}
        aria-label={$t('toolbar.exit')}
        title="{$t('toolbar.exit')} blunderDB (Ctrl-Q)"
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M8.25 9V5.25A2.25 2.25 0 0 1 10.5 3h6a2.25 2.25 0 0 1 2.25 2.25v13.5A2.25 2.25 0 0 1 16.5 21h-6a2.25 2.25 0 0 1-2.25-2.25V15M12 9l3 3m0 0-3 3m3-3H2.25"
            />
        </svg>
    </button>

    <div class="separator"></div>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onImportPosition(e);
        }}
        aria-label={$t('toolbar.importPosition')}
        title="{$t('toolbar.importPositionTip')} (Ctrl-I)"
        disabled={!databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m3.75 9v6m3-3H9m1.5-12H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z"
            />
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onImportFolder(e);
        }}
        aria-label={$t('toolbar.importFolder')}
        title="{$t('toolbar.importFolder')} (Ctrl-Shift-F)"
        disabled={!databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M12 10.5v6m3-3H9m4.06-7.19-2.12-2.12a1.5 1.5 0 0 0-1.061-.44H4.5A2.25 2.25 0 0 0 2.25 6v12a2.25 2.25 0 0 0 2.25 2.25h15A2.25 2.25 0 0 0 21.75 18V9a2.25 2.25 0 0 0-2.25-2.25h-5.379a1.5 1.5 0 0 1-1.06-.44Z"
            />
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onCopyPosition(e);
        }}
        aria-label={$t('toolbar.copyPosition')}
        title={isSearchTab ? `${$t('toolbar.copyPositionSearchTip')} (Ctrl-C)` : `${$t('toolbar.copyPositionTip')} (Ctrl-C)`}
        disabled={!databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 0 1-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 0 1 1.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 0 0-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 0 1-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 0 0-3.375-3.375h-1.5a1.125 1.125 0 0 1-1.125-1.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H9.75"
            />
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onPastePosition(e);
        }}
        aria-label={$t('toolbar.pastePosition')}
        title={isSearchTab ? `${$t('toolbar.pastePositionSearchTip')} (Ctrl-V)` : `${$t('toolbar.pastePosition')} (Ctrl-V)`}
        disabled={!databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M15.666 3.888A2.25 2.25 0 0 0 13.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 0 1-.75.75H9a.75.75 0 0 1-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 0 1-2.25 2.25H6.75A2.25 2.25 0 0 1 4.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 0 1 1.927-.184"
            />
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onSavePosition(e);
        }}
        aria-label={$t('toolbar.savePosition')}
        title="{$t('toolbar.savePosition')} (Ctrl-S)"
        disabled={!isSearchTab || !databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v6m3-3H9m12 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onUpdatePosition(e);
        }}
        aria-label={$t('toolbar.updatePosition')}
        title="{$t('toolbar.updatePosition')} (Ctrl-U)"
        disabled={!isSearchTab || !databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99"
            />
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onDeletePosition(e);
        }}
        aria-label={$t('toolbar.deletePosition')}
        title="{$t('toolbar.deletePosition')} (Del)"
        disabled={!databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0"
            />
        </svg>
    </button>

    <div class="separator"></div>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onLoadAllPositions(e);
        }}
        aria-label={$t('toolbar.loadAllPositions')}
        title="{$t('toolbar.loadAllPositions')} (Ctrl-R)"
        disabled={!databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5M16.5 12 12 16.5m0 0L7.5 12m4.5 4.5V3" />
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onFirstPosition(e);
        }}
        aria-label={$t('toolbar.firstPosition')}
        title="{$t('toolbar.firstPosition')} (PageUp, h)"
        disabled={!databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="m18.75 4.5-7.5 7.5 7.5 7.5m-6-15L5.25 12l7.5 7.5" />
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onPreviousPosition(e);
        }}
        aria-label={$t('toolbar.previousPosition')}
        title="{$t('toolbar.previousPosition')} (Left, k)"
        disabled={!databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5 8.25 12l7.5-7.5" />
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onNextPosition(e);
        }}
        aria-label={$t('toolbar.nextPosition')}
        title="{$t('toolbar.nextPosition')} (Right, j)"
        disabled={!databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onLastPosition(e);
        }}
        aria-label={$t('toolbar.lastPosition')}
        title="{$t('toolbar.lastPosition')} (PageDown, l)"
        disabled={!databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="m5.25 4.5 7.5 7.5-7.5 7.5m6-15 7.5 7.5-7.5 7.5" />
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onGoToPosition(e);
        }}
        aria-label={$t('toolbar.goToPosition')}
        title={$t('toolbar.goToPosition')}
        disabled={!databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M12 6.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5ZM12 12.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5ZM12 18.75a.75.75 0 1 1 0-1.5.75.75 0 0 1 0 1.5Z"
            />
        </svg>
    </button>

    <div class="separator"></div>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onTogglePipcount(e);
        }}
        aria-label={$t('toolbar.togglePipcount')}
        title="{$t('toolbar.togglePipcount')} (p)"
        disabled={!databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5.25 8.25h15m-16.5 7.5h15m-1.8-13.5-3.9 19.5m-2.1-19.5-3.9 19.5" />
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onRandomPosition(e);
        }}
        aria-label={$t('toolbar.randomPosition')}
        title="{$t('toolbar.randomPosition')} (r)"
        disabled={!databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" aria-hidden="true" data-slot="icon" fill="none" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M14.5 14.5c.12.069.384-.166.592-.525.207-.358.278-.705.158-.774-.12-.07-.384.166-.592.524-.207.36-.278.706-.158.775Zm3.25 1.5c.12.069.384-.166.592-.525.207-.358.278-.705.158-.774-.12-.07-.384.166-.592.524-.207.36-.278.706-.158.775ZM9.5 14.5c-.12.069-.384-.166-.592-.525-.207-.358-.278-.705-.158-.774.12-.07.384.166.592.524.207.36.278.706.158.775Zm0 3.25c-.12.069-.384-.166-.592-.525-.207-.358-.278-.705-.158-.774.12-.07.384.166.592.524.207.36.278.706.158.775ZM6.25 16c-.12.069-.384-.166-.592-.525-.207-.358-.278-.705-.158-.774.12-.07.384.166.592.524.207.36.278.706.158.775Zm0-3.201c-.12.07-.384-.166-.592-.524-.207-.36-.278-.706-.158-.775.12-.069.384.166.592.525.207.358.278.705.158.774Zm6.5-5.549a.75.25 0 0 1-.75.25.75.25 0 0 1-.75-.25A.75.25 0 0 1 12 7a.75.25 0 0 1 .75.25ZM21 7.5l-9-5.25L3 7.5m18 0-9 5.25m9-5.25v9l-9 5.25M3 7.5l9 5.25M3 7.5v9l9 5.25m0-9v9"
            ></path>
        </svg>
    </button>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onCopyBoardImage(e);
        }}
        aria-label={$t('toolbar.copyBoardImage')}
        title="{$t('toolbar.copyBoardImageTip')} (Ctrl-X), {$t('toolbar.copyBoardImageWithAnalysis')} (Ctrl-X Ctrl-X)"
        disabled={!databasePath}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="m2.25 15.75 5.159-5.159a2.25 2.25 0 0 1 3.182 0l5.159 5.159m-1.5-1.5 1.409-1.409a2.25 2.25 0 0 1 3.182 0l2.909 2.909m-18 3.75h16.5a1.5 1.5 0 0 0 1.5-1.5V6a1.5 1.5 0 0 0-1.5-1.5H3.75A1.5 1.5 0 0 0 2.25 6v12a1.5 1.5 0 0 0 1.5 1.5Zm10.5-11.25h.008v.008h-.008V8.25Zm.375 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Z"
            />
        </svg>
    </button>

    <div class="separator"></div>

    <button
        onclick={(e) => {
            e.stopPropagation();
            onToggleConfig(e);
        }}
        aria-label={$t('toolbar.settings')}
        title={$t('toolbar.settings')}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 0 1 0 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.369-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 0 1 0-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281Z"
            />
            <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
        </svg>
    </button>

    <button
        data-tour="tour"
        onclick={(e) => {
            e.stopPropagation();
            onToggleTour(e);
        }}
        aria-label={$t('toolbar.tour')}
        title={$t('toolbar.tour')}
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M4.26 10.147a60.438 60.438 0 0 0-.491 6.347A48.62 48.62 0 0 1 12 20.904a48.62 48.62 0 0 1 8.232-4.41 60.46 60.46 0 0 0-.491-6.347m-15.482 0a50.636 50.636 0 0 0-2.658-.813A59.906 59.906 0 0 1 12 3.493a59.903 59.903 0 0 1 10.399 5.84c-.896.248-1.783.52-2.658.814m-15.482 0A50.717 50.717 0 0 1 12 13.489a50.702 50.702 0 0 1 7.74-3.342M6.75 15a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Zm0 0v-3.675A55.378 55.378 0 0 1 12 8.443m-7.007 11.55A5.981 5.981 0 0 0 6.75 15.75v-1.5"
            />
        </svg>
    </button>

    <button
        data-tour="help"
        onclick={(e) => {
            e.stopPropagation();
            onToggleHelp(e);
        }}
        aria-label={$t('toolbar.help')}
        title="{$t('toolbar.help')} (?)"
    >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 5.25h.008v.008H12v-.008Z"
            />
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
