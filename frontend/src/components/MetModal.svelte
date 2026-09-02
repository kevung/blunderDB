<script>
    import Modal from './Modal.svelte';
    import { t } from '../i18n';
    import { tableData } from '../stores/metTable';

    let { visible = false, onClose } = $props();

    function formatCell(value) {
        return value.toFixed(1);
    }
</script>

<Modal open={visible} onclose={onClose} size="auto" closeOnOverlay closeButton={false} label={$t('met.title')}>
    <table>
        <thead>
            <tr>
                <th></th>
                {#each Array(15) as _, colIndex (colIndex)}
                    <th><strong>{colIndex + 1}</strong></th>
                {/each}
            </tr>
        </thead>
        <tbody>
            {#each tableData as row, rowIndex (rowIndex)}
                <tr class={rowIndex % 2 === 0 ? 'even-row' : 'odd-row'}>
                    <td><strong>{rowIndex + 1}</strong></td>
                    {#each row as cell, cellIndex (cellIndex)}
                        <td>{formatCell(cell)}</td>
                    {/each}
                </tr>
            {/each}
        </tbody>
    </table>
</Modal>

<style>
    table {
        width: 100%;
        border-collapse: collapse;
    }

    th,
    td {
        border: 1px solid #ddd;
        padding: 8px;
        text-align: center;
        width: calc(100% / 16); /* Ensure cells are square */
    }

    .even-row {
        background-color: #f2f2f2; /* Increase contrast for alternating row color */
    }

    .odd-row {
        background-color: #ffffff; /* Increase contrast for alternating row color */
    }
</style>
