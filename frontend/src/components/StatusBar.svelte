<script>
  import { statusBarTextStore, statusBarModeStore, currentPositionIndexStore } from '../stores/uiStore';
  import { positionsStore } from '../stores/positionStore';
  import { analysisStore } from '../stores/analysisStore';
  import { tableData as metTable } from '../stores/metTable';
  import { takePoint2LiveTable } from '../stores/takePoint2LiveTable';
  import { takePoint2LastTable } from '../stores/takePoint2LastTable';
  import { gammonValue1Table } from '../stores/gammonValue1Table';
  import { gammonValue2Table } from '../stores/gammonValue2Table';
  import { gammonValue4Table } from '../stores/gammonValue4Table';
  import { takePoint4LiveTable } from '../stores/takePoint4LiveTable';
  import { takePoint4LastTable } from '../stores/takePoint4LastTable';
  import { get } from 'svelte/store';

  function showDatesAndMetadata() {
    const analysis = get(analysisStore);
    const positions = get(positionsStore);
    const currentIndex = get(currentPositionIndexStore);

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
    let statusText = `Created: ${creationDate} | Modified: ${lastModifiedDate}`;

    if (positions.length === 0 || currentIndex < 0 || currentIndex >= positions.length) {
      statusText += ' | No position data available';
    } else {
      const position = positions[currentIndex];
      const cubeValue = position.cube.value;
      let metValue = 'N/A';
      let tp2LiveValue = 'N/A';
      let tp2LastValue = 'N/A';
      let gv1Value = 'N/A';
      let gv2Value = 'N/A';
      let gv4Value = 'N/A';
      let tp4LiveValue = 'N/A';
      let tp4LastValue = 'N/A';

      if (position.score[0] - 1 < metTable.length && position.score[1] - 1 < metTable[0].length) {
        metValue = metTable[position.score[0] - 1][position.score[1] - 1].toFixed(1);
      }

      if (position.score[0] - 2 >= 0 && position.score[0] - 2 < takePoint2LiveTable.length && position.score[1] - 2 >= 0 && position.score[1] - 2 < takePoint2LiveTable[0].length) {
        tp2LiveValue = takePoint2LiveTable[position.score[0] - 2][position.score[1] - 2].toFixed(1);
      }

      if (position.score[0] - 2 >= 0 && position.score[0] - 2 < takePoint2LastTable.length && position.score[1] - 2 >= 0 && position.score[1] - 2 < takePoint2LastTable[0].length) {
        tp2LastValue = takePoint2LastTable[position.score[0] - 2][position.score[1] - 2].toFixed(1);
      }

      if (position.score[0] - 2 >= 0 && position.score[0] - 2 < gammonValue1Table.length && position.score[1] - 2 >= 0 && position.score[1] - 2 < gammonValue1Table[0].length) {
        gv1Value = gammonValue1Table[position.score[0] - 2][position.score[1] - 2].toFixed(2);
      }

      if (position.score[0] - 3 >= 0 && position.score[0] - 3 < gammonValue2Table.length && position.score[1] - 2 >= 0 && position.score[1] - 2 < gammonValue2Table[0].length) {
        gv2Value = gammonValue2Table[position.score[0] - 3][position.score[1] - 2].toFixed(2);
      }

      if (position.score[0] - 5 >= 0 && position.score[0] - 5 < gammonValue4Table.length && position.score[1] - 2 >= 0 && position.score[1] - 2 < gammonValue4Table[0].length) {
        gv4Value = gammonValue4Table[position.score[0] - 5][position.score[1] - 2].toFixed(2);
      }

      if (position.score[0] - 3 >= 0 && position.score[0] - 3 < takePoint4LiveTable.length && position.score[1] - 3 >= 0 && position.score[1] - 3 < takePoint4LiveTable[0].length) {
        tp4LiveValue = takePoint4LiveTable[position.score[0] - 3][position.score[1] - 3].toFixed(1);
      }

      if (position.score[0] - 3 >= 0 && position.score[0] - 3 < takePoint4LastTable.length && position.score[1] - 3 >= 0 && position.score[1] - 3 < takePoint4LastTable[0].length) {
        tp4LastValue = takePoint4LastTable[position.score[0] - 3][position.score[1] - 3].toFixed(1);
      }

      let metadata = `met: ${metValue}`;
      if (cubeValue === 0) {
        metadata += ` | tp2_live: ${tp2LiveValue} | tp2_last: ${tp2LastValue} | gv1: ${gv1Value} | gv2: ${gv2Value}`;
      } else if (cubeValue === 1) {
        metadata += ` | tp4_live: ${tp4LiveValue} | tp4_last: ${tp4LastValue} | gv2: ${gv2Value} | gv4: ${gv4Value}`;
      } else if (cubeValue === 2) {
        metadata += ` | gv4: ${gv4Value}`;
      }
      statusText += ` | ${metadata}`;
    }

    statusBarTextStore.set(statusText);
  }

  window.addEventListener('keydown', (event) => {
    if (event.ctrlKey && event.key === 'g') {
      if (get(statusBarModeStore) === 'NORMAL') {
        showDatesAndMetadata();
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

