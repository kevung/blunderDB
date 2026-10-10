// The opening position, in `domain.Position`'s shape, at an Away score: no roll, nobody's decision.

/** @param {number[]} away */
export function openingPosition(away) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    /** @type {[number, number, number][]} */
    const layout = [
        [24, 2, 0],
        [13, 5, 0],
        [8, 3, 0],
        [6, 5, 0],
        [1, 2, 1],
        [12, 5, 1],
        [17, 3, 1],
        [19, 5, 1]
    ];
    for (const [point, checkers, color] of layout) points[point] = { checkers, color };
    return {
        board: { points, bearoff: [0, 0] },
        cube: { owner: -1, value: 0 },
        dice: [0, 0],
        score: [away[0], away[1]],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
        has_beaver: 0
    };
}
