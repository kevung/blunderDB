<script>
    // La phrase d'explication (#298, fiche J.8).
    //
    // Une ligne, sous les tableaux, et seulement quand une règle est
    // confiante. Le moteur rend un thème et ses écarts mesurés ; le gabarit
    // traduit est ici. Quand le thème est vide — le cas le plus fréquent — ce
    // composant ne rend rien du tout, pas même un cadre vide.
    import { explainDecision, playedFromAnalysis } from '../services/explainService.js';
    import { positionStore } from '../stores/positionStore.js';
    import { t } from '../i18n';

    let { analysis = null } = $props();

    let explanation = $state(null);

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
</script>

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
