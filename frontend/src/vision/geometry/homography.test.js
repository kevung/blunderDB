import { describe, it, expect } from 'vitest';
import { computeHomography, applyHomography, invertHomography, jacobian } from './homography.js';
import { lcg } from './synthetic.fixture.js';

const SQUARE = [
    [0, 0],
    [13, 0],
    [13, 13],
    [0, 13]
];

function expectSameH(actual, expected, tol) {
    const e = expected.map((v) => v / expected[8]);
    actual.forEach((v, i) => expect(Math.abs(v - e[i])).toBeLessThan(tol * Math.max(1, Math.abs(e[i]))));
}

/** A random camera: perspective terms large enough to bend lines, small enough to keep the square in front. */
function randomH(rand) {
    const a = 10 + 20 * rand();
    const theta = 2 * Math.PI * rand();
    return [
        a * Math.cos(theta) + 4 * (rand() - 0.5),
        -a * Math.sin(theta) + 4 * (rand() - 0.5),
        100 + 200 * rand(),
        a * Math.sin(theta) + 4 * (rand() - 0.5),
        a * Math.cos(theta) + 4 * (rand() - 0.5),
        50 + 100 * rand(),
        0.03 * (rand() - 0.5),
        0.03 * (rand() - 0.5),
        1
    ];
}

describe('computeHomography', () => {
    it('recovers a known homography from the four projected corners', () => {
        const rand = lcg(7);
        let tried = 0;
        for (let k = 0; k < 200; k++) {
            const Htrue = randomH(rand);
            const dst = SQUARE.map((p) => applyHomography(Htrue, p));
            const res = computeHomography(SQUARE, dst);
            if (!res.ok) continue;
            tried++;
            expectSameH(res.H, Htrue, 1e-9);
            // Points other than the four used land where the true camera puts them.
            for (const p of [
                [6.5, 6.5],
                [2, 11],
                [-1, 3]
            ]) {
                const [x, y] = applyHomography(res.H, p);
                const [xt, yt] = applyHomography(Htrue, p);
                expect(Math.hypot(x - xt, y - yt)).toBeLessThan(1e-8);
            }
        }
        expect(tried).toBeGreaterThan(150);
    });

    it('keeps full precision on a small quad far from the origin', () => {
        const Htrue = [0.5, 0.01, 40000, -0.02, 0.48, 30000, 1e-6, 2e-6, 1];
        const dst = SQUARE.map((p) => applyHomography(Htrue, p));
        const res = computeHomography(SQUARE, dst);
        expect(res.ok).toBe(true);
        for (const p of [
            [6.5, 6.5],
            [1, 12],
            [12.5, 0.5]
        ]) {
            const [x, y] = applyHomography(res.H, p);
            const [xt, yt] = applyHomography(Htrue, p);
            expect(Math.hypot(x - xt, y - yt)).toBeLessThan(1e-6);
        }
    });

    it('accepts a mirrored view: home board on the other side', () => {
        const dst = [
            [300, 10],
            [10, 10],
            [10, 170],
            [300, 170]
        ];
        const res = computeHomography(SQUARE, dst);
        expect(res.ok).toBe(true);
        dst.forEach((q, i) => {
            const [x, y] = applyHomography(res.H, SQUARE[i]);
            expect(x).toBeCloseTo(q[0], 9);
            expect(y).toBeCloseTo(q[1], 9);
        });
    });

    it('inverts', () => {
        const Htrue = randomH(lcg(3));
        const Hinv = invertHomography(Htrue);
        for (const p of SQUARE) {
            const [x, y] = applyHomography(Hinv, applyHomography(Htrue, p));
            expect(x).toBeCloseTo(p[0], 9);
            expect(y).toBeCloseTo(p[1], 9);
        }
        expect(invertHomography([1, 2, 3, 2, 4, 6, 0, 0, 1])).toBeNull();
    });

    const good = [
        [10, 10],
        [100, 12],
        [95, 80],
        [8, 75]
    ];
    const refusals = [
        ['three points', [good[0], good[1], good[2]], 'points'],
        ['five points', [...good, [50, 50]], 'points'],
        ['a NaN click', [good[0], [NaN, 3], good[2], good[3]], 'points'],
        ['an infinite click', [good[0], good[1], [Infinity, 1], good[3]], 'points'],
        ['two clicks on the same spot', [good[0], good[0], good[2], good[3]], 'aligned'],
        ['three aligned clicks', [good[0], [55, 11], good[1], good[3]], 'aligned'],
        [
            'nearly aligned clicks',
            [
                [0, 0],
                [50, 0.2],
                [100, 0],
                [50, 80]
            ],
            'aligned'
        ],
        [
            'a dart (reflex vertex)',
            [
                [10, 10],
                [100, 10],
                [40, 30],
                [10, 80]
            ],
            'non-convex'
        ],
        ['a bow-tie: clicks out of order', [good[0], good[2], good[1], good[3]], 'non-convex']
    ];
    for (const [name, dst, reason] of refusals) {
        it(`refuses ${name}`, () => {
            expect(computeHomography(SQUARE, dst)).toEqual({ ok: false, reason });
            expect(computeHomography(dst, SQUARE)).toEqual({ ok: false, reason });
        });
    }
});

describe('jacobian', () => {
    it('matches finite differences', () => {
        const H = randomH(lcg(11));
        const p = [4, 9];
        const J = jacobian(H, p);
        const h = 1e-6;
        const [x0, y0] = applyHomography(H, p);
        const [xu, yu] = applyHomography(H, [p[0] + h, p[1]]);
        const [xv, yv] = applyHomography(H, [p[0], p[1] + h]);
        expect(J[0][0]).toBeCloseTo((xu - x0) / h, 4);
        expect(J[1][0]).toBeCloseTo((yu - y0) / h, 4);
        expect(J[0][1]).toBeCloseTo((xv - x0) / h, 4);
        expect(J[1][1]).toBeCloseTo((yv - y0) / h, 4);
    });
});
