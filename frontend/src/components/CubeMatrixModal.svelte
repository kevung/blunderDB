<script>
    // La matrice du videau (#267, fiche I.11).
    //
    // Une décision de videau n'est pas une propriété du damier : les mêmes
    // pions, à 2-away/4-away, se doublent, et à 4-away/2-away ne se doublent
    // pas. blunderDB n'a jamais montré que la case que la position portait ;
    // cette fenêtre montre la grille entière.
    //
    // Deux paliers, comme l'escalade du panneau Eval (#125) : 0-ply au geste
    // — quelques millisecondes, la forme de la réponse est là tout de suite —
    // puis la profondeur d'affichage configurée une fois la fenêtre au repos.
    // Un balayage supplanté est annulé, jamais affiché.
    import Modal from './Modal.svelte';
    import { ComputeCubeMatrix, CancelCubeMatrix } from '../../wailsjs/go/gui/App.js';
    import { GetGammonNetDisplayPly, GetGammonNetPruneK } from '../../wailsjs/go/main/Config.js';
    import { positionStore } from '../stores/positionStore';
    import { currentScoreCell } from '../utils/cubeDecision.js';
    import { logger } from '../utils/logger.js';
    import { t } from '../i18n';

    let { visible = false, onClose } = $props();

    /** Les trois longueurs que la grille propose. Trois, pas un nombre libre :
     *  la grille se lit à l'œil, et 9×9 est déjà le bord de ce qui tient. */
    const LENGTHS = [5, 7, 9];
    const REST_DELAY_MS = 400;

    let matchLength = $state(7);
    let matrix = $state(null);
    let error = $state('');
    let pending = $state(false);
    /** La profondeur affichée : ce que la grille a réellement coûté, jamais
     *  celle qu'on a demandée. */
    let ply = $state(0);

    let restTimer = null;
    let generation = 0;

    let positionSignature = $derived(JSON.stringify($positionStore ?? null));

    $effect(() => {
        const signature = positionSignature;
        const open = visible;
        const length = matchLength;
        if (!open || !signature) return;
        void length;
        run();
        return () => {
            if (restTimer) clearTimeout(restTimer);
            restTimer = null;
        };
    });

    $effect(() => {
        if (!visible) {
            if (restTimer) clearTimeout(restTimer);
            restTimer = null;
            CancelCubeMatrix().catch(() => {});
        }
    });

    async function run() {
        const pos = $positionStore;
        if (!pos) return;
        const mine = ++generation;
        error = '';
        pending = true;

        let pruneK = 0;
        try {
            pruneK = await GetGammonNetPruneK();
        } catch {
            pruneK = 0;
        }

        try {
            const quick = await ComputeCubeMatrix(pos, matchLength, 0, pruneK);
            if (mine !== generation) return;
            matrix = quick;
            ply = quick.ply;
        } catch (e) {
            if (mine !== generation) return;
            error = String(e);
            pending = false;
            return;
        }

        if (restTimer) clearTimeout(restTimer);
        restTimer = setTimeout(async () => {
            let displayPly;
            try {
                displayPly = await GetGammonNetDisplayPly();
            } catch {
                displayPly = 2;
            }
            if (displayPly <= 0 || mine !== generation) {
                pending = false;
                return;
            }
            try {
                const deep = await ComputeCubeMatrix($positionStore, matchLength, displayPly, pruneK);
                if (mine !== generation) return;
                matrix = deep;
                ply = deep.ply;
            } catch (e) {
                if (mine === generation) logger.error('cube matrix: the deep sweep failed', e);
            } finally {
                if (mine === generation) pending = false;
            }
        }, REST_DELAY_MS);
    }

    /** @param {number} i @param {number} j */
    function cellAt(i, j) {
        return matrix?.cells?.find((c) => c.awayOnRoll === i && c.awayOpponent === j) ?? null;
    }

    /** Les quatre verdicts dans les deux caractères qu'une grille peut se
     *  payer ; la légende est sous la grille plutôt que confiée à la mémoire. */
    function glyph(cell) {
        if (!cell || cell.refused) return '?';
        switch (cell.verdict) {
            case 'double_take':
                return $t('cubeMatrix.glyphDoubleTake');
            case 'double_pass':
                return $t('cubeMatrix.glyphDoublePass');
            case 'too_good':
                return $t('cubeMatrix.glyphTooGood');
            default:
                return $t('cubeMatrix.glyphNoDouble');
        }
    }

    function cellTitle(cell) {
        if (!cell) return '';
        const here = isCurrent(cell.awayOnRoll, cell.awayOpponent) ? `\n${$t('cubeMatrix.currentScore')}` : '';
        if (cell.refused) return cell.reason + here;
        const fmt = (v) => v.toFixed(3);
        return `${cell.awayOnRoll}-away / ${cell.awayOpponent}-away\nND ${fmt(cell.noDouble)}  DT ${fmt(cell.doubleTake)}  DP ${fmt(cell.doublePass)}${here}`;
    }

    let rows = $derived(Array.from({ length: matchLength }, (_, i) => i + 1));

    /** La case du score que la position porte réellement, ou null quand il n'y
     *  en a pas à désigner (money, Crawford, un away au-delà de la grille).
     *  Une mise en évidence, pas une sélection : la case reste ce qu'elle est,
     *  elle est seulement reconnaissable — un cadre et un sigle gras, pas une
     *  couleur de plus, puisque les quatre fonds portent déjà le verdict. */
    let current = $derived(currentScoreCell($positionStore, matchLength));

    /** @param {number} i @param {number} j */
    function isCurrent(i, j) {
        return current !== null && current.awayOnRoll === i && current.awayOpponent === j;
    }
</script>

<Modal open={visible} onclose={onClose} size="auto" closeOnOverlay label={$t('cubeMatrix.title')}>
    <div class="cube-matrix">
        <div class="controls">
            <span class="depth">{pending ? $t('cubeMatrix.computing') : $t('cubeMatrix.depth', { ply })}</span>
            <span class="lengths">
                {#each LENGTHS as len (len)}
                    <button type="button" class:selected={matchLength === len} onclick={() => (matchLength = len)}>
                        {$t('cubeMatrix.points', { n: len })}
                    </button>
                {/each}
            </span>
        </div>

        {#if error}
            <div class="error">{error}</div>
        {:else}
            <table>
                <thead>
                    <tr>
                        <th class="corner" title={$t('cubeMatrix.axes')}></th>
                        {#each rows as j (j)}
                            <th class:axis-current={current?.awayOpponent === j}>{j}</th>
                        {/each}
                    </tr>
                </thead>
                <tbody>
                    {#each rows as i (i)}
                        <tr>
                            <th class:axis-current={current?.awayOnRoll === i}>{i}</th>
                            {#each rows as j (j)}
                                {@const cell = cellAt(i, j)}
                                <td
                                    class="verdict {cell?.refused ? 'refused' : (cell?.verdict ?? 'pending')}"
                                    class:current={isCurrent(i, j)}
                                    aria-current={isCurrent(i, j) ? 'true' : undefined}
                                    title={cellTitle(cell)}
                                >
                                    {glyph(cell)}
                                </td>
                            {/each}
                        </tr>
                    {/each}
                </tbody>
            </table>
            <div class="legend">
                <span><b>{$t('cubeMatrix.glyphNoDouble')}</b> {$t('cubeMatrix.legendNoDouble')}</span>
                <span><b>{$t('cubeMatrix.glyphDoubleTake')}</b> {$t('cubeMatrix.legendDoubleTake')}</span>
                <span><b>{$t('cubeMatrix.glyphDoublePass')}</b> {$t('cubeMatrix.legendDoublePass')}</span>
                <span><b>{$t('cubeMatrix.glyphTooGood')}</b> {$t('cubeMatrix.legendTooGood')}</span>
                <span><b>?</b> {$t('cubeMatrix.legendRefused')}</span>
                {#if current}
                    <span><span class="swatch"></span> {$t('cubeMatrix.currentScore')}</span>
                {/if}
            </div>
            <p class="note">{$t('cubeMatrix.note')}</p>
        {/if}
    </div>
</Modal>

<style>
    .cube-matrix {
        display: flex;
        flex-direction: column;
        gap: 0.6em;
        min-width: 22em;
    }

    .controls {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 1em;
    }

    .depth {
        color: var(--color-text-muted);
    }

    .lengths button {
        margin-left: 0.3em;
        padding: 0.15em 0.6em;
        cursor: pointer;
    }

    .lengths button.selected {
        font-weight: 600;
    }

    table {
        border-collapse: collapse;
    }

    th,
    td {
        border: 1px solid var(--color-border);
        padding: 0.3em 0.45em;
        text-align: center;
        min-width: 2.2em;
    }

    thead th,
    tbody th {
        font-weight: 600;
        color: var(--color-text-muted);
    }

    .corner {
        border: none;
    }

    /* La couleur porte la même information que le sigle, jamais elle seule :
       le sigle reste lisible sans distinguer les teintes. Les quatre fonds
       sont mélangés à partir de la palette unique de style.css (ADR-0031),
       pas de quatre teintes inventées ici. */
    td.no_double {
        background-color: color-mix(in srgb, var(--color-text-muted) 12%, transparent);
    }

    td.double_take {
        background-color: color-mix(in srgb, var(--color-primary) 18%, transparent);
    }

    td.double_pass {
        background-color: color-mix(in srgb, var(--color-danger) 14%, transparent);
    }

    td.too_good {
        background-color: color-mix(in srgb, var(--color-danger) 32%, transparent);
    }

    td.refused,
    td.pending {
        color: var(--color-text-muted);
    }

    /* La case du score courant : un cadre à l'intérieur de la bordure (donc
       sans décaler la grille d'un pixel), le sigle en gras, et les deux
       en-têtes soulignés. Une forme, pas une teinte de plus — la couleur est
       déjà prise par le verdict, et la case doit rester reconnaissable sans
       distinguer les teintes. Le cadre est tracé dans l'encre de la case
       (`currentColor`) plutôt que dans un jeton de couleur : il garde alors
       exactement le contraste du sigle qu'il entoure, quel que soit le thème
       et quelle que soit la surface sur laquelle la fenêtre est peinte. */
    td.current {
        outline: 2px solid currentColor;
        outline-offset: -2px;
        font-weight: 700;
    }

    tbody th.axis-current,
    thead th.axis-current {
        font-weight: 700;
        text-decoration: underline;
    }

    .swatch {
        display: inline-block;
        width: 0.8em;
        height: 0.8em;
        vertical-align: -0.1em;
        border: 2px solid currentColor;
    }

    .legend {
        display: flex;
        flex-wrap: wrap;
        gap: 0.9em;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .note {
        margin: 0;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        max-width: 34em;
    }

    .error {
        color: var(--color-danger);
    }
</style>
