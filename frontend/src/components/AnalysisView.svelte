<script>
    // The stored analysis of a position, rendered, for the Analysis panel and
    // an Anki answer (ADR-0025 rule 6). Stateless: sorting, tabs, keys and
    // `kind` come from the caller, which knows about MATCH mode.
    import { moverFactsToSides } from '../utils/positionFacts.js';
    import { cubeDecision } from '../utils/cubeDecision.js';
    import CandidateMovesTable from './CandidateMovesTable.svelte';
    import CubeVerdictTable from './CubeVerdictTable.svelte';
    import PositionFactsTable from './PositionFactsTable.svelte';

    // isMoney/jacoby/beaver (ADR-0016 point 6): read by the caller, passed down.
    let {
        analysis,
        kind,
        turnability,
        cubeValue = 0,
        onRoll = 0,
        moves = [],
        sortColumn = 'equity',
        sortDirection = 'desc',
        selectedMove = null,
        isPlayedMove = () => false,
        isPlayedCubeAction = () => false,
        onSort = () => {},
        onRowClick = () => {},
        isMoney = undefined,
        jacoby = false,
        beaver = false,
        maxCube = 0
    } = $props();

    // Facts table shared with EvalPanel (ADR-0017 rule 5), fed by the record.
    // Stored chances are percentages [0,100]; the table takes fractions [0,1],
    // so divide here rather than touch the stored scale.
    function cubeFacts(cubeAnalysis) {
        const frac = (x) => (x == null ? null : x / 100);
        return moverFactsToSides(
            {
                win: frac(cubeAnalysis.playerWinChances),
                gammon: frac(cubeAnalysis.playerGammonChances),
                backgammon: frac(cubeAnalysis.playerBackgammonChances),
                cubeless: cubeAnalysis.cubelessNoDoubleEquity
            },
            { win: frac(cubeAnalysis.opponentWinChances), gammon: frac(cubeAnalysis.opponentGammonChances), backgammon: frac(cubeAnalysis.opponentBackgammonChances) },
            onRoll
        );
    }

    // A record may carry the same position analysed by several engines. They
    // are shown one under the other, XG first, then GNUbg, then the rest.
    let cubeAnalysesList = $derived.by(() => {
        if (!analysis) return [];
        let list = [];
        if (analysis.allCubeAnalyses && analysis.allCubeAnalyses.length > 0) {
            list = [...analysis.allCubeAnalyses];
        } else if (analysis.doublingCubeAnalysis) {
            list = [analysis.doublingCubeAnalysis];
        }
        const enginePriority = (engine) => {
            const e = (engine || '').toLowerCase();
            if (e === 'xg') return 0;
            if (e === 'gnubg') return 1;
            return 2;
        };
        list.sort((a, b) => enginePriority(a.analysisEngine) - enginePriority(b.analysisEngine));
        return list;
    });

    let hasMoves = $derived(moves && moves.length > 0);
</script>

{#if kind === 'cube' && cubeAnalysesList.length > 0}
    {#each cubeAnalysesList as cubeAnalysis, cubeIdx (cubeIdx)}
        {@const facts = cubeFacts(cubeAnalysis)}
        {@const decision = cubeDecision({ cubeAnalysis, turnability, stored: true })}
        <div class="tables-container" class:multi-engine-cube={cubeAnalysesList.length > 1}>
            <PositionFactsTable bottom={facts.bottom} top={facts.top} />
            <CubeVerdictTable {decision} {cubeAnalysis} {cubeValue} {isPlayedCubeAction} engineVersionFallback={analysis?.analysisEngineVersion} {isMoney} {jacoby} {beaver} {maxCube} />
        </div>
    {/each}
{/if}

{#if kind === 'checker' && hasMoves}
    <CandidateMovesTable {moves} {sortColumn} {sortDirection} {selectedMove} {isPlayedMove} {onSort} {onRowClick} {isMoney} />
{/if}

<style>
    /* Blocks at a constant gap, leftover width left over, never spread (ADR-0021).
       The query container is the host panel's. */
    .tables-container {
        display: flex;
        flex-wrap: wrap;
        align-items: flex-start;
        gap: 8px 20px;
    }

    .multi-engine-cube {
        margin-bottom: 6px;
    }

    @container (max-width: 600px) {
        .tables-container {
            flex-direction: column;
            gap: 6px;
        }
    }
</style>
