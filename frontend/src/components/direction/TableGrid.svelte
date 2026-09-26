<script>
    /*
     * La grille des tables (tasks/nicomaque/ux.md §2.2), lisible à deux mètres ; le temps
     * écoulé passe en alerte quand un match traîne. La fiche de résultat s'ouvre sur la case.
     * Un match sans table a sa case après la salle : plusieurs cases portent la table 0, d'où
     * une clé par match.
     */
    import { t } from '../../i18n';
    import ResultCard from './ResultCard.svelte';

    /** @typedef {import('../../../wailsjs/go/models').database.TableCell} TableCell */

    /**
     * @type {{
     *     cells?: TableCell[],
     *     busy?: boolean,
     *     onResult?: (matchId: string, winner: string, scoreA: number, scoreB: number, note: string) => void,
     *     onForfeit?: (matchId: string, winner: string, note: string) => void,
     *     onMove?: (matchId: string, table: number) => void,
     *     onCancel?: (matchId: string) => void,
     *     actions?: import('svelte').Snippet
     * }}
     */
    let { cells = [], busy = false, onResult = () => {}, onForfeit = () => {}, onMove = () => {}, onCancel = () => {}, actions } = $props();

    let openKey = $state('');

    /** @param {TableCell} c */
    function key(c) {
        return c.noTable ? `m:${c.matchId}` : `t:${c.table}`;
    }

    /** Le titre compte les tables de la salle, pas les matchs qui n'en ont pas. */
    let tableCount = $derived(cells.filter((c) => !c.noTable).length);

    /**
     * Le temps écoulé d'un match. Zéro seconde est omis par le backend (`omitempty`) : un match
     * qui vient d'être lancé arrive sans `elapsedSeconds`, d'où le `|| 0` à l'appel.
     *
     * @param {number} seconds
     */
    function elapsed(seconds) {
        const h = Math.floor(seconds / 3600);
        const m = Math.floor((seconds % 3600) / 60);
        return h > 0 ? `${h} h ${String(m).padStart(2, '0')}` : `${m} min`;
    }

    /**
     * La fiche ne s'ouvre que sur une table où un match est en cours : ses deux joueurs et son
     * identifiant sont là. Rend la case elle-même, typée comme telle.
     *
     * @param {TableCell} c
     * @returns {TableCell & { matchId: string, a: string, b: string }}
     */
    function runningCell(c) {
        return /** @type {TableCell & { matchId: string, a: string, b: string }} */ (c);
    }

    /** @param {TableCell} c */
    function label(c) {
        if (c.unavailable) return $t('direction.table.unavailable');
        if (c.reserved) return $t('direction.table.reserved');
        return $t('direction.table.free');
    }
</script>

<section class="grid-wrap">
    <header>
        <h3>{$t('direction.table.title', { n: tableCount })}</h3>
        {#if actions}{@render actions()}{/if}
    </header>
    <div class="grid">
        {#each cells as c (key(c))}
            <div class="cell-wrap">
                <button
                    type="button"
                    class="cell"
                    data-testid={c.noTable ? `direction-table-none-${c.matchId}` : `direction-table-${c.table}`}
                    class:busy={c.matchId}
                    class:slow={c.slow}
                    class:idle={!c.matchId}
                    class:no-table={c.noTable}
                    disabled={!c.matchId}
                    onclick={() => (openKey = openKey === key(c) ? '' : key(c))}
                >
                    <span class="num">{c.noTable ? $t('direction.table.noTable') : c.table}</span>
                    {#if c.matchId}
                        <span class="players">{c.aName} – {c.bName}</span>
                        <span class="meta">
                            {$t('direction.proposals.points', { n: c.length })} &middot;
                            {elapsed(c.elapsedSeconds || 0)}
                            {#if c.slow}
                                &middot; {$t('direction.table.slow')}
                            {/if}
                        </span>
                    {:else}
                        <span class="state">{label(c)}</span>
                    {/if}
                </button>

                {#if openKey === key(c) && c.matchId}
                    <ResultCard cell={runningCell(c)} {busy} onClose={() => (openKey = '')} {onResult} {onForfeit} {onMove} {onCancel} />
                {/if}
            </div>
        {/each}
        {#if cells.length === 0}
            <p class="empty">{$t('direction.table.none')}</p>
        {/if}
    </div>
</section>

<style>
    .grid-wrap {
        padding: var(--space-2);
        min-width: 0;
    }

    header {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: var(--space-2);
        margin: 0 0 var(--space-1);
    }

    h3 {
        margin: 0;
        font-size: var(--font-size-base);
        font-weight: 600;
        color: var(--color-text);
    }

    .grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
        gap: var(--space-1);
    }

    /* Sous 900 px la grille devient une colonne : les cases redeviennent des lignes, comme le
       document d'ergonomie le prévoit. */
    @media (max-width: 900px) {
        .grid {
            grid-template-columns: 1fr;
        }
    }

    .cell-wrap {
        position: relative;
    }

    .cell {
        display: flex;
        flex-direction: column;
        align-items: flex-start;
        gap: 2px;
        width: 100%;
        min-height: 62px;
        padding: var(--space-1) var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        text-align: left;
        cursor: pointer;
    }

    .cell.idle {
        background: var(--color-surface-alt);
        cursor: default;
    }

    .cell.busy {
        border-color: var(--color-primary);
    }

    /* Sans table, le match reste une case pleine ; seul le cadre en pointillé dit qu'il attend
       qu'on lui en donne une (« Changer de table » dans sa fiche). */
    .cell.no-table {
        border-style: dashed;
    }

    /* Un match lent se voit sans qu'on le cherche, et sans clignoter : la couleur d'alerte et
       le temps en gras suffisent. */
    .cell.slow {
        border-color: var(--color-danger);
    }

    .cell.slow .meta {
        color: var(--color-danger);
        font-weight: 600;
    }

    .num {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .players {
        font-weight: 600;
    }

    .meta,
    .state {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .empty {
        margin: 0;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }
</style>
