<script>
    import { takePoint4LastTable } from '../stores/takePoint4LastTable';
    import { takePoint4LiveTable } from '../stores/takePoint4LiveTable';
    import { showTakePoint4ModalStore } from '../stores/uiStore';
    let visible = false;

    showTakePoint4ModalStore.subscribe(value => {
        visible = value;
    });

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            showTakePoint4ModalStore.set(false);
        }
    }

    function handleWheel(event) {
        event.preventDefault();
    }

    function formatCell(value) {
        return value.toFixed(0);
    }

    $: if (visible) {
        window.addEventListener('keydown', handleKeyDown);
        window.addEventListener('wheel', handleWheel, { passive: false });
    } else {
        window.removeEventListener('keydown', handleKeyDown);
        window.removeEventListener('wheel', handleWheel);
    }
</script>

{#if visible}
    <div class="modal-overlay" on:click={e => showTakePoint4ModalStore.set(false)}>
        <div class="modal-content" on:click|stopPropagation>
            <div class="table-container">
                <div class="table-section">
                    <h3>Long Races</h3>
                    <table>
                        <thead>
                            <tr>
                                <th></th>
                                {#each Array(7) as _, colIndex}
                                    <th><strong>{colIndex + 3}</strong></th>
                                {/each}
                            </tr>
                        </thead>
                        <tbody>
                            {#each takePoint4LiveTable as row, rowIndex}
                                <tr class={rowIndex % 2 === 0 ? 'even-row' : 'odd-row'}>
                                    <td><strong>{rowIndex + 3}</strong></td>
                                    {#each row as cell}
                                        <td>{formatCell(cell)}</td>
                                    {/each}
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
                <div class="table-section">
                    <h3>Last Roll</h3>
                    <table>
                        <thead>
                            <tr>
                                <th></th>
                                {#each Array(7) as _, colIndex}
                                    <th><strong>{colIndex + 3}</strong></th>
                                {/each}
                            </tr>
                        </thead>
                        <tbody>
                            {#each takePoint4LastTable as row, rowIndex}
                                <tr class={rowIndex % 2 === 0 ? 'even-row' : 'odd-row'}>
                                    <td><strong>{rowIndex + 3}</strong></td>
                                    {#each row as cell}
                                        <td>{formatCell(cell)}</td>
                                    {/each}
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.5);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 1000;
    }

    .modal-content {
        background: white;
        padding: 20px;
        border-radius: 4px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        max-width: 90%;
        max-height: 90%;
        overflow: auto;
        display: flex;
        flex-direction: column;
    }

    .table-container {
        display: flex;
        justify-content: space-between;
    }

    .table-section {
        flex: 1;
        margin: 0 10px;
    }

    h3 {
        margin-top: 0px; /* Reduce space above table titles */
    }

    table {
        width: 100%;
        border-collapse: collapse;
    }

    th, td {
        border: 1px solid #ddd;
        padding: 8px;
        text-align: center;
        width: calc(100% / 8); /* Ensure cells are square */
    }

    .even-row {
        background-color: #f2f2f2; /* Increase contrast for alternating row color */
    }

    .odd-row {
        background-color: #ffffff; /* Increase contrast for alternating row color */
    }
</style>
