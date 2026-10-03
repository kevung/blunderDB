/**
 * La fiche de résultat : un raccourci d'une touche ne vaut jamais dans un champ, Entrée valide
 * le vainqueur choisi, la fiche attend le backend et reste ouverte sur un échec, et les gestes
 * qui retirent un match se confirment.
 */
import { describe, test, expect, vi, afterEach } from 'vitest';
import { answerConfirm } from './confirmHelper.js';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import ResultCard from '../components/direction/ResultCard.svelte';

const cell = { table: 3, length: 7, matchId: 'm1', a: 'pa', b: 'pb', aName: 'Alice', bName: 'Bruno' };

/** @param {Record<string, unknown>} [props] */
function mount(props = {}) {
    const handlers = { onResult: vi.fn(), onForfeit: vi.fn(), onMove: vi.fn(), onCancel: vi.fn(), onClose: vi.fn(), ...props };
    render(ResultCard, { props: { cell, ...handlers } });
    return handlers;
}

const flush = async () => {
    await tick();
    await Promise.resolve();
    await tick();
};

afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
});

describe('clavier', () => {
    test('← / → dans le champ de score ne valident rien', async () => {
        const h = mount();
        const input = screen.getByTestId('direction-result-score-a');
        await fireEvent.keyDown(input, { key: 'ArrowLeft' });
        await fireEvent.keyDown(input, { key: 'ArrowRight' });
        await fireEvent.keyDown(input, { key: 'Enter' });
        expect(h.onResult).not.toHaveBeenCalled();
    });

    test('→ choisit un vainqueur sans valider ; Entrée valide', async () => {
        const h = mount();
        const card = screen.getByTestId('direction-result-card');
        await fireEvent.keyDown(card, { key: 'ArrowRight' });
        expect(h.onResult).not.toHaveBeenCalled();
        await fireEvent.keyDown(card, { key: 'Enter' });
        await flush();
        expect(h.onResult).toHaveBeenCalledWith('m1', 'pb', 0, 0, '');
        expect(h.onClose).toHaveBeenCalled();
    });

    test('Entrée dans le champ de score valide le vainqueur choisi, score compris', async () => {
        const h = mount();
        const card = screen.getByTestId('direction-result-card');
        await fireEvent.keyDown(card, { key: 'ArrowLeft' });
        const a = screen.getByTestId('direction-result-score-a');
        await fireEvent.input(a, { target: { value: '7' } });
        await fireEvent.keyDown(a, { key: 'Enter' });
        await flush();
        expect(h.onResult).toHaveBeenCalledWith('m1', 'pa', 7, 0, '');
    });
});

describe('Entrée dans le champ de table', () => {
    test('déplace le match, même avec un vainqueur choisi', async () => {
        const h = mount();
        const card = screen.getByTestId('direction-result-card');
        await fireEvent.keyDown(card, { key: 'ArrowLeft' });
        await fireEvent.click(screen.getByTestId('direction-result-more'));
        const table = screen.getByTestId('direction-result-move-table');
        await fireEvent.input(table, { target: { value: '5' } });
        await fireEvent.keyDown(table, { key: 'Enter' });
        await flush();
        expect(h.onResult).not.toHaveBeenCalled();
        expect(h.onMove).toHaveBeenCalledWith('m1', 5);
    });
});

describe('attente du backend', () => {
    test('un échec garde la fiche ouverte et le dit', async () => {
        const h = mount({ onResult: vi.fn().mockResolvedValue(false) });
        await fireEvent.click(screen.getByTestId('direction-result-winner-a'));
        await flush();
        expect(h.onClose).not.toHaveBeenCalled();
        expect(screen.getByTestId('direction-result-error')).toBeTruthy();
    });

    test('une exception garde aussi la fiche ouverte', async () => {
        const h = mount({ onResult: vi.fn().mockRejectedValue(new Error('x')) });
        await fireEvent.click(screen.getByTestId('direction-result-winner-a'));
        await flush();
        expect(h.onClose).not.toHaveBeenCalled();
    });

    test('un succès ferme la fiche après la réponse', async () => {
        /** @type {(v: boolean) => void} */
        let resolve = () => {};
        const h = mount({ onResult: vi.fn(() => new Promise((r) => (resolve = r))) });
        await fireEvent.click(screen.getByTestId('direction-result-winner-a'));
        await flush();
        expect(h.onClose).not.toHaveBeenCalled();
        resolve(true);
        await flush();
        expect(h.onClose).toHaveBeenCalled();
    });
});

describe('confirmations', () => {
    test('annuler le match se confirme ; refuser ne fait rien', async () => {
        const h = mount();
        await fireEvent.click(screen.getByTestId('direction-result-more'));
        await fireEvent.click(screen.getByTestId('direction-result-cancel'));
        await answerConfirm(false);
        expect(h.onCancel).not.toHaveBeenCalled();
        await fireEvent.click(screen.getByTestId('direction-result-cancel'));
        await answerConfirm(true);
        await flush();
        expect(h.onCancel).toHaveBeenCalledWith('m1');
    });

    test('le forfait nomme le vainqueur et se confirme', async () => {
        const h = mount();
        await fireEvent.click(screen.getByTestId('direction-result-more'));
        const btn = screen.getByTestId('direction-result-forfeit-a');
        expect(btn.textContent).toContain('Bruno');
        await fireEvent.click(btn);
        await answerConfirm(true);
        await flush();
        expect(h.onForfeit).toHaveBeenCalledWith('m1', 'pb', '');
    });
});
