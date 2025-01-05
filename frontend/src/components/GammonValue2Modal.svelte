<script>
    import { gammonValue2Table } from '../stores/gammonValue2Table';
    export let visible = false;
    export let onClose;

    function closeModal() {
        onClose();
    }

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            closeModal();
        }
        // Prevent browsing position shortcuts
        if (['PageUp', 'PageDown', 'ArrowLeft', 'ArrowRight', 'h', 'k', 'j', 'l'].includes(event.key)) {
            event.preventDefault();
        }
    }

    const tableData = gammonValue2Table;

    function formatCell(value) {
        return value.toFixed(2);
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
        width: calc(100% / 9); /* Ensure cells are square */
    }

    .even-row {
        background-color: #f2f2f2; /* Increase contrast for alternating row color */
    }

    .odd-row {
        background-color: #ffffff; /* Increase contrast for alternating row color */
    }
</style>
