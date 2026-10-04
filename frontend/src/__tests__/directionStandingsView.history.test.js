/**
 * Le Classement mène à l'historique d'un joueur, où chaque résultat se corrige : le nom d'une
 * ligne est un bouton quand la vue sait où mener.
 */
import { test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, screen } from '@testing-library/svelte';

import StandingsView from '../components/direction/StandingsView.svelte';

afterEach(cleanup);

const view = /** @type {any} */ ({ sections: [{ name: '', rows: [{ id: 'p1', rank: 1, name: 'Alice', wins: 1, losses: 0 }] }] });

test('un clic sur le nom ouvre l’historique de ce joueur', () => {
    const onHistory = vi.fn();
    render(StandingsView, { props: { view, onHistory } });
    screen.getByTestId('direction-standings-history').click();
    expect(onHistory).toHaveBeenCalledWith('Alice');
});

test('sans destination, le nom reste un simple texte', () => {
    render(StandingsView, { props: { view } });
    expect(screen.queryByTestId('direction-standings-history')).toBeNull();
    expect(screen.getByText('Alice')).toBeTruthy();
});
