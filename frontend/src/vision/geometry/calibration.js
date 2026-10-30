// Four clicks on the image become the model→image homography of the board.

import { CALIBRATION_CORNERS } from './boardModel.js';
import { computeHomography, invertHomography } from './homography.js';

/**
 * @param {[number, number][]} clicks image positions of the field corners, in the order
 *   of CALIBRATION_CORNERS: 1-point, 12-point, 13-point, 24-point of the bottom player
 * @returns {{ ok: true, modelToImage: number[], imageToModel: number[] } | { ok: false, reason: string }}
 */
export function calibrateBoard(clicks) {
    const res = computeHomography(CALIBRATION_CORNERS, clicks);
    if (!res.ok) return res;
    const imageToModel = invertHomography(res.H);
    if (!imageToModel) return { ok: false, reason: 'singular' };
    return { ok: true, modelToImage: res.H, imageToModel };
}
