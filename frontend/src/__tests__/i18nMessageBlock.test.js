/**
 * `messageBlock` : le bloc de messages que le front passe au backend (issue #386).
 *
 * La page d'affichage d'un tournoi est écrite en Go, et le moteur Nicomaque n'émet que des
 * codes. Plutôt que de recopier neuf catalogues côté Go, le front lui passe le sien — d'où ce
 * test : ce qui part doit être complet, et replié sur l'anglais là où une langue manque.
 */
import { describe, test, expect, beforeEach } from 'vitest';
import { messageBlock, language, setLanguage } from '../i18n';

describe('le bloc de messages passé au backend', () => {
    beforeEach(() => {
        language.set('en');
    });

    test('porte tout ce dont la page a besoin pour ne montrer aucun code', () => {
        const block = messageBlock('direction');
        for (const section of ['label', 'note', 'warning', 'reason', 'section', 'format', 'term', 'credit']) {
            expect(Object.keys(block[section] || {}).length, section).toBeGreaterThan(0);
        }
        // Les mots du rendu : ce sont eux que le moteur demande par `Term`.
        expect(block.term.tables).toBeTruthy();
        expect(block.term.players).toContain('{n}');
        // Et le crédit, qui part en pied de page dans la langue de l'utilisateur.
        expect(block.credit.by).toContain('Nicolas Harmand');
    });

    test('parle la langue courante', async () => {
        await setLanguage('fr');
        expect(messageBlock('direction').term.tables).toBe('Tables');
        expect(messageBlock('direction').label.final).toBe('Finale');
        await setLanguage('de');
        expect(messageBlock('direction').term.tables).toBe('Tische');
    });

    test('une langue chargée porte déjà tout le bloc anglais, clé par clé', async () => {
        await setLanguage('fi');
        const own = messageBlock('direction');
        language.set('en');
        const en = messageBlock('direction');
        /** @type {(o: any, p?: string) => string[]} */
        const keys = (o, p = '') => Object.entries(o).flatMap(([k, v]) => (v && typeof v === 'object' ? keys(v, `${p}${k}.`) : [`${p}${k}`]));
        const has = (/** @type {any} */ o, /** @type {string} */ path) => path.split('.').reduce((x, k) => (x == null ? undefined : x[k]), o) !== undefined;
        for (const k of keys(en)) expect(has(own, k), k).toBe(true);
    });

    test('rend un objet vide plutôt que undefined pour une section inconnue', () => {
        expect(messageBlock('cettesectionnexistepas')).toEqual({});
    });
});
