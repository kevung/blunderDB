<script>
    import Modal from './Modal.svelte';
    import { t } from '../i18n';

    let { visible = false, onClose, tables = [] } = $props();

    function handleWheel(event) {
        event.preventDefault();
    }

    $effect(() => {
        if (visible) {
            window.addEventListener('wheel', handleWheel, { passive: false });
        } else {
            window.removeEventListener('wheel', handleWheel);
        }
    });
</script>

<Modal open={visible} onclose={onClose} size="auto" closeOnOverlay closeButton={false} label={$t('datatable.title')}>
    <div class="table-container" class:multi={tables.length > 1}>
        {#each tables as { title, data, precision, colCount, colOffset, rowOffset } (title)}
            <div class="table-section">
                {#if title}<h3>{title}</h3>{/if}
                <table>
                    <thead>
                        <tr>
                            <th></th>
                            {#each Array(colCount) as _, colIndex (colIndex)}
                                <th><strong>{colIndex + colOffset}</strong></th>
                            {/each}
                        </tr>
                    </thead>
                    <tbody>
                        {#each data as row, rowIndex (rowIndex)}
                            <tr class={rowIndex % 2 === 0 ? 'even-row' : 'odd-row'}>
                                <td><strong>{rowIndex + rowOffset}</strong></td>
                                {#each row as cell, cellIndex (cellIndex)}
                                    <td>{cell.toFixed(precision)}</td>
                                {/each}
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/each}
    </div>
</Modal>

<style>
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
