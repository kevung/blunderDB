<script>
    // The stored analysis of a position, rendered — and nothing else. This is
    // what the Analysis panel shows, and what an Anki review reveals as the
    // answer of a card (ADR-0025 rule 6).
    //
    // Presentational on purpose: it holds no state. Sorting, the cube/checker
    // tabs of MATCH mode, keyboard handling and the reveal of a review card
    // all live in the panel that owns them and arrive here as props. The two
    // callers ask different questions of the same record — the Analysis panel
    // lets the user sort and switch tabs, a review card takes the sort as it
    // comes — so `kind` is decided by the caller rather than re-derived here
    // from a mode this component would then have to know about.
    import { moverFactsToSides } from '../utils/positionFacts.js';
    import { cubeDecision } from '../utils/cubeDecision.js';
    import CandidateMovesTable from './CandidateMovesTable.svelte';
    import CubeVerdictTable from './CubeVerdictTable.svelte';
    import PositionFactsTable from './PositionFactsTable.svelte';

    // isMoney/jacoby/beaver (ADR-0016 point 6, #190/C.3): the position's own
    // referential and money-game rule flags, read by the caller off its own
    // position (only it knows about MATCH mode / the current position, this
    // component stays presentational) and passed straight down to the two
    // tables that show them next to the equity column and the verdict.
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
        beaver = false
    } = $props();

    // ADR-0017 rule 5: the facts table is shared with EPCPanel, fed here by
    // the stored record instead of a live evaluation. cubelessNoDoubleEquity
    // is the single cubeless figure this table shows (see CubeVerdictTable's
    // own doc comment on why the pair it used to render collapsed to one
    // column).
    //
    // DoublingCubeAnalysis's win/gammon/backgammon chances are stored as
    // percentages [0,100] (the scale every importer and RoundToHundredthPercent
    // use), but moverFactsToSides/PositionFactsTable's shared contract is
    // fractions [0,1] — divide here, at this caller's boundary, rather than
    // touch the stored scale.
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
            <CubeVerdictTable {decision} {cubeAnalysis} {cubeValue} {isPlayedCubeAction} engineVersionFallback={analysis?.analysisEngineVersion} {isMoney} {jacoby} {beaver} />
        </div>
    {/each}
{/if}

{#if kind === 'checker' && hasMoves}
    <CandidateMovesTable {moves} {sortColumn} {sortDirection} {selectedMove} {isPlayedMove} {onSort} {onRowClick} {isMoney} />
{/if}

<style>
    /* Facts, decision and the depth/engine footer, side by side, each next to
       the last. `justify-content: space-between` used to live here and pushed
       the facts against the left edge and the decision against the right one,
       manufacturing a band of white down the middle of the panel — the same
       defect ADR-0020 rule 8 removed from the Eval panel's badge strip, and
       what ADR-0021 states as a rule: blocks are laid out at a constant gap
       and the leftover width is left over, never spread between them. The
       query container is established by the panel that hosts this view, so
       the layout follows the panel's own width — bottom band or side column,
       Analysis tab or Anki answer. */
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
