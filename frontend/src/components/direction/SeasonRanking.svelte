<script>
    /*
     * Le classement de saison d'une Rencontre (ADR-0062) : les tournois clos de l'Événement,
     * chaque place convertie en points par un barème, et au choix un Elo de club. Calculé à la
     * demande : il rejoue tous les journaux de l'Événement.
     */
    import { t } from '../../i18n';
    import { seasonRanking, seasonCSV } from '../../stores/rencontreStore.js';

    /** @type {{ rencontreId: number }} */
    let { rencontreId } = $props();

    let points = $state('25,18,15,12,10,8,6,4,2,1');
    let elo = $state(false);
    /** @type {import('../../../wailsjs/go/models').service.SeasonView | null} */
    let view = $state(null);
    let error = $state('');
    let copied = $state(false);

    function query() {
        const scale = points
            .split(',')
            .map((s) => s.trim())
            .filter((s) => s !== '')
            .map(Number)
            .filter((n) => Number.isFinite(n));
        return { rencontreId, points: scale, elo };
    }

    async function compute() {
        error = '';
        copied = false;
        try {
            view = await seasonRanking(query());
        } catch (e) {
            view = null;
            error = String(e);
        }
    }

    async function copy() {
        try {
            await navigator.clipboard.writeText(await seasonCSV(query()));
            copied = true;
        } catch (e) {
            error = String(e);
        }
    }
</script>

<fieldset class="season" data-testid="season-ranking">
    <legend title={$t('direction.season.hint')}>{$t('direction.season.title')}</legend>
    <p class="facts">{$t('direction.season.hint')}</p>
    <div class="row">
        <label>{$t('direction.season.points')} <input type="text" data-testid="season-points" bind:value={points} /></label>
        <label><input type="checkbox" data-testid="season-elo" bind:checked={elo} /> {$t('direction.season.elo')}</label>
        <button type="button" data-testid="season-compute" onclick={compute}>{$t('direction.season.compute')}</button>
        {#if view && view.rows.length}
            <button type="button" data-testid="season-copy" onclick={copy}>{copied ? $t('direction.season.copied') : $t('direction.season.copy')}</button>
        {/if}
    </div>
    {#if view}
        {#if view.rows.length}
            <table data-testid="season-table">
                <thead>
                    <tr>
                        <th>{$t('direction.season.rank')}</th>
                        <th>{$t('direction.season.player')}</th>
                        <th>{$t('direction.season.total')}</th>
                        <th>{$t('direction.season.played')}</th>
                        {#if view.elo}<th>Elo</th>{/if}
                    </tr>
                </thead>
                <tbody>
                    {#each view.rows as r (r.name)}
                        <tr>
                            <td>{r.rank}</td>
                            <td>{r.name}{r.club ? ` (${r.club})` : ''}</td>
                            <td>{r.total}</td>
                            <td>{r.played}</td>
                            {#if view.elo}<td>{r.elo}</td>{/if}
                        </tr>
                    {/each}
                </tbody>
            </table>
        {:else}
            <p class="facts" data-testid="season-empty">{$t('direction.season.empty')}</p>
        {/if}
    {/if}
    {#if error}<p class="error">{error}</p>{/if}
</fieldset>

<style>
    .row {
        display: flex;
        flex-wrap: wrap;
        gap: 0.4em;
        align-items: center;
    }
    .facts {
        color: var(--color-text-muted);
    }
    .error {
        color: var(--color-danger);
    }
    table {
        border-collapse: collapse;
    }
    th,
    td {
        padding: 0.1em 0.6em;
        text-align: left;
    }
</style>
