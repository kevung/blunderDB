/**
 * Le nom proposé quand on enregistre le classement ou l'annuaire (#454, D8.1) :
 * `<tournoi>-classement-<date>.csv`. Un nom lisible dans un dossier de téléchargements, sans
 * caractère qu'un système de fichiers refuse.
 */
import { describe, test, expect } from 'vitest';
import { csvFilename } from '../components/direction/labels.js';

const TUESDAY = new Date(2026, 8, 22, 9, 0);

describe('csvFilename', () => {
    test('tournoi, mot, date du jour', () => {
        expect(csvFilename('Open de Lyon', 'classement', TUESDAY)).toBe('Open-de-Lyon-classement-2026-09-22.csv');
    });

    test('les caractères interdits dans un nom de fichier disparaissent', () => {
        expect(csvFilename('Coupe : A/B ?', 'classement', TUESDAY)).toBe('Coupe-A-B-classement-2026-09-22.csv');
    });

    test('les accents restent : ce sont des lettres', () => {
        expect(csvFilename('Été à Évian', 'annuaire', TUESDAY)).toBe('Été-à-Évian-annuaire-2026-09-22.csv');
    });

    test('sans tournoi, le mot et la date suffisent', () => {
        expect(csvFilename('', 'annuaire', TUESDAY)).toBe('annuaire-2026-09-22.csv');
    });
});
