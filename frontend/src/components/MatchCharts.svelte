<script>
    // The time chart and the two loss charts of a match, with the one setting
    // they share: which players are shown. It starts with both for every match.
    import { untrack } from 'svelte';
    import { togglePlayer } from '../utils/chartAxis.js';
    import MatchLosses from './MatchLosses.svelte';
    import MatchTimes from './MatchTimes.svelte';

    /** @type {{ matchId: number | string, times: any, losses: any[] | null, movePositions: any[], player1: string, player2: string, hovered?: number | null, onhover?: (moveId: number | null) => void, onselect?: (index: number) => void }} */
    let { matchId, times, losses, movePositions, player1, player2, hovered = null, onhover = () => {}, onselect = () => {} } = $props();

    /** @type {[boolean, boolean]} */
    let shown = $state([true, true]);
    let current = untrack(() => matchId);
    $effect.pre(() => {
        if (matchId !== current) {
            current = matchId;
            shown = [true, true];
        }
    });
    /** @param {0 | 1} p */
    const toggle = (p) => (shown = togglePlayer(shown, p));
</script>

{#if times}
    <MatchTimes summary={times} {movePositions} {player1} {player2} {hovered} {onhover} {onselect} {shown} ontoggle={toggle} />
{/if}
<MatchLosses {losses} {movePositions} {player1} {player2} {hovered} {onhover} {onselect} {shown} ontoggle={toggle} />
