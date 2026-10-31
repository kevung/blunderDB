// Board geometry for the vision pipeline: pure functions over numbers and
// ImageData-shaped buffers, usable from the main thread and from a Worker.

export { computeHomography, applyHomography, invertHomography } from './homography.js';
export * from './boardModel.js';
export { calibrateBoard } from './calibration.js';
export { rectifyZone, sampleImage } from './rectify.js';
export { presenceScore, nextPresence, createPresenceTracker, PRESENCE_LOST, PRESENCE_FOUND } from './presence.js';
export { boardRois, WITNESS_MAX } from './rois.js';
