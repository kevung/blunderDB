<!--
  DuelBoardPrompt — les confirmations d'un Duel posées sur le plateau (ADR-0072 : le Duel se joue
  au plateau) : Doubler / Annuler après un clic sur le videau, Prendre / Passer quand le Bot
  double, Valider quand le coup est complet. Rien n'est décidé ici : les gestes partent par
  duelService, et l'Arbitre répond.
-->
<script>
    import { t } from '../i18n';
    import { duelStore, duelBoardStore, duelAnimatingStore } from '../stores/duelStore.js';
    import { quizPlayStore } from '../stores/quizPlayStore.js';
    import { humanSide } from '../services/duel.js';
    import { boardPrompt } from '../services/duelBoard.js';
    import { completedPlay } from '../services/quizPlay.js';
    import { decide, confirmDouble, cancelDouble, validateMove } from '../services/duelService.js';

    /**
     * `anchor`: the centre of the board's right half, in pixels from the centre of the board's
     * container (Board.svelte measures it from the drawing's own geometry, so it follows the
     * resizing and the orientation).
     * @type {{ anchor?: { dx: number, dy: number } }}
     */
    let { anchor = { dx: 0, dy: 0 } } = $props();

    let ctx = $derived({
        awaiting: $duelStore?.state && !$duelStore.state.ended ? ($duelStore.state.awaiting ?? null) : null,
        human: humanSide($duelStore?.state),
        animating: $duelAnimatingStore,
        play: $quizPlayStore,
        swapped: $duelBoardStore.swapped,
        prompt: $duelBoardStore.prompt
    });
    let prompt = $derived(boardPrompt(ctx));
    let complete = $derived(!prompt && ctx.awaiting?.kind === 'move' && ctx.awaiting.side === ctx.human && !ctx.animating && !!ctx.play && !!completedPlay(ctx.play));
</script>

{#if prompt || complete}
    <div
        class="duel-board-prompt"
        class:centred={!!prompt}
        style={prompt ? `left: calc(50% + ${anchor.dx}px); top: calc(50% + ${anchor.dy}px);` : undefined}
        role="group"
        aria-label={$t('duel.gestures')}
        data-testid="duel-board-prompt"
    >
        {#if prompt === 'double'}
            <span class="question">{$t('duel.board.doubleQuestion')}</span>
            <button type="button" class="primary" data-testid="duel-confirm-double" onclick={confirmDouble}>{$t('duel.double')}</button>
            <button type="button" data-testid="duel-cancel-double" onclick={cancelDouble}>{$t('duel.board.cancel')}</button>
        {:else if prompt === 'answer'}
            <span class="question">{$t('duel.awaitAnswer')}</span>
            <button type="button" class="primary" data-testid="duel-take" onclick={() => decide('take')}>{$t('duel.take')}</button>
            <button type="button" data-testid="duel-pass" onclick={() => decide('pass')}>{$t('duel.pass')}</button>
        {:else}
            <button type="button" class="primary" data-testid="duel-validate" title={$t('duel.board.validateHint')} onclick={validateMove}>{$t('duel.validate')}</button>
        {/if}
    </div>
{/if}

<style>
    .duel-board-prompt {
        position: absolute;
        left: 50%;
        bottom: 8px;
        transform: translateX(-50%);
        display: flex;
        align-items: center;
        gap: 0.5em;
        padding: 0.35em 0.7em;
        border: 1px solid var(--color-border);
        border-radius: 4px;
        background: var(--color-surface);
        color: var(--color-text);
        box-shadow: 0 2px 6px rgba(0, 0, 0, 0.25);
    }

    /* A cube decision sits in the middle of the right half, where the dice are usually thrown. */
    .duel-board-prompt.centred {
        left: 75%;
        bottom: auto;
        top: 50%;
        transform: translate(-50%, -50%);
        flex-wrap: wrap;
        justify-content: center;
        max-width: 90%;
        gap: 0.6em 0.8em;
        padding: 0.8em 1.2em;
        border: 2px solid var(--color-primary);
        border-radius: 8px;
        box-shadow: 0 4px 14px rgba(0, 0, 0, 0.45);
        z-index: 5;
    }

    .centred .question {
        flex-basis: 100%;
        text-align: center;
        font-size: 1.15em;
    }

    .centred button {
        min-width: 6em;
        padding: 0.45em 1.4em;
        font-size: 1.1em;
    }

    .question {
        font-weight: 600;
    }

    button {
        cursor: pointer;
        padding: 0.15em 0.8em;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        background: var(--color-surface);
        color: var(--color-text);
    }

    button.primary {
        border-color: var(--color-primary);
        color: var(--color-primary);
        font-weight: 600;
    }
</style>
