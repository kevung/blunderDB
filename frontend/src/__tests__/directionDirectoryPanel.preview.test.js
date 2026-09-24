/**
 * L'aperçu d'un CSV collé dans l'annuaire (#442).
 *
 * Cinquante lignes dont une sans séparateur et un doublon s'annonçaient « 50 lignes prêtes à
 * être inscrites », sans une erreur. L'aperçu doit montrer l'erreur et le doublon AVANT
 * « Inscrire », et le doublon n'entre que si le directeur le coche.
 */

import { describe, test, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

import DirectoryPanel from '../components/direction/DirectoryPanel.svelte';

afterEach(cleanup);

const PREVIEW = {
    rows: [
        { name: 'Hugo Andrieu', club: 'Lyon', rating: 5, entries: 0 },
        { name: 'Léa Bonnet', club: 'Lyon', rating: 4, entries: 0 }
    ],
    errors: [{ line: 3, code: 'noSeparator', text: 'ligne sans séparateur' }],
    skipped: [],
    warnings: [{ line: 4, code: 'duplicate', firstLine: 1, row: { name: 'Hugo Andrieu', club: 'Lyon', rating: 5, entries: 0 } }]
};

async function openPreview(onImport) {
    const utils = render(DirectoryPanel, { props: { onParse: async () => structuredClone(PREVIEW), onImport } });
    await fireEvent.click(utils.getByTestId('direction-directory-toggle'));
    await fireEvent.input(utils.getByTestId('direction-directory-paste'), { target: { value: 'x' } });
    await fireEvent.click(utils.getByTestId('direction-directory-read'));
    await tick();
    await tick();
    return utils;
}

describe("l'aperçu de l'annuaire", () => {
    test("l'erreur et le doublon se lisent avant « Inscrire », et le doublon n'entre pas par défaut", async () => {
        const onImport = vi.fn();
        const { getByTestId, container } = await openPreview(onImport);
        const preview = getByTestId('direction-directory-preview');
        expect(container.querySelectorAll('.errors li')).toHaveLength(1);
        expect(preview.querySelector('.errors')?.textContent).toContain('ligne sans séparateur');
        expect(getByTestId('direction-directory-warnings').textContent).toContain('Hugo Andrieu');
        expect(/** @type {HTMLInputElement} */ (getByTestId('direction-directory-force-4')).checked).toBe(false);
        expect(preview.textContent).toContain('2');

        await fireEvent.click(getByTestId('direction-directory-confirm'));
        expect(onImport).toHaveBeenCalledTimes(1);
        expect(onImport.mock.calls[0][0].map((r) => r.name)).toEqual(['Hugo Andrieu', 'Léa Bonnet']);
    });

    test('un doublon coché est inscrit quand même', async () => {
        const onImport = vi.fn();
        const { getByTestId } = await openPreview(onImport);
        await fireEvent.click(getByTestId('direction-directory-force-4'));
        await fireEvent.click(getByTestId('direction-directory-confirm'));
        expect(onImport.mock.calls[0][0]).toHaveLength(3);
    });
});
