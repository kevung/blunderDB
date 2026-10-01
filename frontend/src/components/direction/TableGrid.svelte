<script>
    /*
     * La grille des tables (tasks/nicomaque/ux.md §2.2), lisible à deux mètres ; le temps
     * écoulé passe en alerte quand un match traîne. La fiche de résultat s'ouvre sur la case.
     * Un match sans table a sa case après la salle : plusieurs cases portent la table 0, d'où
     * une clé par match.
     */
    import { t } from '../../i18n';
    import ResultCard from './ResultCard.svelte';
    import ContextMenu from '../ContextMenu.svelte';
    import { seatLabel } from './labels.js';
    import { cellMenu } from '../../services/directionMenus.js';
    import { menuRequest } from '../../services/contextMenuTrigger.js';
    import { directionPageShown, somethingOpenAbove } from '../../services/directionKeys.js';
    import { gridKeyAction, TABLE_DIGIT_DELAY_MS } from '../../services/directionGridKeys.js';

    /** @typedef {import('../../../wailsjs/go/models').service.TableCell} TableCell */

    /**
     * @type {{
     *     cells?: TableCell[],
     *     busy?: boolean,
     *     onResult?: (matchId: string, winner: string, scoreA: number, scoreB: number, note: string) => void,
     *     onForfeit?: (matchId: string, winner: string, note: string) => void,
     *     onMove?: (matchId: string, table: number) => void,
     *     onCancel?: (matchId: string) => void,
     *     onHistory?: (name: string) => void,
     *     onOutOfService?: (table: number, out: boolean) => void,
     *     onLaunchHere?: (table: number) => void,
     *     reveal?: { table: number, open?: boolean, seq: number } | null,
     *     actions?: import('svelte').Snippet
     * }}
     */
    let { cells = [], busy = false, onResult = () => {}, onForfeit = () => {}, onMove = () => {}, onCancel = () => {}, onHistory, onOutOfService, onLaunchHere, reveal = null, actions } = $props();

    let openKey = $state('');
    /** La fiche s'ouvre (ou, déjà ouverte, passe) sur le champ de table quand M, X ou le menu le demandent. */
    let moveFor = $state({ key: '', seq: 0 });
    let moveSeq = 0;
    /** La case qui prend le Tab : une seule, les flèches font le reste (grille ARIA). */
    let rovingKey = $state('');
    let menu = $state(/** @type {import('../../services/contextMenuTrigger.js').MenuRequest | null} */ (null));
    /** @type {HTMLElement | null} */
    let gridEl = $state(null);

    /** @param {TableCell} c @param {boolean} move */
    function openCard(c, move = false) {
        if (!c.matchId) return;
        moveFor = move ? { key: key(c), seq: ++moveSeq } : { key: '', seq: 0 };
        openKey = key(c);
    }

    /** @param {TableCell} c */
    function menuItems(c) {
        return cellMenu((k, p) => $t(k, p), c, {
            busy,
            onLaunchHere,
            openResult: () => openCard(c),
            openMove: () => openCard(c, true),
            onForfeit,
            onCancel,
            onHistory,
            onOutOfService
        });
    }

    /**
     * Menu sur la case (clic droit, Menu, Maj+F10) et clavier de la grille : flèches entre
     * cases, M / X ouvrent le champ de table, Entrée la fiche.
     *
     * @param {KeyboardEvent} e
     * @param {TableCell} c
     */
    function onCellKey(e, c) {
        const req = menuRequest(e, () => menuItems(c));
        if (req) {
            menu = req;
            return;
        }
        if (e.target !== e.currentTarget || !gridEl) return;
        const cellEls = /** @type {HTMLElement[]} */ ([...gridEl.querySelectorAll('.cell')]);
        const action = gridKeyAction(e, cellEls, /** @type {HTMLElement} */ (e.currentTarget), !!c.matchId);
        if (!action) return;
        e.preventDefault();
        e.stopPropagation();
        if (action.focus) action.focus.focus();
        else if (action.move) openCard(c, true);
    }

    /** @param {MouseEvent} e @param {TableCell} c */
    function onCellContext(e, c) {
        const req = menuRequest(e, () => menuItems(c));
        if (req) menu = req;
    }

    /** @param {number} table */
    function cellOfTable(table) {
        return cells.find((c) => !c.noTable && !c.shared && c.table === table);
    }

    /**
     * Mène à la table N : le focus sur sa case, et sa fiche si un match y joue.
     *
     * @param {number} table
     * @param {boolean} open
     */
    function goTo(table, open) {
        const c = cellOfTable(table);
        if (!c || !gridEl) return false;
        /** @type {HTMLElement | null} */ (gridEl.querySelector(`[data-testid="direction-table-${table}"]`))?.focus();
        if (open) openCard(c);
        return true;
    }

    let lastReveal = 0;
    $effect(() => {
        if (reveal && reveal.seq !== lastReveal) {
            lastReveal = reveal.seq;
            const r = reveal;
            // Attend que la grille soit montée (la vue vient de changer d'onglet).
            queueMicrotask(() => goTo(r.table, !!r.open));
        }
    });

    /* Chiffres = table N (un ou deux chiffres, le second dans les 400 ms), comme un numéro de
       canal. En capture pour passer avant le répartiteur ; jamais dans un champ, sous une
       surcouche, ni pendant qu'une fiche est ouverte (ses champs prennent les chiffres). */
    let digits = '';
    /** @type {ReturnType<typeof setTimeout> | undefined} */
    let digitTimer;

    /** @param {KeyboardEvent} e */
    function onDigit(e) {
        if (!directionPageShown() || somethingOpenAbove() || openKey) return;
        const m = /^(?:Digit|Numpad)([0-9])$/.exec(e.code);
        if (!m || e.ctrlKey || e.metaKey || e.altKey || e.shiftKey) return;
        if (e.target instanceof Element && e.target.closest('input, select, textarea, [contenteditable]')) return;
        const tables = cells.filter((c) => !c.noTable && !c.shared).map((c) => c.table);
        // Un chiffre qui ne prolonge pas le numéro en cours repart de zéro : « 1 » puis « 5 »
        // sans table 15 mène à la table 5, pas nulle part.
        let next = digits + m[1];
        if (digits && !tables.some((n) => String(n).startsWith(next))) next = m[1];
        clearTimeout(digitTimer);
        digits = '';
        if (!tables.some((n) => String(n).startsWith(next))) return;
        e.preventDefault();
        e.stopImmediatePropagation();
        const n = parseInt(next, 10);
        const longer = tables.some((t) => t !== n && String(t).startsWith(next));
        if (longer) {
            digits = next;
            digitTimer = setTimeout(() => {
                digits = '';
                goTo(n, true);
            }, TABLE_DIGIT_DELAY_MS);
        } else {
            goTo(n, true);
        }
    }

    $effect(() => {
        window.addEventListener('keydown', onDigit, true);
        return () => {
            window.removeEventListener('keydown', onDigit, true);
            clearTimeout(digitTimer);
        };
    });

    /**
     * Une table partagée par deux matchs (hérité d'un journal ancien) donne deux cases au même
     * numéro : la clé est alors le match, sans quoi l'un des deux disparaîtrait.
     *
     * @param {TableCell} c
     */
    function key(c) {
        return c.noTable || c.shared ? `m:${c.matchId}` : `t:${c.table}`;
    }

    /** Le titre compte les tables de la salle, une seule fois chacune. */
    let tableCount = $derived(new Set(cells.filter((c) => !c.noTable).map((c) => c.table)).size);

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
        if (c.elsewhere) return $t('direction.table.elsewhere', { event: c.elsewhere });
        if (c.reserved) return $t('direction.table.reserved');
        return $t('direction.table.free');
    }
</script>

<section class="grid-wrap">
    <header>
        <h3>{$t('direction.table.title', { n: tableCount })}</h3>
        {#if actions}{@render actions()}{/if}
    </header>
    <div class="grid" role="grid" aria-label={$t('direction.table.title', { n: tableCount })} bind:this={gridEl}>
        {#each cells as c (key(c))}
            <div class="cell-wrap" role="row">
                <button
                    type="button"
                    role="gridcell"
                    class="cell"
                    tabindex={(rovingKey && cells.some((x) => key(x) === rovingKey) ? rovingKey === key(c) : c === cells[0]) ? 0 : -1}
                    onfocus={() => (rovingKey = key(c))}
                    data-testid={c.noTable ? `direction-table-none-${c.matchId}` : c.shared ? `direction-table-shared-${c.matchId}` : `direction-table-${c.table}`}
                    class:busy={c.matchId}
                    class:slow={c.slow}
                    class:idle={!c.matchId}
                    class:elsewhere={!!c.elsewhere}
                    class:no-table={c.noTable}
                    class:shared={c.shared}
                    aria-haspopup={menuItems(c).length > 0 ? 'menu' : undefined}
                    onclick={() => (openKey === key(c) ? (openKey = '') : openCard(c))}
                    oncontextmenu={(e) => onCellContext(e, c)}
                    onkeydown={(e) => onCellKey(e, c)}
                >
                    <span class="num">{c.noTable ? $t('direction.table.noTable') : c.table}</span>
                    {#if c.matchId}
                        <span class="players">{c.aName} – {c.bName}</span>
                        {#if c.aElsewhere || c.bElsewhere}
                            <!-- Un appariement à la main d'un joueur qui joue aussi à côté : accepté, signalé. -->
                            <span class="meta seat" data-testid="direction-table-seat">
                                {#if c.aElsewhere}{c.aName} {seatLabel($t, c.aElsewhere)}{/if}
                                {#if c.aElsewhere && c.bElsewhere}&middot;{/if}
                                {#if c.bElsewhere}{c.bName} {seatLabel($t, c.bElsewhere)}{/if}
                            </span>
                        {/if}
                        <span class="meta">
                            {$t('direction.proposals.points', { n: c.length })} &middot;
                            {elapsed(c.elapsedSeconds || 0)}
                            {#if c.slow}
                                &middot; {$t('direction.table.slow')}
                            {/if}
                        </span>
                        {#if c.shared}
                            <span class="meta conflict" data-testid="direction-table-conflict">{$t('direction.table.shared')}</span>
                        {/if}
                    {:else}
                        <span class="state">{label(c)}</span>
                    {/if}
                </button>

                {#if openKey === key(c) && c.matchId}
                    <div role="gridcell">
                        <ResultCard cell={runningCell(c)} {busy} moveRequest={moveFor.key === key(c) ? moveFor.seq : 0} onClose={() => (openKey = '')} {onResult} {onForfeit} {onMove} {onCancel} />
                    </div>
                {/if}
            </div>
        {/each}
        {#if cells.length === 0}
            <p class="empty">{$t('direction.table.none')}</p>
        {/if}
    </div>
</section>

{#if menu}
    <ContextMenu x={menu.x} y={menu.y} items={menu.items} onClose={() => (menu = null)} />
{/if}

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

    /* Occupée par une épreuve sœur de la Rencontre : pas libre ici, sans être à nous. */
    .cell.elsewhere {
        border-style: dashed;
        color: var(--color-text-muted, inherit);
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

    /* Deux matchs sur une table : chacun garde sa case, et la case dit qu'il faut en déplacer un. */
    .cell.shared {
        border-color: var(--color-danger);
    }

    .cell .meta.conflict,
    .cell .meta.seat {
        color: var(--color-danger);
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
