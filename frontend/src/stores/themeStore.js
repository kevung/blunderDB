import { writable, get } from 'svelte/store';
import { GetTheme, SaveTheme } from '../../wailsjs/go/main/Config.js';
import { applyThemeTokens, THEME_SYSTEM, THEME_NAMES } from '../utils/themes.js';
import { applyBoardPalette } from './boardColorsStore.js';
import { logger } from '../utils/logger.js';

// Le thème nommé courant (#286, fiche I.30). `system` par défaut : un outil
// n'impose pas son clair ou son sombre à un bureau qui a déjà tranché.
export const themeStore = writable(THEME_SYSTEM);

/**
 * Applique un thème : les jetons de l'interface, puis la palette du plateau.
 * `withBoard` à false n'applique que la chrome — c'est ce qu'on veut au
 * démarrage, où la palette persistée de l'utilisateur doit primer sur celle du
 * thème, et ce qu'on ne veut PAS quand il vient d'en choisir un.
 * @param {string} name
 * @param {{withBoard?: boolean}} [opts]
 */
export function applyTheme(name, { withBoard = true } = {}) {
    const theme = applyThemeTokens(name);
    if (withBoard) applyBoardPalette(theme.board);
}

/** Charge le thème persisté et l'applique, sans toucher à la palette. */
export async function initTheme() {
    let name = THEME_SYSTEM;
    try {
        name = (await GetTheme()) || THEME_SYSTEM;
    } catch (err) {
        logger.error('could not read the theme:', err);
    }
    if (!THEME_NAMES.includes(name)) name = THEME_SYSTEM;
    themeStore.set(name);
    // Sans la palette : celle que l'utilisateur a réglée est déjà chargée par
    // initBoardColors, et la réécrire au démarrage effacerait son travail à
    // chaque lancement.
    applyTheme(name, { withBoard: false });

    // `system` suit le bureau, y compris quand il change en cours de session.
    if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
        const query = window.matchMedia('(prefers-color-scheme: dark)');
        const listener = () => {
            if (get(themeStore) === THEME_SYSTEM) applyTheme(THEME_SYSTEM, { withBoard: false });
        };
        if (typeof query.addEventListener === 'function') query.addEventListener('change', listener);
    }
}

/** Choisit un thème : l'applique, palette comprise, et le persiste. */
export async function setTheme(name) {
    if (!THEME_NAMES.includes(name)) return;
    themeStore.set(name);
    applyTheme(name);
    try {
        await SaveTheme(name);
    } catch (err) {
        logger.error('could not save the theme:', err);
    }
}
