/**
 * The import of a database reports its matches in the preview, and in the result what it kept
 * apart: matches already held, living collections frozen or left on a manual list, families
 * the source was too old to read.
 */

import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';

import ImportProgressModal from '../components/ImportProgressModal.svelte';
import { importAnalysisStore, importResultStore } from '../stores/importModalStore';
import { get } from 'svelte/store';
import en from '../i18n/locales/en.json';
import fr from '../i18n/locales/fr.json';
import de from '../i18n/locales/de.json';
import el from '../i18n/locales/el.json';
import es from '../i18n/locales/es.json';
import fi from '../i18n/locales/fi.json';
import it from '../i18n/locales/it.json';
import ja from '../i18n/locales/ja.json';
import ru from '../i18n/locales/ru.json';

afterEach(cleanup);

/** @param {any} props */
const mount = (props) => render(ImportProgressModal, props);

const analysis = { toAdd: 0, toMerge: 0, toSkip: 3, total: 3, matches: 4, matchesKnown: 2, importPath: '' };
const result = {
    added: 0,
    merged: 0,
    skipped: 3,
    total: 3,
    matches: 4,
    matchesKnown: 2,
    livingCollectionsFrozen: ['Blunders'],
    livingCollectionsOnManual: ['Mine', 'Other'],
    unreadable: ['transcriptions']
};

/** @param {HTMLElement} c @param {string} id */
const text = (c, id) => c.querySelector(`[data-testid="${id}"]`)?.textContent ?? null;

describe('the preview', () => {
    test('shows the matches to add and those already held, and is not empty when a match comes', () => {
        const { container } = mount({ visible: true, mode: 'preview', analysis });
        expect(text(container, 'import-matches')).toContain('4');
        expect(text(container, 'import-matches-known')).toContain('2');
        expect(container.textContent).not.toContain('Nothing to import');
    });

    test('shows no match block when the source holds none', () => {
        const { container } = mount({ visible: true, mode: 'preview', analysis: { ...analysis, matches: 0, matchesKnown: 0 } });
        expect(text(container, 'import-matches')).toBeNull();
    });
});

describe('the result', () => {
    test('shows the matches held already and names what was frozen, left on a manual list or unreadable', () => {
        const { container } = mount({ visible: true, mode: 'completed', result });
        expect(text(container, 'result-matches-known')).toContain('2');
        expect(text(container, 'import.livingFrozen')).toContain('Blunders');
        expect(text(container, 'import.livingOnManual')).toContain('Mine, Other');
        expect(text(container, 'import.unreadable')).toContain('transcriptions');
    });

    test('shows the directions read back and the events created, as the CLI does', () => {
        const { container } = mount({ visible: true, mode: 'completed', result: { ...result, directions: 2, rencontres: 1 } });
        expect(text(container, 'result-directions')).toContain('2');
        expect(text(container, 'result-rencontres')).toContain('1');
    });

    test('shows no direction block when the source brought none', () => {
        const { container } = mount({ visible: true, mode: 'completed', result });
        expect(text(container, 'result-directions')).toBeNull();
    });

    test('shows no note when nothing was set apart', () => {
        const { container } = mount({
            visible: true,
            mode: 'completed',
            result: { ...result, livingCollectionsFrozen: [], livingCollectionsOnManual: [], unreadable: [] }
        });
        expect(container.querySelector('.warning')).toBeNull();
    });

    test('the stores start with the new fields', () => {
        expect(get(importAnalysisStore)).toMatchObject({ matches: 0, matchesKnown: 0 });
        expect(get(importResultStore)).toMatchObject({ matchesKnown: 0, directions: 0, rencontres: 0, unreadable: [] });
    });
});

describe('the keys exist in the nine languages', () => {
    test.each(['matchesToAdd', 'matchesToSkip', 'matchesAdded', 'matchesKnown', 'directionsRead', 'rencontresCreated', 'livingFrozen', 'livingOnManual', 'unreadable'])('%s', (k) => {
        for (const l of [en, fr, de, el, es, fi, it, ja, ru]) expect(/** @type {Record<string, string>} */ (l.import)[k]).toBeTruthy();
    });
});
