/**
 * ResignLevelModal.test.js — la fenêtre des niveaux d'abandon par-dessus le
 * théâtre : trois niveaux de partie, le match entier, et « Annuler ».
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

import ResignLevelModal from '../components/ResignLevelModal.svelte';
import { FORFEIT_LEVEL } from '../services/transcriptionKeys.js';

const levels = () => [...document.querySelectorAll('[data-testid="resign-modal"] .resign-level')];

afterEach(cleanup);

describe('la fenêtre des niveaux d’abandon', () => {
    test('fermée, elle ne montre rien', () => {
        render(ResignLevelModal, { props: { open: false } });
        expect(document.querySelector('[data-testid="resign-modal"]')).toBeNull();
    });

    test('ouverte : simple, gammon, backgammon, puis le match sous la touche 4', () => {
        render(ResignLevelModal, { props: { open: true, player: 'Alice' } });
        expect(levels().map((b) => b.textContent.trim())).toEqual(['1Single', '2Gammon', '3Backgammon', '4The match']);
    });

    test('le match rend FORFEIT_LEVEL, « Annuler » n’en rend aucun', async () => {
        const onLevel = vi.fn();
        const onCancel = vi.fn();
        render(ResignLevelModal, { props: { open: true, onLevel, onCancel } });
        await fireEvent.click(levels()[3]);
        expect(onLevel).toHaveBeenCalledExactlyOnceWith(FORFEIT_LEVEL);
        await fireEvent.click(/** @type {HTMLElement} */ (document.querySelector('.resign-cancel')));
        expect(onCancel).toHaveBeenCalledTimes(1);
        expect(onLevel).toHaveBeenCalledTimes(1);
    });
});
