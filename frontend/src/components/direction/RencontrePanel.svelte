<script>
    /*
     * La Rencontre du tournoi ouvert (ADR-0056) : la rattacher à une salle, l'en détacher,
     * supprimer la salle, et y déclarer une table hors service — une fois, pour toutes les
     * épreuves. Rattacher montre d'abord ce qui va changer : les tables de l'épreuve deviennent
     * celles de la salle.
     */
    import { t } from '../../i18n';
    import { renderConfigChange } from './labels.js';
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
        writeRencontrePage
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
        if (!window.confirm($t('direction.rencontre.trashConfirm', { name: current.name }))) return;
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

    const tableNumbers = $derived(current ? Array.from({ length: current.room?.tables || current.tables }, (_, i) => i + 1) : []);
</script>

<section class="rencontre" data-testid="rencontre-panel">
    <h3>{$t('direction.rencontre.title')}</h3>
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
</section>

<style>
    .rencontre {
        display: flex;
        flex-direction: column;
        gap: 0.4em;
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
    .out-of-service {
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
