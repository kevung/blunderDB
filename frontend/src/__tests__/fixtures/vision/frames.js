// Synthetic 320×180 frames for the vision tests: a textured scene whose regions can be
// changed one at a time, so a test names exactly what moved between two frames.

export const W = 320;
export const H = 180;

/** A deterministic texture: census needs local contrast, a flat field has none. */
export function texture(x, y, seed = 0) {
    return 40 + ((x * 37 + y * 91 + seed * 53 + ((x * y) % 7) * 17) % 160);
}

/**
 * @param {(x: number, y: number) => number} lum luminance 0–255 at each pixel
 * @param {number} [gain] multiplies every pixel, as an auto-exposure would
 */
export function makeFrame(lum, gain = 1) {
    const data = new Uint8ClampedArray(W * H * 4);
    for (let y = 0; y < H; y++) {
        for (let x = 0; x < W; x++) {
            const v = Math.min(255, Math.round(lum(x, y) * gain));
            const i = (y * W + x) * 4;
            data[i] = data[i + 1] = data[i + 2] = v;
            data[i + 3] = 255;
        }
    }
    return data;
}

/** @param {{ x: number, y: number, w: number, h: number }} r */
export const inside = (r, x, y) => x >= r.x && x < r.x + r.w && y >= r.y && y < r.y + r.h;

export const CALIBRATION = Object.freeze({
    width: W,
    height: H,
    rois: [
        { name: 'diceLeft', x: 60, y: 70, w: 40, h: 30 },
        { name: 'diceRight', x: 220, y: 70, w: 40, h: 30 }
    ],
    witness: { x: 0, y: 0, w: 48, h: 24 }
});

/** The base scene, with `changed` regions retextured by `seed`. */
export function scene(changed = [], seed = 1) {
    return makeFrame((x, y) => (changed.some((r) => inside(r, x, y)) ? texture(y, x, seed) : texture(x, y)));
}
