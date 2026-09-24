/**
 * Clore et rouvrir un tournoi, confirmés sur place (#441, D5.8).
 *
 * « Clore » terminait le tournoi en un clic, avec sept ou dix matchs en cours ; « Rouvrir »
 * passait par un `window.confirm` natif, le seul dialogue natif de la vue. Les deux se
 * confirment désormais comme « Tout lancer » : sur place, un second clic — et clore sans match
 * en cours reste un clic.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

import StandingsView from '../components/direction/StandingsView.svelte';

afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
});

const VIEW = { finished: false, pool: 0, retained: 0, payable: 0, entrants: 4, sections: [] };
const q = (/** @type {Element} */ c, /** @type {string} */ id) => /** @type {Element} */ (c.querySelector(`[data-testid="${id}"]`));

describe('clore le tournoi', () => {
    test('sans match en cours, un clic suffit', async () => {
        const onClose = vi.fn();
        const { container } = render(StandingsView, { props: { view: VIEW, running: 0, onClose } });
        await fireEvent.click(q(container, 'direction-standings-close'));
        expect(onClose).toHaveBeenCalledTimes(1);
        expect(q(container, 'direction-standings-confirm')).toBeNull();
    });

    test('avec des matchs en cours, il dit combien et attend un second clic', async () => {
        const onClose = vi.fn();
        const { container } = render(StandingsView, { props: { view: VIEW, running: 7, onClose } });
        await fireEvent.click(q(container, 'direction-standings-close'));
        expect(onClose).not.toHaveBeenCalled();
        expect(q(container, 'direction-standings-confirm').textContent).toContain('7');
        await fireEvent.click(q(container, 'direction-standings-confirm-go'));
        expect(onClose).toHaveBeenCalledTimes(1);
        expect(q(container, 'direction-standings-confirm')).toBeNull();
    });

    test('annuler ne clôt rien', async () => {
        const onClose = vi.fn();
        const { container } = render(StandingsView, { props: { view: VIEW, running: 3, onClose } });
        await fireEvent.click(q(container, 'direction-standings-close'));
        await fireEvent.click(q(container, 'direction-standings-confirm-cancel'));
        expect(onClose).not.toHaveBeenCalled();
        expect(q(container, 'direction-standings-confirm')).toBeNull();
    });
});

describe('rouvrir le tournoi', () => {
    test('la confirmation est sur place, sans dialogue natif', async () => {
        const native = vi.spyOn(window, 'confirm');
        const onReopen = vi.fn();
        const { container } = render(StandingsView, { props: { view: { ...VIEW, finished: true }, onReopen } });
        await fireEvent.click(q(container, 'direction-standings-reopen'));
        expect(onReopen).not.toHaveBeenCalled();
        await fireEvent.click(q(container, 'direction-standings-confirm-go'));
        expect(onReopen).toHaveBeenCalledTimes(1);
        expect(native).not.toHaveBeenCalled();
    });
});
