<script>
    import { confirmAction } from '../../services/confirmService.js';
    /*
     * L'Événement du tournoi ouvert (ADR-0056, ADR-0058) : le rattacher, l'en détacher, le
     * supprimer, y déclarer une table hors service et les propriétés de ses tables — une fois,
     * pour toutes les épreuves — et choisir les salles où joue cette épreuve. Rattacher montre
     * d'abord ce qui va changer : les tables de l'épreuve deviennent celles de l'Événement.
     */
    import { untrack } from 'svelte';
    import { t } from '../../i18n';
    import { renderConfigChange } from './labels.js';
    import TableSettingsEditor from './TableSettingsEditor.svelte';
    import SeasonRanking from './SeasonRanking.svelte';
    import { rencontreParticipantsStore } from '../../stores/directionStore.js';
    import {
        listRencontres,
        createRencontre,
        previewAttach,
        attachToRencontre,
        detachFromRencontre,
        trashRencontre,
        setTableOutOfService,
        chooseRencontreOutputDir,
        forgetRencontreOutputDir,
        writeRencontrePage,
        setRencontreTables,
        setEventRooms
    } from '../../stores/rencontreStore.js';
    import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';

    /** @typedef {import('../../../wailsjs/go/models').service.RencontreView} RencontreView */
    /** @typedef {import('../../../wailsjs/go/models').service.ConfigPreview} ConfigPreview */

    /** @type {{ tournamentId: number, rencontreId?: number }} */
    let { tournamentId, rencontreId = 0 } = $props();

    /** @type {RencontreView[]} */
    let rencontres = $state([]);
    let chosen = $state(0);
    let name = $state('');
    let tables = $state(8);
    let startsOn = $state('');
    let endsOn = $state('');
    /** @type {ConfigPreview | null} */
    let preview = $state(null);
    let error = $state('');
    // Replié à l'arrivée sur une épreuve déjà rattachée ; rattacher ne le referme pas sous la main.
    let expanded = $state(untrack(() => rencontreId === 0));

    const current = $derived(rencontres.find((r) => r.id === rencontreId) || null);
    const others = $derived(rencontres.filter((r) => r.id !== rencontreId));

    async function reload() {
        rencontres = await listRencontres();
        if (!chosen && rencontres.length) chosen = rencontres[0].id;
    }

    $effect(() => {
        // Relu à chaque changement de rattachement.
        void rencontreId;
        reload();
    });

    /** @param {unknown} e */
    function fail(e) {
        error = String(e);
    }

    async function askAttach(id = chosen) {
        error = '';
        try {
            chosen = id;
            preview = await previewAttach(tournamentId, id);
        } catch (e) {
            fail(e);
        }
    }

    async function confirmAttach() {
        try {
            await attachToRencontre(tournamentId, chosen);
            preview = null;
            await reload();
        } catch (e) {
            fail(e);
        }
    }

    async function create() {
        error = '';
        try {
            const r = await createRencontre(name.trim(), startsOn, endsOn, Number(tables));
            name = '';
            await reload();
            await askAttach(r.id);
        } catch (e) {
            fail(e);
        }
    }

    async function detach() {
        try {
            await detachFromRencontre(tournamentId);
            await reload();
        } catch (e) {
            fail(e);
        }
    }

    async function remove() {
        if (!current) return;
        if (!(await confirmAction($t('direction.rencontre.trashConfirm', { name: current.name })))) return;
        try {
            await trashRencontre(current.id);
            await reload();
        } catch (e) {
            fail(e);
        }
    }

    /** @param {number} n @param {boolean} out */
    async function toggle(n, out) {
        if (!current) return;
        try {
            await setTableOutOfService(current.id, n, out);
            await reload();
        } catch (e) {
            fail(e);
        }
    }

    /* Page murale : un dossier choisi une fois (D6.5) ; réécrite ensuite sans aucun geste. */
    async function onChooseOutput() {
        if (!current) return;
        try {
            await chooseRencontreOutputDir(current.id);
            await reload();
        } catch (e) {
            fail(e);
        }
    }

    async function onForgetOutput() {
        if (!current) return;
        try {
            await forgetRencontreOutputDir(current.id);
            await reload();
        } catch (e) {
            fail(e);
        }
    }

    async function onOpenPage() {
        if (!current) return;
        try {
            const path = await writeRencontrePage(current.id);
            if (path) BrowserOpenURL('file://' + path);
        } catch (e) {
            fail(e);
        }
    }

    /** @param {import('../../../wailsjs/go/models').domain.TableSetting[]} settings */
    async function saveTables(settings) {
        if (current) await setRencontreTables(current.id, settings);
        await reload();
    }

    const roomNames = $derived([...new Set((current?.tableSettings || []).map((s) => s.room).filter(Boolean))]);
    const myRooms = $derived(current?.eventRooms?.[tournamentId] || []);

    /** @param {string} room @param {boolean} on */
    async function toggleRoom(room, on) {
        if (!current) return;
        const next = on ? [...myRooms, room] : myRooms.filter((r) => r !== room);
        try {
            error = '';
            await setEventRooms(current.id, tournamentId, next);
            await reload();
        } catch (e) {
            fail(String(e instanceof Error ? e.message : e).replace(/^direction:\s*/, ''));
        }
    }

    const tableNumbers = $derived(current ? Array.from({ length: current.room?.tables || current.tables }, (_, i) => i + 1) : []);
</script>

<!-- Replié sur son titre une fois l'épreuve rattachée : en tête des Réglages, il reste visible sans défilement. -->
<details class="rencontre" data-testid="rencontre-panel" bind:open={expanded}>
    <summary data-testid="rencontre-summary">
        <h3>{$t('direction.rencontre.title')}</h3>
        {#if current}<span class="facts">&middot; {current.name}</span>{/if}
    </summary>
    <p class="facts">{$t('direction.rencontre.hint')}</p>
    {#if current}
        <p data-testid="rencontre-current">
            <strong>{current.name}</strong> &middot; {$t('direction.rencontre.tableCount', { n: current.tables })} &middot;
            {$t('direction.rencontre.members', { names: (current.members || []).map((m) => m.name).join(', ') })}
        </p>
        <fieldset class="out-of-service">
            <legend title={$t('direction.rencontre.outOfServiceHint')}>{$t('direction.rencontre.outOfService')}</legend>
            {#each tableNumbers as n (n)}
                <label>
                    <input type="checkbox" data-testid="rencontre-out-{n}" checked={(current.room?.unavailable || []).includes(n)} onchange={(e) => toggle(n, e.currentTarget.checked)} />
                    {n}
                </label>
            {/each}
        </fieldset>
        {#if roomNames.length}
            <fieldset class="rooms" data-testid="rencontre-rooms">
                <legend title={$t('direction.rooms.hint')}>{$t('direction.rooms.title')}</legend>
                {#each roomNames as room (room)}
                    <label>
                        <input type="checkbox" data-testid="rencontre-room-{room}" checked={myRooms.includes(room)} onchange={(e) => toggleRoom(room, e.currentTarget.checked)} />
                        {room}
                    </label>
                {/each}
                <p class="facts">{$t('direction.rooms.hint')}</p>
            </fieldset>
        {/if}
        <TableSettingsEditor testid="rencontre-tables-editor" tables={tableNumbers.length} settings={current.tableSettings || []} participants={$rencontreParticipantsStore} onSave={saveTables} />
        <fieldset class="display">
            <legend>{$t('direction.display.title')}</legend>
            <p class="facts">{$t('direction.display.hint')}</p>
            {#if current.outputDir}
                <p class="facts path">{current.outputDir}</p>
            {/if}
            <div class="row">
                <button type="button" data-testid="rencontre-output" onclick={onChooseOutput}>
                    {current.outputDir ? $t('direction.display.changeFolder') : $t('direction.display.chooseFolder')}
                </button>
                {#if current.outputDir}
                    <button type="button" data-testid="rencontre-open-page" onclick={onOpenPage}>{$t('direction.display.open')}</button>
                    <button type="button" class="link" data-testid="rencontre-forget-output" onclick={onForgetOutput}>{$t('direction.display.forget')}</button>
                {/if}
            </div>
        </fieldset>
        <SeasonRanking rencontreId={current.id} />
        <div class="row">
            <button type="button" data-testid="rencontre-detach" onclick={detach}>{$t('direction.rencontre.detach')}</button>
            <button type="button" data-testid="rencontre-trash" onclick={remove}>{$t('direction.rencontre.trash')}</button>
        </div>
    {:else}
        {#if others.length}
            <div class="row">
                <select data-testid="rencontre-choose" bind:value={chosen}>
                    {#each others as r (r.id)}
                        <option value={r.id}>{r.name}</option>
                    {/each}
                </select>
                <button type="button" data-testid="rencontre-attach" onclick={() => askAttach()}>{$t('direction.rencontre.attach')}</button>
            </div>
        {/if}
        <div class="row">
            <input type="text" data-testid="rencontre-name" placeholder={$t('direction.rencontre.name')} bind:value={name} />
            <label>{$t('direction.rencontre.tables')} <input type="number" min="1" data-testid="rencontre-tables" bind:value={tables} /></label>
            <input type="date" aria-label={$t('direction.rencontre.startsOn')} bind:value={startsOn} />
            <input type="date" aria-label={$t('direction.rencontre.endsOn')} bind:value={endsOn} />
            <button type="button" data-testid="rencontre-create" disabled={!name.trim() || tables < 1} onclick={create}>{$t('direction.rencontre.create')}</button>
        </div>
    {/if}
    {#if preview}
        <div class="confirm" data-testid="rencontre-attach-confirm">
            <p>{$t('direction.rencontre.confirmTitle')}</p>
            {#if (preview.changes || []).length}
                <ul>
                    {#each preview.changes || [] as change, i (i)}
                        <li>{renderConfigChange($t, change)}</li>
                    {/each}
                </ul>
            {:else}
                <p class="facts">{$t('direction.rencontre.noChange')}</p>
            {/if}
            <div class="row">
                <button type="button" class="primary" data-testid="rencontre-attach-yes" onclick={confirmAttach}>{$t('direction.rencontre.confirm')}</button>
                <button type="button" onclick={() => (preview = null)}>{$t('common.cancel')}</button>
            </div>
        </div>
    {/if}
    {#if error}<p class="error">{error}</p>{/if}
</details>

<style>
    .rencontre {
        padding: 0 0.75rem;
    }
    .rencontre[open] {
        display: flex;
        flex-direction: column;
        gap: 0.4em;
    }
    .rencontre > summary {
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 0.5em;
        min-height: 24px;
    }
    .rencontre > summary h3 {
        display: inline;
        margin: 0;
    }
    /* Cibles d'au moins 24 px, y compris celles des éditeurs enfants. */
    .rencontre :global(button),
    .rencontre :global(select),
    .rencontre :global(input:not([type='checkbox'])) {
        min-height: 24px;
        box-sizing: border-box;
    }
    .rencontre :global(input[type='checkbox']) {
        width: 24px;
        height: 24px;
    }
    .rencontre :global(label) {
        display: inline-flex;
        align-items: center;
        gap: 0.3em;
        min-height: 24px;
    }
    .row {
        display: flex;
        flex-wrap: wrap;
        gap: 0.4em;
        align-items: center;
    }
    .rencontre input[type='number'] {
        width: 4em;
    }
    .out-of-service,
    .rooms {
        display: flex;
        flex-wrap: wrap;
        gap: 0.2em 0.8em;
    }
    .facts {
        color: var(--text-muted, #666);
    }
    .error {
        color: var(--error-color, #b00020);
    }
</style>
