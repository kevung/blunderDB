// Rectification of a model-space zone by inverse mapping: each output pixel centre is
// sent through the model→image homography and the source is sampled bilinearly.
// Images are ImageData-shaped ({ width, height, data }) with one channel (luminance)
// or four (RGBA); nothing here touches a canvas.

import { applyHomography } from './homography.js';

/**
 * @typedef {{ width: number, height: number, data: ArrayLike<number> }} Image
 * @typedef {{ width: number, height: number, data: Float32Array, coverage: number }} GrayImage
 *   `coverage` is the share of output pixels whose source lies inside the image.
 */

/**
 * Rectified luminance of a model rectangle.
 *
 * Pixels whose source falls outside the image are NaN, so a statistic over the zone
 * can skip them instead of reading a fabricated border; `coverage` says how many are
 * real.
 *
 * @param {Image} image one channel or RGBA
 * @param {number[]} H model → image homography
 * @param {import('./boardModel.js').Rect} rect zone in model units
 * @param {number} width output width in pixels
 * @param {number} height output height in pixels
 * @returns {GrayImage}
 */
export function rectifyZone(image, H, rect, width, height) {
    if (!(Number.isInteger(width) && width > 0 && Number.isInteger(height) && height > 0)) throw new RangeError(`output size ${width}×${height}`);
    if (!(rect.x1 > rect.x0 && rect.y1 > rect.y0)) throw new RangeError('empty zone');
    const channels = channelCount(image);
    const out = new Float32Array(width * height);
    const sx = (rect.x1 - rect.x0) / width;
    const sy = (rect.y1 - rect.y0) / height;
    let inside = 0;
    for (let j = 0; j < height; j++) {
        const y = rect.y0 + (j + 0.5) * sy;
        for (let i = 0; i < width; i++) {
            const [px, py] = applyHomography(H, [rect.x0 + (i + 0.5) * sx, y]);
            const v = sampleLuma(image, channels, px, py);
            out[j * width + i] = v;
            if (!Number.isNaN(v)) inside++;
        }
    }
    return { width, height, data: out, coverage: inside / (width * height) };
}

/**
 * Luminance at a continuous image position, pixel centres at integer + 0.5.
 * NaN outside the image.
 * @param {Image} image
 * @param {number} x
 * @param {number} y
 */
export function sampleImage(image, x, y) {
    return sampleLuma(image, channelCount(image), x, y);
}

function channelCount(image) {
    const n = image.data.length / (image.width * image.height);
    if (n !== 1 && n !== 4) throw new RangeError(`image with ${n} channels`);
    return n;
}

function luma(image, channels, x, y) {
    const o = (y * image.width + x) * channels;
    const d = image.data;
    // Rec. 601 weights, the ones canvas-based pipelines and the 320×180 descriptors use.
    return channels === 1 ? d[o] : 0.299 * d[o] + 0.587 * d[o + 1] + 0.114 * d[o + 2];
}

function sampleLuma(image, channels, px, py) {
    const { width: w, height: h } = image;
    if (!(px >= 0 && px <= w && py >= 0 && py <= h)) return NaN;
    // Clamp to the outermost pixel centres so the half-pixel border still samples.
    const fx = Math.min(Math.max(px - 0.5, 0), w - 1);
    const fy = Math.min(Math.max(py - 0.5, 0), h - 1);
    const x0 = Math.floor(fx);
    const y0 = Math.floor(fy);
    const x1 = Math.min(x0 + 1, w - 1);
    const y1 = Math.min(y0 + 1, h - 1);
    const ax = fx - x0;
    const ay = fy - y0;
    const top = luma(image, channels, x0, y0) * (1 - ax) + luma(image, channels, x1, y0) * ax;
    const bottom = luma(image, channels, x0, y1) * (1 - ax) + luma(image, channels, x1, y1) * ax;
    return top * (1 - ay) + bottom * ay;
}
