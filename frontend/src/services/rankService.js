import { get } from 'svelte/store';
import { positionsStore } from '../stores/positionStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import { loadPositionsByFilters } from './positionService.js';
import { setStatusBarMessage } from './databaseService.js';
import { tMsg } from '../i18n';

// « Positions voisines » (ADR-0043), le geste plutôt que le jeton.
//
// Le menu contextuel du plateau et le raccourci clavier lancent exactement la
// requête que l'utilisateur aurait tapée — `s like<id>` — et pas une seconde
// façon de classer : le classement a un seul chemin, et c'est la grammaire.
// Ce qui change ici est seulement d'où part le geste.

/** Classe les voisines de la position courante. */
export async function rankNeighboursOfCurrentPosition() {
    const id = positionsStore.idAt(get(currentPositionIndexStore));
    if (!id) {
        setStatusBarMessage(tMsg('similar.noPosition'));
        return;
    }
    const command = `s like${id}`;
    await loadPositionsByFilters({
        filters: [`like${id}`],
        likeFilter: true,
        likeTargetId: id,
        searchCommand: command
    });
}
