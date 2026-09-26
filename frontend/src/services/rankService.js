import { get } from 'svelte/store';
import { positionsStore } from '../stores/positionStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import { loadPositionsByFilters } from './positionService.js';
import { setStatusBarMessage } from './databaseService.js';
import { tMsg } from '../i18n';

// « Positions voisines » (ADR-0043) : menu contextuel et raccourci lancent
// exactement `s like<id>`, le seul chemin du classement.

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
