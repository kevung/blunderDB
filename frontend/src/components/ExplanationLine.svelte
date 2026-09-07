<script>
    // La phrase d'explication (#298, fiche J.8).
    //
    // Une ligne, sous les tableaux, et seulement quand une règle est
    // confiante. Le moteur rend un thème et ses écarts mesurés ; le gabarit
    // traduit est ici. Quand le thème est vide — le cas le plus fréquent — ce
    // composant ne rend rien du tout, pas même un cadre vide.
    //
    // Elle porte aussi, quand la liste parcourue est un CLASSEMENT, la
    // distance qui a rangé cette position-là (ADR-0043). C'est une phrase
    // contextuelle sur la position courante, exactement comme l'explication —
    // et c'est le seul endroit où la distance survit au premier geste, là où
    // le message de la barre d'état s'efface.
    import { explainDecision, playedFromAnalysis } from '../services/explainService.js';
    import { positionStore } from '../stores/positionStore.js';
    import { rankedDistancesStore, rankedTargetStore } from '../stores/rankedStore.js';
    import { t } from '../i18n';

    let { analysis = null } = $props();

    /** @typedef {import('../../wailsjs/go/models').engine.Explanation} Explanation */
    let explanation = $state(/** @type {Explanation | null} */ (null));

    $effect(() => {
        const id = $positionStore?.id;
        const played = playedFromAnalysis(analysis);
        let cancelled = false;
        explanation = null;
        if (!id || !played) return;
        explainDecision(id, played).then((result) => {
            if (!cancelled) explanation = result;
        });
        return () => {
            cancelled = true;
        };
    });

    let sentence = $derived(
        explanation
            ? $t(`explain.${explanation.theme}`, {
                  cost: explanation.costMp,
                  best: explanation.best,
                  blots: explanation.blots ?? 0,
                  bestBlots: explanation.bestBlots ?? 0,
                  gammon: explanation.gammonPct ?? 0,
                  points: explanation.points ?? 0,
                  bestPoints: explanation.bestPoints ?? 0
              })
            : ''
    );

    // La distance de la position courante, s'il y en a une. Un classement
    // contre un plateau dessiné n'a pas de cible numérotée : on dit alors la
    // distance seule, sans prétendre nommer une position qui n'existe pas.
    let distance = $derived($rankedDistancesStore.get($positionStore?.id));
    let neighbourLine = $derived(distance == null ? '' : $rankedTargetStore > 0 ? $t('similar.fromPosition', { distance, id: $rankedTargetStore }) : $t('similar.fromBoard', { distance }));
</script>

{#if neighbourLine}
    <p class="explanation">{neighbourLine}</p>
{/if}
{#if sentence}
    <p class="explanation">{sentence}</p>
{/if}

<style>
    .explanation {
        margin: 0.4em 0 0;
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }
</style>
