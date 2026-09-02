<script>
    // A Modal with a title, two controls and a footer, the way a real modal fills it.
    import Modal from '../../components/Modal.svelte';

    let { open = false, onclose = () => {}, onkeydown, closeOnOverlay = false, closeOnEscape = true, closeButton = true, withTitle = true, withControls = true, label = '' } = $props();
</script>

{#snippet heading()}A title{/snippet}

<Modal {open} {onclose} {onkeydown} {closeOnOverlay} {closeOnEscape} {closeButton} {label} title={withTitle ? heading : undefined}>
    {#if withControls}
        <input id="first" />
    {:else}
        <p>Plain text</p>
    {/if}
    {#snippet footer()}
        {#if withControls}<button id="last">OK</button>{/if}
    {/snippet}
</Modal>
