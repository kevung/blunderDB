/** La question d'Ouverture : consigne, puis le tableau du livre et une marque juste/faux, sans prose. */
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
    test('la catégorie est nommée à côté du score', () => {
        const { getByTestId, queryByTestId, rerender } = render(OpeningBookStep, { question: { ...question, score: '2-1' }, verdict: null, revealed: false });
        expect(getByTestId('opening-category').textContent).toBe(en.training.opening.category.gammonSave);
        rerender({ question: { ...question, score: '4-2' }, verdict: null, revealed: false });
        expect(queryByTestId('opening-category')).toBeNull();
    });

    test('avant le coup : la consigne seule, sans tableau', () => {
        const { getByTestId, queryByTestId } = render(OpeningBookStep, { question, verdict: null, revealed: false });
        expect(getByTestId('opening-prompt').textContent).toContain('31');
        expect(queryByTestId('opening-ranked')).toBeNull();
        expect(queryByTestId('opening-after')).toBeNull();
    });

    test('un coup faux : la marque et le tableau, coup joué surligné, aucune phrase de verdict', () => {
        const { getByTestId, container } = render(OpeningBookStep, { question, verdict: verdict(), revealed: true });
        expect(getByTestId('training-verdict').textContent).toContain(en.training.opening.markWrong);
        const rows = getByTestId('opening-ranked').querySelectorAll('tbody tr');
        expect(rows).toHaveLength(2);
        expect(rows[1].classList.contains('played')).toBe(true);
        expect(container.querySelector('.hint')).toBeNull();
        expect(container.textContent).not.toContain('behind');
    });

    test('un coup juste montre aussi le tableau', () => {
        const { getByTestId } = render(OpeningBookStep, { question, verdict: verdict({ right: true, played: 0, notation: '8/5 6/5' }), revealed: true });
        expect(getByTestId('training-verdict').textContent).toContain(en.training.opening.markRight);
        expect(getByTestId('training-verdict').classList.contains('correct')).toBe(true);
        expect(getByTestId('opening-ranked').querySelectorAll('tbody tr')).toHaveLength(2);
    });

    test('un coup légal hors des candidats s’ajoute en bas du tableau, sans écart', () => {
        const { getByTestId } = render(OpeningBookStep, { question, verdict: verdict({ played: -1, notation: '24/21 24/23' }), revealed: true });
        expect(getByTestId('opening-unranked').textContent).toContain('24/21 24/23');
    });

    test('hors délai : le tableau seul, sans marque', () => {
        const { getByTestId, queryByTestId } = render(OpeningBookStep, { question, verdict: verdict({ legal: false, played: -1, notation: '' }), revealed: true, outOfTime: true });
        expect(queryByTestId('training-verdict')).toBeNull();
        expect(getByTestId('opening-ranked')).toBeTruthy();
    });

    test('la précision du rollout est en infobulle du tableau, la table de bearoff dite', () => {
        const { getByTestId } = render(OpeningBookStep, { question, verdict: verdict({ exact_bearoff: true }), revealed: true });
        const title = getByTestId('opening-ranked').getAttribute('title') ?? '';
        expect(title).toContain('216');
        expect(title).toContain(en.training.opening.withBearoff);
    });

    test('2nd roll : la consigne dit le coup d’ouverture déjà joué', () => {
        const reply = { ...question, step: 'reply', openingPlay: '8/5 6/5' };
        const { getByTestId } = render(OpeningBookStep, { question: reply, revealed: false });
        expect(getByTestId('opening-prompt').textContent).toContain('21');
        expect(getByTestId('opening-after').textContent).toContain('8/5 6/5');
    });
});
