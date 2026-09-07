<script>
    /*
     * La vue Classement (issue #374, fonctionnel.md §5.6 et §3.6).
     *
     * La dernière chose que fait un directeur, et la seule que les joueurs emportent.
     *
     * Deux règles du moteur tiennent ici et ne sont pas à cette couche de les adoucir : il n'y a
     * PAS de départage, donc les ex æquo restent ex æquo et se partagent les prix des places
     * qu'ils occupent ; et une note de classement est un code, rendu par l'interface.
     *
     * Une section qui a ses propres prix a son propre classement : le vainqueur de la consolante
     * n'est pas le finaliste du tournoi, et le payer sur le classement général le laisserait
     * entendre.
     */
    import { t } from '../../i18n';
    import { renderNote, renderSectionName } from './labels.js';

    let { view = null, busy = false, onClose = () => {}, onReopen = () => {}, onCSV = () => {} } = $props();

    function money(v) {
        if (!v) return '';
        return v.toLocaleString(undefined, { maximumFractionDigits: 0 });
    }
</script>

{#if view}
    <div class="standings">
        <header>
            {#if view.pool}
                <span class="pool">
                    {$t('direction.standings.pool', {
                        pool: money(view.pool),
                        payable: money(view.payable),
                        retained: money(view.retained)
                    })}
                </span>
            {/if}
            <span class="grow"></span>
            <button type="button" onclick={onCSV}>{$t('direction.standings.csv')}</button>
            {#if view.finished}
                <button type="button" disabled={busy} onclick={onReopen}>{$t('direction.standings.reopen')}</button>
            {:else}
                <button type="button" class="primary" disabled={busy} onclick={onClose}>{$t('direction.standings.close')}</button>
            {/if}
        </header>

        {#each view.sections as s, si (s.name || si)}
            <section>
                <h4>
                    {s.name ? renderSectionName($t, s.name) : $t('direction.standings.overall')}
                </h4>
                <table>
                    <thead>
                        <tr>
                            <th class="rank">#</th>
                            <th>{$t('direction.players.name')}</th>
                            <th>{$t('direction.players.club')}</th>
                            <th>{$t('direction.standings.note')}</th>
                            {#if s.rows.some((r) => r.prize)}
                                <th class="prize">{$t('direction.standings.prize')}</th>
                            {/if}
                        </tr>
                    </thead>
                    <tbody>
                        {#each s.rows as r (r.id)}
                            <tr>
                                <td class="rank">
                                    {r.rank}{#if r.shared}<span class="tied" title={$t('direction.standings.tiedHint')}>=</span>{/if}
                                </td>
                                <td class="name">{r.name}</td>
                                <td class="muted">{r.club || ''}</td>
                                <td class="muted">{renderNote($t, r.note)}</td>
                                {#if s.rows.some((x) => x.prize)}
                                    <td class="prize">{money(r.prize)}</td>
                                {/if}
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </section>
        {/each}
    </div>
{/if}

<style>
    .standings {
        display: flex;
        flex-direction: column;
        gap: var(--space-2);
        padding: var(--space-2);
        min-width: 0;
    }

    header {
        display: flex;
        align-items: center;
        gap: var(--space-1);
        flex-wrap: wrap;
    }

    .pool,
    .muted {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .grow {
        flex: 1;
    }

    h4 {
        margin: 0 0 var(--space-1);
        font-size: var(--font-size-small);
        font-weight: 600;
        color: var(--color-text-muted);
    }

    table {
        border-collapse: collapse;
        width: 100%;
        font-size: var(--font-size-small);
    }

    th {
        text-align: left;
        font-weight: 600;
        color: var(--color-text-muted);
        border-bottom: 1px solid var(--color-border);
        padding: 0.2rem 0.4rem;
    }

    td {
        padding: 0.15rem 0.4rem;
        border-bottom: 1px solid var(--color-surface-alt);
    }

    td.name {
        font-weight: 600;
    }

    .rank {
        width: 3rem;
        text-align: right;
    }

    /* Un ex æquo est un FAIT du résultat, pas un artefact d'arrondi : la marque le dit. */
    .tied {
        color: var(--color-text-muted);
        margin-left: 1px;
    }

    .prize {
        text-align: right;
        white-space: nowrap;
    }

    button {
        padding: 0.15rem 0.6rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
        font-size: var(--font-size-small);
    }

    button.primary {
        border-color: var(--color-primary);
    }

    button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
</style>
