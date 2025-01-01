<script>
    export let visible = false;
    export let onClose;

    function closeModal() {
        onClose();
    }

    const tableData = [
        [25, 40, 33, 29, 30, 33, 32],
        [19, 33, 30, 25, 26, 29, 29],
        [16, 26, 25, 25, 25, 27, 28],
        [11, 20, 22, 23, 24, 26, 26],
        [9, 16, 18, 20, 22, 24, 25],
        [7, 12, 16, 18, 20, 22, 23],
        [7, 12, 15, 17, 19, 21, 22]
    ];

    function formatCell(value) {
        return value.toFixed(1);
    }
</script>

{#if visible}
    <div class="modal-overlay" on:click={closeModal}>
        <div class="modal-content" on:click|stopPropagation>
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
                    {#each tableData as row, rowIndex}
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
