<script>
  import { statusBarTextStore, statusBarModeStore, currentPositionIndexStore } from '../stores/uiStore';
  import { positionsStore } from '../stores/positionStore';
  import { analysisStore } from '../stores/analysisStore';
  import { get } from 'svelte/store';

  function showDates() {
    const analysis = get(analysisStore);
    if (!analysis || !analysis.creationDate || !analysis.lastModifiedDate) {
      statusBarTextStore.set('No database opened');
      return;
    }
    const options = { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' };
    const formatDate = (date) => {
      const [year, month, day] = date.toLocaleDateString('sv-SE').split('-');
      const time = date.toLocaleTimeString('sv-SE', { hour: '2-digit', minute: '2-digit' });
      return `${year}/${month}/${day} ${time}`;
    };
    const creationDate = formatDate(new Date(analysis.creationDate));
    const lastModifiedDate = formatDate(new Date(analysis.lastModifiedDate));
    statusBarTextStore.set(`Created: ${creationDate} | Modified: ${lastModifiedDate}`);
  }

  window.addEventListener('keydown', (event) => {
    if (event.ctrlKey && event.key === 'g') {
      if (get(statusBarModeStore) === 'NORMAL') {
        showDates();
      }
    }
  });
</script>

<div class="status-bar">
  <span class="mode">{$statusBarModeStore}</span>
  <div class="separator"></div>
  <span class="info-message">{$statusBarTextStore}</span>
  <div class="separator"></div>
  <span class="position">{$positionsStore.length > 0 ? $currentPositionIndexStore + 1 : 0} / {$positionsStore.length}</span>
</div>

<style>
  .status-bar {
    display: flex;
    align-items: center; /* Center items vertically */
    background-color: #f0f0f0;
    border-bottom: 1px solid #ccc;
    border-top: 1px solid #ccc;
    padding: 4px 0px; /* Padding for the status bar */
    position: fixed; /* Fixed position at the bottom */
    bottom: 0; /* Align to bottom */
    left: 0; /* Align to left */
    right: 0; /* Align to right */
    font-size: 14px; /* Font size */
    z-index: 10;
  }

  .mode {
      width: 84px; /* Fixed width for mode */
      text-align: center;
      justify-content: center;
  }

  .info-message {
      flex: 1; /* Allow this to expand and take available space */
      text-align: center; /* Center text */
      margin: 0 8px; /* Space on the sides */
  }

  .position {
      width: 80px; /* Fixed width for position */
      text-align: center;
      justify-content: center;
  }

  .separator {
      width: 1px; /* Width of the separator */
      height: 20px; /* Height of the separator */
      background-color: rgba(0, 0, 0, 0.2); /* Light color for the separator */
      margin: 0 8px; /* Add some space between the separators */
  }

</style>

