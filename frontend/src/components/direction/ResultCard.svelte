<script>
    /*
     * La fiche de résultat (issue #371, tasks/nicomaque/ux.md §2.2).
     *
     * Le geste que le directeur répète le plus, et son budget est de DEUX clics : la case, puis
     * le nom du vainqueur. Cliquer un nom VALIDE — il n'y a pas de bouton à chercher ensuite.
     *
     * Le score est facultatif : les deux, ou aucun. Le forfait, la remarque, le changement de
     * table et l'annulation sont dans un menu discret, parce qu'ils servent une fois par
     * tournoi et ne doivent encombrer personne (décision D17 du cadrage).
     *
     * La fiche s'ouvre SUR la case, jamais au centre de l'écran : le regard ne quitte pas la
     * grille.
     */
    import { t } from '../../i18n';

    let { cell, busy = false, onClose = () => {}, onResult = () => {}, onForfeit = () => {}, onMove = () => {}, onCancel = () => {} } = $props();

    let scoreA = $state('');
    let scoreB = $state('');
    let more = $state(false);
    let note = $state('');
    let moveTo = $state('');

    function num(v) {
        const n = parseInt(v, 10);
        return Number.isFinite(n) && n >= 0 ? n : 0;
    }

    /* Cliquer un nom valide : avec ou sans score, c'est le même geste. */
    function win(winner) {
        onResult(cell.matchId, winner, num(scoreA), num(scoreB), note.trim());
        onClose();
    }

    function forfeit(winner) {
        onForfeit(cell.matchId, winner, note.trim());
        onClose();
    }

    function move() {
        const n = num(moveTo);
        if (n > 0) {
            onMove(cell.matchId, n);
            onClose();
        }
    }

    function cancel() {
        onCancel(cell.matchId);
        onClose();
    }

    function onKey(e) {
        if (e.key === 'Escape') {
            e.stopPropagation();
            onClose();
        } else if (e.key === 'Enter' && !more) {
            e.stopPropagation();
            // Sans vainqueur choisi, Entrée ne fait rien : le vainqueur est la seule chose exigée.
        } else if (e.key === 'ArrowLeft') {
            win(cell.a);
        } else if (e.key === 'ArrowRight') {
            win(cell.b);
        }
    }
</script>

<div class="card" role="dialog" aria-label={$t('direction.result.title')} onkeydown={onKey}>
    <header>
        <span class="where"
            >{$t('direction.proposals.table', { n: cell.table })} &middot;
            {$t('direction.proposals.points', { n: cell.length })}</span
        >
        <span class="grow"></span>
        <button type="button" class="more-btn" onclick={() => (more = !more)} title={$t('direction.result.more')}>⋯</button>
        <button type="button" class="close" onclick={onClose} title={$t('common.close')}>×</button>
    </header>

    <!-- Deux grosses cibles aux noms des joueurs : on clique dessus en se penchant, d'où
         l'exception typographique nommée dans l'ADR-0008. -->
    <div class="winners">
        <button type="button" class="winner" disabled={busy} onclick={() => win(cell.a)}>{cell.aName}</button>
        <button type="button" class="winner" disabled={busy} onclick={() => win(cell.b)}>{cell.bName}</button>
    </div>

    <div class="score">
        <label>
            {$t('direction.result.score')}
            <input type="number" min="0" max="99" bind:value={scoreA} />
        </label>
        <span>–</span>
        <input type="number" min="0" max="99" bind:value={scoreB} />
        <span class="optional">{$t('direction.result.optional')}</span>
    </div>

    {#if more}
        <div class="more">
            <div class="row">
                <span class="row-label">{$t('direction.result.forfeit')}</span>
                <button type="button" disabled={busy} onclick={() => forfeit(cell.b)}>{cell.aName}</button>
                <button type="button" disabled={busy} onclick={() => forfeit(cell.a)}>{cell.bName}</button>
            </div>
            <label class="row">
                <span class="row-label">{$t('direction.result.note')}</span>
                <input type="text" bind:value={note} placeholder={$t('direction.result.notePlaceholder')} />
            </label>
            <div class="row">
                <span class="row-label">{$t('direction.result.move')}</span>
                <input type="number" min="1" max="200" bind:value={moveTo} />
                <button type="button" disabled={busy} onclick={move}>{$t('direction.result.apply')}</button>
            </div>
            <div class="row">
                <button type="button" class="danger" disabled={busy} onclick={cancel}>{$t('direction.result.cancelMatch')}</button>
            </div>
        </div>
    {/if}
</div>

<style>
    /* Sur la case, pas au centre : le regard ne quitte pas la grille. */
    .card {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        z-index: 20;
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        padding: var(--space-2);
        border: 1px solid var(--color-primary);
        border-radius: var(--radius);
        background: var(--color-surface);
        box-shadow: 0 6px 18px rgba(0, 0, 0, 0.18);
        text-align: left;
    }

    header {
        display: flex;
        align-items: center;
        gap: var(--space-1);
    }

    .where {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .grow {
        flex: 1;
    }

    .winners {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: var(--space-1);
    }

    /* Ce qui rend ces deux noms faciles à cliquer en se penchant sur un portable posé au coin
       d'une table, c'est la TAILLE DE LA CIBLE — 44 px de haut, toute la moitié de la fiche —
       et non la taille de la police. Le cadrage avait prévu de les agrandir d'une fois et
       demie ; l'ADR-0008 fait venir la hiérarchie du poids et de la couleur, et il a raison
       ici. Si le premier tournoi réel montre que les noms sont trop petits, ce sera à
       l'ADR-0008 d'être amendé, avec la mesure qui le justifie. */
    .winner {
        min-height: 44px;
        font-weight: 600;
        padding: var(--space-1);
        border: 1px solid var(--color-primary);
        border-radius: var(--radius);
        background: var(--color-surface-alt);
        color: var(--color-text);
        cursor: pointer;
    }

    .winner:hover:not(:disabled) {
        background: var(--color-surface);
    }

    .score {
        display: flex;
        align-items: center;
        gap: var(--space-1);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .optional {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .more {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        border-top: 1px solid var(--color-border);
        padding-top: var(--space-1);
    }

    .row {
        display: flex;
        align-items: center;
        gap: var(--space-1);
        font-size: var(--font-size-small);
    }

    .row-label {
        color: var(--color-text-muted);
        min-width: 6rem;
    }

    input {
        width: 4rem;
        padding: 0.15rem 0.3rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface-alt);
        color: var(--color-text);
    }

    .row input[type='text'] {
        width: auto;
        flex: 1;
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

    button.danger {
        border-color: var(--color-danger);
        color: var(--color-danger);
    }

    .close,
    .more-btn {
        border-color: transparent;
        color: var(--color-text-muted);
    }

    button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
</style>
