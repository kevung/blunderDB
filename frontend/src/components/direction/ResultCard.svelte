<script>
    /*
     * La fiche de résultat : deux clics, la case puis le nom du
     * vainqueur, qui valide. Score facultatif (les deux ou aucun) ; forfait, remarque, table
     * et annulation dans un menu discret. Ouverte sur la case.
     */
    import { tick } from 'svelte';
    import { t } from '../../i18n';
    import { closeOnEscape } from '../../services/escapeService.js';
    import { isTypingTarget } from '../../utils/panelFocus.js';

    /**
     * La case d'une table où un match est en cours : la fiche ne s'ouvre que sur celle-là.
     *
     * @typedef {{ table?: number, noTable?: boolean, length?: number, matchId: string, a: string, b: string, aName?: string, bName?: string }} ResultCell
     */

    /**
     * Chaque geste rend la promesse du backend : `false` (ou une exception) dit qu'il a échoué,
     * et la fiche reste ouverte avec l'erreur plutôt que de laisser croire qu'il a eu lieu.
     *
     * @typedef {Promise<boolean | void> | boolean | void} Outcome
     */

    /**
     * @type {{
     *     cell: ResultCell,
     *     busy?: boolean,
     *     onClose?: () => void,
     *     onResult?: (matchId: string, winner: string, scoreA: number, scoreB: number, note: string) => Outcome,
     *     onForfeit?: (matchId: string, winner: string, note: string) => Outcome,
     *     onMove?: (matchId: string, table: number) => Outcome,
     *     onCancel?: (matchId: string) => Outcome,
     *     moveRequest?: number
     * }}
     */
    let { cell, busy = false, onClose = () => {}, onResult = () => {}, onForfeit = () => {}, onMove = () => {}, onCancel = () => {}, moveRequest = 0 } = $props();

    let scoreA = $state('');
    let scoreB = $state('');
    let more = $state(false);
    let note = $state('');
    let moveTo = $state('');
    let moveInput = $state(/** @type {HTMLInputElement | null} */ (null));
    /** Le vainqueur choisi au clavier (← / →), que Entrée valide. */
    let chosen = $state('');
    let failed = $state(false);
    let sending = $state(false);

    /** @param {string} v */
    function num(v) {
        const n = parseInt(v, 10);
        return Number.isFinite(n) && n >= 0 ? n : 0;
    }

    /**
     * Attend le backend avant de fermer : sur un échec, la fiche reste ouverte.
     *
     * @param {() => Outcome} fn
     */
    async function submit(fn) {
        if (sending) return;
        sending = true;
        failed = false;
        let ok;
        try {
            ok = (await fn()) !== false;
        } catch {
            ok = false;
        }
        sending = false;
        if (ok) onClose();
        else failed = true;
    }

    /* Cliquer un nom valide : avec ou sans score, c'est le même geste. */
    /** @param {string} winner */
    function win(winner) {
        submit(() => onResult(cell.matchId, winner, num(scoreA), num(scoreB), note.trim()));
    }

    /**
     * Un forfait donne la victoire à l'autre : il se confirme, car il ne se lit pas comme un
     * résultat et pèse sur le parcours du forfait.
     *
     * @param {string} winner
     * @param {string} loserName
     * @param {string} winnerName
     */
    function forfeit(winner, loserName, winnerName) {
        if (!window.confirm($t('direction.result.forfeitConfirm', { loser: loserName, winner: winnerName }))) return;
        submit(() => onForfeit(cell.matchId, winner, note.trim()));
    }

    function move() {
        const n = num(moveTo);
        if (n > 0) submit(() => onMove(cell.matchId, n));
    }

    function cancel() {
        if (!window.confirm($t('direction.result.cancelConfirm', { a: cell.aName ?? cell.a, b: cell.bName ?? cell.b }))) return;
        submit(() => onCancel(cell.matchId));
    }

    /* « Changer de table » (menu, M, X) : le champ de table passe devant le curseur, que la fiche
       vienne de s'ouvrir ou le soit déjà. Chaque demande a son numéro. */
    $effect(() => {
        if (moveRequest <= 0) return;
        more = true;
        tick().then(() => moveInput?.focus());
    });

    /* Échap ferme la fiche avant tout geste global, même quand le focus l'a quittée. */
    $effect(() => closeOnEscape(() => onClose()));

    /**
     * ← et → choisissent un vainqueur, Entrée le valide. Dans un champ, les flèches appartiennent
     * au champ : aucun raccourci d'une touche n'y est pris.
     *
     * @param {KeyboardEvent} e
     */
    function onKey(e) {
        if (e.key === 'Enter') {
            // Sans vainqueur choisi, Entrée ne fait rien : le vainqueur est la seule chose exigée.
            // Rien ne sort de la fiche : Entrée lancerait sinon la proposition sélectionnée.
            e.stopPropagation();
            if (e.target instanceof HTMLButtonElement) return;
            e.preventDefault();
            // Le champ de table a son propre geste : Entrée y déplace, jamais n'enregistre.
            if (e.target === moveInput) {
                if (!busy) move();
                return;
            }
            if (chosen && !busy) win(chosen);
            return;
        }
        if (isTypingTarget(/** @type {Element | null} */ (e.target)) || e.ctrlKey || e.metaKey || e.altKey) return;
        if (e.key === 'ArrowLeft') {
            chosen = cell.a;
            e.preventDefault();
        } else if (e.key === 'ArrowRight') {
            chosen = cell.b;
            e.preventDefault();
        }
    }
</script>

<div class="card" data-testid="direction-result-card" role="dialog" aria-label={$t('direction.result.title')} tabindex="-1" onkeydown={onKey}>
    <header>
        <span class="where"
            >{cell.noTable ? $t('direction.table.noTable') : $t('direction.proposals.table', { n: cell.table })} &middot;
            {$t('direction.proposals.points', { n: cell.length })}</span
        >
        <span class="grow"></span>
        <button type="button" class="more-btn" data-testid="direction-result-more" onclick={() => (more = !more)} title={$t('direction.result.more')}>⋯</button>
        <button type="button" class="close" onclick={onClose} title={$t('common.close')}>×</button>
    </header>

    <!-- Deux grosses cibles aux noms des joueurs : on clique dessus en se penchant, d'où
         l'exception typographique nommée dans l'ADR-0008. -->
    <div class="winners">
        <button type="button" class="winner" class:chosen={chosen === cell.a} data-testid="direction-result-winner-a" disabled={busy || sending} onclick={() => win(cell.a)}>{cell.aName}</button>
        <button type="button" class="winner" class:chosen={chosen === cell.b} data-testid="direction-result-winner-b" disabled={busy || sending} onclick={() => win(cell.b)}>{cell.bName}</button>
    </div>
    {#if failed}
        <p class="error" role="alert" data-testid="direction-result-error">{$t('direction.result.error')}</p>
    {/if}

    <div class="score">
        <label>
            {$t('direction.result.score')}
            <input type="number" data-testid="direction-result-score-a" min="0" max="99" bind:value={scoreA} />
        </label>
        <span>–</span>
        <input type="number" data-testid="direction-result-score-b" min="0" max="99" bind:value={scoreB} />
        <span class="optional">{$t('direction.result.optional')}</span>
    </div>

    {#if more}
        <div class="more">
            <div class="row">
                <span class="row-label">{$t('direction.result.forfeit')}</span>
                <button type="button" data-testid="direction-result-forfeit-a" disabled={busy || sending} onclick={() => forfeit(cell.b, cell.aName ?? cell.a, cell.bName ?? cell.b)}
                    >{$t('direction.result.forfeitWins', { loser: cell.aName ?? cell.a, winner: cell.bName ?? cell.b })}</button
                >
                <button type="button" data-testid="direction-result-forfeit-b" disabled={busy || sending} onclick={() => forfeit(cell.a, cell.bName ?? cell.b, cell.aName ?? cell.a)}
                    >{$t('direction.result.forfeitWins', { loser: cell.bName ?? cell.b, winner: cell.aName ?? cell.a })}</button
                >
            </div>
            <label class="row">
                <span class="row-label">{$t('direction.result.note')}</span>
                <input type="text" bind:value={note} placeholder={$t('direction.result.notePlaceholder')} />
            </label>
            <div class="row">
                <span class="row-label">{$t('direction.result.move')}</span>
                <input type="number" data-testid="direction-result-move-table" bind:this={moveInput} min="1" max="200" bind:value={moveTo} />
                <button type="button" data-testid="direction-result-move" disabled={busy || sending} onclick={move}>{$t('direction.result.apply')}</button>
            </div>
            <div class="row">
                <button type="button" class="danger" data-testid="direction-result-cancel" disabled={busy || sending} onclick={cancel}>{$t('direction.result.cancelMatch')}</button>
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

    /* La cible fait 44 px, la police reste au jeton (ADR-0008). */
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

    .winner.chosen {
        background: var(--color-primary);
        color: var(--color-surface);
    }

    .error {
        margin: 0;
        font-size: var(--font-size-small);
        color: var(--color-danger);
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
