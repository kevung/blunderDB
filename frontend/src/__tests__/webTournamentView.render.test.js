/**
 * TournamentView.svelte : la page du moteur porte des noms saisis ; son cadre peut lire son
 * document (origine conservée, pour mesurer sa hauteur) mais ne doit jamais exécuter de script.
 */
import { describe, test, expect, afterEach, vi } from 'vitest';
import { render, cleanup, waitFor } from '@testing-library/svelte';

vi.mock('../web/api.js', () => ({
    callConditional: () => Promise.resolve({ changed: true, etag: 'x', body: { html: '<html><head></head><body><table></table></body></html>' } })
}));

import TournamentView from '../web/TournamentView.svelte';

afterEach(() => cleanup());

describe('TournamentView', () => {
    test('the frame keeps allow-same-origin and never allows scripts', async () => {
        const { container } = render(TournamentView, { props: { tournamentId: 4 } });
        await waitFor(() => expect(container.querySelector('iframe')).not.toBeNull());
        const tokens = (container.querySelector('iframe')?.getAttribute('sandbox') || '').split(/\s+/).filter(Boolean);
        expect(tokens).toContain('allow-same-origin');
        expect(tokens).not.toContain('allow-scripts');
        expect(tokens).toEqual(['allow-same-origin']);
    });
});
