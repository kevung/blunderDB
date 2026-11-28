<script>
    // A die face drawn small enough to sit in a line of text. value 0 draws the empty face of a
    // roll not chosen yet.
    let { value = 0, size = 14 } = $props();

    // Pip centres on a 3×3 lattice, per face.
    /** @type {Record<number, Array<[number, number]>>} */
    const PIPS = {
        0: [],
        1: [[1, 1]],
        2: [
            [0, 0],
            [2, 2]
        ],
        3: [
            [0, 0],
            [1, 1],
            [2, 2]
        ],
        4: [
            [0, 0],
            [2, 0],
            [0, 2],
            [2, 2]
        ],
        5: [
            [0, 0],
            [2, 0],
            [1, 1],
            [0, 2],
            [2, 2]
        ],
        6: [
            [0, 0],
            [2, 0],
            [0, 1],
            [2, 1],
            [0, 2],
            [2, 2]
        ]
    };
    let pips = $derived(PIPS[value] ?? []);
</script>

<svg class="mini-die" width={size} height={size} viewBox="0 0 12 12" aria-hidden="true" data-value={value}>
    <rect x="0.5" y="0.5" width="11" height="11" rx="2" class:blank={!value} />
    {#each pips as [x, y], i (i)}
        <circle cx={3 + x * 3} cy={3 + y * 3} r="1.1" />
    {/each}
</svg>

<style>
    .mini-die {
        display: inline-block;
        vertical-align: middle;
        flex: none;
    }
    rect {
        fill: var(--color-surface);
        stroke: currentColor;
        stroke-width: 1;
    }
    rect.blank {
        stroke-dasharray: 2 1.5;
    }
    circle {
        fill: currentColor;
    }
</style>
