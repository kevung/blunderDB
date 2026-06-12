// Pure board geometry/parsing helpers extracted from Board.svelte so they can
// be unit-tested without two.js or a mounted component. Everything that touches
// the canvas (drawBoard, drawCheckers, …) stays in the component; only these
// side-effect-free transforms live here.

/**
 * Parse a move string (e.g. "24/23 13/11(2)", "bar/23", "6/off(4)") into a flat
 * list of `{ from, to, index }` moves. `bar` → 0, `off` → -1; the `(n)` suffix
 * expands to n moves. Empty/"bar"/"cannot move" strings yield [].
 */
export function parseMoveNotation(moveString) {
    if (!moveString || moveString === 'bar' || moveString.toLowerCase().includes('cannot move')) {
        return [];
    }

    const moves = [];
    // Split by spaces to get individual moves like "24/23" or "13/11(2)"
    const parts = moveString.trim().split(/\s+/);

    for (const part of parts) {
        // Match pattern like "24/23" or "24/23(2)" or "bar/23" or "6/off(4)"
        const match = part.match(/(\d+|bar)\/(\d+|off)(?:\((\d+)\))?/i);
        if (match) {
            const from = match[1].toLowerCase() === 'bar' ? 0 : parseInt(match[1]);
            const to = match[2].toLowerCase() === 'off' ? -1 : parseInt(match[2]);
            const count = match[3] ? parseInt(match[3]) : 1;

            // Add multiple moves if count > 1
            for (let i = 0; i < count; i++) {
                moves.push({ from, to, index: i });
            }
        }
    }

    return moves;
}

/**
 * Mirror a position so the other player is brought to the bottom: points are
 * reversed (i ↔ 25-i) with colours swapped, bearoff/scores swapped, the
 * on-roll player flipped, and the cube owner flipped when owned. Returns a deep
 * copy; the input is not mutated. `mirrorPosition(mirrorPosition(p))` ≈ `p`.
 */
export function mirrorPosition(pos) {
    const mirrored = JSON.parse(JSON.stringify(pos)); // Deep copy

    // Mirror the board points
    const tempPoints = [...mirrored.board.points];
    for (let i = 0; i < 26; i++) {
        mirrored.board.points[25 - i] = {
            color: tempPoints[i].color === -1 ? -1 : 1 - tempPoints[i].color,
            checkers: tempPoints[i].checkers
        };
    }

    // Swap bearoff
    [mirrored.board.bearoff[0], mirrored.board.bearoff[1]] = [mirrored.board.bearoff[1], mirrored.board.bearoff[0]];

    // Swap player on roll
    mirrored.player_on_roll = 1 - mirrored.player_on_roll;

    // Swap scores
    [mirrored.score[0], mirrored.score[1]] = [mirrored.score[1], mirrored.score[0]];

    // Swap cube owner if owned
    if (mirrored.cube.owner !== -1) {
        mirrored.cube.owner = 1 - mirrored.cube.owner;
    }

    return mirrored;
}

/**
 * Compute both players' pip counts from a (display) position: color-0 checkers
 * count their point index, color-1 checkers count 25 - index.
 */
export function computePipCount(position) {
    let pipCount1 = 0;
    let pipCount2 = 0;

    position.board.points.forEach((point, index) => {
        if (point.color === 0) {
            pipCount1 += point.checkers * index;
        } else if (point.color === 1) {
            pipCount2 += point.checkers * (25 - index);
        }
    });

    return { pipCount1, pipCount2 };
}
