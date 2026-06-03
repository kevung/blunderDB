import { writable } from 'svelte/store';
import { GetBoardColors, SaveBoardColors } from '../../wailsjs/go/main/Config.js';
import { logger } from '../utils/logger.js';

// User-customisable board palette. Defaults mirror the historical hard-coded
// values in Board.svelte (and DefaultBoardColors() in config.go).
export const DEFAULT_BOARD_COLORS = {
    background: '#f0f0f0',
    border: '#333333',
    point1: '#d9d9d9',
    point2: '#a6a6a6',
    checker1: '#333333',
    checker2: '#ffffff',
    dice: '#ffffff',
    diceDot: '#000000',
    cube: '#ffffff'
};

export const BOARD_COLOR_KEYS = Object.keys(DEFAULT_BOARD_COLORS);

export const boardColorsStore = writable({ ...DEFAULT_BOARD_COLORS });

// Keep only known keys and fall back to defaults for missing/empty values.
function sanitize(colors) {
    const out = { ...DEFAULT_BOARD_COLORS };
    if (colors && typeof colors === 'object') {
        for (const key of BOARD_COLOR_KEYS) {
            if (typeof colors[key] === 'string' && colors[key].trim() !== '') {
                out[key] = colors[key];
            }
        }
    }
    return out;
}

// Load persisted colours at startup. Falls back to defaults on any error.
export async function initBoardColors() {
    try {
        const persisted = await GetBoardColors();
        boardColorsStore.set(sanitize(persisted));
    } catch (err) {
        logger.error('Failed to load board colors, using defaults:', err);
        boardColorsStore.set({ ...DEFAULT_BOARD_COLORS });
    }
}

// Update a single colour and persist the whole palette.
export function setBoardColor(key, value) {
    if (!BOARD_COLOR_KEYS.includes(key)) return;
    boardColorsStore.update((current) => {
        const next = { ...current, [key]: value };
        SaveBoardColors(next).catch((err) => logger.error('Failed to save board colors:', err));
        return next;
    });
}

// Restore the default palette and persist it.
export function resetBoardColors() {
    const next = { ...DEFAULT_BOARD_COLORS };
    boardColorsStore.set(next);
    SaveBoardColors(next).catch((err) => logger.error('Failed to save board colors:', err));
}
