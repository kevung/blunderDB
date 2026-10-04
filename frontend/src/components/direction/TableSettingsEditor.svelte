<script>
    /*
     * Les propriétés des tables (ADR-0058) : une ligne par table, un nom, une salle (libellé
     * libre), une réservation, des joueurs attitrés. La salle se pose aussi par lot, sur une
     * plage de tables : c'est le geste des quarante tables. Une table sans propriété est une
     * table ordinaire, elle n'est pas enregistrée.
     */
    import { t } from '../../i18n';

    /** @typedef {import('../../../wailsjs/go/models').domain.TableSetting} TableSetting */

    /**
     * @type {{
     *     tables: number,
     *     settings?: TableSetting[],
     *     participants?: string[],
     *     onSave: (settings: TableSetting[]) => Promise<unknown> | unknown,
     *     testid?: string
     * }}
     */
    let { tables, settings = [], participants: participantNames = [], onSave, testid = 'table-settings' } = $props();
    // Assignments are by name, and two entrants may share one: a keyed list needs each name once.
    const participants = $derived([...new Set(participantNames)]);

    /** @typedef {{ number: number, name: string, room: string, reserved: boolean, assignedTo: string[] }} Row */

    /** @param {number} count @param {TableSetting[]} saved @returns {Row[]} */
    function build(count, saved) {
        const byNumber = new Map(saved.map((s) => [s.number, s]));
        return Array.from({ length: count }, (_, i) => {
            const s = byNumber.get(i + 1);
            return { number: i + 1, name: s?.name || '', room: s?.room || '', reserved: !!s?.reserved, assignedTo: [...(s?.assignedTo || [])] };
        });
    }

    /** @param {Row[]} rows */
    function kept(rows) {
        return rows
            .map((r) => ({ number: r.number, name: r.name.trim(), room: r.room.trim(), reserved: r.reserved, assignedTo: r.assignedTo }))
            .filter((r) => r.name || r.room || r.reserved || r.assignedTo.length);
    }

    let rows = $state(/** @type {Row[]} */ ([]));
    let from = $state(1);
    let to = $state(1);
    let bulkRoom = $state('');
    let busy = $state(false);
    let error = $state('');

    $effect(() => {
        rows = build(tables, settings);
        to = tables;
    });

    const dirty = $derived(JSON.stringify(kept(rows)) !== JSON.stringify(kept(build(tables, settings))));
    const roomNames = $derived([...new Set(rows.map((r) => r.room.trim()).filter(Boolean))]);

    function applyBulk() {
        const lo = Math.min(from, to);
        const hi = Math.max(from, to);
        for (const r of rows) if (r.number >= lo && r.number <= hi) r.room = bulkRoom.trim();
    }

    /** @param {Row} r @param {string} name */
    function assign(r, name) {
        if (name && !r.assignedTo.includes(name)) r.assignedTo = [...r.assignedTo, name];
    }

    /** @param {Row} r @param {string} name */
    function unassign(r, name) {
        r.assignedTo = r.assignedTo.filter((n) => n !== name);
    }

    async function save() {
        busy = true;
        error = '';
        try {
            await onSave(kept(rows));
        } catch (e) {
            error = String(e instanceof Error ? e.message : e).replace(/^direction:\s*/, '');
        } finally {
            busy = false;
        }
    }

    function reset() {
        rows = build(tables, settings);
        error = '';
    }
</script>

<section class="table-settings" data-testid={testid}>
    <h4>{$t('direction.tableProps.title')}</h4>
    <p class="facts">{$t('direction.tableProps.hint')}</p>
    <div class="bulk" data-testid="{testid}-bulk">
        <label>{$t('direction.tableProps.bulkFrom')} <input type="number" min="1" max={tables} bind:value={from} data-testid="{testid}-bulk-from" /></label>
        <label>{$t('direction.tableProps.bulkTo')} <input type="number" min="1" max={tables} bind:value={to} data-testid="{testid}-bulk-to" /></label>
        <label>{$t('direction.tableProps.room')} <input type="text" list="{testid}-rooms" bind:value={bulkRoom} data-testid="{testid}-bulk-room" /></label>
        <button type="button" onclick={applyBulk} data-testid="{testid}-bulk-apply">{$t('direction.tableProps.bulkApply')}</button>
    </div>
    <datalist id="{testid}-rooms">
        {#each roomNames as room (room)}<option value={room}></option>{/each}
    </datalist>
    <div class="scroll">
        <table>
            <thead>
                <tr>
                    <th scope="col">{$t('direction.tableProps.number')}</th>
                    <th scope="col">{$t('direction.tableProps.name')}</th>
                    <th scope="col">{$t('direction.tableProps.room')}</th>
                    <th scope="col">{$t('direction.tableProps.reserved')}</th>
                    <th scope="col">{$t('direction.tableProps.assignedTo')}</th>
                </tr>
            </thead>
            <tbody>
                {#each rows as r (r.number)}
                    <tr data-testid="{testid}-row-{r.number}">
                        <th scope="row">{r.number}</th>
                        <td><input type="text" aria-label={$t('direction.tableProps.nameOf', { n: r.number })} bind:value={r.name} data-testid="{testid}-name-{r.number}" /></td>
                        <td><input type="text" list="{testid}-rooms" aria-label={$t('direction.tableProps.roomOf', { n: r.number })} bind:value={r.room} data-testid="{testid}-room-{r.number}" /></td>
                        <td><input type="checkbox" aria-label={$t('direction.tableProps.reservedOf', { n: r.number })} bind:checked={r.reserved} data-testid="{testid}-reserved-{r.number}" /></td>
                        <td class="assignees">
                            {#each r.assignedTo as name (name)}
                                <span class="chip">
                                    {name}
                                    <button type="button" class="x" aria-label={$t('direction.tableProps.unassign', { name })} onclick={() => unassign(r, name)}>&times;</button>
                                </span>
                            {/each}
                            {#if participants.some((p) => !r.assignedTo.includes(p))}
                                <select
                                    aria-label={$t('direction.tableProps.assignOf', { n: r.number })}
                                    data-testid="{testid}-assign-{r.number}"
                                    value=""
                                    onchange={(e) => (assign(r, e.currentTarget.value), (e.currentTarget.value = ''))}
                                >
                                    <option value="">{$t('direction.tableProps.assign')}</option>
                                    {#each participants.filter((p) => !r.assignedTo.includes(p)) as p (p)}
                                        <option value={p}>{p}</option>
                                    {/each}
                                </select>
                            {/if}
                        </td>
                    </tr>
                {/each}
            </tbody>
        </table>
    </div>
    <div class="row">
        <button type="button" class="primary" disabled={!dirty || busy} onclick={save} data-testid="{testid}-save">{$t('direction.tableProps.save')}</button>
        <button type="button" disabled={!dirty || busy} onclick={reset} data-testid="{testid}-reset">{$t('direction.tableProps.reset')}</button>
    </div>
    {#if error}<p class="error" role="alert" data-testid="{testid}-error">{error}</p>{/if}
</section>

<style>
    .table-settings {
        display: flex;
        flex-direction: column;
        gap: 0.4em;
    }
    h4 {
        margin: 0;
    }
    .facts {
        margin: 0;
        color: var(--color-text-muted);
    }
    .bulk,
    .row {
        display: flex;
        flex-wrap: wrap;
        gap: 0.4em 0.8em;
        align-items: center;
    }
    .bulk input[type='number'] {
        width: 4em;
    }
    .scroll {
        max-height: 22em;
        overflow: auto;
        border: 1px solid var(--color-border);
    }
    table {
        border-collapse: collapse;
        width: 100%;
    }
    th,
    td {
        padding: 1px 6px;
        text-align: left;
        vertical-align: middle;
    }
    thead th {
        position: sticky;
        top: 0;
        background: var(--color-surface, var(--color-bg));
    }
    tbody th {
        width: 3em;
        font-weight: 600;
    }
    td input[type='text'] {
        width: 100%;
        min-width: 7em;
    }
    .assignees {
        display: flex;
        flex-wrap: wrap;
        gap: 0.3em;
        align-items: center;
    }
    .chip {
        padding: 0 0.4em;
        border: 1px solid var(--color-border);
        border-radius: 999px;
        font-size: var(--font-size-small);
    }
    .x {
        border: none;
        background: none;
        cursor: pointer;
        color: inherit;
    }
    .error {
        margin: 0;
        color: var(--color-danger);
    }
</style>
