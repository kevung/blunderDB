/**
 * t3-lib.js — données et petits outils de T3 (Rencontre : épreuve A poules → tableau, épreuve B
 * suisse 2 vies). Les résultats sont tirés d'une graine fixe à partir des cotes (PR : plus bas =
 * meilleur), indépendamment de l'écran.
 */
import fs from 'node:fs';

export const A_PLAYERS = [
    ['Agathe Lemoine', 4.8], ['Benoît Carrel', 5.6], ['Chloé Ferrand', 6.1], ['David Pujol', 6.9],
    ['Elsa Marchand', 7.2], ['Fabien Roux', 7.8], ['Gaëlle Tessier', 8.3], ['Hugo Bastide', 8.9],
    ['Inès Vautrin', 9.4], ['Jules Perrin', 9.9], ['Karim Delorme', 10.5], ['Laure Gauthier', 11.0],
    ['Mathis Aubry', 11.6], ['Nadia Lefort', 12.2], ['Olivier Caron', 12.9], ['Pauline Brunet', 13.5]
];
/** B : 8 joueurs, dont 3 aussi inscrits en A (test de la Rencontre) et Léa, la directrice. */
export const B_PLAYERS = [
    ['Léa Chauvin', 7.0], ['Agathe Lemoine', 4.8], ['Hugo Bastide', 8.9], ['Pauline Brunet', 13.5],
    ['Quentin Morel', 6.4], ['Rose Valette', 9.1], ['Simon Arnaud', 10.2], ['Thaïs Guérin', 11.8]
];

/** Générateur déterministe (mulberry32). */
export function rng(seed) {
    let a = seed >>> 0;
    return () => {
        a = (a + 0x6d2b79f5) >>> 0;
        let t = a;
        t = Math.imul(t ^ (t >>> 15), t | 1);
        t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
        return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
}

/** Probabilité que a batte b d'après les PR (écart de 4 PR ≈ 3 contre 1). */
export function pWin(prA, prB) {
    return 1 / (1 + Math.pow(10, (prA - prB) / 8));
}

/** id blunderDB d'un nom (slug, comme entries.go) : sert seulement à relire la feuille. */
export function slug(name) {
    return name
        .normalize('NFD')
        .replace(/[̀-ͯ]/g, '')
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/^-|-$/g, '');
}

/** Lit la page murale : [{table, text}] par ligne de table, texte brut. */
export function readWall(file) {
    if (!fs.existsSync(file)) return null;
    return fs.readFileSync(file, 'utf8');
}
