/**
 * helpers/screenshotTools.js
 *
 * Fonctions partagées par les specs de capture documentaire
 * (screenshot.spec.js, screenshot-panels.spec.js) : optimisation PNG et
 * capture-plus-vérification de taille. Un seul endroit pour choisir
 * l'optimiseur disponible sur le PATH, plutôt qu'une copie par spec.
 */

import { spawnSync } from 'node:child_process';
import { statSync } from 'node:fs';

/** Lance le premier optimiseur PNG disponible sur le PATH ; sans-bruit s'il n'y en a aucun. */
export function optimise(file) {
    const candidates = [
        ['pngquant', ['--force', '--skip-if-larger', '--quality', '70-95', '--output', file, file]],
        ['oxipng', ['-o', '4', '--strip', 'safe', file]],
        ['optipng', ['-quiet', '-o2', file]]
    ];
    for (const [bin, args] of candidates) {
        if (spawnSync('which', [bin]).status !== 0) continue;
        const run = spawnSync(bin, args, { stdio: 'inherit' });
        return { bin, ok: run.status === 0 };
    }
    return null;
}

/**
 * Capture `page` vers `outputPath`, l'optimise, journalise son poids et
 * renvoie sa taille en octets (au test d'imposer sa propre limite).
 *
 * @param {import('@playwright/test').Page} page
 * @param {string} outputPath
 */
export async function captureAndOptimise(page, outputPath) {
    await page.screenshot({ path: outputPath, scale: 'css', animations: 'disabled' });
    const optimised = optimise(outputPath);
    const size = statSync(outputPath).size;
    // eslint-disable-next-line no-console -- le poids du fichier produit est la sortie utile de ce script
    console.log(`${outputPath}: ${(size / 1024).toFixed(0)} KiB${optimised ? ` (${optimised.bin})` : ''}`);
    return size;
}
