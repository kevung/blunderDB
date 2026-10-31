// The known tournaments feed the autocomplete of the tournament field, on the list and on the sheet.
import { GetAllTournaments } from '../../wailsjs/go/database/Database.js';
import { tournamentsStore } from '../stores/tournamentStore.js';
import { readOfLibrary, isLibraryLeft } from '../stores/groupActivityStore.js';
import { logger } from '../utils/logger.js';

export async function loadTournaments() {
    try {
        const loaded = await readOfLibrary(() => GetAllTournaments());
        tournamentsStore.set(loaded || []);
    } catch (error) {
        if (isLibraryLeft(error)) return;
        logger.error('Error loading tournaments:', error);
    }
}
