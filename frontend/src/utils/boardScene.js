// Drawing of the backgammon board, extracted from Board.svelte so the scene
// can be asserted without mounting the component or a real two.js backend.
//
// Every function takes the same leading arguments:
//   two   — anything exposing two.js's shape factories (makePath, makeText,
//           makeCircle, makeRectangle, makeLine). In production it is the
//           Two instance or a layer adapter around it; in tests, a recorder.
//   geom  — boardMetrics() for the current drawing surface.
//   cfg   — Board.svelte's boardCfg (orientation, widthFactor, palette).
// and draws into `two` as a side effect. Nothing here reads a store: the
// component resolves the display position, the offered-cube state, the label
// flip and the selected move, then hands them over as plain values.
//
// The scene splits in two: drawStaticScene() — triangles, point labels, bar —
// depends only on the geometry, the palette and the label flip, while
// drawDynamicScene() — checkers, cube, bearoff, pip counts, dice, scores and
// move arrows — depends on the position. drawFrame() sits on top of both so
// the outline keeps a consistent linewidth over the checkers.

import { computePipCount } from './boardGeometry.js';

// Sentinel colour stored on an exclude-structure point that must hold no checker.
export const EXCLUDE_EMPTY = 2;

// Bearoff tray pseudo-point used by move notation ("6/off").
export const BEAROFF_POINT = -1;

/** X of the column of `point` (1..24, or 0/25 for the bar) in the given orientation. */
export function pointColumnX(geom, orientation, point) {
    const { originX, checkerSize } = geom;
    const sign = orientation === 'left' ? -1 : 1;
    if (point === 0 || point === 25) return originX;
    if (point <= 6) return originX + sign * (7 - point) * checkerSize;
    if (point <= 12) return originX - sign * (point - 6) * checkerSize;
    if (point <= 18) return originX - sign * (19 - point) * checkerSize;
    return originX + sign * (point - 18) * checkerSize;
}

/** True when `point` stacks from the bottom edge upwards (points 1-12 and the top bar). */
function stacksUpward(point) {
    return (point !== 0 && point <= 12) || point === 25;
}

/**
 * Centre of the `slot`-th checker (0-based) on `point` — the exact spot
 * drawCheckers() paints it, also used to anchor move arrows. Points 0 and 25
 * are the bars (stacking from the middle out), BEAROFF_POINT the tray beside
 * the board. Returns null for an unknown point.
 */
export function stackSlotCenter(geom, cfg, point, slot) {
    const { originX, originY, boardWidth, boardHeight, checkerSize } = geom;
    const orientation = cfg.orientation;
    const step = cfg.checker.sizeFactor * checkerSize;
    if (point === BEAROFF_POINT) {
        const sign = orientation === 'left' ? -1 : 1;
        return { x: originX + sign * (0.5 * boardWidth + 0.75 * checkerSize), y: originY };
    }
    if (point < 0 || point > 25) return null;
    let yBase;
    if (point === 0) yBase = originY + 0.5 * checkerSize;
    else if (point === 25) yBase = originY - 0.5 * checkerSize;
    else if (point <= 12) yBase = originY + 0.5 * boardHeight;
    else yBase = originY - 0.5 * boardHeight;
    const dir = stacksUpward(point) ? -1 : 1;
    return { x: pointColumnX(geom, orientation, point), y: yBase + dir * (slot + 0.5) * step };
}

function makeTriangle(two, cfg, geom, x, y, flip) {
    const cs = geom.checkerSize;
    const th = geom.triangleHeight;
    const triangle = flip ? two.makePath(x, y + th, x + cs, y + th, x + 0.5 * cs, y + th - 5 * cs) : two.makePath(x, y, x + cs, y, x + 0.5 * cs, y + 5 * cs);
    triangle.stroke = cfg.triangle.stroke;
    triangle.linewidth = cfg.triangle.linewidth;
    return triangle;
}

function drawQuadrant(two, geom, cfg, x, y, flip) {
    for (let i = 0; i < 6; i++) {
        const t = makeTriangle(two, cfg, geom, x + i * geom.checkerSize, y, flip);
        // Alternate the two point colours; a flipped (bottom) quadrant starts on
        // the other colour so opposite points share a shade.
        const odd = i % 2 === 1;
        t.fill = odd !== flip ? cfg.triangle.fill1 : cfg.triangle.fill2;
    }
}

/** The 24 triangles: four quadrants of six, top ones pointing down, bottom ones up. */
export function drawTriangles(two, geom, cfg) {
    const { originX, originY, boardWidth, checkerSize, triangleHeight } = geom;
    const topY = originY - triangleHeight - 0.5 * checkerSize;
    const bottomY = originY + 0.5 * checkerSize;
    drawQuadrant(two, geom, cfg, originX + 0.5 * checkerSize, topY, false);
    drawQuadrant(two, geom, cfg, originX - 0.5 * boardWidth, topY, false);
    drawQuadrant(two, geom, cfg, originX - 0.5 * boardWidth, bottomY, true);
    drawQuadrant(two, geom, cfg, originX + 0.5 * checkerSize, bottomY, true);
}

/**
 * The 24 point numbers, one under (points 1-12) or over (13-24) each column.
 * `flip` numbers the board from the other player's side (point p reads 25-p):
 * match mode with player 2 on roll, or an edited position with player 2 on roll.
 */
export function drawLabels(two, geom, cfg, flip) {
    const { originY, boardHeight, checkerSize } = geom;
    const offset = 0.5 * boardHeight + cfg.label.distanceToBoard * checkerSize;
    for (let p = 1; p <= 24; p++) {
        const bottom = p <= 12;
        const t = two.makeText((flip ? 25 - p : p).toString(), pointColumnX(geom, cfg.orientation, p), bottom ? originY + offset : originY - offset);
        t.size = cfg.label.size;
        t.alignment = 'center';
        t.baseline = bottom ? 'top' : 'middle';
    }
}

/** The bar, painted before the checkers so those on the bar sit above it. */
export function drawBar(two, geom, cfg) {
    const bar = two.makeRectangle(geom.originX, geom.originY, geom.checkerSize, geom.boardHeight);
    bar.fill = cfg.fill;
    bar.stroke = cfg.stroke;
    bar.linewidth = 3.5;
    return bar;
}

/** The board outline, painted last so its linewidth is not eaten by the checkers. */
export function drawFrame(two, geom, cfg) {
    const board = two.makeRectangle(geom.originX, geom.originY, geom.boardWidth, geom.boardHeight);
    board.fill = 'transparent';
    board.stroke = cfg.stroke;
    board.linewidth = 3.5;
    return board;
}

/** Everything that survives a position change: triangles, labels, bar. */
export function drawStaticScene(two, geom, cfg, flip) {
    drawLabels(two, geom, cfg, flip);
    drawTriangles(two, geom, cfg);
    drawBar(two, geom, cfg);
}

// "Must be empty" exclusion marker: a red hatched, crossed-out cell spanning
// the point's checker column to make the block obvious.
function drawExcludeMarker(two, geom, cfg, point) {
    const cs = cfg.checker.sizeFactor * geom.checkerSize;
    const spanSlots = 3; // cover ~3 checker slots
    const { x, y: firstSlotY } = stackSlotCenter(geom, cfg, point, 0);
    const dir = stacksUpward(point) ? -1 : 1;
    const cy = firstSlotY - dir * 0.5 * cs + dir * (spanSlots / 2) * cs;
    const w = cs;
    const h = spanSlots * cs;
    const cell = two.makeRectangle(x, cy, w, h);
    cell.fill = 'rgba(192,57,43,0.18)';
    cell.stroke = '#c0392b';
    cell.linewidth = 2;
    // Diagonal hatching across the cell.
    const top = cy - h / 2;
    const left = x - w / 2;
    const step = cs / 2;
    for (let d = step; d < w + h; d += step) {
        let ax = left + d,
            ay = top;
        let bx = left,
            by = top + d;
        if (ax > left + w) {
            ay = top + (ax - (left + w));
            ax = left + w;
        }
        if (by > top + h) {
            bx = left + (by - (top + h));
            by = top + h;
        }
        const hatch = two.makeLine(ax, ay, bx, by);
        hatch.stroke = '#c0392b';
        hatch.linewidth = 1;
    }
}

/**
 * The checkers of every point and both bars: at most five per stack, the
 * fifth carrying the true count when the stack is taller. An EXCLUDE_EMPTY
 * point draws the exclusion marker instead.
 */
export function drawCheckers(two, geom, cfg, position) {
    const radius = (cfg.checker.sizeFactor * geom.checkerSize) / 2;
    position.board.points.forEach((point, index) => {
        if (point.color === EXCLUDE_EMPTY) {
            drawExcludeMarker(two, geom, cfg, index);
            return;
        }
        const checkersToDraw = Math.min(point.checkers, 5);
        for (let i = 0; i < checkersToDraw; i++) {
            const { x, y } = stackSlotCenter(geom, cfg, index, i);
            const checker = two.makeCircle(x, y, radius);
            checker.fill = cfg.checker.colors[point.color];
            checker.stroke = cfg.triangle.stroke;
            checker.linewidth = cfg.checker.linewidth;
            if (i === 4 && point.checkers > 5) {
                const text = two.makeText(point.checkers.toString(), x, y);
                text.size = 20;
                text.alignment = 'center';
                text.baseline = 'middle';
                text.weight = 'bold';
                // Contrast against the checker it sits on.
                if (point.color === 0) text.fill = '#ffffff';
                else if (point.color === 1) text.fill = '#333333';
            }
        }
    });
}

/**
 * Where the doubling cube sits: centred on the left when nobody owns it,
 * beside the owner's home board otherwise, and in the middle of the left pan
 * when it has been offered (take/pass decision). Returns the square so the
 * click handler can hit-test it.
 */
export function cubeBox(geom, position, offered) {
    const { originX, originY, boardWidth, boardHeight, checkerSize } = geom;
    const size = 0.9 * checkerSize;
    const gap = 0.75 * checkerSize;
    const restX = originX - boardWidth / 2 - size / 2 - gap;
    if (offered) {
        // The board is 13 checkers wide (6 + bar + 6), so the left pan spans
        // [-6.5, -0.5] checkers from centre and its midpoint is -3.5. Kept on
        // the left — the same side the cube normally sits — so it never
        // clashes with the bear-off (checker-off) indication on the right.
        return { x: originX - 3.5 * checkerSize, y: originY, size };
    }
    if (position.cube.owner === 0) return { x: restX, y: originY + 0.5 * boardHeight - 1.5 * checkerSize, size };
    if (position.cube.owner === 1) return { x: restX, y: originY - 0.5 * boardHeight + 1.5 * checkerSize, size };
    return { x: restX, y: originY, size };
}

/** The doubling cube with its face value (2^value). Returns its box. */
export function drawDoublingCube(two, geom, cfg, position, offered) {
    const box = cubeBox(geom, position, offered);
    const cube = two.makeRectangle(box.x, box.y, box.size, box.size);
    cube.fill = cfg.cube.fill;
    cube.stroke = cfg.stroke; // follows the board border colour
    cube.linewidth = 2.5;
    const text = two.makeText(Math.pow(2, position.cube.value).toString(), box.x, box.y);
    text.size = 34;
    text.alignment = 'center';
    text.baseline = 'middle';
    text.translation.set(box.x, box.y + 0.05 * box.size); // optically centred
    return box;
}

function boldText(two, content, x, y) {
    const t = two.makeText(content, x, y);
    t.size = 20;
    t.alignment = 'center';
    t.baseline = 'middle';
    t.weight = 'bold';
    return t;
}

/** Both pip counts, on the left at the height of the scores. */
export function drawPipCounts(two, geom, position) {
    const { pipCount1, pipCount2 } = computePipCount(position);
    const { originX, originY, boardWidth, boardHeight, checkerSize } = geom;
    const x = originX - boardWidth / 2 - 1.2 * checkerSize;
    boldText(two, `pip: ${pipCount1}`, x, originY + boardHeight / 2 + 0.2 * checkerSize);
    boldText(two, `pip: ${pipCount2}`, x, originY - boardHeight / 2 - 0.2 * checkerSize);
}

/** Layout of the side column (bearoff counts, dice, scores) beside the board. */
export function sideLayout(geom, cfg, playerOnRoll) {
    const { originX, originY, boardWidth, boardHeight, checkerSize } = geom;
    const sign = cfg.orientation === 'left' ? -1 : 1;
    const diceGap = 0.325 * checkerSize;
    const diceSize = 0.7 * checkerSize;
    return {
        // Scores and dice always sit on the right; the bearoff counts follow
        // the orientation (the tray is on the side the checkers leave from).
        bearoffX: originX + sign * (boardWidth / 2 + 1.2 * checkerSize),
        bearoff1Y: originY + boardHeight / 2 - 3.7 * checkerSize,
        bearoff2Y: originY - boardHeight / 2 + 3.7 * checkerSize,
        scoreX: originX + boardWidth / 2 + 1.2 * checkerSize,
        score1Y: originY + boardHeight / 2 + 0.2 * checkerSize,
        score2Y: originY - boardHeight / 2 - 0.2 * checkerSize,
        scoreWidth: 1.5 * checkerSize,
        scoreHeight: 0.5 * checkerSize,
        diceGap,
        diceSize,
        diceX: originX + boardWidth / 2 + 2 * diceGap,
        diceY: playerOnRoll === 0 ? originY + 0.5 * boardHeight - 1.5 * checkerSize : originY - 0.5 * boardHeight + 1.5 * checkerSize
    };
}

/** "(n OFF)" for both players. */
export function drawBearoff(two, geom, cfg, position) {
    const side = sideLayout(geom, cfg, position.player_on_roll);
    for (const [i, y] of [side.bearoff1Y, side.bearoff2Y].entries()) {
        const t = two.makeText(`(${position.board.bearoff[i]} OFF)`, side.bearoffX, y);
        t.size = 20;
        t.alignment = 'center';
        t.baseline = 'middle';
    }
}

// Pip layout of each die face, in thirds of the die size.
const DIE_DOTS = [
    [],
    [[0, 0]],
    [
        [-0.7, -0.7],
        [0.7, 0.7]
    ],
    [
        [-0.7, -0.7],
        [0, 0],
        [0.7, 0.7]
    ],
    [
        [-0.7, -0.7],
        [0.7, -0.7],
        [-0.7, 0.7],
        [0.7, 0.7]
    ],
    [
        [-0.7, -0.7],
        [0.7, -0.7],
        [0, 0],
        [-0.7, 0.7],
        [0.7, 0.7]
    ],
    [
        [-0.7, -0.7],
        [0.7, -0.7],
        [-0.7, 0],
        [0.7, 0],
        [-0.7, 0.7],
        [0.7, 0.7]
    ]
];

/** Two dice on the roller's side; blank faces for a cube decision. */
export function drawDice(two, geom, cfg, position) {
    const side = sideLayout(geom, cfg, position.player_on_roll);
    const { diceSize, diceGap, diceY } = side;
    position.dice.forEach((die, index) => {
        const dieX = side.diceX + index * (diceSize + diceGap);
        const face = two.makeRectangle(dieX, diceY, diceSize, diceSize);
        face.fill = cfg.dice.fill;
        face.stroke = cfg.stroke; // follows the board border colour
        face.linewidth = 2.5;
        if (position.decision_type !== 0) return;
        (DIE_DOTS[die] || []).forEach(([dx, dy]) => {
            const dot = two.makeCircle(dieX + (dx * diceSize) / 3, diceY + (dy * diceSize) / 3, diceSize / 12);
            dot.fill = cfg.dice.dot;
        });
    });
}

/** The score label for one player: away count, crawford, post-crawford or money. */
export function scoreLabel(score) {
    if (score === 1) return 'crawford';
    if (score === 0) return 'post';
    if (score === -1) return 'unlimited';
    return `${score} away`;
}

/** Both scores on the right; post-crawford takes two lines. */
export function drawScores(two, geom, cfg, position) {
    const side = sideLayout(geom, cfg, position.player_on_roll);
    for (const [i, y] of [side.score1Y, side.score2Y].entries()) {
        const score = position.score[i];
        boldText(two, scoreLabel(score), side.scoreX, y - (score === 0 ? 10 : 0));
        if (score === 0) boldText(two, 'crawford', side.scoreX, y + 10);
    }
}

/**
 * Arrows for a candidate move, one per checker moved. `moves` is
 * parseMoveNotation()'s output already mirrored to the display position when
 * needed; each arrow leaves from the top checker of its source stack and
 * lands on the next free slot of its destination, simulating the intermediate
 * board so a second checker on the same point stacks on the first.
 */
export function drawMoveArrows(two, geom, cfg, position, moves) {
    if (!moves || moves.length === 0) return;
    const counts = {};
    position.board.points.forEach((point, index) => {
        counts[index] = point.checkers;
    });
    counts[BEAROFF_POINT] = 0;

    const cs = geom.checkerSize;
    const arrowColor = 'rgba(255, 107, 107, 0.85)';
    const arrowWidth = Math.max(cs * 0.22, 6);
    const headLength = cs * 0.45;
    const headWidth = cs * 0.38;

    for (const move of moves) {
        const fromCount = counts[move.from] || 0;
        const toCount = counts[move.to] || 0;
        const from = stackSlotCenter(geom, cfg, move.from, Math.min(Math.max(fromCount - 1, 0), 4));
        const to = stackSlotCenter(geom, cfg, move.to, Math.min(toCount, 4));
        if (counts[move.from] > 0) counts[move.from]--;
        counts[move.to] = toCount + 1;
        if (!from || !to) continue;

        const dx = to.x - from.x;
        const dy = to.y - from.y;
        const length = Math.sqrt(dx * dx + dy * dy);
        if (length < 1) continue;
        const ndx = dx / length;
        const ndy = dy / length;

        // Shaft from the source centre to the base of the arrowhead.
        const line = two.makeLine(from.x, from.y, to.x - headLength * ndx, to.y - headLength * ndy);
        line.stroke = arrowColor;
        line.linewidth = arrowWidth;
        line.cap = 'round';

        // Arrowhead pointing into the destination centre.
        const baseX = to.x - headLength * ndx;
        const baseY = to.y - headLength * ndy;
        const head = two.makePath(to.x, to.y, baseX - headWidth * ndy, baseY + headWidth * ndx, baseX + headWidth * ndy, baseY - headWidth * ndx);
        head.fill = arrowColor;
        head.stroke = arrowColor;
        head.linewidth = 1;
        head.closed = true;
    }
}

/**
 * Everything that depends on the position. `opts`: offeredCube (draw the
 * cube as offered, take/pass), showPipcount, moves (arrows). Returns the
 * cube's box for hit-testing.
 */
export function drawDynamicScene(two, geom, cfg, position, opts = {}) {
    const box = drawDoublingCube(two, geom, cfg, position, !!opts.offeredCube);
    drawCheckers(two, geom, cfg, position);
    drawBearoff(two, geom, cfg, position);
    if (opts.showPipcount) drawPipCounts(two, geom, position);
    drawDice(two, geom, cfg, position);
    drawScores(two, geom, cfg, position);
    drawMoveArrows(two, geom, cfg, position, opts.moves);
    return box;
}
