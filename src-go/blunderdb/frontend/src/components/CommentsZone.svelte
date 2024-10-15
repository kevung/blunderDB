<script>
  export let hideCommentsZone;

  let textAreaValue = '';

  import { onMount } from "svelte";

  onMount(() => {
    const textAreaEl = document.getElementById('commentsTextArea');
    if (textAreaEl) {
      textAreaEl.focus();
    }
  });
</script>

<div class="comments-zone">
  <textarea
    id="commentsTextArea"
    rows="5"
    cols="30"
    bind:value={textAreaValue}
    placeholder="Type your comments here..."
  ></textarea>
  <button on:click={hideCommentsZone}>Close</button>
</div>

<style>

  textarea {
    font-size: 20px;
    padding: 8px;
    border: 1px solid #ccc;
    border-radius: 4px;
    outline: none;
    width: 100%;
    resize: none;
  }

  button {
    border: none;
    border-radius: 4px;
    cursor: pointer;
    transition: background-color 0.3s ease, opacity 0.3s ease;
  }

  button:hover {
    opacity: 0.7;
  }

</style>

