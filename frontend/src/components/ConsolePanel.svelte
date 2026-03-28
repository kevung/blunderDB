<script>
   import { tick } from 'svelte';
   import { activeTabStore, logEntriesStore } from '../stores/uiStore';

   let logContainer;
   let logEntries = [];

   logEntriesStore.subscribe(async value => {
      logEntries = value;
      await tick();
      if (logContainer) {
         logContainer.scrollTop = logContainer.scrollHeight;
      }
   });

   // Auto-scroll when the log tab becomes active
   activeTabStore.subscribe(async value => {
      if (value === 'log') {
         await tick();
         if (logContainer) {
            logContainer.scrollTop = logContainer.scrollHeight;
         }
      }
   });
</script>

<div class="log-panel">
   <div class="log-output" bind:this={logContainer}>
      {#if logEntries.length === 0}
         <div class="empty-msg">No log entries yet.</div>
      {:else}
         {#each logEntries as entry}
            <div class="log-line {entry.type || 'info'}">
               <span class="log-time">{entry.timestamp.toLocaleTimeString('sv-SE', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}</span>
               <span class="log-msg">{entry.message}</span>
            </div>
         {/each}
      {/if}
   </div>
</div>

<style>
   .log-panel {
      display: flex;
      flex-direction: column;
      height: 100%;
      background: #fff;
      font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
      font-size: 13px;
   }

   .log-output {
      flex: 1;
      overflow-y: auto;
      padding: 4px 8px;
      min-height: 0;
   }

   .log-line {
      display: flex;
      gap: 10px;
      line-height: 1.6;
      border-bottom: 1px solid #f5f5f5;
      padding: 1px 0;
   }

   .log-line.command {
      color: #555;
      font-weight: 500;
   }

   .log-line.command .log-msg {
      color: #555;
   }

   .log-line.result .log-msg {
      color: #2e7d32;
   }

   .log-line.error .log-msg {
      color: #c62828;
   }

   .log-line.info .log-msg {
      color: #333;
   }

   .log-time {
      color: #999;
      white-space: nowrap;
      min-width: 64px;
      font-size: 11px;
   }

   .log-msg {
      color: #333;
   }

   .empty-msg {
      color: #999;
      text-align: center;
      padding: 20px;
   }
</style>
