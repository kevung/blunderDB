import { logger } from '../utils/logger.js';

// Un seul rendu du plateau (#278, fiche I.22).
//
// Le plateau est déjà du SVG : two.js le dessine ainsi. Ce qui manquait,
// c'était UNE fonction pour en prendre une copie — le clone, les styles
// calculés recopiés dans les attributs, la sérialisation — au lieu de ce bloc
// réécrit dans chaque exportateur. Deux copies d'un même rendu finissent
// toujours par diverger, et celle qui diverge est celle qu'on ne regarde pas.
//
// Le SVG est ici le format PREMIER, et le PNG en dérive. C'est l'inverse de ce
// qui existait : on rasterisait pour copier, et il n'y avait pas de moyen
// d'obtenir le vectoriel — celui qu'on met dans un article, qu'on agrandit
// sans le flouter, et que le rapport HTML (#279) veut en ligne.

/** L'identifiant de l'élément qui contient le plateau. */
const BOARD_ELEMENT_ID = 'backgammon-board';

/**
 * Les propriétés de style que le clone doit porter en dur. Un SVG sérialisé
 * quitte le document, donc ses feuilles de style : sans cette recopie, il
 * s'ouvre en noir et blanc.
 */
const STYLE_PROPS =
    'fill stroke stroke-width stroke-linecap stroke-linejoin stroke-miterlimit opacity font-family font-size font-weight font-style text-anchor dominant-baseline visibility display'.split(' ');

/** La couleur de fond du plateau, celle que le rendu suppose sous lui. */
export const BOARD_BACKGROUND = '#f7f0e6';

/**
 * Prend une copie autonome du plateau affiché.
 *
 * @returns {{svg: string, width: number, height: number} | null} null quand il
 *   n'y a pas de plateau à l'écran — un cas que l'appelant doit nommer à
 *   l'utilisateur, pas une erreur à lancer.
 */
export function snapshotBoardSVG() {
    const boardEl = document.getElementById(BOARD_ELEMENT_ID);
    if (!boardEl) return null;
    const svgEl = boardEl.querySelector('svg');
    if (!svgEl) return null;

    const width = parseInt(svgEl.getAttribute('width')) || svgEl.clientWidth;
    const height = parseInt(svgEl.getAttribute('height')) || svgEl.clientHeight;

    const clone = /** @type {SVGSVGElement} */ (svgEl.cloneNode(true));
    clone.setAttribute('xmlns', 'http://www.w3.org/2000/svg');
    clone.setAttribute('width', String(width));
    clone.setAttribute('height', String(height));

    const origElements = svgEl.querySelectorAll('*');
    const clonedElements = clone.querySelectorAll('*');
    for (let i = 0; i < origElements.length; i++) {
        const cloned = clonedElements[i];
        if (!cloned || !(cloned instanceof SVGElement)) continue;
        const computed = window.getComputedStyle(origElements[i]);
        for (const prop of STYLE_PROPS) {
            const val = computed.getPropertyValue(prop);
            if (val) cloned.style.setProperty(prop, val);
        }
    }

    // Le fond est peint DANS le SVG, en premier enfant : un fichier ouvert
    // dans un navigateur ou glissé dans un traitement de texte n'a pas de
    // « fond du plateau » à lui prêter, et le damier sur du transparent se
    // lit mal partout où le blanc n'est pas garanti.
    const background = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
    background.setAttribute('x', '0');
    background.setAttribute('y', '0');
    background.setAttribute('width', String(width));
    background.setAttribute('height', String(height));
    background.setAttribute('fill', BOARD_BACKGROUND);
    clone.insertBefore(background, clone.firstChild);

    return { svg: new XMLSerializer().serializeToString(clone), width, height };
}

/**
 * Rasterise un SVG dans un canvas. `scale` 2 donne une image lisible sur un
 * écran dense sans que le fichier double de poids inutilement.
 *
 * @param {{svg: string, width: number, height: number}} snapshot
 * @param {number} [scale]
 * @returns {Promise<HTMLCanvasElement>}
 */
export function svgToCanvas(snapshot, scale = 2) {
    return new Promise((resolve, reject) => {
        const blob = new Blob([snapshot.svg], { type: 'image/svg+xml;charset=utf-8' });
        const url = URL.createObjectURL(blob);
        const img = new Image();
        img.onload = () => {
            try {
                const canvas = document.createElement('canvas');
                canvas.width = snapshot.width * scale;
                canvas.height = snapshot.height * scale;
                const ctx = canvas.getContext('2d');
                ctx.scale(scale, scale);
                // Le fond est déjà dans le SVG ; on le repeint ici pour le cas
                // où le rendu du navigateur laisse passer du transparent.
                ctx.fillStyle = BOARD_BACKGROUND;
                ctx.fillRect(0, 0, snapshot.width, snapshot.height);
                ctx.drawImage(img, 0, 0, snapshot.width, snapshot.height);
                resolve(canvas);
            } catch (err) {
                reject(err);
            } finally {
                URL.revokeObjectURL(url);
            }
        };
        img.onerror = () => {
            URL.revokeObjectURL(url);
            reject(new Error('the board SVG could not be rendered'));
        };
        img.src = url;
    });
}

/**
 * Le PNG du plateau, en base64 sans son préfixe `data:` — la forme que le
 * backend attend pour écrire un fichier ou alimenter le presse-papier.
 *
 * @param {{svg: string, width: number, height: number}} snapshot
 * @returns {Promise<string>}
 */
export async function snapshotToPNGBase64(snapshot) {
    const canvas = await svgToCanvas(snapshot);
    return canvas.toDataURL('image/png').replace(/^data:image\/png;base64,/, '');
}

/** Un nom de fichier lisible, daté, pour une image de plateau. */
export function boardImageFilename(extension) {
    const now = new Date();
    const stamp = [
        now.getFullYear(),
        String(now.getMonth() + 1).padStart(2, '0'),
        String(now.getDate()).padStart(2, '0'),
        '-',
        String(now.getHours()).padStart(2, '0'),
        String(now.getMinutes()).padStart(2, '0')
    ].join('');
    return `blunderdb-${stamp}.${extension}`;
}

/** @param {unknown} err */
export function logSnapshotFailure(err) {
    logger.error('board snapshot failed:', err);
}
