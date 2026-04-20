<script>
    import { trapFocus } from '../utils/focusTrap.js';

    let { visible = false, onClose, tables = [] } = $props();

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            onClose();
        }
    }

    function handleWheel(event) {
        event.preventDefault();
    }

    $effect(() => {
        if (visible) {
            window.addEventListener('keydown', handleKeyDown);
            window.addEventListener('wheel', handleWheel, { passive: false });
        } else {
            window.removeEventListener('keydown', handleKeyDown);
            window.removeEventListener('wheel', handleWheel);
        }
    });
</script>

{#if visible}
    <div class="modal-overlay" onclick={() => onClose()} role="dialog" aria-modal="true" aria-label="Data table" use:trapFocus>
        <div class="modal-content" onclick={(e) => e.stopPropagation()}>
            <div class="table-container" class:multi={tables.length > 1}>
                {#each tables as { title, data, precision, colCount, colOffset, rowOffset }}
                    <div class="table-section">
                        {#if title}<h3>{title}</h3>{/if}
                        <table>
                            <thead>
                                <tr>
                                    <th></th>
                                    {#each Array(colCount) as _, colIndex}
                                        <th><strong>{colIndex + colOffset}</strong></th>
                                    {/each}
                                </tr>
                            </thead>
                            <tbody>
                                {#each data as row, rowIndex}
                                    <tr class={rowIndex % 2 === 0 ? 'even-row' : 'odd-row'}>
                                        <td><strong>{rowIndex + rowOffset}</strong></td>
                                        {#each row as cell}
                                            <td>{cell.toFixed(precision)}</td>
                                        {/each}
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                {/each}
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
    }

    .table-container.multi {
        display: flex;
        justify-content: space-between;
    }

    .multi .table-section {
        flex: 1;
        margin: 0 10px;
    }

    h3 {
        margin-top: 0;
    }

    table {
        width: 100%;
        border-collapse: collapse;
    }

    th,
    td {
        border: 1px solid #ddd;
        padding: 8px;
        text-align: center;
    }

    .even-row {
        background-color: #f2f2f2;
    }

    .odd-row {
        background-color: #ffffff;
    }
</style>
