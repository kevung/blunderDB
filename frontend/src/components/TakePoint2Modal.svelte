<script>
    export let visible = false;
    export let onClose;

    function closeModal() {
        onClose();
    }

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            closeModal();
        }
    }

    const liveTableData = [
        [32.5, 26, 20, 17.5, 22.5, 22, 21.5, 21],
        [25, 25, 21.5, 19.5, 22.5, 23, 22.5, 23],
        [18.5, 24, 22, 19.5, 23, 22.5, 22.5, 21.5],
        [23.5, 21.5, 24, 20, 23, 22.5, 23, 22],
        [22.5, 22, 24.5, 20, 23, 22, 22.5, 21],
        [23, 19.5, 25, 20, 23, 21.5, 22.5, 21.5],
        [20.5, 19.5, 24, 20, 22.5, 21, 22.5, 21.5],
        [22, 17.5, 24, 20, 22.5, 21, 22.5, 21.5]
    ];

    const lastTableData = [
        [32.5, 26, 20, 17.5, 22.5, 22, 21.5, 21],
        [37, 30, 24, 21, 24, 24.5, 23, 23.5],
        [37, 35, 29, 22.5, 26, 24.5, 24.5, 23],
        [39.5, 28.5, 30.5, 24, 27, 25.5, 25, 24],
        [34, 28, 29.5, 23.5, 27, 25, 25.5, 24],
        [36, 25, 30.5, 24, 27.5, 25, 26, 24.5],
        [33.5, 26, 30.5, 24.5, 27.5, 25.5, 26.5, 24.5],
        [35.5, 23, 30.5, 24.5, 28, 25.5, 26.5, 25]
    ];

    function formatCell(value) {
        return value.toFixed(1);
    }

    $: if (visible) {
        window.addEventListener('keydown', handleKeyDown);
    } else {
        window.removeEventListener('keydown', handleKeyDown);
    }
</script>

{#if visible}
    <div class="modal-overlay" on:click={closeModal}>
        <div class="modal-content" on:click|stopPropagation>
            <div class="table-container">
                <div class="table-section">
                    <h3>Long Races</h3>
                    <table>
                        <thead>
                            <tr>
                                <th></th>
                                {#each Array(8) as _, colIndex}
                                    <th><strong>{colIndex + 2}</strong></th>
                                {/each}
                            </tr>
                        </thead>
                        <tbody>
                            {#each liveTableData as row, rowIndex}
                                <tr class={rowIndex % 2 === 0 ? 'even-row' : 'odd-row'}>
                                    <td><strong>{rowIndex + 2}</strong></td>
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
                                {#each Array(8) as _, colIndex}
                                    <th><strong>{colIndex + 2}</strong></th>
                                {/each}
                            </tr>
                        </thead>
                        <tbody>
                            {#each lastTableData as row, rowIndex}
                                <tr class={rowIndex % 2 === 0 ? 'even-row' : 'odd-row'}>
                                    <td><strong>{rowIndex + 2}</strong></td>
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
        width: calc(100% / 9); /* Ensure cells are square */
    }

    .even-row {
        background-color: #f2f2f2; /* Increase contrast for alternating row color */
    }

    .odd-row {
        background-color: #ffffff; /* Increase contrast for alternating row color */
    }
</style>
