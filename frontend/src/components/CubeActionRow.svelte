<!--
  CubeActionRow — la rangée [D] [T] [P] [R] de la maquette (ux.md §2).

  Quatre gestes de videau à la souris, un clic chacun : doubler, prendre,
  passer, résigner. Ce sont les touches `d`, `t`, `p` et `r` du panneau et rien
  d'autre — le composant n'appelle aucune fonction du moteur et ne connaît ni
  brouillon ni document : il rend un `kind` au panneau, qui le fait passer par
  la même machine que les touches (services/transcriptionKeys.js).

  # Ce qui s'allume, et pourquoi

  Un bouton est éteint là où son geste ne répond à rien, et le clavier, lui, ne
  refuse jamais (ADR-0044). Les deux ne se contredisent pas : la touche est
  l'entrée de qui sait ce qu'il tape, la rangée est une cible offerte, et offrir
  une cible qui ne produirait qu'une Incohérence vaut moins que ne pas
  l'offrir — c'est déjà la règle du triangle des jets, qui disparaît devant une
  réponse au videau.

  La rangée dit donc de qui est le tour : « le camp au trait agit » — `[D]` et
  `[R]` — ou « le camp d'en face répond » — `[T]` et `[P]`. Jamais les quatre à
  la fois.

  # La résignation, en deux gestes

  `[R]` n'enregistre rien : il annonce la résignation et la rangée devient les
  trois niveaux, simple, gammon, backgammon, plus l'abandon qui double `Échap`.
  C'est exactement les deux frappes `r` puis `1`/`2`/`3` d'ux.md §4.2, et c'est
  la raison du choix contre les deux autres formes envisagées : trois boutons
  permanents auraient coûté trois cases de plus, à demeure, pour un geste qui
  sert une fois par match, quand `[D]` et `[T]` servent à chaque tour ; un menu
  surgissant aurait ajouté un placement et une règle de fermeture à un choix
  qui a déjà un endroit où se poser — la rangée elle-même, dont les quatre
  cases deviennent trois le temps de la réponse.
-->
<script>
    import { t } from '../i18n';
    import { COMMAND } from '../services/transcriptionKeys.js';

    let {
        /** Le camp au trait peut annoncer (aucune réponse attendue). */
        canAct = false,
        /** Une offre attend sa réponse : prise ou passe. */
        canAnswer = false,
        /** La résignation est annoncée, son niveau attendu. */
        resigning = false,
        /** Le geste choisi : `(kind) => void`, avec les COMMAND du videau. */
        onGesture = null,
        /** `[R]` : la résignation annoncée, son niveau attendu. */
        onResign = null,
        /** Le niveau donné : `(level) => void`, 1, 2 ou 3. */
        onLevel = null,
        /** La résignation abandonnée. */
        onCancelResign = null
    } = $props();

    const LEVELS = [
        { level: 1, key: 'transcription.resignSingle' },
        { level: 2, key: 'transcription.resignGammon' },
        { level: 3, key: 'transcription.resignBackgammon' }
    ];
</script>

<div class="cube-row" role="group" aria-label={$t('transcription.cubeRow')}>
    {#if resigning}
        {#each LEVELS as entry (entry.level)}
            <button class="cube-btn" type="button" onclick={() => onLevel?.(entry.level)} title={$t('transcription.resignLevelTooltip', { n: entry.level })}>
                {$t(entry.key)}
            </button>
        {/each}
        <button class="cube-btn" type="button" onclick={() => onCancelResign?.()} title={$t('transcription.resignCancelTooltip')}>
            {$t('transcription.resignCancel')}
        </button>
    {:else}
        <button class="cube-btn" type="button" disabled={!canAct} onclick={() => onGesture?.(COMMAND.DOUBLE)} title={$t('transcription.doubleTooltip')}>
            {$t('transcription.double')}
        </button>
        <button class="cube-btn" type="button" disabled={!canAnswer} onclick={() => onGesture?.(COMMAND.TAKE)} title={$t('transcription.takeTooltip')}>
            {$t('transcription.take')}
        </button>
        <button class="cube-btn" type="button" disabled={!canAnswer} onclick={() => onGesture?.(COMMAND.PASS)} title={$t('transcription.passTooltip')}>
            {$t('transcription.pass')}
        </button>
        <button class="cube-btn" type="button" disabled={!canAct} onclick={() => onResign?.()} title={$t('transcription.resignTooltip')}>
            {$t('transcription.resign')}
        </button>
    {/if}
</div>

<style>
    .cube-row {
        display: flex;
        flex-wrap: wrap;
        gap: var(--space-1);
    }

    .cube-btn {
        padding: var(--space-1) var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }

    .cube-btn:hover:not(:disabled) {
        background: var(--color-surface-alt);
    }

    .cube-btn:disabled {
        color: var(--color-text-muted);
        cursor: default;
    }
</style>
