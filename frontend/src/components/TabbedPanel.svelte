<script>
   import { activeTabStore } from '../stores/uiStore';
   
   import ConsolePanel from './ConsolePanel.svelte';
   import AnalysisPanel from './AnalysisPanel.svelte';
   import CommentPanel from './CommentPanel.svelte';
   import SearchPanel from './SearchPanel.svelte';
   import CollectionPanel from './CollectionPanel.svelte';
   import MetadataPanel from './MetadataPanel.svelte';
   import MatchPanel from './MatchPanel.svelte';
   import TournamentPanel from './TournamentPanel.svelte';

   // Props passed through to panels
   export let onToggleHelp;
   export let onNewDatabase;
   export let onOpenDatabase;
   export let onImportDatabase;
   export let onExportDatabase;
   export let importPosition;
   export let onSavePosition;
   export let onUpdatePosition;
   export let onDeletePosition;
   export let onToggleAnalysis;
   export let onToggleComment;
   export let exitApp;
   export let onLoadPositionsByFilters;
   export let onLoadAllPositions;
   export let toggleFilterLibraryPanel;
   export let toggleSearchHistoryPanel;
   export let toggleMatchPanel;
   export let toggleCollectionPanel;
   export let toggleEPCMode;
   export let toggleMatchMode;
   export let onCloseAnalysis;
   export let onCloseComment;
   export let onOpenCollection;
   export let onAddToFilterLibrary;

   let consolePanelRef;

   const tabs = [
      { id: 'analysis', label: 'Analysis', icon: 'analysis', shortcut: 'Ctrl+L' },
      { id: 'comments', label: 'Comments', icon: 'comments', shortcut: 'Ctrl+P' },
      { id: 'search', label: 'Search', icon: 'search', shortcut: 'Ctrl+F' },
      { id: 'collections', label: 'Collections', icon: 'collections', shortcut: 'Ctrl+K' },
      { id: 'matches', label: 'Matchs', icon: 'matches', shortcut: 'Ctrl+Tab' },
      { id: 'tournaments', label: 'Tournaments', icon: 'tournaments', shortcut: 'Ctrl+Y' },
      { id: 'metadata', label: 'Metadata', icon: 'metadata', shortcut: 'Ctrl+M' },
      { id: 'console', label: 'Console', icon: 'console', shortcut: 'Esc' },
   ];

   function selectTab(tabId) {
      activeTabStore.set(tabId);
   }

   export function focusConsole() {
      activeTabStore.set('console');
      setTimeout(() => {
         consolePanelRef?.focusInput();
      }, 50);
   }
</script>

<div class="tabbed-panel">
   <div class="tab-bar">
      {#each tabs as tab}
         <button
            class="tab-button"
            class:active={$activeTabStore === tab.id}
            on:click={() => selectTab(tab.id)}
            title={tab.shortcut ? `${tab.label} (${tab.shortcut})` : tab.label}
         >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="tab-icon">
               {#if tab.icon === 'console'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="m6.75 7.5 3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0 0 21 18V6a2.25 2.25 0 0 0-2.25-2.25H5.25A2.25 2.25 0 0 0 3 6v12a2.25 2.25 0 0 0 2.25 2.25Z" />
               {:else if tab.icon === 'analysis'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="M8.242 5.992h12m-12 6.003H20.24m-12 5.999h12M4.117 7.495v-3.75H2.99m1.125 3.75H2.99m1.125 0H5.24m-1.92 2.577a1.125 1.125 0 1 1 1.591 1.59l-1.83 1.83h2.16M2.99 15.745h1.125a1.125 1.125 0 0 1 0 2.25H3.74m0-.002h.375a1.125 1.125 0 0 1 0 2.25H2.99" />
               {:else if tab.icon === 'comments'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.76c0 1.6 1.123 2.994 2.707 3.227 1.087.16 2.185.283 3.293.369V21l4.076-4.076a1.526 1.526 0 0 1 1.037-.443 48.282 48.282 0 0 0 5.68-.494c1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0 0 12 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018Z" />
               {:else if tab.icon === 'search'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z" />
               {:else if tab.icon === 'collections'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="M17.593 3.322c1.1.128 1.907 1.077 1.907 2.185V21L12 17.25 4.5 21V5.507c0-1.108.806-2.057 1.907-2.185a48.507 48.507 0 0 1 11.186 0Z" />
               {:else if tab.icon === 'matches'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 12h16.5m-16.5 3.75h16.5M3.75 19.5h16.5M5.625 4.5h12.75a1.875 1.875 0 0 1 0 3.75H5.625a1.875 1.875 0 0 1 0-3.75Z" />
               {:else if tab.icon === 'tournaments'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="M16.5 18.75h-9m9 0a3 3 0 0 1 3 3h-15a3 3 0 0 1 3-3m9 0v-3.375c0-.621-.503-1.125-1.125-1.125h-.871M7.5 18.75v-3.375c0-.621.504-1.125 1.125-1.125h.872m5.007 0H9.497m5.007 0a7.454 7.454 0 0 1-.982-3.172M9.497 14.25a7.454 7.454 0 0 0 .981-3.172M5.25 4.236c-.982.143-1.954.317-2.916.52A6.003 6.003 0 0 0 7.73 9.728M5.25 4.236V4.5c0 2.108.966 3.99 2.48 5.228M5.25 4.236V2.721C7.456 2.41 9.71 2.25 12 2.25c2.291 0 4.545.16 6.75.47v1.516M7.73 9.728a6.726 6.726 0 0 0 2.748 1.35m8.272-6.842V4.5c0 2.108-.966 3.99-2.48 5.228m2.48-5.492a46.32 46.32 0 0 1 2.916.52 6.003 6.003 0 0 1-5.395 4.972m0 0a6.726 6.726 0 0 1-2.749 1.35m0 0a6.772 6.772 0 0 1-3.044 0" />
               {:else if tab.icon === 'metadata'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="m11.25 11.25.041-.02a.75.75 0 0 1 1.063.852l-.708 2.836a.75.75 0 0 0 1.063.853l.041-.021M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9-3.75h.008v.008H12V8.25Z" />
               {/if}
            </svg>
            <span class="tab-label">{tab.label}</span>
         </button>
      {/each}
   </div>

   <div class="tab-content">
      {#if $activeTabStore === 'analysis'}
         <AnalysisPanel
            visible={true}
            onClose={onCloseAnalysis}
         />
      {:else if $activeTabStore === 'comments'}
         <CommentPanel
            visible={true}
            onClose={onCloseComment}
         />
      {:else if $activeTabStore === 'search'}
         <SearchPanel
            onLoadPositionsByFilters={onLoadPositionsByFilters}
            onAddToFilterLibrary={onAddToFilterLibrary}
         />
      {:else if $activeTabStore === 'collections'}
         <CollectionPanel onOpenCollection={onOpenCollection} />
      {:else if $activeTabStore === 'matches'}
         <MatchPanel />
      {:else if $activeTabStore === 'tournaments'}
         <TournamentPanel />
      {:else if $activeTabStore === 'metadata'}
         <MetadataPanel />
      {:else if $activeTabStore === 'console'}
         <ConsolePanel
            bind:this={consolePanelRef}
            onToggleHelp={onToggleHelp}
            onNewDatabase={onNewDatabase}
            onOpenDatabase={onOpenDatabase}
            onImportDatabase={onImportDatabase}
            onExportDatabase={onExportDatabase}
            importPosition={importPosition}
            onSavePosition={onSavePosition}
            onUpdatePosition={onUpdatePosition}
            onDeletePosition={onDeletePosition}
            onToggleAnalysis={onToggleAnalysis}
            onToggleComment={onToggleComment}
            exitApp={exitApp}
            onLoadPositionsByFilters={onLoadPositionsByFilters}
            onLoadAllPositions={onLoadAllPositions}
            toggleFilterLibraryPanel={toggleFilterLibraryPanel}
            toggleSearchHistoryPanel={toggleSearchHistoryPanel}
            toggleMatchPanel={toggleMatchPanel}
            toggleCollectionPanel={toggleCollectionPanel}
            toggleEPCMode={toggleEPCMode}
            toggleMatchMode={toggleMatchMode}
         />
      {/if}
   </div>
</div>

<style>
   .tabbed-panel {
      display: flex;
      flex-direction: column;
      height: 100%;
      min-height: 0;
      border-top: 1px solid #ccc;
      background: #fff;
   }

   .tab-bar {
      display: flex;
      align-items: center;
      background: #f0f0f0;
      border-bottom: 1px solid #ccc;
      overflow-x: auto;
      flex-shrink: 0;
      height: 28px;
   }

   .tab-button {
      display: flex;
      align-items: center;
      gap: 4px;
      padding: 4px 10px;
      border: none;
      border-bottom: 2px solid transparent;
      background: transparent;
      cursor: pointer;
      font-size: 12px;
      color: #555;
      white-space: nowrap;
      height: 100%;
      transition: background-color 0.15s, border-color 0.15s;
   }

   .tab-button:hover {
      background: #e0e0e0;
   }

   .tab-button.active {
      color: #1a73e8;
      border-bottom-color: #1a73e8;
      background: #fff;
   }

   .tab-icon {
      width: 14px;
      height: 14px;
      flex-shrink: 0;
   }

   .tab-label {
      font-size: 11px;
   }

   .tab-content {
      flex: 1;
      min-height: 0;
      overflow-y: auto;
      overflow-x: hidden;
   }
</style>
