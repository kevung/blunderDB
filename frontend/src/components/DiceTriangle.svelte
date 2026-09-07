<!--
  DiceTriangle — les 21 jets distincts, une case chacun, et la rangée des six
  dés de l'ouverture (T2.1, ux.md §4.1).

  C'est une CIBLE, pas un clavier de remplacement. Deux chiffres coûtent 2 K =
  0,56 s ; un clic coûte H + P + B + B, mesuré de 1,15 à 1,21 s sur une
  fenêtre de 1024 px (voir ci-dessous). La souris ne rattrapera jamais le clavier sur les
  dés, et le triangle ne cherche pas à le faire : il donne une entrée à qui
  transcrit d'une main, la souris posée, et laisse le clavier intact à côté.

  Pourquoi 21 cases et non la grille de 36. Un jet est une PAIRE NON ORDONNÉE :
  3-1 et 1-3 sont le même jet, et une grille complète offre deux cibles pour la
  même réponse — 36 cibles à balayer, dont quinze doublons, avec l'hésitation
  que fabrique tout choix sans différence. Le triangle n'en montre qu'une, les
  doubles sur la diagonale.

  Taille de case, MESURÉE le 2026-09-07 sur un prototype jetable posé dans une
  fenêtre de 1024 px (trois variantes, Playwright, loi de Fitts de ux.md §1,
  b = 0,15 s/bit, D pris du centre du plateau au centre de la case) :

    case 28 px fixes, bloc de 178 px : P de 0,55 à 0,61 s → clic 1,15 à 1,21 s
    case élastique (37,6 px), bloc de 236 px : P 0,50 à 0,58 s → clic 1,10 à 1,18 s
    grille de 36 à 20 px, bloc de 130 px : P 0,57 à 0,66 s → clic 1,17 à 1,26 s

  La case de 28 px est retenue. L'élastique gagne 0,03 s — sous la résolution du
  modèle — pour 60 px de largeur en plus et une cible dont la taille change avec
  le panneau, ce qui défait la position apprise, seul vrai gain d'une cible
  répétée deux cents fois dans un match. La prédiction d'ux.md §4.1 (1,14 s)
  était optimiste de 0,07 s : la distance réelle du plateau au triangle dépasse
  les 300 px supposés. Le classement, lui, tient : le triangle bat la grille de
  36 de 0,05 s ET de quinze cibles à lire.

  L'ouverture ne prend PAS le triangle. Elle demande un dé par camp, deux
  réponses séparées, et lire une paire dans une case pour la partager entre deux
  joueurs donnerait à la case (3,1) un second sens — « J1 fait 3, J2 fait 1 » —
  quand elle en a déjà un, « le jet 3-1 ». Une même cible à deux sens est
  exactement l'ambiguïté que le triangle existe pour supprimer. L'ouverture
  reçoit donc une rangée de six dés (`single`), un clic = un dé = une touche.
-->
<script>
    import { t } from '../i18n';

    // `single` : la rangée des six dés (l'ouverture, un dé par camp).
    // Sinon le triangle des 21 jets.
    //
    // `onPick(high, low)` rend le jet, dé fort d'abord — l'ordre de la
    // notation, celui que porte l'étiquette de la case.
    // `onDie(die)` rend le dé unique de la rangée.
    let { single = false, onPick = () => {}, onDie = () => {} } = $props();

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

    .die-cell:hover {
        background: var(--color-surface-alt);
        color: var(--color-primary);
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
