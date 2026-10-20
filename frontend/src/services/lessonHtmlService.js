import { LessonHTML, LessonPositionIDs, GetLesson, LoadPositionsByIDs } from '../../wailsjs/go/database/Database.js';
import { SaveBoardImageDialog, SaveBoardSVG } from '../../wailsjs/go/gui/App.js';
import { renderPositionSVG } from './diagramService.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';
import { language, tMsg } from '../i18n';
import { get } from 'svelte/store';

// La Leçon en un fichier HTML autonome. Le document est construit par le moteur
// (package report), le même que pour la ligne de commande et le démon ; l'écran
// ne fournit que les plateaux, dessinés par renderPositionSVG, le dessinateur
// du rapport, dans la palette choisie.

/**
 * Le HTML de la leçon id, sans rien écrire. Exporté pour être testable.
 * @param {number} id
 */
export async function buildLessonHTML(id) {
    const ids = [...new Set((await LessonPositionIDs(id)) || [])];
    const positions = ids.length > 0 ? (await LoadPositionsByIDs(ids)) || [] : [];
    /** @type {Record<number, string>} */
    const diagrams = {};
    for (const p of positions) {
        try {
            diagrams[p.id] = renderPositionSVG(p);
        } catch (err) {
            // Un plateau qui ne se dessine pas laisse la place à celui du moteur.
            logger.error('could not draw a lesson board:', err);
        }
    }
    return LessonHTML(id, get(language), diagrams);
}

/**
 * Construit la page de la leçon et propose de l'enregistrer.
 * @param {number} id
 * @returns {Promise<string|null>} le chemin écrit, ou null.
 */
export async function exportLessonHTML(id) {
    try {
        const html = await buildLessonHTML(id);
        const lesson = await GetLesson(id);
        const path = await SaveBoardImageDialog('html', lessonFilename(lesson?.name));
        if (!path) return null;
        await SaveBoardSVG(path, html);
        setStatusBarMessage(tMsg('lessonEditor.htmlSaved', { path }));
        return path;
    } catch (err) {
        logger.error('could not export the lesson page:', err);
        setStatusBarMessage(tMsg('lessonEditor.htmlFailed', { err }));
        return null;
    }
}

/** @param {string} [name] */
function lessonFilename(name) {
    const slug = (name || 'lecon')
        .normalize('NFD')
        .replace(/[̀-ͯ]/g, '')
        .replace(/[^a-zA-Z0-9]+/g, '-')
        .replace(/^-+|-+$/g, '')
        .toLowerCase();
    return `${slug || 'lecon'}.html`;
}
