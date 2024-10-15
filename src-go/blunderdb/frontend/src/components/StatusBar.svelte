<script>
  export let mode = "normal"; // Current mode
  export let infoMessage = ""; // Info message to display
  export let position = 0; // Current position
</script>

<div class="status-bar">
  <span class="mode">{mode}</span>
  <div class="separator"></div>
  <span class="info-message">{infoMessage}</span>
  <div class="separator"></div>
  <span class="position">{position}/1000</span>
</div>

<style>
  .status-bar {
    display: flex;
    align-items: center; /* Center items vertically */
    background-color: rgba(0, 0, 0, 0.09); /* Slightly dark background */
    padding: 8px 12px; /* Padding for the status bar */
    border-top: 1px solid rgba(0, 0, 0, 0.2); /* Top border */
    position: fixed; /* Fixed position at the bottom */
    bottom: 0; /* Align to bottom */
    left: 0; /* Align to left */
    right: 0; /* Align to right */
    box-shadow: 0 -2px 5px rgba(0, 0, 0, 0.2); /* Shadow for depth */
    font-size: 16px; /* Font size */
  }

  .mode {
      width: 90px; /* Fixed width for mode */
      text-align: center;
      margin-right: 8px; /* Space to the right */
  }

  .info-message {
      flex: 1; /* Allow this to expand and take available space */
      text-align: center; /* Center text */
      margin: 0 8px; /* Space on the sides */
  }

  .position {
      width: 80px; /* Fixed width for position */
      text-align: center;
      margin-left: 8px;
  }

  .separator {
      width: 1px; /* Width of the separator */
      height: 24px; /* Height of the separator */
      background-color: rgba(0, 0, 0, 0.2); /* Light color for the separator */
  }

</style>

