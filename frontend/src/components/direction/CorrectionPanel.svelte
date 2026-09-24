<script>
    /*
     * La correction d'un résultat : « qui a gagné ? », les deux noms, le score (#372, #436).
     *
     * Un seul panneau, ouvert depuis deux endroits : sous la dernière décision (le geste vu
     * aussitôt, F28) et sur une ligne de l'Historique (le résultat ancien, F8). Les deux
     * corrigent de la même façon — un événement de plus, le premier résultat reste au journal —
     * et doivent donc se présenter de la même façon.
     */
    import { t } from '../../i18n';

    /**
     * @type {{
     *     a: string, b: string, aName: string, bName: string,
     *     busy?: boolean,
     *     testid?: string, // préfixe des deux boutons : `<testid>-winner-a`, `-winner-b`
     *     onPick?: (winner: string, scoreA: number, scoreB: number) => void
     * }}
     */
    let { a, b, aName, bName, busy = false, testid = 'direction-last', onPick = () => {} } = $props();

    let scoreA = $state('');
    let scoreB = $state('');

    /** @param {string} v */
    function num(v) {
        const n = parseInt(v, 10);
        return Number.isFinite(n) && n >= 0 ? n : 0;
    }

    /** @param {string} winner */
    function pick(winner) {
        onPick(winner, num(scoreA), num(scoreB));
        scoreA = '';
        scoreB = '';
    }
</script>

<div class="correct">
    <span class="hint">{$t('direction.last.whoWon')}</span>
    <button type="button" class="winner" data-testid="{testid}-winner-a" disabled={busy} onclick={() => pick(a)}>{aName}</button>
    <button type="button" class="winner" data-testid="{testid}-winner-b" disabled={busy} onclick={() => pick(b)}>{bName}</button>
    <input type="number" min="0" max="99" bind:value={scoreA} />
    <span>–</span>
    <input type="number" min="0" max="99" bind:value={scoreB} />
</div>

<style>
    .correct {
        display: flex;
        align-items: center;
        gap: var(--space-1);
        padding: var(--space-1) var(--space-2);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        border-top: 1px solid var(--color-border);
    }

    button {
        padding: 0.1rem 0.5rem;
        border: 1px solid var(--color-primary);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
        font-size: var(--font-size-small);
        font-weight: 600;
        min-height: 30px;
    }

    button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    input {
        width: 3.2rem;
        padding: 0.1rem 0.3rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface-alt);
        color: var(--color-text);
        font-size: var(--font-size-small);
    }
</style>
