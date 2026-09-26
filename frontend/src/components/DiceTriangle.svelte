<!--
  DiceTriangle — les 21 jets distincts, et la rangée des six dés de
  l'ouverture (ux.md §4.1). Une cible pour qui transcrit la souris en main, pas
  un substitut du clavier (deux chiffres 0,56 s, un clic ~1,2 s).

  21 cases et non 36 : un jet est une paire non ordonnée, et 3-1/1-3 en double
  cible ne ferait que de l'hésitation. Case fixe de 28 px (mesuré à 1024 px,
  Fitts, ux.md §1) : l'élastique ne gagnait que 0,03 s et déplaçait la position
  apprise. L'ouverture a sa rangée de six (`single`), un clic = un dé : lire
  (3,1) comme « J1 fait 3, J2 fait 1 » donnerait deux sens à une case.
-->
<script>
    import { t } from '../i18n';

    // `single` : la rangée des six dés, sinon le triangle. `onPick(high, low)`,
    // dé fort d'abord ; `onDie(die)` pour la rangée. `allowed` : clés « 31 »
    // encore possibles (celles que les pas joués laissent), `null` = toutes.
    let { single = false, allowed = null, onPick = () => {}, onDie = () => {} } = $props();

    const enabled = (high, low) => allowed === null || allowed.has(`${high}${low}`);

    const FACES = [1, 2, 3, 4, 5, 6];
</script>

{#if single}
    <div class="dice-row" role="group" aria-label={$t('transcription.dieRow')}>
        {#each FACES as die (die)}
            <button type="button" class="die-cell" title={$t('transcription.dieTitle', { n: die })} aria-label={$t('transcription.dieTitle', { n: die })} onclick={() => onDie(die)}>{die}</button>
        {/each}
    </div>
{:else}
    <div class="dice-triangle" role="group" aria-label={$t('transcription.diceTriangle')}>
        {#each FACES as high (high)}
            {#each FACES as low (low)}
                {#if low <= high}
                    <button
                        type="button"
                        class="die-cell"
                        class:double={low === high}
                        disabled={!enabled(high, low)}
                        title={$t('transcription.rollTitle', { a: high, b: low })}
                        aria-label={$t('transcription.rollTitle', { a: high, b: low })}
                        onclick={() => onPick(high, low)}>{high}{low}</button
                    >
                {:else}
                    <span class="gap" aria-hidden="true"></span>
                {/if}
            {/each}
        {/each}
    </div>
{/if}

<style>
    /* 28 px : la taille mesurée, pas une taille de police — l'échelle de type
       (ADR-0008) passe par les tokens juste en dessous. */
    .dice-triangle {
        display: grid;
        grid-template-columns: repeat(6, 28px);
        gap: 2px;
    }

    .dice-row {
        display: flex;
        gap: 2px;
    }

    .die-cell {
        display: flex;
        align-items: center;
        justify-content: center;
        box-sizing: border-box;
        width: 28px;
        height: 28px;
        padding: 0;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        font-size: var(--font-size-small);
        font-variant-numeric: tabular-nums;
        cursor: pointer;
    }

    .die-cell:hover:not(:disabled) {
        background: var(--color-surface-alt);
        color: var(--color-primary);
    }

    /* Une case éteinte est un jet que les pions déjà joués démentent : elle se
       lit encore — le triangle garde sa forme, donc les positions apprises —
       mais elle ne se clique plus. */
    .die-cell:disabled {
        color: var(--color-text-muted);
        cursor: default;
    }

    /* Les doubles tiennent la diagonale ; le gras la dit sans ajouter de
       couleur, la case restant une cible avant d'être une décoration. */
    .die-cell.double {
        font-weight: 600;
    }

    .gap {
        width: 28px;
        height: 28px;
    }
</style>
