<script>
    /*
     * La grille des tables, lisible à deux mètres ; le temps
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
    import { pastThreshold, canGrab, dropAction } from '../../services/directionDrag.js';
    import { closeOnEscape } from '../../services/escapeService.js';
    import { keepFocus } from '../../utils/keepFocus.js';
    import { confirmAction } from '../../services/confirmService.js';
    import { registerKeys } from '../../services/keyDispatch.js';

    /**
     * Une case de salle (HallView) porte en plus son épreuve : la même grille sert aux deux, et
     * chaque geste reçoit la case pour savoir à quelle épreuve il s'adresse.
     *
     * @typedef {import('../../../wailsjs/go/models').service.TableCell & { tournamentId?: number, event?: string, eventIndex?: number }} TableCell
     */

    /**
     * @type {{
     *     cells?: TableCell[],
     *     busy?: boolean,
     *     onResult?: (matchId: string, winner: string, scoreA: number, scoreB: number, note: string, cell?: TableCell) => any,
     *     onForfeit?: (matchId: string, winner: string, note: string, cell?: TableCell, withdrawLoser?: string) => any,
     *     canWithdraw?: boolean,
     *     onMove?: (matchId: string, table: number, cell?: TableCell) => any,
     *     onCancel?: (matchId: string, cell?: TableCell) => any,
     *     onHistory?: (name: string, cell?: TableCell) => void,
     *     onOutOfService?: (table: number, out: boolean) => void,
     *     onLaunchHere?: (table: number) => void,
     *     reveal?: { table: number, open?: boolean, seq: number } | null,
     *     actions?: import('svelte').Snippet
     * }}
     */
    let {
        cells = [],
        busy = false,
        onResult = () => {},
        onForfeit = () => {},
        onMove = () => {},
        onCancel = () => {},
        onHistory,
        onOutOfService,
        onLaunchHere,
        canWithdraw = false,
        reveal = null,
        actions
    } = $props();

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
            onForfeit: (/** @type {string} */ m, /** @type {string} */ w, /** @type {string} */ n) => onForfeit(m, w, n, c),
            onCancel: (/** @type {string} */ m) => onCancel(m, c),
            onHistory: onHistory ? (/** @type {string} */ name) => onHistory(name, c) : undefined,
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

    /*
     * Glisser une case occupée sur une autre : déplacer le match sur une table libre, échanger
     * les deux sur une table occupée (après confirmation). Événements pointeur, pas l'API HTML5.
     * Le service décide de ce qui est permis ; ici on n'envoie que le geste.
     */
    let drag = $state(/** @type {{ key: string, cell: TableCell, x: number, y: number, startX: number, startY: number, started: boolean, over: string } | null} */ (null));
    /** Le clic qui suit un glissement ne doit pas ouvrir la fiche. */
    let swallowClick = false;

    /** @param {number} x @param {number} y */
    function cellAt(x, y) {
        const el = document.elementFromPoint(x, y)?.closest('.cell');
        const k = el?.getAttribute('data-key');
        return k ? (cells.find((c) => key(c) === k) ?? null) : null;
    }

    /** @param {PointerEvent} e @param {TableCell} c */
    function onCellDown(e, c) {
        if (e.button !== 0 || busy || !canGrab(c)) return;
        drag = { key: key(c), cell: c, x: e.clientX, y: e.clientY, startX: e.clientX, startY: e.clientY, started: false, over: '' };
    }

    /** @param {PointerEvent} e */
    function onDragMove(e) {
        if (!drag) return;
        const started = drag.started || pastThreshold({ x: drag.startX, y: drag.startY }, { x: e.clientX, y: e.clientY });
        const over = started ? cellAt(e.clientX, e.clientY) : null;
        drag = { ...drag, x: e.clientX, y: e.clientY, started, over: over ? key(over) : '' };
    }

    function endDrag() {
        drag = null;
    }

    /** @param {PointerEvent} e */
    async function onDragUp(e) {
        const d = drag;
        drag = null;
        if (!d || !d.started) return;
        swallowClick = true;
        setTimeout(() => (swallowClick = false), 0);
        const target = cellAt(e.clientX, e.clientY);
        const act = dropAction(d.cell, target);
        if (act.kind === 'none') return;
        if (act.kind === 'swap') {
            // Deux épreuves de la salle : le service écrit un changement dans chaque journal, le
            // directeur voit lesquelles avant de dire oui.
            const across = !!target?.event && !!d.cell.event && target.tournamentId !== d.cell.tournamentId;
            const question = across
                ? $t('direction.hall.swapConfirm', { from: act.from, to: act.to, a: d.cell.event ?? '', b: target?.event ?? '' })
                : $t('direction.table.swapConfirm', { from: act.from, to: act.to });
            const ok = await confirmAction(question, { confirmLabel: $t('direction.table.swap') });
            if (!ok) return;
        }
        onMove(act.matchId, act.table, d.cell);
    }

    $effect(() => {
        if (!drag) return;
        window.addEventListener('pointermove', onDragMove);
        window.addEventListener('pointerup', onDragUp);
        window.addEventListener('pointercancel', endDrag);
        const unEsc = closeOnEscape(endDrag);
        return () => {
            window.removeEventListener('pointermove', onDragMove);
            window.removeEventListener('pointerup', onDragUp);
            window.removeEventListener('pointercancel', endDrag);
            unEsc();
        };
    });

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
        // Deux grilles peuvent être montées (l'épreuve et la salle) : seule la visible répond.
        if (!directionPageShown() || somethingOpenAbove() || openKey || !gridEl || gridEl.closest('[hidden]')) return;
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
        const unregister = registerKeys('directionGrid', onDigit);
        return () => {
            unregister();
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
        const ev = c.tournamentId ? `${c.tournamentId}:` : '';
        return c.noTable || c.shared ? `m:${ev}${c.matchId}` : `t:${c.table}`;
    }

    /*
     * Plusieurs salles : les tables se groupent par salle, dans l'ordre de leur première table ;
     * celles sans salle, et les matchs sans table, ferment la grille. Une seule salle, ou aucune,
     * laisse la grille telle quelle.
     */
    const roomOrder = $derived([...new Set(cells.map((c) => c.room).filter((r) => !!r))]);
    const shown = $derived(roomOrder.length > 1 ? [...cells].sort((a, b) => rank(a) - rank(b)) : cells);

    /** @param {TableCell} c */
    function rank(c) {
        const i = c.room ? roomOrder.indexOf(c.room) : -1;
        return i < 0 ? roomOrder.length : i;
    }

    /** Le titre de salle à poser avant cette case, s'il y en a un. @param {number} i */
    function roomTitleAt(i) {
        const c = shown[i];
        if (roomOrder.length < 2 || !c.room) return '';
        return i === 0 || shown[i - 1].room !== c.room ? c.room : '';
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

<section class="grid-wrap" use:keepFocus={() => gridEl?.querySelector('.cell[tabindex="0"]')}>
    <header>
        <h3>{$t('direction.table.title', { n: tableCount })}</h3>
        {#if actions}{@render actions()}{/if}
    </header>
    <div class="grid" role="grid" aria-label={$t('direction.table.title', { n: tableCount })} bind:this={gridEl}>
        {#each shown as c, i (key(c))}
            {#if roomTitleAt(i)}
                <h4 class="room-title" role="presentation" data-testid="direction-room-{roomTitleAt(i)}">{$t('direction.table.room', { name: roomTitleAt(i) })}</h4>
            {/if}
            <div class="cell-wrap" role="row">
                <button
                    type="button"
                    role="gridcell"
                    class="cell"
                    tabindex={(rovingKey && cells.some((x) => key(x) === rovingKey) ? rovingKey === key(c) : c === shown[0]) ? 0 : -1}
                    onfocus={() => (rovingKey = key(c))}
                    data-testid={c.noTable
                        ? `direction-table-none-${c.tournamentId ? c.tournamentId + '-' : ''}${c.matchId}`
                        : c.shared
                          ? `direction-table-shared-${c.tournamentId ? c.tournamentId + '-' : ''}${c.matchId}`
                          : `direction-table-${c.table}`}
                    class:busy={c.matchId}
                    class:slow={c.slow}
                    class:idle={!c.matchId}
                    class:elsewhere={!!c.elsewhere}
                    class:no-table={c.noTable}
                    class:shared={c.shared}
                    class:in-event={!!c.event}
                    style={c.event ? `--ev: var(--td-event-${(c.eventIndex ?? 0) % 6})` : undefined}
                    aria-haspopup={menuItems(c).length > 0 ? 'menu' : undefined}
                    data-key={key(c)}
                    class:grabbed={drag?.started && drag.key === key(c)}
                    class:drop-over={drag?.started && drag.over === key(c) && drag.key !== key(c)}
                    onpointerdown={(e) => onCellDown(e, c)}
                    onclick={() => {
                        if (swallowClick) return;
                        openKey === key(c) ? (openKey = '') : openCard(c);
                    }}
                    oncontextmenu={(e) => onCellContext(e, c)}
                    onkeydown={(e) => onCellKey(e, c)}
                >
                    <span class="num">{c.noTable ? $t('direction.table.noTable') : c.table}</span>
                    {#if c.name}<span class="table-name" data-testid="direction-table-name">{c.name}</span>{/if}
                    {#if c.reserved && !c.unavailable}<span class="mark" title={$t('direction.table.reserved')} data-testid="direction-table-reserved">&#9873;</span>{/if}
                    {#if c.assignedTo?.length}<span class="mark" title={$t('direction.table.assignedTo', { names: c.assignedTo.join(', ') })} data-testid="direction-table-assigned"
                            >&#9733; {c.assignedTo.join(', ')}</span
                        >{/if}
                    {#if c.event}
                        <span class="event-chip" data-testid="hall-event-chip">{c.event}</span>
                    {/if}
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
                        <ResultCard
                            cell={runningCell(c)}
                            {busy}
                            moveRequest={moveFor.key === key(c) ? moveFor.seq : 0}
                            onClose={() => (openKey = '')}
                            onResult={(m, w, a, b, note) => onResult(m, w, a, b, note, c)}
                            {canWithdraw}
                            onForfeit={(m, w, note, wd) => onForfeit(m, w, note, c, wd)}
                            onMove={(m, table) => onMove(m, table, c)}
                            onCancel={(m) => onCancel(m, c)}
                        />
                    </div>
                {/if}
            </div>
        {/each}
        {#if cells.length === 0}
            <p class="empty">{$t('direction.table.none')}</p>
        {/if}
    </div>
</section>

{#if drag?.started}
    <div class="ghost" data-testid="direction-drag-ghost" style="left: {drag.x + 12}px; top: {drag.y + 12}px">
        {drag.cell.noTable ? '' : `${drag.cell.table} · `}{drag.cell.aName} – {drag.cell.bName}
    </div>
{/if}

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
        grid-template-columns: 1fr;
        gap: var(--td-gap, var(--space-1));
    }

    /* Le nombre de colonnes suit la largeur de la zone (le conteneur de défilement de l'onglet),
       non celle de la fenêtre : le panneau latéral la réduit. */
    @container (min-width: 600px) {
        .grid {
            grid-template-columns: repeat(3, 1fr);
        }
    }

    @container (min-width: 900px) {
        .grid {
            grid-template-columns: repeat(4, 1fr);
        }
    }

    @container (min-width: 1300px) {
        .grid {
            grid-template-columns: repeat(6, 1fr);
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
        min-height: calc(2 * var(--td-target, 31px));
        padding: var(--space-1) var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        text-align: left;
        cursor: pointer;
    }

    .cell.grabbed {
        opacity: 0.5;
    }

    /* La case sous le pointeur : prête à recevoir, que ce soit un déplacement ou un échange. */
    .cell.drop-over {
        outline: 2px solid var(--color-primary);
        outline-offset: 1px;
    }

    .ghost {
        position: fixed;
        z-index: 1000;
        pointer-events: none;
        padding: var(--space-1) var(--space-2);
        border: 1px solid var(--color-primary);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        font-weight: 600;
        box-shadow: 0 2px 8px rgb(0 0 0 / 0.25);
    }

    .cell.idle {
        background: var(--color-surface-alt);
        cursor: default;
    }

    .cell.busy {
        border-color: var(--color-primary);
        user-select: none;
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

    .room-title {
        margin: var(--space-2) 0 0;
        font-size: var(--font-size-base);
        font-weight: 600;
    }
    .table-name {
        margin-left: var(--space-1);
        font-weight: 600;
    }
    .mark {
        margin-left: var(--space-1);
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }
    .num {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .cell.in-event {
        border-left: 4px solid var(--ev);
    }

    .event-chip {
        max-width: 100%;
        padding: 0 var(--space-1);
        border-left: 3px solid var(--ev);
        border-radius: 3px;
        background: color-mix(in srgb, var(--ev) 15%, transparent);
        color: var(--color-text);
        font-size: var(--font-size-small);
        font-weight: 600;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
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
