// Ce que le plateau montre quand ce n'est pas la position courante : en EDIT (onglet Recherche)
// et en EVAL (onglet Eval) il porte un brouillon sans lien avec la position étudiée, alors que la
// barre d'info du match et la barre d'état continuent de décrire celle-ci. Le plateau le dit, et
// la barre d'info se tait tant qu'elle ne décrit pas ce qu'on voit.
import { derived } from 'svelte/store';
import { statusBarModeStore } from '../stores/uiStore.js';

/** Mode du plateau brouillon → clé i18n de son bandeau. */
/** @type {Record<string, string>} */
const SCRATCH_BANNER = { EDIT: 'board.situation.search', EVAL: 'board.situation.eval' };

/** La clé du bandeau à afficher, ou null quand le plateau montre la position courante. */
export const boardBannerKeyStore = derived(statusBarModeStore, ($mode) => SCRATCH_BANNER[$mode] ?? null);
