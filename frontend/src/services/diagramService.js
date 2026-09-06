import Two from 'two.js';
import { get } from 'svelte/store';
import { boardMetrics } from '../utils/boardGeometry.js';
import { layerOf, drawStaticScene, drawDynamicScene, drawFrame } from '../utils/boardScene.js';
import { defaultBoardConfig, applyPalette } from '../utils/boardConfig.js';
import { boardColorsStore } from '../stores/boardColorsStore.js';
import { BOARD_BACKGROUND } from './boardSnapshot.js';

// Dessiner une position QUELCONQUE, hors de l'écran (#279, fiche I.23).
//
// Le rapport HTML a besoin d'un diagramme par décision, et ces positions ne
// sont pas celle du plateau. Plutôt qu'un second dessinateur — ce que #278
// vient précisément d'éviter — on instancie une seconde surface two.js et on
// lui fait exécuter LES MÊMES fonctions de scène. Le rendu est donc le même
// par construction, palette de l'utilisateur comprise.
//
// L'élément n'est jamais attaché au document : two.js sait rendre dans un
// nœud SVG détaché, et un diagramme qui apparaîtrait une fraction de seconde
// à l'écran serait une nuisance visible.

/** La taille d'un diagramme de rapport. Assez grand pour être lisible
 *  imprimé, assez petit pour que dix tiennent dans un document. */
export const DIAGRAM_WIDTH = 520;
export const DIAGRAM_HEIGHT = 380;

/**
 * Rend une position en un document SVG autonome.
 *
 * @param {object} position La position au format du store (board, cube, dés…).
 * @param {{width?: number, height?: number, showPipcount?: boolean}} [opts]
 * @returns {string} Le SVG sérialisé, fond compris.
 */
export function renderPositionSVG(position, { width = DIAGRAM_WIDTH, height = DIAGRAM_HEIGHT, showPipcount = true } = {}) {
    const cfg = applyPalette(defaultBoardConfig(), get(boardColorsStore));
    const geom = boardMetrics(width, height, cfg.widthFactor);

    const two = new Two({ type: Two.Types.svg, width, height });

    const staticLayer = two.makeGroup();
    const dynamicLayer = two.makeGroup();
    const frameLayer = two.makeGroup();

    // Le repère des points est celui du joueur au trait, comme à l'écran.
    const flip = position?.player_on_roll === 1;
    drawStaticScene(layerOf(two, staticLayer), geom, cfg, flip);
    drawDynamicScene(layerOf(two, dynamicLayer), geom, cfg, position, { offeredCube: false, showPipcount, moves: [] });
    drawFrame(layerOf(two, frameLayer), geom, cfg);
    two.update();

    const svg = /** @type {SVGSVGElement} */ (two.renderer.domElement);
    svg.setAttribute('xmlns', 'http://www.w3.org/2000/svg');
    svg.setAttribute('width', String(width));
    svg.setAttribute('height', String(height));

    // Le fond dans le document, pour la même raison qu'à l'export d'image : un
    // SVG lu ailleurs n'a pas de fond à lui prêter.
    const background = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
    background.setAttribute('x', '0');
    background.setAttribute('y', '0');
    background.setAttribute('width', String(width));
    background.setAttribute('height', String(height));
    background.setAttribute('fill', BOARD_BACKGROUND);
    svg.insertBefore(background, svg.firstChild);

    const serialised = new XMLSerializer().serializeToString(svg);
    // two.js garde la surface vivante ; on la relâche pour qu'un rapport de
    // dix diagrammes ne laisse pas dix scènes derrière lui.
    two.clear();
    return serialised;
}
