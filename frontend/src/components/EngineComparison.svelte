<script>
    // La comparaison inter-moteurs (#269, fiche I.13).
    //
    // Plusieurs moteurs coexistent depuis longtemps dans une même Analyse —
    // chaque coup candidat porte le sien, et les analyses de videau sont
    // gardées une par moteur. Rien ne les montrait CÔTE À CÔTE : les blocs de
    // videau s'empilent, et la liste des coups mélange les rangs de deux
    // moteurs dans un seul classement trié par équité. Il fallait lire deux
    // tableaux en diagonale pour répondre à « sont-ils d'accord ? ».
    //
    // Cette bande y répond en une ligne par moteur, et dit surtout quand ils
    // ne le sont pas. Elle vit dans le panneau Analyse et nulle part ailleurs :
    // l'ADR-0017 réserve au panneau Eval UNE décision, celle du moteur
    // embarqué, et une comparaison n'y a pas sa place.
    import { t } from '../i18n';

    let { analysis = null, kind = 'checker' } = $props();

    /** Une ligne par moteur ayant jugé le videau. */
    let cubeRows = $derived.by(() => {
        if (!analysis) return [];
        const list = analysis.allCubeAnalyses?.length ? analysis.allCubeAnalyses : analysis.doublingCubeAnalysis ? [analysis.doublingCubeAnalysis] : [];
        return list
            .filter((ca) => ca && ca.analysisEngine)
            .map((ca) => ({
                engine: ca.analysisEngine,
                depth: ca.analysisDepth || '',
                verdict: ca.bestCubeAction || ''
            }));
    });

    /**
     * Une ligne par moteur ayant classé les coups, avec SON meilleur coup —
     * celui d'équité maximale parmi ceux qu'il a produits, et non le premier de
     * la liste commune, qui est triée tous moteurs confondus.
     */
    let checkerRows = $derived.by(() => {
        const moves = analysis?.checkerAnalysis?.moves ?? [];
        /** @type {Record<string, {engine: string, depth: string, move: string, equity: number}>} */
        const best = {};
        for (const m of moves) {
            const engine = m.analysisEngine;
            if (!engine) continue;
            const equity = typeof m.equity === 'number' ? m.equity : Number.NEGATIVE_INFINITY;
            if (!best[engine] || equity > best[engine].equity) {
                best[engine] = { engine, depth: m.analysisDepth || '', move: m.move, equity };
            }
        }
        return Object.values(best);
    });

    let rows = $derived(kind === 'cube' ? cubeRows : checkerRows);

    /** Les moteurs sont-ils d'accord ? Deux lignes suffisent à le dire. */
    let disagree = $derived.by(() => {
        if (rows.length < 2) return false;
        const answers = rows.map((r) => (kind === 'cube' ? r.verdict : r.move));
        return answers.some((a) => a !== answers[0]);
    });
</script>

<!-- Un seul moteur n'est pas une comparaison : la bande ne s'affiche pas. -->
{#if rows.length > 1}
    <div class="engine-comparison" class:disagree>
        <span class="label">{disagree ? $t('engines.disagree') : $t('engines.agree')}</span>
        {#each rows as row (row.engine)}
            <span class="engine-row">
                <span class="engine-name">{row.engine}{row.depth ? ` ${row.depth}` : ''}</span>
                <span class="engine-answer">{kind === 'cube' ? row.verdict : row.move}</span>
            </span>
        {/each}
    </div>
{/if}

<style>
    .engine-comparison {
        display: flex;
        align-items: baseline;
        flex-wrap: wrap;
        gap: 0.2em 1em;
        padding: 0.2em 0.4em;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    /* L'accord est l'information la moins intéressante : il se lit, il ne
       s'annonce pas. Le désaccord, lui, est ce pour quoi la bande existe. */
    .engine-comparison.disagree .label {
        color: var(--color-danger);
        font-weight: 600;
    }

    .engine-row {
        display: inline-flex;
        gap: 0.4em;
    }

    .engine-answer {
        color: var(--color-text);
    }
</style>
