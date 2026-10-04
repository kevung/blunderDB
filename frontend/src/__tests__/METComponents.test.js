import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup, waitFor } from '@testing-library/svelte';

const status = { name: 'Rockwell-Kazaross', current: 'Kazaross-XG2', different: true };
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    AnalysisMETStatus: vi.fn(() => Promise.resolve(status)),
    ListMETs: vi.fn(() =>
        Promise.resolve([
            { id: 0, name: 'Kazaross-XG2', current: false },
            { id: 3, name: 'Rockwell-Kazaross', current: true }
        ])
    ),
    ImportMET: vi.fn(),
    SetCurrentMET: vi.fn(() => Promise.resolve())
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({ OpenMETDialog: vi.fn(() => Promise.resolve('')) }));

import METBadge from '../components/METBadge.svelte';
import METSettings from '../components/METSettings.svelte';

afterEach(cleanup);

describe('MET de la base', () => {
    test('le badge nomme la table d’une analyse « MET différente »', async () => {
        render(METBadge, { props: { positionId: 7, analysis: { analysisType: 'DoublingCube' } } });
        await waitFor(() => expect(screen.getByText(/Rockwell-Kazaross/)).toBeTruthy());
    });

    test('aucun badge sans analyse', async () => {
        const { container } = render(METBadge, { props: { positionId: 7, analysis: null } });
        expect(container.querySelector('.met-different')).toBeNull();
    });

    test('les réglages sélectionnent la table courante', async () => {
        render(METSettings);
        const select = await waitFor(() => screen.getByRole('combobox'));
        await waitFor(() => expect(select.value).toBe('3'));
    });
});
