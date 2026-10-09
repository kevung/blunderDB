<script>
    import { onMount } from 'svelte';
    import { t } from '../i18n';
    import { logger } from '../utils/logger.js';
    import { ListMETs, ImportMET, SetCurrentMET } from '../../wailsjs/go/database/Database.js';
    import { OpenMETDialog } from '../../wailsjs/go/gui/App.js';

    // The library's match equity table (ADR-0068): Kazaross-XG2 by default,
    // or a gnubg .xml imported here. Choosing one rewrites no analysis.
    let tables = $state([]);
    let busy = $state(false);
    let error = $state('');
    let notice = $state('');

    let currentId = $derived(tables.find((m) => m.current)?.id ?? 0);

    async function load() {
        try {
            tables = (await ListMETs()) || [];
        } catch (e) {
            logger.error('METSettings: list failed', e);
            error = String(e);
        }
    }

    onMount(load);

    async function onChoose(/** @type {Event & { currentTarget: HTMLInputElement }} */ event) {
        const id = Number(event.currentTarget.value);
        error = '';
        try {
            await SetCurrentMET(id);
        } catch (e) {
            logger.error('METSettings: set current failed', e);
            error = String(e);
        }
        await load();
    }

    async function onImport() {
        error = '';
        notice = '';
        busy = true;
        try {
            const path = await OpenMETDialog();
            if (!path) return;
            const table = await ImportMET(path);
            notice = table.id === 0 ? $t('config.metBuiltInImported') : $t('config.metImported', { name: table.name });
            await load();
        } catch (e) {
            logger.error('METSettings: import failed', e);
            error = String(e);
        } finally {
            busy = false;
        }
    }
</script>

<div class="setting-row">
    <label for="config-met-current">{$t('config.metCurrent')}</label>
    <select id="config-met-current" class="setting-select" value={currentId} onchange={onChoose}>
        {#each tables as table (table.id)}
            <option value={table.id}>{table.name}</option>
        {/each}
    </select>
    <button class="secondary-button" disabled={busy} onclick={onImport}>{$t('config.metImport')}</button>
</div>
<p class="setting-note">{$t('config.metNote')}</p>
{#if notice}<p class="setting-note">{notice}</p>{/if}
{#if error}<p class="setting-note met-error">{error}</p>{/if}

<style>
    .met-error {
        color: var(--color-danger);
    }
</style>
