<script>
  export let mode = "normal"; // Current mode
  export let text = ""; // Info message to display
  export let currentPosition = 0; // Current position
  export let totalPositions = 0; // Total positions
</script>

<div class="status-bar">
  <span class="mode">{mode}</span>
  <div class="separator"></div>
  <span class="info-message">{text}</span>
  <div class="separator"></div>
  <span class="position">{totalPositions > 0 ? currentPosition + 1 : 0} / {totalPositions}</span>
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

