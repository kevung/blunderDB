import { get } from 'svelte/store';
import { ComputeStats, LoadPositionsByIDs, LoadAnalysis } from '../../wailsjs/go/database/Database.js';
import { SaveBoardImageDialog, SaveBoardSVG } from '../../wailsjs/go/gui/App.js';
import { statsFilterStore } from '../stores/statsStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { renderPositionSVG } from './diagramService.js';
import { logger } from '../utils/logger.js';
import { t, tMsg } from '../i18n';

// Le rapport HTML : un fichier AUTONOME (ni image externe, ni style distant,
// ni script), diagrammes en SVG en ligne du même rendu qu'à l'écran. Le
// navigateur l'imprime en PDF : pas de générateur à embarquer. Il nomme en
// tête le filtre courant du panneau Stats, son périmètre.

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

/** Le HTML du rapport, sans rien écrire. Exporté pour être testable. */
export async function buildReportHTML() {
    const filter = get(statsFilterStore);
    const stats = await ComputeStats(filter);
    const blunders = (stats?.TopBlunders ?? []).slice(0, REPORT_BLUNDERS);

    const ids = blunders.map((b) => b.PositionID).filter((id) => id > 0);
    const positions = ids.length > 0 ? (await LoadPositionsByIDs(ids)) || [] : [];
    const byID = new Map(positions.map((p) => [p.id, p]));

    const sections = [];
    for (const [i, b] of blunders.entries()) {
        const position = byID.get(b.PositionID);
        let best;
        try {
            const analysis = await LoadAnalysis(b.PositionID);
            best = analysis?.checkerAnalysis?.moves?.[0]?.move ?? '';
        } catch {
            // Une analyse illisible ne doit pas coûter le rapport : la
            // section perd sa ligne « meilleur coup » et garde son diagramme.
            best = '';
        }
        sections.push(blunderSection(i + 1, b, position, best));
    }

    return document_(stats, sections.join('\n'));
}

/**
 * @param {number} rank
 * @param {any} blunder une entrée de `stats.TopBlunders`
 * @param {any} position la position correspondante, ou undefined
 * @param {string} bestMove
 */
function blunderSection(rank, blunder, position, bestMove) {
    const diagram = position ? renderPositionSVG(position) : '';
    const kind = blunder.DecisionType === 1 ? get(t)('report.cube') : get(t)('report.checker');
    const cost = (blunder.ErrorMP / 1000).toFixed(3);
    return `
<section class="blunder">
  <h3>${rank}. ${escapeHTML(kind)} — ${cost}</h3>
  <p class="meta">${escapeHTML(blunder.PlayerNames || '')}${blunder.MatchDate ? ' · ' + escapeHTML(String(blunder.MatchDate).slice(0, 10)) : ''}</p>
  <div class="diagram">${diagram}</div>
  ${bestMove ? `<p class="best">${escapeHTML(get(t)('report.bestMove'))} <b>${escapeHTML(bestMove)}</b></p>` : ''}
</section>`;
}

/**
 * @param {any} stats
 * @param {string} body
 */
function document_(stats, body) {
    const tr = get(t);
    const totals = stats?.Totals ?? {};
    const rows = [
        [tr('report.positions'), totals.NumPositions ?? 0],
        [tr('report.matches'), totals.NumMatches ?? 0],
        [tr('report.decisions'), totals.NumDecisions ?? 0],
        [tr('report.prGlobal'), fmt(stats?.PRGlobal)],
        [tr('report.prChecker'), fmt(stats?.PRChecker)],
        [tr('report.prCube'), fmt(stats?.PRCube)]
    ];
    return `<!doctype html>
<html lang="${escapeHTML(documentLanguage())}">
<head>
<meta charset="utf-8">
<title>${escapeHTML(tr('report.title'))}</title>
<style>
  body { font-family: system-ui, sans-serif; margin: 2rem auto; max-width: 46rem; color: #222; }
  h1 { font-size: 1.4rem; }
  h3 { font-size: 1rem; margin-bottom: 0.2rem; }
  table { border-collapse: collapse; margin: 1rem 0; }
  td { padding: 0.15rem 1rem 0.15rem 0; }
  td.value { font-weight: 600; text-align: right; }
  .meta { color: #666; margin: 0 0 0.4rem 0; font-size: 0.9rem; }
  .diagram svg { max-width: 100%; height: auto; }
  .blunder { break-inside: avoid; page-break-inside: avoid; margin-bottom: 1.6rem; }
  .footer { color: #666; font-size: 0.85rem; margin-top: 2rem; }
  @media print { body { margin: 0; max-width: none; } }
</style>
</head>
<body>
<h1>${escapeHTML(tr('report.title'))}</h1>
<p class="meta">${escapeHTML(tr('report.generated', { date: new Date().toLocaleString() }))}</p>
<table>
${rows.map(([label, value]) => `  <tr><td>${escapeHTML(String(label))}</td><td class="value">${escapeHTML(String(value))}</td></tr>`).join('\n')}
</table>
<h2>${escapeHTML(tr('report.worst'))}</h2>
${body || `<p>${escapeHTML(tr('report.noBlunder'))}</p>`}
<p class="footer">${escapeHTML(tr('report.footer'))}</p>
</body>
</html>
`;
}

/** @param {unknown} value */
function fmt(value) {
    return typeof value === 'number' && value > 0 ? value.toFixed(2) : '—';
}

function documentLanguage() {
    return (typeof document !== 'undefined' && document.documentElement.lang) || 'fr';
}

/**
 * Pas d'échappement en bloc (le rapport contient du SVG) : chaque fragment
 * venant des données importées est échappé un par un.
 */
/** @param {unknown} s */
function escapeHTML(s) {
    return String(s).replace(/[&<>"']/g, (c) => /** @type {Record<string, string>} */ ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);
}

function reportFilename() {
    const now = new Date();
    const stamp = [now.getFullYear(), String(now.getMonth() + 1).padStart(2, '0'), String(now.getDate()).padStart(2, '0')].join('');
    return `blunderdb-rapport-${stamp}.html`;
}
