<script>
    // La phrase d'explication, quand une règle est confiante (gabarit traduit
    // ici), sinon rien. Porte aussi la distance d'un classement `like` (ADR-0043).
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
