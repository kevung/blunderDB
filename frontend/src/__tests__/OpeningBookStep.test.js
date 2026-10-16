/** L'étape d'Ouverture : consigne, tableau d'analyse et résumé du rollout. */
import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import OpeningBookStep from '../components/OpeningBookStep.svelte';
import en from '../i18n/locales/en.json';

afterEach(cleanup);

const SIGNATURE = 'v1; cubeful; 0-ply plays, cube and leaves; seed 1; truncation 7 half-moves; exact bearoff when covered';
const question = { kind: 'opening', step: 'opening', opening: '31', reply: '21', score: '4-2', openingPlay: '' };
const verdict = (extra = {}) => ({
    legal: true,
    notation: '13/10 6/5',
    played: 1,
    right: false,
    ranked: [
        { move: '8/5 6/5', gap: 0, ci95: 0, tied: true },
        { move: '13/10 6/5', gap: 0.06, ci95: 0.008, tied: false }
    ],
    games: 216,
    signature: SIGNATURE,
    exact_bearoff: false,
    ...extra
});

describe('OpeningBookStep', () => {
    test('le résumé du rollout est court, dit la vérité du bearoff, et garde la signature au survol', () => {
        const { getByTestId } = render(OpeningBookStep, { question, verdict: verdict(), revealed: true });
        const line = getByTestId('opening-precision');
        expect(line.textContent?.trim()).toBe('Rollout 0-ply, truncated at 7, 216 games, without the exact bearoff table');
        expect(line.getAttribute('title')).toBe(SIGNATURE);
        expect(line.textContent).not.toContain('exact bearoff when covered');
    });

    test('la table de bearoff exacte servie est dite', () => {
        const { getByTestId } = render(OpeningBookStep, { question, verdict: verdict({ exact_bearoff: true }), revealed: true });
        expect(getByTestId('opening-precision').textContent).toContain(en.training.opening.withBearoff);
    });
});
