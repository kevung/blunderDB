<script>
    // La planche-contact de la liste parcourue (#287, fiche I.31).
    //
    // Une grille de mini-plateaux, une page à la fois ; choisir une vignette
    // ouvre sa position sur le plateau. Le découpage et le clavier sont dans
    // services/contactSheet.js ; ce composant tient ce qui coûte : charger une
    // page de positions et la dessiner.
    //
    // Performance. Une liste peut compter cinquante mille positions. On ne
    // charge que la page (getPositions, qui ne touche pas au cache du
    // plateau), on ne dessine que la page, et on dessine UNE vignette par
    // tâche : la grille s'affiche tout de suite, les plateaux s'y posent l'un
    // après l'autre, et une page qu'on quitte en cours de dessin s'arrête là.
    // Les vignettes déjà dessinées sont gardées le temps que la planche reste
    // ouverte (bornées), pour qu'un aller-retour de page soit immédiat.
    //
    // Chaque vignette est une <img> au SVG en data: plutôt qu'un SVG en
    // ligne : vingt-quatre SVG dans le document, ce sont des milliers de
    // nœuds de plus et des identifiants two.js qui pourraient se croiser.
    import { tick, untrack } from 'svelte';
    import { get } from 'svelte/store';
    import { SvelteMap } from 'svelte/reactivity';
    import Modal from './Modal.svelte';
    import { positionsStore } from '../stores/positionStore.js';
    import { currentPositionIndexStore } from '../stores/uiStore.js';
    import { renderPositionSVG } from '../services/diagramService.js';
    import { PAGE_SIZE, pageOf, pageCount, pageBounds, columnsFor, targetIndex } from '../services/contactSheet.js';
    import { logger } from '../utils/logger.js';
    import { t } from '../i18n';

    let { visible = false, onClose } = $props();

    /** La taille du dessin. La vignette le réduit (le SVG porte un viewBox). */
    const THUMB_WIDTH = 260;
    const THUMB_HEIGHT = 190;
    /** Largeur minimale d'une vignette et espace entre deux, en pixels. */
    const TILE_MIN = 150;
    const GAP = 8;
    /** Vignettes gardées en mémoire : dix pages. */
    const MAX_IMAGES = PAGE_SIZE * 10;

    const hintId = $props.id();

    let focusIndex = $state(0);
    let gridWidth = $state(0);
    /** @type {HTMLElement | undefined} */
    let gridEl = $state();

    /** id → l'image en data:, ou null quand la position n'existe plus. Absent : pas encore dessinée. */
    const images = new SvelteMap();

    let total = $derived($positionsStore.length);
    let columns = $derived(columnsFor(gridWidth, TILE_MIN, GAP));
    let page = $derived(pageOf(focusIndex));
    let pages = $derived(pageCount(total));
    let bounds = $derived(pageBounds(page, total));
    let tiles = $derived.by(() => {
        const out = [];
        for (let i = bounds.from; i < bounds.to; i++) out.push({ index: i, id: $positionsStore.ids[i] });
        return out;
    });

    // À l'ouverture : la page de la position courante, le focus sur elle.
    $effect(() => {
        if (!visible) return;
        untrack(() => {
            const n = get(positionsStore).length;
            const current = get(currentPositionIndexStore);
            focusIndex = n > 0 ? Math.min(Math.max(0, current), n - 1) : 0;
            images.clear();
        });
        focusTile();
    });

    // Charger puis dessiner la page affichée.
    $effect(() => {
        if (!visible) return;
        const { from, to } = bounds;
        const ids = $positionsStore.ids;
        let cancelled = false;
        (async () => {
            let positions;
            try {
                positions = await positionsStore.getPositions(from, to);
            } catch (e) {
                logger.error('contact sheet: could not load positions:', e);
                return;
            }
            if (cancelled) return;
            const byId = new Map(positions.map((p) => [p.id, p]));
            untrack(() => evict(from, to, ids));
            for (let i = from; i < to; i++) {
                const id = ids[i];
                if (id == null || untrack(() => images.has(id))) continue;
                const position = byId.get(id);
                if (!position) {
                    images.set(id, null);
                    continue;
                }
                // Une vignette par tâche : le reste de l'interface respire.
                await new Promise((resolve) => setTimeout(resolve, 0));
                if (cancelled) return;
                try {
                    const svg = renderPositionSVG(position, { width: THUMB_WIDTH, height: THUMB_HEIGHT, showPipcount: false });
                    images.set(id, `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`);
                } catch (e) {
                    logger.error('contact sheet: could not draw position', id, e);
                    images.set(id, null);
                }
            }
        })();
        return () => {
            cancelled = true;
        };
    });

    /**
     * Oublie les vignettes les plus anciennes au-delà de MAX_IMAGES, jamais
     * celles de la page affichée.
     * @param {number} from
     * @param {number} to
     * @param {(number|null)[]} ids
     */
    function evict(from, to, ids) {
        if (images.size <= MAX_IMAGES) return;
        const keep = new Set(ids.slice(from, to));
        for (const id of [...images.keys()]) {
            if (images.size <= MAX_IMAGES - PAGE_SIZE) break;
            if (!keep.has(id)) images.delete(id);
        }
    }

    // La largeur mesurée à l'ouverture et à chaque redimensionnement de la
    // fenêtre — pas par bind:clientWidth, qui demande un ResizeObserver pour
    // une grille dont seule la fenêtre change la largeur.
    function measure() {
        gridWidth = gridEl?.clientWidth ?? 0;
    }

    async function focusTile() {
        await tick();
        measure();
        /** @type {HTMLElement | null | undefined} */
        const tile = gridEl?.querySelector(`[data-index="${focusIndex}"]`);
        tile?.focus();
    }

    /** @param {number} index */
    function openAt(index) {
        currentPositionIndexStore.set(index);
        onClose?.();
    }

    /** @param {KeyboardEvent} event */
    function handleGridKeydown(event) {
        const next = targetIndex(focusIndex, event, { columns, total });
        if (next === null) return;
        event.preventDefault();
        focusIndex = next;
        focusTile();
    }

    /** @param {number} delta */
    function turnPage(delta) {
        focusIndex = pageBounds(page + delta, total).from;
        focusTile();
    }
</script>

<svelte:window onresize={measure} />

<Modal open={visible} onclose={onClose} size="wide" closeOnOverlay label={$t('contactSheet.title')}>
    <div class="sheet-header">
        <h2 class="sheet-title">{$t('contactSheet.title')}</h2>
        <span class="sheet-count">{$t('contactSheet.count', { n: total })}</span>
        <div class="sheet-pager">
            <button type="button" onclick={() => turnPage(-1)} disabled={page === 0}>{$t('contactSheet.previous')}</button>
            <span>{$t('contactSheet.page', { page: page + 1, pages })}</span>
            <button type="button" onclick={() => turnPage(1)} disabled={page >= pages - 1}>{$t('contactSheet.next')}</button>
        </div>
    </div>
    <p class="sheet-hint" id={hintId}>{$t('contactSheet.hint')}</p>
    <ul
        class="sheet-grid"
        style:grid-template-columns="repeat({columns}, minmax(0, 1fr))"
        style:gap="{GAP}px"
        aria-label={$t('contactSheet.page', { page: page + 1, pages })}
        aria-describedby={hintId}
        bind:this={gridEl}
        onkeydown={handleGridKeydown}
    >
        {#each tiles as tile (tile.index)}
            {@const image = tile.id == null ? null : images.get(tile.id)}
            <li>
                <button
                    type="button"
                    class="tile"
                    class:current={tile.index === $currentPositionIndexStore}
                    data-index={tile.index}
                    tabindex={tile.index === focusIndex ? 0 : -1}
                    aria-label={$t('contactSheet.tile', { n: tile.index + 1, total })}
                    aria-current={tile.index === $currentPositionIndexStore ? 'true' : undefined}
                    onfocus={() => (focusIndex = tile.index)}
                    onclick={() => openAt(tile.index)}
                >
                    {#if image}
                        <img src={image} alt="" width={THUMB_WIDTH} height={THUMB_HEIGHT} />
                    {:else}
                        <span class="tile-placeholder">{image === null ? $t('contactSheet.missing') : ''}</span>
                    {/if}
                    <span class="tile-number">{tile.index + 1}</span>
                </button>
            </li>
        {/each}
    </ul>
</Modal>

<style>
    .sheet-header {
        display: flex;
        flex-wrap: wrap;
        align-items: baseline;
        gap: 0.4em 1em;
        margin-bottom: 0.3em;
    }

    .sheet-title {
        margin: 0;
        font-size: var(--font-size-title);
        font-weight: 600;
    }

    .sheet-count {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .sheet-pager {
        display: flex;
        align-items: baseline;
        gap: 0.4em;
        margin-left: auto;
        font-size: var(--font-size-small);
    }

    .sheet-pager button {
        cursor: pointer;
    }

    .sheet-hint {
        margin: 0 0 0.5em;
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .sheet-grid {
        display: grid;
        list-style: none;
        margin: 0;
        padding: 0;
        max-height: 70vh;
        overflow: auto;
    }

    .tile {
        position: relative;
        display: block;
        width: 100%;
        aspect-ratio: 260 / 190;
        padding: 0;
        border: 2px solid var(--color-border);
        background: var(--color-surface-alt);
        cursor: pointer;
    }

    .tile.current {
        border-color: var(--color-primary);
    }

    .tile:focus-visible {
        outline: 3px solid var(--color-primary);
        outline-offset: 1px;
    }

    .tile img {
        display: block;
        width: 100%;
        height: 100%;
    }

    .tile-placeholder {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100%;
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .tile-number {
        position: absolute;
        top: 2px;
        left: 2px;
        padding: 0 0.3em;
        background: var(--color-surface);
        color: var(--color-text);
        font-size: var(--font-size-small);
        font-weight: 600;
    }
</style>
