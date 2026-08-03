<script>
    // Mirrors how App.svelte mounts the export modal: the options are bound to a store, so
    // the child receives a plain object through a two-way binding rather than a $state
    // proxy. That binding is what the focus regression needed to show itself.
    import ExportDatabaseModal from '../../components/ExportDatabaseModal.svelte';

    let { optionsStore, metadataStore } = $props();
</script>

<ExportDatabaseModal
    visible={true}
    mode="metadata"
    positionCount={3}
    bind:metadata={$metadataStore}
    bind:exportOptions={$optionsStore}
    matches={[]}
    onCancel={() => {}}
    onExport={() => {}}
    onClose={() => {}}
/>
