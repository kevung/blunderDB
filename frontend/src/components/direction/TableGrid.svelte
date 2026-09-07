<script>
    /*
     * La grille des tables (issue #371, tasks/nicomaque/ux.md §2.2).
     *
     * C'est ce que le directeur regarde le plus souvent dans une salle, et la contrainte de
     * dessin vient de là : une case se lit de deux mètres. D'où des cases larges, les deux noms
     * en évidence, et le temps écoulé en petit — sauf quand le match traîne, où il passe en
     * alerte avant qu'un joueur vienne se plaindre.
     *
     * Un clic sur une case ouvre la fiche de résultat EN SURIMPRESSION SUR LA CASE, pas au
     * centre de l'écran : le regard ne quitte pas la grille.
     */
    import { t } from '../../i18n';
    import ResultCard from './ResultCard.svelte';

    let { cells = [], busy = false, onResult = () => {}, onForfeit = () => {}, onMove = () => {}, onCancel = () => {} } = $props();

    let openTable = $state(0);

    function elapsed(seconds) {
        const h = Math.floor(seconds / 3600);
        const m = Math.floor((seconds % 3600) / 60);
        return h > 0 ? `${h} h ${String(m).padStart(2, '0')}` : `${m} min`;
    }

    function label(c) {
        if (c.unavailable) return $t('direction.table.unavailable');
        if (c.reserved) return $t('direction.table.reserved');
        return $t('direction.table.free');
    }
</script>

<section class="grid-wrap">
    <h3>{$t('direction.table.title', { n: cells.length })}</h3>
    <div class="grid">
        {#each cells as c (c.table)}
            <div class="cell-wrap">
                <button
                    type="button"
                    class="cell"
                    class:busy={c.matchId}
                    class:slow={c.slow}
                    class:idle={!c.matchId}
                    disabled={!c.matchId}
                    onclick={() => (openTable = openTable === c.table ? 0 : c.table)}
                >
                    <span class="num">{c.table}</span>
                    {#if c.matchId}
                        <span class="players">{c.aName} – {c.bName}</span>
                        <span class="meta">
                            {$t('direction.proposals.points', { n: c.length })} &middot;
                            {elapsed(c.elapsedSeconds)}
                            {#if c.slow}
                                &middot; {$t('direction.table.slow')}
                            {/if}
                        </span>
                    {:else}
                        <span class="state">{label(c)}</span>
                    {/if}
                </button>

                {#if openTable === c.table && c.matchId}
                    <ResultCard cell={c} {busy} onClose={() => (openTable = 0)} {onResult} {onForfeit} {onMove} {onCancel} />
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

    h3 {
        margin: 0 0 var(--space-1);
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
