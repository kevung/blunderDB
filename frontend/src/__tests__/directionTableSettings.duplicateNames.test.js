/**
 * Deux inscrits de même nom (un homonyme, ou les inscrits d'un tournoi repris deux fois) ne
 * doivent pas faire tomber la page : la liste des joueurs à affecter à une table est clée par
 * nom, et une clé dupliquée gèlerait toute la Direction.
 */
import { test, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';

import TableSettingsEditor from '../components/direction/TableSettingsEditor.svelte';

afterEach(cleanup);

test('un nom inscrit deux fois n est proposé qu une fois, sans erreur de rendu', () => {
    const { container } = render(TableSettingsEditor, {
        props: { tables: 2, settings: [], participants: ['Hugo Andrieu', 'Léa Bonnet', 'Hugo Andrieu'], onSave: () => {} }
    });
    const hugo = [...container.querySelectorAll('option')].filter((o) => (o.value || o.textContent).includes('Hugo Andrieu'));
    // Un seul choix par nom et par table (deux tables).
    expect(hugo.length).toBe(2);
});
