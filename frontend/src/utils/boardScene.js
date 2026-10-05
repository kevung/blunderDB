// Drawing of the backgammon board, outside Board.svelte so the scene can be asserted without the
// component or a real two.js backend. Every function takes `two` (anything exposing two.js's
// shape factories: the Two instance, a layer adapter, or a test recorder), `geom`
// (boardMetrics()) and `cfg` (boardCfg), and reads no store: the component passes plain values.
// drawStaticScene() depends only on geometry, palette and label flip; drawDynamicScene() on the
// position; drawFrame() goes on top so the outline keeps its linewidth over the checkers.

import { computePipCount } from './boardGeometry.js';

/** @typedef {import('./boardGeometry.js').BoardMetrics} BoardMetrics */
/** @typedef {import('./boardGeometry.js').BoardPosition} BoardPosition */
/** @typedef {import('./boardGeometry.js').StepMove} StepMove */
/** @typedef {import('./boardConfig.js').BoardConfig} BoardConfig */
/** @typedef {import('two.js').default} Two */
/** @typedef {Pick<Two, 'makeText' | 'makeCircle' | 'makeRectangle' | 'makeLine'> & { makePath: (...args: number[]) => ReturnType<Two['makePath']> }} Surface */
/** @typedef {Two['scene']} Group */
/** @typedef {{ x: number, y: number }} Pt */
/** @typedef {{ x: number, y: number, size: number }} CubeBox */

/**
 * A drawing surface that puts every shape into `group` (two.js factories add to the root and
 * Group.add() reparents). Keeps Board.svelte's static layer (rebuilt on resize, orientation or
 * palette change) and dynamic layer (emptied on every redraw) apart with the same functions.
 *
 * @param {Surface} two
 * @param {Group} group
 * @returns {Surface}
 */
export function layerOf(two, group) {
    /**
     * @template {(...args: never[]) => object} F
     * @param {F} factory
     * @returns {(...args: Parameters<F>) => ReturnType<F>}
     */
    const into =
        (factory) =>
        (...args) => {
            const shape = /** @type {ReturnType<F>} */ (Reflect.apply(factory, two, args));
            group.add(/** @type {Parameters<Group['add']>[0]} */ (shape));
            return shape;
        };
    return {
        makePath: into(two.makePath),
        makeText: into(two.makeText),
        makeCircle: into(two.makeCircle),
        makeRectangle: into(two.makeRectangle),
        makeLine: into(two.makeLine)
    };
}

// Sentinel colour stored on an exclude-structure point that must hold no checker.
export const EXCLUDE_EMPTY = 2;

// Bearoff tray pseudo-point used by move notation ("6/off").
export const BEAROFF_POINT = -1;

/**
 * X of the column of `point` (1..24, or 0/25 for the bar) in the given orientation.
 *
 * @param {BoardMetrics} geom
 * @param {string} orientation
 * @param {number} point
 * @returns {number}
 */
export function pointColumnX(geom, orientation, point) {
    const { originX, checkerSize } = geom;
    const sign = orientation === 'left' ? -1 : 1;
    if (point === 0 || point === 25) return originX;
    if (point <= 6) return originX + sign * (7 - point) * checkerSize;
    if (point <= 12) return originX - sign * (point - 6) * checkerSize;
    if (point <= 18) return originX - sign * (19 - point) * checkerSize;
    return originX + sign * (point - 18) * checkerSize;
}

/**
 * True when `point` stacks from the bottom edge upwards (points 1-12 and the top bar).
 *
 * @param {number} point
 * @returns {boolean}
 */
function stacksUpward(point) {
    return (point !== 0 && point <= 12) || point === 25;
}

/**
 * Centre of the `slot`-th checker (0-based) on `point`, as drawCheckers() paints it (also anchors
 * move arrows). Points 0 and 25 are the bars (stacking from the middle out), BEAROFF_POINT the
 * tray. Null for an unknown point.
 *
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {number} point
 * @param {number} slot
 * @returns {Pt | null}
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

/**
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {number} x
 * @param {number} y
 * @param {boolean} flip
 */
function makeTriangle(two, cfg, geom, x, y, flip) {
    const cs = geom.checkerSize;
    const th = geom.triangleHeight;
    const triangle = flip ? two.makePath(x, y + th, x + cs, y + th, x + 0.5 * cs, y + th - 5 * cs) : two.makePath(x, y, x + cs, y, x + 0.5 * cs, y + 5 * cs);
    triangle.stroke = cfg.triangle.stroke;
    triangle.linewidth = cfg.triangle.linewidth;
    return triangle;
}

/**
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {number} x
 * @param {number} y
 * @param {boolean} flip
 */
function drawQuadrant(two, geom, cfg, x, y, flip) {
    for (let i = 0; i < 6; i++) {
        const t = makeTriangle(two, cfg, geom, x + i * geom.checkerSize, y, flip);
        // Alternate the two point colours; a flipped (bottom) quadrant starts on
        // the other colour so opposite points share a shade.
        const odd = i % 2 === 1;
        t.fill = odd !== flip ? cfg.triangle.fill1 : cfg.triangle.fill2;
    }
}

/**
 * The 24 triangles: four quadrants of six, top ones pointing down, bottom ones up.
 *
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 */
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
 * The 24 point numbers under (1-12) or over (13-24) each column. `flip` numbers the board from
 * player 2's side (point p reads 25-p).
 *
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {boolean} flip
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

/**
 * The bar, painted before the checkers so those on the bar sit above it.
 *
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 */
export function drawBar(two, geom, cfg) {
    const bar = two.makeRectangle(geom.originX, geom.originY, geom.checkerSize, geom.boardHeight);
    bar.fill = cfg.fill;
    bar.stroke = cfg.stroke;
    bar.linewidth = 3.5;
    return bar;
}

/**
 * The board outline, painted last so its linewidth is not eaten by the checkers.
 *
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 */
export function drawFrame(two, geom, cfg) {
    const board = two.makeRectangle(geom.originX, geom.originY, geom.boardWidth, geom.boardHeight);
    board.fill = 'transparent';
    board.stroke = cfg.stroke;
    board.linewidth = 3.5;
    return board;
}

/**
 * Everything that survives a position change: triangles, labels, bar.
 *
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {boolean} flip
 */
export function drawStaticScene(two, geom, cfg, flip) {
    drawLabels(two, geom, cfg, flip);
    drawTriangles(two, geom, cfg);
    drawBar(two, geom, cfg);
}

// "Must be empty" exclusion marker: a red hatched, crossed-out cell spanning
// the point's checker column to make the block obvious.
/**
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {number} point
 */
function drawExcludeMarker(two, geom, cfg, point) {
    const cs = cfg.checker.sizeFactor * geom.checkerSize;
    const spanSlots = 3; // cover ~3 checker slots
    const { x, y: firstSlotY } = /** @type {Pt} */ (stackSlotCenter(geom, cfg, point, 0));
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
 *
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {BoardPosition} position
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
            const { x, y } = /** @type {Pt} */ (stackSlotCenter(geom, cfg, index, i));
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
 * Where the cube sits: centred on the left when nobody owns it, beside the owner's home board
 * otherwise, mid left pan when offered. Returns the square for hit-testing.
 *
 * @param {BoardMetrics} geom
 * @param {BoardPosition} position
 * @param {boolean} offered
 * @returns {CubeBox}
 */
export function cubeBox(geom, position, offered) {
    const { originX, originY, boardWidth, boardHeight, checkerSize } = geom;
    const size = 0.9 * checkerSize;
    const gap = 0.75 * checkerSize;
    const restX = originX - boardWidth / 2 - size / 2 - gap;
    if (offered) {
        // 13 checkers wide (6 + bar + 6): the left pan's midpoint is -3.5 from centre, clear of
        // the bear-off indication on the right.
        return { x: originX - 3.5 * checkerSize, y: originY, size };
    }
    if (position.cube.owner === 0) return { x: restX, y: originY + 0.5 * boardHeight - 1.5 * checkerSize, size };
    if (position.cube.owner === 1) return { x: restX, y: originY - 0.5 * boardHeight + 1.5 * checkerSize, size };
    return { x: restX, y: originY, size };
}

/**
 * The doubling cube with its face value (2^value). Returns its box.
 *
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {BoardPosition} position
 * @param {boolean} offered
 * @returns {CubeBox}
 */
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

/** @typedef {(key: string, params?: Record<string, unknown>) => string} SceneText */

const ENGLISH_TEXT = {
    pip: 'pip: {n}',
    away: '{n} away',
    crawford: 'crawford',
    post: 'post',
    unlimited: 'unlimited',
    off: '({n} OFF)'
};

/**
 * The scene reads no store, so its words come in as a function; this fallback keeps the
 * recorder-based tests and any caller without a translator on the English text.
 *
 * @type {SceneText}
 */
export const defaultSceneText = (key, params = {}) => ENGLISH_TEXT[/** @type {keyof typeof ENGLISH_TEXT} */ (key)].replace(/\{(\w+)\}/g, (m, k) => (k in params ? String(params[k]) : m));

const LABEL_MAX_SIZE = 20;
const LABEL_MIN_SIZE = 9;

/**
 * Rendered width of `content` at `size`, estimated per character: the canvas has no text metrics
 * before it paints, and a label must be sized before it is drawn. Bold Latin is ~0.62 em, an
 * ideographic character a full em.
 *
 * @param {string} content
 * @param {number} size
 */
export function estimateTextWidth(content, size) {
    let em = 0;
    for (const ch of content) em += /** @type {number} */ (ch.codePointAt(0)) >= 0x2e80 ? 1 : 0.62;
    return em * size;
}

/**
 * Half the room a side label may take: centred 1.2 checkers from the board edge, it must stay
 * clear of the board (point numbers sit against it) and of the canvas edge.
 *
 * @param {BoardMetrics} geom
 */
function sideLabelHalfWidth(geom) {
    const margin = (geom.width - geom.boardWidth) / 2;
    return Math.min(1.2 * geom.checkerSize, margin - 1.2 * geom.checkerSize) - 3;
}

/**
 * One font size for a whole side column: the largest that keeps its widest label inside the
 * margin, bounded so a very narrow canvas degrades to small text rather than to none.
 *
 * @param {BoardMetrics} geom
 * @param {string[]} contents
 */
export function sideLabelSize(geom, contents) {
    const widest = Math.max(...contents.map((c) => estimateTextWidth(c, 1)));
    const fitted = Math.floor((2 * sideLabelHalfWidth(geom)) / widest);
    return Math.max(LABEL_MIN_SIZE, Math.min(LABEL_MAX_SIZE, fitted));
}

/**
 * @param {Surface} two
 * @param {string} content
 * @param {number} x
 * @param {number} y
 * @param {number} [size]
 */
function boldText(two, content, x, y, size = LABEL_MAX_SIZE) {
    const t = two.makeText(content, x, y);
    t.size = size;
    t.alignment = 'center';
    t.baseline = 'middle';
    t.weight = 'bold';
    return t;
}

/**
 * Both pip counts, on the left at the height of the scores.
 *
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardPosition} position
 * @param {SceneText} [text]
 */
export function drawPipCounts(two, geom, position, text = defaultSceneText) {
    const { pipCount1, pipCount2 } = computePipCount(position);
    const { originX, originY, boardWidth, boardHeight, checkerSize } = geom;
    const x = originX - boardWidth / 2 - 1.2 * checkerSize;
    const labels = [text('pip', { n: pipCount1 }), text('pip', { n: pipCount2 })];
    const size = sideLabelSize(geom, labels);
    boldText(two, labels[0], x, originY + boardHeight / 2 + 0.2 * checkerSize, size);
    boldText(two, labels[1], x, originY - boardHeight / 2 - 0.2 * checkerSize, size);
}

/**
 * Layout of the side column (bearoff counts, dice, scores) beside the board.
 *
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {number} playerOnRoll
 */
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

/**
 * "(n OFF)" for both players.
 *
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {BoardPosition} position
 * @param {SceneText} [text]
 */
export function drawBearoff(two, geom, cfg, position, text = defaultSceneText) {
    const side = sideLayout(geom, cfg, position.player_on_roll);
    const labels = [0, 1].map((i) => text('off', { n: position.board.bearoff[i] }));
    const size = sideLabelSize(geom, labels);
    for (const [i, y] of [side.bearoff1Y, side.bearoff2Y].entries()) {
        const t = two.makeText(labels[i], side.bearoffX, y);
        t.size = size;
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

/** A played die stays readable, but steps back. */
const USED_DIE_OPACITY = 0.35;

/**
 * Two dice on the roller's side; blank faces for a cube decision.
 *
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {BoardPosition} position
 * @param {boolean[] | null} [used] a die already played is drawn faded (a Duel's move)
 */
export function drawDice(two, geom, cfg, position, used = null) {
    const side = sideLayout(geom, cfg, position.player_on_roll);
    const { diceSize, diceGap, diceY } = side;
    position.dice.forEach((die, index) => {
        const dieX = side.diceX + index * (diceSize + diceGap);
        const opacity = used?.[index] ? USED_DIE_OPACITY : 1;
        const face = two.makeRectangle(dieX, diceY, diceSize, diceSize);
        face.fill = cfg.dice.fill;
        face.stroke = cfg.stroke; // follows the board border colour
        face.linewidth = 2.5;
        face.opacity = opacity;
        if (position.decision_type !== 0) return;
        (DIE_DOTS[die] || []).forEach(([dx, dy]) => {
            const dot = two.makeCircle(dieX + (dx * diceSize) / 3, diceY + (dy * diceSize) / 3, diceSize / 12);
            dot.fill = cfg.dice.dot;
            dot.opacity = opacity;
        });
    });
}

/**
 * The score label for one player: away count, crawford, post-crawford or money.
 *
 * @param {number} score
 * @param {SceneText} [text]
 * @returns {string}
 */
export function scoreLabel(score, text = defaultSceneText) {
    if (score === 1) return text('crawford');
    if (score === 0) return text('post');
    if (score === -1) return text('unlimited');
    return text('away', { n: score });
}

/**
 * Both scores on the right; post-crawford takes two lines.
 *
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {BoardPosition} position
 * @param {SceneText} [text]
 */
export function drawScores(two, geom, cfg, position, text = defaultSceneText) {
    const side = sideLayout(geom, cfg, position.player_on_roll);
    const lines = position.score.slice(0, 2).map((score) => scoreLabel(score, text));
    const size = sideLabelSize(geom, [...lines, text('crawford')]);
    for (const [i, y] of [side.score1Y, side.score2Y].entries()) {
        const score = position.score[i];
        boldText(two, lines[i], side.scoreX, y - (score === 0 ? size / 2 : 0), size);
        if (score === 0) boldText(two, text('crawford'), side.scoreX, y + size / 2, size);
    }
}

/**
 * Arrows for a candidate move, one per checker moved (`moves` already mirrored to the display
 * position). Each leaves from the top of its source stack and lands on the next free slot,
 * simulating the intermediate board so two checkers on one point stack.
 *
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {BoardPosition} position
 * @param {StepMove[] | null | undefined} moves
 */
export function drawMoveArrows(two, geom, cfg, position, moves) {
    if (!moves || moves.length === 0) return;
    /** @type {Record<number, number>} */
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
 * Les points qu'offre le coup en cours (Duel, transcription) : un anneau autour du pion choisi,
 * un disque là où il irait — rien d'autre que `quizPlayTargetsStore`. Les pions jouables ne sont
 * pas marqués : presque tous le sont à chaque jet. Dessiné après les pions (sinon invisible) et avant les flèches.
 *
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {BoardPosition} position
 * @param {{targets?: Iterable<number>, selected?: number|null}} opts
 */
export function drawPlayHighlights(two, geom, cfg, position, opts = {}) {
    const targets = [...(opts.targets ?? [])];
    const selected = opts.selected ?? null;
    if (selected === null && targets.length === 0) return;

    const cs = geom.checkerSize;
    const radius = cs * 0.42;
    const count = (/** @type {number} */ point) => (point === BEAROFF_POINT ? 0 : (position.board.points[point]?.checkers ?? 0));

    for (const point of targets) {
        const centre = stackSlotCenter(geom, cfg, point, Math.min(count(point), 4));
        if (!centre) continue;
        const disc = two.makeCircle(centre.x, centre.y, radius);
        disc.fill = 'rgba(255, 214, 102, 0.38)';
        disc.stroke = 'rgba(255, 214, 102, 0.9)';
        disc.linewidth = Math.max(cs * 0.06, 1.5);
    }

    if (selected !== null) {
        const centre = stackSlotCenter(geom, cfg, selected, Math.min(Math.max(count(selected) - 1, 0), 4));
        if (centre) {
            const ring = two.makeCircle(centre.x, centre.y, radius);
            ring.fill = 'transparent';
            ring.stroke = 'rgba(255, 107, 107, 0.95)';
            ring.linewidth = Math.max(cs * 0.12, 2);
        }
    }
}

/**
 * Everything that depends on the position. `opts`: offeredCube, showPipcount, moves (arrows),
 * play (highlights of the move in progress). Returns the cube's box for hit-testing.
 *
 * @param {Surface} two
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {BoardPosition} position
 * @param {{ text?: SceneText, offeredCube?: boolean, showPipcount?: boolean, diceUsed?: boolean[] | null, moves?: StepMove[] | null, play?: { targets?: Iterable<number>, selected?: number | null } }} [opts]
 * @returns {CubeBox}
 */
export function drawDynamicScene(two, geom, cfg, position, opts = {}) {
    const box = drawDoublingCube(two, geom, cfg, position, !!opts.offeredCube);
    drawCheckers(two, geom, cfg, position);
    drawBearoff(two, geom, cfg, position, opts.text);
    if (opts.showPipcount) drawPipCounts(two, geom, position, opts.text);
    drawDice(two, geom, cfg, position, opts.diceUsed ?? null);
    drawScores(two, geom, cfg, position, opts.text);
    drawPlayHighlights(two, geom, cfg, position, opts.play ?? {});
    drawMoveArrows(two, geom, cfg, position, opts.moves);
    return box;
}
