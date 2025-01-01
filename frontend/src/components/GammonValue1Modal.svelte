<script>
    export let visible = false;
    export let onClose;

    function closeModal() {
        onClose();
    }

    const tableData = [
        [0.91, 0.99, 0.86, 0.91, 0.78, 0.86, 0.75, 0.82],
        [0.71, 0.76, 0.86, 0.57, 0.58, 0.5, 0.5, 0.43],
        [0.46, 0.59, 0.68, 0.65, 0.66, 0.67, 0.67, 0.66],
        [0.4, 0.42, 0.48, 0.46, 0.48, 0.47, 0.49, 0.47],
        [0.52, 0.51, 0.58, 0.54, 0.58, 0.58, 0.59, 0.57],
        [0.52, 0.48, 0.55, 0.5, 0.53, 0.51, 0.53, 0.51],
        [0.48, 0.45, 0.54, 0.5, 0.54, 0.54, 0.56, 0.55],
        [0.48, 0.44, 0.49, 0.47, 0.49, 0.49, 0.51, 0.5]
    ];

    function formatCell(value) {
        return value.toFixed(2);
    }
</script>

{#if visible}
    <div class="modal-overlay" on:click={closeModal}>
        <div class="modal-content" on:click|stopPropagation>
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
                    {#each tableData as row, rowIndex}
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
        width: calc(100% / 9); /* Ensure cells are square */
    }

    .even-row {
        background-color: #f2f2f2; /* Increase contrast for alternating row color */
    }

    .odd-row {
        background-color: #ffffff; /* Increase contrast for alternating row color */
    }
</style>
