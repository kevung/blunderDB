/**
 * boardScene.test.js
 *
 * The board scene is drawn through two.js's shape factories only, so a
 * recorder standing in for the Two instance captures every primitive with
 * its arguments. The assertions below pin the shape of the scene for a known
 * position — how many checkers, where the cube and dice sit, where a move's
 * arrows leave from and land — and the two symmetries the component relies
 * on: mirrorPosition() reflects the stacks across the bar's horizontal axis,
 * and the 'left' orientation reflects the board across its vertical axis.
 */

import { describe, test, expect } from 'vitest';
import { boardMetrics, mirrorPosition, parseMoveNotation } from '../utils/boardGeometry.js';
import {
    EXCLUDE_EMPTY,
    BEAROFF_POINT,
    pointColumnX,
    stackSlotCenter,
    cubeBox,
    drawStaticScene,
    drawDynamicScene,
    drawTriangles,
    drawLabels,
    drawCheckers,
    drawDoublingCube,
    drawDice,
    drawScores,
    drawBearoff,
    drawPipCounts,
    drawMoveArrows,
    drawFrame
} from '../utils/boardScene.js';

// ── Fixtures ────────────────────────────────────────────────────────────────

const W = 1000;
const H = 720;

function makeCfg(overrides = {}) {
    return {
        widthFactor: 0.75,
        orientation: 'right',
        fill: '#f0f0f0',
        stroke: '#333333',
        triangle: { fill1: '#d9d9d9', fill2: '#a6a6a6', stroke: '#333333', linewidth: 1.3 },
        label: { size: 20, distanceToBoard: 0.3 },
        checker: { sizeFactor: 0.97, colors: ['#333333', '#ffffff'], linewidth: 2.5 },
        dice: { fill: '#ffffff', dot: '#000000' },
        cube: { fill: '#ffffff' },
        ...overrides
    };
}

function emptyPos() {
    return {
        board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
        cube: { owner: -1, value: 0 },
        dice: [3, 1],
        score: [7, 7],
        player_on_roll: 0,
        decision_type: 0
    };
}

// The opening position, colour 0 at the bottom bearing off towards point 1.
function startPos() {
    const pos = emptyPos();
    const put = (p, checkers, color) => (pos.board.points[p] = { checkers, color });
    put(24, 2, 0);
    put(13, 5, 0);
    put(8, 3, 0);
    put(6, 5, 0);
    put(1, 2, 1);
    put(12, 5, 1);
    put(17, 3, 1);
    put(19, 5, 1);
    return pos;
}

// Stand-in for the Two instance: records each primitive with its arguments.
function recorder() {
    const shapes = [];
    const make =
        (kind) =>
        (...args) => {
            const shape = {
                kind,
                args,
                translation: {
                    set(x, y) {
                        shape.translated = [x, y];
                    }
                }
            };
            shapes.push(shape);
            return shape;
        };
    return {
        shapes,
        of: (kind) => shapes.filter((s) => s.kind === kind),
        makePath: make('path'),
        makeText: make('text'),
        makeCircle: make('circle'),
        makeRectangle: make('rect'),
        makeLine: make('line')
    };
}

const geom = boardMetrics(W, H, 0.75);
const cs = geom.checkerSize;
const near = (v) => expect.closeTo(v, 6);

// ── Static scene ────────────────────────────────────────────────────────────

describe('drawStaticScene — triangles, labels, bar', () => {
    test('paints 24 triangles, 24 labels and the bar', () => {
        const two = recorder();
        drawStaticScene(two, geom, makeCfg(), false);
        expect(two.of('path')).toHaveLength(24);
        expect(two.of('text')).toHaveLength(24);
        expect(two.of('rect')).toHaveLength(1);
        expect(two.of('circle')).toHaveLength(0);
    });

    test('every triangle is five checkers tall, twelve of each shade, alternating along a quadrant', () => {
        const two = recorder();
        const cfg = makeCfg();
        drawTriangles(two, geom, cfg);
        const paths = two.of('path');
        for (const p of paths) {
            const [, y1, , , , y3] = p.args;
            expect(Math.abs(y3 - y1)).toBeCloseTo(5 * cs, 6);
        }
        expect(paths.filter((p) => p.fill === cfg.triangle.fill1)).toHaveLength(12);
        expect(paths.filter((p) => p.fill === cfg.triangle.fill2)).toHaveLength(12);
        // Sort a quadrant's six by x: neighbours never share a shade.
        const bottomRight = paths.filter((p) => p.args[0] > geom.originX && p.args[1] > geom.originY).sort((a, b) => a.args[0] - b.args[0]);
        expect(bottomRight).toHaveLength(6);
        for (let i = 1; i < 6; i++) expect(bottomRight[i].fill).not.toBe(bottomRight[i - 1].fill);
    });

    test('labels number the points 1..24 under their own column, 1-12 below and 13-24 above', () => {
        const two = recorder();
        const cfg = makeCfg();
        drawLabels(two, geom, cfg, false);
        const texts = two.of('text');
        expect(texts.map((t) => t.args[0]).sort((a, b) => a - b)).toEqual(Array.from({ length: 24 }, (_, i) => String(i + 1)));
        for (const t of texts) {
            const p = Number(t.args[0]);
            expect(t.args[1], `x of label ${p}`).toBeCloseTo(pointColumnX(geom, 'right', p), 6);
            if (p <= 12) {
                expect(t.args[2]).toBeGreaterThan(geom.originY + geom.boardHeight / 2);
                expect(t.baseline).toBe('top');
            } else {
                expect(t.args[2]).toBeLessThan(geom.originY - geom.boardHeight / 2);
                expect(t.baseline).toBe('middle');
            }
        }
    });

    test('flip numbers the board from the other side: the column of point p reads 25 - p', () => {
        const plain = recorder();
        const flipped = recorder();
        drawLabels(plain, geom, makeCfg(), false);
        drawLabels(flipped, geom, makeCfg(), true);
        // Points p and 25 - p share a column (one below the bar, one above),
        // so a label is found by its column AND its half.
        const at = (rec, x, bottom) => rec.of('text').find((t) => Math.abs(t.args[1] - x) < 1e-6 && t.args[2] > geom.originY === bottom).args[0];
        for (let p = 1; p <= 24; p++) {
            const x = pointColumnX(geom, 'right', p);
            expect(at(plain, x, p <= 12)).toBe(String(p));
            expect(at(flipped, x, p <= 12)).toBe(String(25 - p));
        }
    });

    test("orientation 'left' reflects every triangle and label across the vertical axis", () => {
        const right = recorder();
        const left = recorder();
        drawStaticScene(right, geom, makeCfg({ orientation: 'right' }), false);
        drawStaticScene(left, geom, makeCfg({ orientation: 'left' }), false);
        const labelXs = (rec) =>
            rec
                .of('text')
                .map((t) => [t.args[0], t.args[1]])
                .sort();
        const mirrored = labelXs(right).map(([label, x]) => [label, 2 * geom.originX - x]);
        for (const [i, [label, x]] of labelXs(left).entries()) {
            expect(label).toBe(mirrored[i][0]);
            expect(x).toBeCloseTo(mirrored[i][1], 6);
        }
    });

    test('the frame is a transparent rectangle the size of the board in the border colour', () => {
        const two = recorder();
        const cfg = makeCfg();
        drawFrame(two, geom, cfg);
        const [frame] = two.of('rect');
        expect(frame.args).toEqual([geom.originX, geom.originY, near(geom.boardWidth), near(geom.boardHeight)]);
        expect(frame.fill).toBe('transparent');
        expect(frame.stroke).toBe(cfg.stroke);
    });
});

// ── Checkers ────────────────────────────────────────────────────────────────

describe('drawCheckers', () => {
    test('the opening position shows its 30 checkers, each once, in the owner colour', () => {
        const two = recorder();
        const cfg = makeCfg();
        drawCheckers(two, geom, cfg, startPos());
        const circles = two.of('circle');
        expect(circles).toHaveLength(30);
        expect(circles.filter((c) => c.fill === cfg.checker.colors[0])).toHaveLength(15);
        expect(circles.filter((c) => c.fill === cfg.checker.colors[1])).toHaveLength(15);
        expect(two.of('text')).toHaveLength(0); // no stack taller than five
    });

    test('a stack rises from the bottom edge on points 1-12 and hangs from the top edge on 13-24', () => {
        const two = recorder();
        const cfg = makeCfg();
        drawCheckers(two, geom, cfg, startPos());
        const step = cfg.checker.sizeFactor * cs;
        // Same column, different half: filter on y as well as x.
        const column = (p) => two.of('circle').filter((c) => Math.abs(c.args[0] - pointColumnX(geom, 'right', p)) < 1e-6 && c.args[1] > geom.originY === p <= 12);
        const six = column(6)
            .map((c) => c.args[1])
            .sort((a, b) => b - a);
        expect(six).toHaveLength(5);
        six.forEach((y, i) => expect(y).toBeCloseTo(geom.originY + geom.boardHeight / 2 - (i + 0.5) * step, 6));
        const nineteen = column(19)
            .map((c) => c.args[1])
            .sort((a, b) => a - b);
        expect(nineteen).toHaveLength(5);
        nineteen.forEach((y, i) => expect(y).toBeCloseTo(geom.originY - geom.boardHeight / 2 + (i + 0.5) * step, 6));
    });

    test('a stack taller than five is capped at five checkers, the fifth carrying the count', () => {
        const two = recorder();
        const pos = emptyPos();
        pos.board.points[4] = { checkers: 7, color: 0 };
        drawCheckers(two, geom, makeCfg(), pos);
        expect(two.of('circle')).toHaveLength(5);
        const [count] = two.of('text');
        expect(count.args[0]).toBe('7');
        const fifth = stackSlotCenter(geom, makeCfg(), 4, 4);
        expect(count.args[1]).toBeCloseTo(fifth.x, 6);
        expect(count.args[2]).toBeCloseTo(fifth.y, 6);
        expect(count.fill).toBe('#ffffff'); // white on a dark checker
    });

    test('bar checkers stack outward from the middle, and are painted exactly once each', () => {
        // Board.svelte used to paint the bar twice (a general loop over all 26
        // points, then a bar-only loop) — identical shapes on top of each other.
        const two = recorder();
        const cfg = makeCfg();
        const pos = emptyPos();
        pos.board.points[0] = { checkers: 2, color: 1 };
        pos.board.points[25] = { checkers: 3, color: 0 };
        drawCheckers(two, geom, cfg, pos);
        const circles = two.of('circle');
        expect(circles).toHaveLength(5);
        for (const c of circles) expect(c.args[0]).toBeCloseTo(geom.originX, 6);
        const step = cfg.checker.sizeFactor * cs;
        const lower = circles.filter((c) => c.args[1] > geom.originY).map((c) => c.args[1]);
        const upper = circles.filter((c) => c.args[1] < geom.originY).map((c) => c.args[1]);
        expect(lower.sort((a, b) => a - b)).toEqual([0, 1].map((i) => near(geom.originY + 0.5 * cs + (i + 0.5) * step)));
        expect(upper.sort((a, b) => b - a)).toEqual([0, 1, 2].map((i) => near(geom.originY - 0.5 * cs - (i + 0.5) * step)));
    });

    test('an EXCLUDE_EMPTY point shows the hatched marker and no checker', () => {
        const two = recorder();
        const pos = emptyPos();
        pos.board.points[5] = { checkers: 1, color: EXCLUDE_EMPTY };
        drawCheckers(two, geom, makeCfg(), pos);
        expect(two.of('circle')).toHaveLength(0);
        const [cell] = two.of('rect');
        expect(cell.args[0]).toBeCloseTo(pointColumnX(geom, 'right', 5), 6);
        expect(cell.stroke).toBe('#c0392b');
        expect(two.of('line').length).toBeGreaterThan(0);
    });

    test('mirrorPosition() reflects the stacks across the horizontal axis and swaps colours', () => {
        const cfg = makeCfg();
        const plain = recorder();
        const mirrored = recorder();
        drawCheckers(plain, geom, cfg, startPos());
        drawCheckers(mirrored, geom, cfg, mirrorPosition(startPos()));
        const key = (c, flipY, swap) =>
            `${c.args[0].toFixed(3)},${(flipY ? 2 * geom.originY - c.args[1] : c.args[1]).toFixed(3)},${swap ? (c.fill === cfg.checker.colors[0] ? 1 : 0) : c.fill === cfg.checker.colors[0] ? 0 : 1}`;
        const expected = plain
            .of('circle')
            .map((c) => key(c, true, true))
            .sort();
        const actual = mirrored
            .of('circle')
            .map((c) => key(c, false, false))
            .sort();
        expect(actual).toEqual(expected);
    });

    test("orientation 'left' reflects the stacks across the vertical axis", () => {
        const right = recorder();
        const left = recorder();
        drawCheckers(right, geom, makeCfg({ orientation: 'right' }), startPos());
        drawCheckers(left, geom, makeCfg({ orientation: 'left' }), startPos());
        const key = (c, flipX) => `${(flipX ? 2 * geom.originX - c.args[0] : c.args[0]).toFixed(3)},${c.args[1].toFixed(3)},${c.fill}`;
        expect(
            left
                .of('circle')
                .map((c) => key(c, false))
                .sort()
        ).toEqual(
            right
                .of('circle')
                .map((c) => key(c, true))
                .sort()
        );
    });
});

// ── Cube, dice, scores, bearoff, pip counts ─────────────────────────────────

describe('drawDoublingCube', () => {
    const restX = geom.originX - geom.boardWidth / 2 - (0.9 * cs) / 2 - 0.75 * cs;

    test.each([
        [-1, restX, geom.originY],
        [0, restX, geom.originY + geom.boardHeight / 2 - 1.5 * cs],
        [1, restX, geom.originY - geom.boardHeight / 2 + 1.5 * cs]
    ])('owner %i puts the cube at (%f, %f)', (owner, x, y) => {
        const pos = emptyPos();
        pos.cube = { owner, value: 2 };
        expect(cubeBox(geom, pos, false)).toEqual({ x: near(x), y: near(y), size: near(0.9 * cs) });
    });

    test('an offered cube sits in the middle of the left pan whoever owns it', () => {
        const pos = emptyPos();
        pos.cube = { owner: 1, value: 1 };
        expect(cubeBox(geom, pos, true)).toEqual({ x: near(geom.originX - 3.5 * cs), y: near(geom.originY), size: near(0.9 * cs) });
    });

    test('shows 2^value and returns the box it was drawn in', () => {
        const two = recorder();
        const pos = emptyPos();
        pos.cube = { owner: 0, value: 3 };
        const box = drawDoublingCube(two, geom, makeCfg(), pos, false);
        const [face] = two.of('rect');
        const [label] = two.of('text');
        expect(face.args).toEqual([near(box.x), near(box.y), near(box.size), near(box.size)]);
        expect(label.args[0]).toBe('8');
        expect(label.translated[0]).toBeCloseTo(box.x, 6);
    });
});

describe('drawDice', () => {
    test('a checker decision shows the pips of both dice on the roller side', () => {
        const two = recorder();
        drawDice(two, geom, makeCfg(), startPos()); // dice [3, 1], player 0 on roll
        expect(two.of('rect')).toHaveLength(2);
        expect(two.of('circle')).toHaveLength(4);
        for (const face of two.of('rect')) expect(face.args[1]).toBeCloseTo(geom.originY + geom.boardHeight / 2 - 1.5 * cs, 6);
        expect(two.of('rect')[0].args[0]).toBeGreaterThan(geom.originX + geom.boardWidth / 2);
    });

    test('a cube decision shows blank faces', () => {
        const two = recorder();
        const pos = startPos();
        pos.decision_type = 1;
        pos.dice = [0, 0];
        drawDice(two, geom, makeCfg(), pos);
        expect(two.of('rect')).toHaveLength(2);
        expect(two.of('circle')).toHaveLength(0);
    });

    test('player 2 on roll moves the dice to the top half', () => {
        const two = recorder();
        const pos = startPos();
        pos.player_on_roll = 1;
        drawDice(two, geom, makeCfg(), pos);
        for (const face of two.of('rect')) expect(face.args[1]).toBeCloseTo(geom.originY - geom.boardHeight / 2 + 1.5 * cs, 6);
    });
});

describe('drawScores / drawBearoff / drawPipCounts', () => {
    test('scores read "n away", crawford, post-crawford (two lines) and unlimited', () => {
        const away = recorder();
        const pos = emptyPos();
        pos.score = [7, 1];
        drawScores(away, geom, makeCfg(), pos);
        expect(away.of('text').map((t) => t.args[0])).toEqual(['7 away', 'crawford']);

        const post = recorder();
        pos.score = [0, -1];
        drawScores(post, geom, makeCfg(), pos);
        expect(post.of('text').map((t) => t.args[0])).toEqual(['post', 'crawford', 'unlimited']);
        const scoreY = geom.originY + geom.boardHeight / 2 + 0.2 * cs;
        expect(post.of('text')[0].args[2]).toBeCloseTo(scoreY - 10, 6);
        expect(post.of('text')[1].args[2]).toBeCloseTo(scoreY + 10, 6);
    });

    test('bearoff counts follow the orientation, scores stay on the right', () => {
        const pos = emptyPos();
        pos.board.bearoff = [3, 0];
        const right = recorder();
        drawBearoff(right, geom, makeCfg({ orientation: 'right' }), pos);
        expect(right.of('text').map((t) => t.args[0])).toEqual(['(3 OFF)', '(0 OFF)']);
        for (const t of right.of('text')) expect(t.args[1]).toBeGreaterThan(geom.originX + geom.boardWidth / 2);
        const left = recorder();
        drawBearoff(left, geom, makeCfg({ orientation: 'left' }), pos);
        for (const t of left.of('text')) expect(t.args[1]).toBeLessThan(geom.originX - geom.boardWidth / 2);
        const scores = recorder();
        drawScores(scores, geom, makeCfg({ orientation: 'left' }), pos);
        for (const t of scores.of('text')) expect(t.args[1]).toBeGreaterThan(geom.originX + geom.boardWidth / 2);
    });

    test('pip counts of the opening position are 167 each, on the left', () => {
        const two = recorder();
        drawPipCounts(two, geom, startPos());
        expect(two.of('text').map((t) => t.args[0])).toEqual(['pip: 167', 'pip: 167']);
        for (const t of two.of('text')) expect(t.args[1]).toBeLessThan(geom.originX - geom.boardWidth / 2);
    });
});

// ── Move arrows ─────────────────────────────────────────────────────────────

describe('drawMoveArrows', () => {
    const cfg = makeCfg();
    const tip = (path) => ({ x: path.args[0], y: path.args[1] });

    test('one shaft and one head per checker moved, from the top of the source stack to the next free slot', () => {
        const two = recorder();
        drawMoveArrows(two, geom, cfg, startPos(), parseMoveNotation('13/7 24/18'));
        expect(two.of('line')).toHaveLength(2);
        expect(two.of('path')).toHaveLength(2);
        const [first, second] = two.of('line');
        const from13 = stackSlotCenter(geom, cfg, 13, 4); // 5 checkers → top slot 4
        expect(first.args[0]).toBeCloseTo(from13.x, 6);
        expect(first.args[1]).toBeCloseTo(from13.y, 6);
        const to7 = stackSlotCenter(geom, cfg, 7, 0); // empty → slot 0
        expect(tip(two.of('path')[0])).toEqual({ x: near(to7.x), y: near(to7.y) });
        const from24 = stackSlotCenter(geom, cfg, 24, 1); // 2 checkers → slot 1
        expect(second.args[0]).toBeCloseTo(from24.x, 6);
        expect(second.args[1]).toBeCloseTo(from24.y, 6);
    });

    test('a doubled move stacks its second checker on the first', () => {
        const two = recorder();
        drawMoveArrows(two, geom, cfg, startPos(), parseMoveNotation('13/7(2)'));
        const [head1, head2] = two.of('path');
        expect(tip(head1)).toEqual({ x: near(stackSlotCenter(geom, cfg, 7, 0).x), y: near(stackSlotCenter(geom, cfg, 7, 0).y) });
        expect(tip(head2)).toEqual({ x: near(stackSlotCenter(geom, cfg, 7, 1).x), y: near(stackSlotCenter(geom, cfg, 7, 1).y) });
        const [shaft1, shaft2] = two.of('line');
        expect(shaft1.args[1]).toBeCloseTo(stackSlotCenter(geom, cfg, 13, 4).y, 6);
        expect(shaft2.args[1]).toBeCloseTo(stackSlotCenter(geom, cfg, 13, 3).y, 6);
    });

    test('bar and off are anchored on the bar and the tray', () => {
        const two = recorder();
        const pos = startPos();
        pos.board.points[0] = { checkers: 1, color: 0 };
        drawMoveArrows(two, geom, cfg, pos, parseMoveNotation('bar/20 6/off'));
        const [fromBar, fromSix] = two.of('line');
        expect(fromBar.args[0]).toBeCloseTo(geom.originX, 6);
        expect(fromBar.args[1]).toBeCloseTo(stackSlotCenter(geom, cfg, 0, 0).y, 6);
        expect(fromSix.args[1]).toBeCloseTo(stackSlotCenter(geom, cfg, 6, 4).y, 6);
        const tray = stackSlotCenter(geom, cfg, BEAROFF_POINT, 0);
        expect(tip(two.of('path')[1])).toEqual({ x: near(tray.x), y: near(tray.y) });
        expect(tray.x).toBeGreaterThan(geom.originX + geom.boardWidth / 2);
    });

    test('nothing is drawn without a move', () => {
        const two = recorder();
        drawMoveArrows(two, geom, cfg, startPos(), []);
        drawMoveArrows(two, geom, cfg, startPos(), undefined);
        expect(two.shapes).toHaveLength(0);
    });
});

// ── Whole dynamic scene ─────────────────────────────────────────────────────

describe('drawDynamicScene', () => {
    test('paints checkers, cube, bearoff, dice and scores, and returns the cube box', () => {
        const two = recorder();
        const pos = startPos();
        const box = drawDynamicScene(two, geom, makeCfg(), pos, {});
        expect(box).toEqual(cubeBox(geom, pos, false));
        expect(two.of('circle')).toHaveLength(30 + 4); // checkers + dice pips
        expect(two.of('rect')).toHaveLength(1 + 2); // cube + two dice
        expect(two.of('text').map((t) => t.args[0])).toEqual(['1', '(0 OFF)', '(0 OFF)', '7 away', '7 away']);
        expect(two.of('line')).toHaveLength(0);
    });

    test('showPipcount adds the two pip counts, moves add their arrows, offeredCube centres the cube', () => {
        const two = recorder();
        const pos = startPos();
        const box = drawDynamicScene(two, geom, makeCfg(), pos, { showPipcount: true, offeredCube: true, moves: parseMoveNotation('8/5 6/5') });
        expect(box.x).toBeCloseTo(geom.originX - 3.5 * cs, 6);
        expect(two.of('text').map((t) => t.args[0])).toContain('pip: 167');
        expect(two.of('line')).toHaveLength(2);
        expect(two.of('path')).toHaveLength(2);
    });
});
