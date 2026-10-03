import { get } from 'svelte/store';
import { ComputeStats, LoadPositionsByIDs, StatsReportHTML } from '../../wailsjs/go/database/Database.js';
import { SaveBoardImageDialog, SaveBoardSVG } from '../../wailsjs/go/gui/App.js';
import { statsFilterStore } from '../stores/statsStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { renderPositionSVG } from './diagramService.js';
import { logger } from '../utils/logger.js';
import { language, tMsg } from '../i18n';

// Le rapport HTML : un fichier AUTONOME (ni image externe, ni style distant,
// ni script). Le document est construit par le moteur (package report), le même
// que pour la ligne de commande et le démon ; l'écran ne fournit que les
// diagrammes, dessinés dans la palette du plateau, et le filtre courant du
// panneau Stats, qui est le périmètre du rapport.

/** Combien de décisions le rapport détaille. Dix : de quoi voir un motif,
 *  assez peu pour tenir dans un document qu'on lit. */
const REPORT_BLUNDERS = 10;

/**
 * Construit le rapport et propose de l'enregistrer.
 * @returns {Promise<string|null>} le chemin écrit, ou null.
 */
export async function exportHTMLReport() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDatabaseOpened'));
        return null;
    }
    setStatusBarMessage(tMsg('report.building'));
    let html;
    try {
        html = await buildReportHTML();
    } catch (err) {
        logger.error('could not build the report:', err);
        setStatusBarMessage(tMsg('report.failed', { err }));
        return null;
    }

    try {
        // Le sélecteur d'image sert aussi ici : il enregistre un texte à un
        // chemin choisi, ce dont un rapport a exactement besoin.
        const path = await SaveBoardImageDialog('html', reportFilename());
        if (!path) return null;
        await SaveBoardSVG(path, html);
        setStatusBarMessage(tMsg('report.saved', { path }));
        return path;
    } catch (err) {
        logger.error('could not save the report:', err);
        setStatusBarMessage(tMsg('report.failed', { err }));
        return null;
    }
}

/**
 * Le HTML du rapport, sans rien écrire. Exporté pour être testable.
 * Les diagrammes des dix décisions les plus coûteuses sont dessinés ici, dans
 * la palette du plateau ; le moteur dessine lui-même ceux qu'on ne lui donne pas.
 */
export async function buildReportHTML() {
    const filter = get(statsFilterStore);
    const stats = await ComputeStats(filter);
    const blunders = (stats?.TopBlunders ?? []).slice(0, REPORT_BLUNDERS);

    const ids = blunders.map((b) => b.PositionID).filter((id) => id > 0);
    const positions = ids.length > 0 ? (await LoadPositionsByIDs(ids)) || [] : [];
    /** @type {Record<number, string>} */
    const diagrams = {};
    for (const p of positions) {
        try {
            diagrams[p.id] = renderPositionSVG(p);
        } catch (err) {
            // Un diagramme qui ne se dessine pas laisse la place à celui du moteur.
            logger.error('could not draw a report diagram:', err);
        }
    }
    return StatsReportHTML(filter, get(language), diagrams);
}

function reportFilename() {
    const now = new Date();
    const stamp = [now.getFullYear(), String(now.getMonth() + 1).padStart(2, '0'), String(now.getDate()).padStart(2, '0')].join('');
    return `blunderdb-rapport-${stamp}.html`;
}
