// Planar homography from exactly four correspondences: Hartley-normalised DLT.
//
// H is a row-major 3×3 array [h0..h8] mapping (x, y, 1) to (u·w, v·w, w), scaled so
// that h8 = 1. Four clicks fix the eight degrees of freedom exactly; with that few
// points the click error dominates, so the work here is refusing the configurations
// that would make the result meaningless rather than refining it.

/** Minimum |sin| of the turn at each vertex: below it three points are taken as aligned. */
const MIN_TURN_SIN = 0.01;
/** Relative pivot below which the 8×8 system is singular. */
const PIVOT_EPS = 1e-12;

/**
 * @typedef {[number, number]} Point
 * @typedef {number[]} Matrix3 row-major, length 9
 * @typedef {{ ok: true, H: Matrix3 } | { ok: false, reason: 'points' | 'aligned' | 'non-convex' | 'singular' | 'fold' }} HomographyResult
 */

/**
 * Homography sending src[i] to dst[i].
 *
 * Refused, with the reason:
 * - `points`: not four finite points on each side;
 * - `aligned`: two points coincide or three are aligned (on either side);
 * - `non-convex`: a quadrilateral, taken in the given order, is not strictly convex
 *   (a dart, or a bow-tie from clicks out of order);
 * - `singular`: the linear system has no unique solution;
 * - `fold`: the line sent to infinity crosses the source quadrilateral, so the sign of
 *   the Jacobian changes inside it and the map tears the board in two.
 *
 * Both orientations are accepted: a board whose home is on the left is the mirror
 * image of one whose home is on the right, not a mistake.
 *
 * @param {Point[]} src
 * @param {Point[]} dst
 * @returns {HomographyResult}
 */
export function computeHomography(src, dst) {
    if (!isQuad(src) || !isQuad(dst)) return { ok: false, reason: 'points' };
    for (const quad of [src, dst]) {
        const shape = quadShape(quad);
        if (shape !== 'convex') return { ok: false, reason: shape };
    }

    const Ts = hartley(src);
    const Td = hartley(dst);
    const ns = src.map((p) => applyHomography(Ts, p));
    const nd = dst.map((p) => applyHomography(Td, p));

    // In normalised coordinates the source centroid sits at the origin, inside a convex
    // quadrilateral whose image is finite, so ĥ8 ≠ 0 and fixing it to 1 loses nothing.
    const A = [];
    const b = [];
    for (let i = 0; i < 4; i++) {
        const [x, y] = ns[i];
        const [u, v] = nd[i];
        A.push([x, y, 1, 0, 0, 0, -u * x, -u * y]);
        b.push(u);
        A.push([0, 0, 0, x, y, 1, -v * x, -v * y]);
        b.push(v);
    }
    const h = solve(A, b);
    if (!h) return { ok: false, reason: 'singular' };
    const Hn = [...h, 1];

    const H = normalise(multiply(invertHartley(Td), multiply(Hn, Ts)));
    if (!H) return { ok: false, reason: 'singular' };

    // w is affine in (x, y): one sign at the four corners means one sign on the whole
    // convex quadrilateral.
    const ws = src.map(([x, y]) => H[6] * x + H[7] * y + H[8]);
    if (!(ws.every((w) => w > 0) || ws.every((w) => w < 0))) return { ok: false, reason: 'fold' };

    return { ok: true, H };
}

/**
 * @param {Matrix3} H
 * @param {Point} p
 * @returns {Point} [NaN, NaN] at infinity
 */
export function applyHomography(H, [x, y]) {
    const w = H[6] * x + H[7] * y + H[8];
    if (w === 0) return [NaN, NaN];
    return [(H[0] * x + H[1] * y + H[2]) / w, (H[3] * x + H[4] * y + H[5]) / w];
}

/**
 * Inverse homography, scaled so that its last entry is 1 when it is non-zero.
 * @param {Matrix3} H
 * @returns {Matrix3 | null} null when H is singular
 */
export function invertHomography(H) {
    const inv = invert(H);
    return inv && normalise(inv);
}

/**
 * Jacobian of the map at p, as [[∂u/∂x, ∂u/∂y], [∂v/∂x, ∂v/∂y]]: the local scale and
 * anisotropy that tell whether a zone is large enough in pixels to be read.
 * @param {Matrix3} H
 * @param {Point} p
 * @returns {[[number, number], [number, number]]}
 */
export function jacobian(H, [x, y]) {
    const w = H[6] * x + H[7] * y + H[8];
    const u = (H[0] * x + H[1] * y + H[2]) / w;
    const v = (H[3] * x + H[4] * y + H[5]) / w;
    return [
        [(H[0] - u * H[6]) / w, (H[1] - u * H[7]) / w],
        [(H[3] - v * H[6]) / w, (H[4] - v * H[7]) / w]
    ];
}

function isQuad(pts) {
    return Array.isArray(pts) && pts.length === 4 && pts.every((p) => Array.isArray(p) && p.length === 2 && Number.isFinite(p[0]) && Number.isFinite(p[1]));
}

/**
 * @param {Point[]} q
 * @returns {'convex' | 'aligned' | 'non-convex'}
 */
function quadShape(q) {
    let sign = 0;
    for (let i = 0; i < 4; i++) {
        const a = q[(i + 3) % 4];
        const b = q[i];
        const c = q[(i + 1) % 4];
        const e1x = b[0] - a[0];
        const e1y = b[1] - a[1];
        const e2x = c[0] - b[0];
        const e2y = c[1] - b[1];
        const n1 = Math.hypot(e1x, e1y);
        const n2 = Math.hypot(e2x, e2y);
        if (n1 === 0 || n2 === 0) return 'aligned';
        const s = (e1x * e2y - e1y * e2x) / (n1 * n2);
        if (Math.abs(s) < MIN_TURN_SIN) return 'aligned';
        if (sign === 0) sign = Math.sign(s);
        else if (Math.sign(s) !== sign) return 'non-convex';
    }
    // Four turns on one side and four vertices make a simple convex polygon, and the
    // four vertex triples are every triple of the quad, so none is aligned either.
    return 'convex';
}

/** Similarity moving the centroid to the origin and the mean distance to it to √2. */
function hartley(pts) {
    const cx = pts.reduce((s, p) => s + p[0], 0) / pts.length;
    const cy = pts.reduce((s, p) => s + p[1], 0) / pts.length;
    const mean = pts.reduce((s, p) => s + Math.hypot(p[0] - cx, p[1] - cy), 0) / pts.length;
    const k = Math.SQRT2 / mean;
    return [k, 0, -k * cx, 0, k, -k * cy, 0, 0, 1];
}

/** Exact inverse of a hartley() similarity, without a determinant test that its translation would fool. */
function invertHartley([k, , tx, , , ty]) {
    return [1 / k, 0, -tx / k, 0, 1 / k, -ty / k, 0, 0, 1];
}

/** Gaussian elimination with partial pivoting; null when a pivot vanishes. */
function solve(A, b) {
    const n = b.length;
    const M = A.map((row, i) => [...row, b[i]]);
    const scale = Math.max(...M.flat().map(Math.abs)) || 1;
    for (let col = 0; col < n; col++) {
        let piv = col;
        for (let r = col + 1; r < n; r++) if (Math.abs(M[r][col]) > Math.abs(M[piv][col])) piv = r;
        if (Math.abs(M[piv][col]) < PIVOT_EPS * scale) return null;
        [M[col], M[piv]] = [M[piv], M[col]];
        for (let r = col + 1; r < n; r++) {
            const f = M[r][col] / M[col][col];
            if (f === 0) continue;
            for (let c = col; c <= n; c++) M[r][c] -= f * M[col][c];
        }
    }
    const x = new Array(n);
    for (let r = n - 1; r >= 0; r--) {
        let s = M[r][n];
        for (let c = r + 1; c < n; c++) s -= M[r][c] * x[c];
        x[r] = s / M[r][r];
    }
    return x;
}

function multiply(A, B) {
    const C = new Array(9);
    for (let r = 0; r < 3; r++) for (let c = 0; c < 3; c++) C[r * 3 + c] = A[r * 3] * B[c] + A[r * 3 + 1] * B[3 + c] + A[r * 3 + 2] * B[6 + c];
    return C;
}

function invert(m) {
    const [a, b, c, d, e, f, g, h, i] = m;
    const A = e * i - f * h;
    const B = -(d * i - f * g);
    const C = d * h - e * g;
    const det = a * A + b * B + c * C;
    // Hadamard's bound makes the test blind to the scale of each row.
    const rows = [0, 3, 6].map((o) => Math.hypot(m[o], m[o + 1], m[o + 2]));
    if (!Number.isFinite(det) || Math.abs(det) <= PIVOT_EPS * rows[0] * rows[1] * rows[2]) return null;
    return [A / det, -(b * i - c * h) / det, (b * f - c * e) / det, B / det, (a * i - c * g) / det, -(a * f - c * d) / det, C / det, -(a * h - b * g) / det, (a * e - b * d) / det];
}

function normalise(H) {
    const s = H[8];
    if (!Number.isFinite(s) || s === 0) {
        const norm = Math.max(...H.map(Math.abs));
        return norm > 0 && Number.isFinite(norm) ? H.map((v) => v / norm) : null;
    }
    return H.map((v) => v / s);
}
