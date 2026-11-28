<script>
    // La planche de vérification : le diagramme d'origine, recadré depuis le fichier, à côté du
    // plateau redessiné dans le même sens ; les champs douteux marqués ; une touche valide, une
    // autre corrige. Chaque verdict est écrit dans le Relevé par Go (VerifyDocumentEntry).
    import { t } from '../i18n';
    import { documentImportStore, sheetCategory } from '../stores/documentImportStore.js';
    import { verifyEntry, setSheetFilter, moveSheet, diagramImage, positionOfXGID } from '../services/documentImportService.js';
    import { renderPositionSVG } from '../services/diagramService.js';
    import { isBareLetter } from '../utils/keys.js';

    /** @type {{ fieldLabel: (f: string) => string, busy?: boolean }} */
    let { fieldLabel, busy = false } = $props();

    const s = $derived($documentImportStore);
    const FILTERS = /** @type {const} */ (['sure', 'to-verify', 'uncertain', 'verified']);
    const counts = $derived.by(() => {
        /** @type {Record<string, number>} */
        const c = { sure: 0, 'to-verify': 0, uncertain: 0, verified: 0 };
        for (const e of s.sheet) c[sheetCategory(e)]++;
        return c;
    });
    const shown = $derived(s.sheet.filter((e) => sheetCategory(e) === s.filter));
    const index = $derived(shown.findIndex((e) => e.rank === s.rank));
    const entry = $derived(index >= 0 ? shown[index] : null);
    // Le plateau redessiné suit le sens de jeu du document : ses plateaux de sortie du côté du
    // jan du joueur du bas.
    const orientation = $derived.by(() => {
        try {
            const doc = JSON.parse(s.releve || '{}').document ?? {};
            const home = doc.defaults?.direction?.home || doc.profile?.home;
            return home === 'left' ? 'left' : 'right';
        } catch {
            return 'right';
        }
    });

    let editing = $state(false);
    let draft = $state('');
    let image = $state('');
    let svg = $state('');
    let input = $state(/** @type {HTMLInputElement|undefined} */ (undefined));

    /** @param {string} f */
    function filterLabel(f) {
        switch (f) {
            case 'sure':
                return $t('documentImport.sheet.filter.sure');
            case 'uncertain':
                return $t('documentImport.sheet.filter.uncertain');
            case 'verified':
                return $t('documentImport.sheet.filter.verified');
            default:
                return $t('documentImport.sheet.filter.to-verify');
        }
    }

    // La vignette de l'entrée montrée ; une réponse arrivée après un changement d'entrée est ignorée.
    $effect(() => {
        const e = entry;
        image = '';
        if (!e) return;
        let live = true;
        diagramImage(e).then((url) => {
            if (live) image = url;
        });
        return () => {
            live = false;
        };
    });

    $effect(() => {
        const xgid = editing ? draft : (entry?.position ?? '');
        const side = orientation;
        let live = true;
        positionOfXGID(xgid).then((pos) => {
            if (!live) return;
            svg = pos ? renderPositionSVG(pos, { width: 360, height: 263, orientation: side }) : '';
        });
        return () => {
            live = false;
        };
    });

    // Changer d'entrée abandonne une correction non écrite ; une erreur rendue par Go, non.
    let shownRank = 0;
    $effect(() => {
        const r = s.rank;
        if (r !== shownRank) {
            shownRank = r;
            editing = false;
        }
    });

    function startEdit() {
        if (!entry) return;
        draft = entry.position ? `XGID=${entry.position}` : 'XGID=';
        editing = true;
        queueMicrotask(() => input?.focus());
    }

    async function writeCorrection() {
        if (!entry) return;
        if (await verifyEntry(entry.rank, true, draft)) editing = false;
    }

    function toggle() {
        if (entry) verifyEntry(entry.rank, !entry.verified);
    }

    /** @param {KeyboardEvent} event */
    export function handleKey(event) {
        if (busy || event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement) return;
        let done = true;
        if (isBareLetter(event, 'v') || event.key === ' ') toggle();
        else if (isBareLetter(event, 'e')) startEdit();
        else if (isBareLetter(event, 'j') || event.key === 'ArrowDown' || event.key === 'ArrowRight') moveSheet(1);
        else if (isBareLetter(event, 'k') || event.key === 'ArrowUp' || event.key === 'ArrowLeft') moveSheet(-1);
        else done = false;
        if (done) {
            event.preventDefault();
            event.stopPropagation();
        }
    }

    /** @param {KeyboardEvent} event */
    function inputKey(event) {
        if (event.key === 'Enter') {
            event.preventDefault();
            writeCorrection();
        } else if (event.key === 'Escape') {
            event.preventDefault();
            event.stopPropagation();
            editing = false;
        }
    }
</script>

<section class="sheet" data-testid="document-sheet">
    <div class="sheet-filters" role="group">
        {#each FILTERS as f (f)}
            <button
                type="button"
                class="sheet-filter {f}"
                class:active={s.filter === f}
                aria-pressed={s.filter === f}
                data-testid="document-sheet-filter-{f}"
                disabled={counts[f] === 0}
                onclick={() => setSheetFilter(f)}>{filterLabel(f)} {counts[f]}</button
            >
        {/each}
    </div>
    {#if entry}
        <div class="sheet-nav">
            <button type="button" data-testid="document-sheet-prev" disabled={index <= 0} onclick={() => moveSheet(-1)} title="K">‹</button>
            <strong data-testid="document-sheet-reference">{entry.reference}</strong>
            <span class="sheet-index">{index + 1} / {shown.length}</span>
            <button type="button" data-testid="document-sheet-next" disabled={index >= shown.length - 1} onclick={() => moveSheet(1)} title="J">›</button>
        </div>
        <div class="sheet-boards">
            <figure class="sheet-origin">
                {#if image}
                    <img src={image} alt={entry.reference} data-testid="document-sheet-image" />
                {:else}
                    <div class="sheet-noimage" data-testid="document-sheet-noimage">—</div>
                {/if}
            </figure>
            <figure class="sheet-read" class:doubt={entry.fields.length > 0 && !entry.verified} data-testid="document-sheet-board">
                <!-- eslint-disable-next-line svelte/no-at-html-tags -- SVG rendu par diagramService depuis une position, sans texte venu du document -->
                {#if svg}{@html svg}{:else}<div class="sheet-noimage">?</div>{/if}
            </figure>
        </div>
        {#if entry.fields.length > 0}
            <p class="sheet-fields" data-testid="document-sheet-fields">
                {#each entry.fields as f, i (f)}<mark class="sheet-field">{fieldLabel(f)}</mark>{i < entry.fields.length - 1 ? ', ' : ''}{/each}
            </p>
        {/if}
        {#if editing}
            <input class="sheet-xgid" type="text" data-testid="document-sheet-xgid" bind:this={input} bind:value={draft} onkeydown={inputKey} spellcheck="false" />
        {/if}
        <div class="sheet-actions">
            <button type="button" class="primary" data-testid="document-sheet-verify" disabled={busy || !entry.position} onclick={toggle}>
                {entry.verified ? $t('documentImport.sheet.unverify') : $t('documentImport.sheet.verify')} <kbd>V</kbd>
            </button>
            {#if editing}
                <button type="button" data-testid="document-sheet-write" disabled={busy} onclick={writeCorrection}>{$t('documentImport.sheet.write')} <kbd>↵</kbd></button>
            {:else}
                <button type="button" data-testid="document-sheet-edit" disabled={busy} onclick={startEdit}>{$t('documentImport.sheet.edit')} <kbd>E</kbd></button>
            {/if}
        </div>
    {/if}
</section>

<style>
    .sheet {
        display: flex;
        flex-direction: column;
        gap: var(--space-2);
    }
    .sheet-filters {
        display: flex;
        flex-wrap: wrap;
        gap: var(--space-1);
    }
    .sheet-filter.active {
        font-weight: 600;
        border-color: var(--color-text);
    }
    .sheet-nav {
        display: flex;
        align-items: center;
        gap: var(--space-2);
    }
    .sheet-index {
        color: var(--color-text-muted);
    }
    .sheet-boards {
        display: flex;
        flex-wrap: wrap;
        gap: var(--space-2);
        align-items: flex-start;
    }
    .sheet-boards figure {
        margin: 0;
        width: 360px;
        max-width: 100%;
    }
    .sheet-origin img {
        width: 100%;
        height: auto;
        display: block;
        border: 1px solid var(--color-border);
    }
    .sheet-read {
        border: 2px solid transparent;
    }
    .sheet-read.doubt {
        border-color: var(--color-danger);
    }
    .sheet-read :global(svg) {
        width: 100%;
        height: auto;
        display: block;
    }
    .sheet-noimage {
        display: flex;
        align-items: center;
        justify-content: center;
        aspect-ratio: 360 / 263;
        border: 1px dashed var(--color-border);
        color: var(--color-text-muted);
    }
    .sheet-fields {
        margin: 0;
    }
    .sheet-field {
        background: none;
        color: var(--color-danger, inherit);
        font-weight: 600;
    }
    .sheet-xgid {
        font-family: var(--font-mono, monospace);
        width: 100%;
    }
    .sheet-actions {
        display: flex;
        gap: var(--space-2);
    }
    kbd {
        opacity: 0.7;
    }
</style>
