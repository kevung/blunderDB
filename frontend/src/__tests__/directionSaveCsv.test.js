/**
 * « Enregistrer… » à côté de « Copier (CSV) », pour le classement et pour l'annuaire (#454,
 * D8.1). Le classement du mardi se montrait en collant le presse-papier quelque part ; il
 * s'enregistre désormais dans un fichier, et le fichier est le CSV copié (tenu en Go).
 */
import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

import StandingsView from '../components/direction/StandingsView.svelte';
import DirectoryPanel from '../components/direction/DirectoryPanel.svelte';

afterEach(cleanup);

const VIEW = { finished: false, pool: 0, retained: 0, payable: 0, entrants: 4, sections: [] };

describe('enregistrer le classement', () => {
    test('le bouton est à côté de la copie et appelle onSave, pas onCSV', async () => {
        const onSave = vi.fn();
        const onCSV = vi.fn();
        const { getByTestId } = render(StandingsView, { props: { view: VIEW, onSave, onCSV } });
        await fireEvent.click(getByTestId('direction-standings-save'));
        expect(onSave).toHaveBeenCalledTimes(1);
        expect(onCSV).not.toHaveBeenCalled();
    });
});

describe("enregistrer l'annuaire", () => {
    test('le bouton est à côté de la copie et appelle onSave', async () => {
        const onSave = vi.fn();
        const onExport = vi.fn();
        const { getByTestId } = render(DirectoryPanel, { props: { onSave, onExport } });
        await fireEvent.click(getByTestId('direction-directory-toggle'));
        await fireEvent.click(getByTestId('direction-directory-save'));
        expect(onSave).toHaveBeenCalledTimes(1);
        expect(onExport).not.toHaveBeenCalled();
    });
});
