/**
 * DirectionSettings.render.test.js
 *
 * Régression : l'écran des Réglages plantait au montage, et rien ne le voyait.
 *
 * La cause est un piège de Svelte 5 : le composant recevait une prop nommée `state`, et `$x`
 * est résolu en abonnement à un store dès qu'un `x` est en portée. La rune `$state(null)`
 * ajoutée pour la liste de contrôle (#385) était donc compilée en « abonne-toi au store
 * `state` », ce qui lève `store_invalid_shape` au montage. Aucun test unitaire ne montait ce
 * composant, aucune spec de bout en bout n'ouvrait encore ce panneau : le défaut a vécu une
 * journée entière sur `main`.
 *
 * Ce test monte le composant. C'est tout ce qu'il faut : le plantage était au montage.
 */

import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

import DirectionSettings from '../components/direction/DirectionSettings.svelte';

afterEach(cleanup);

const config = {
    name: 'Open de Lyon',
    tables: { count: 8 },
    phases: [{ kind: 'swiss_lives', length: 7, lives: 2, target: 16 }]
};

describe('l’écran des Réglages se monte', () => {
    test('en préparation, avec ses cartes de format', () => {
        const { container } = render(DirectionSettings, { props: { config, directionState: 'draft' } });
        expect(container.querySelector('.settings')).toBeTruthy();
        expect(container.querySelectorAll('.formats .card').length).toBeGreaterThan(0);
    });

    test('en cours de tournoi, sans les cartes de format', () => {
        const { container } = render(DirectionSettings, { props: { config, directionState: 'running' } });
        expect(container.querySelector('.settings')).toBeTruthy();
        expect(container.querySelector('.formats')).toBeNull();
        // Et la phrase qui dit ce qui est figé, plutôt que des champs absents.
        expect(container.querySelector('.frozen-note')).toBeTruthy();
    });

    /* Le format d'une phase verrouillée reste À SA PLACE, grisé avec sa raison : un contrôle
       absent se lit comme une erreur de recherche, un contrôle grisé se lit comme une règle. */
    test('un format verrouillé montre sa raison au lieu de disparaître', () => {
        const { container } = render(DirectionSettings, {
            props: {
                config,
                directionState: 'running',
                opened: 1,
                locks: [{ phase: 1, kind: 'swiss_lives', locked: true, reason: 'started' }]
            }
        });
        const frozen = container.querySelector('.phase-kind.frozen');
        expect(frozen).toBeTruthy();
        // La langue par défaut des tests est l'anglais : la raison se rend, quelle qu'elle soit.
        expect(frozen.textContent).toContain('matches under way');
        expect(container.querySelector('select.phase-kind')).toBeNull();
    });

    /* En cours de tournoi, enregistrer montre d'abord ce qui va changer. En préparation, rien
       n'est encore décidé, et la confirmation ne coûterait qu'un clic. */
    test('enregistrer en cours de tournoi passe par la liste de contrôle', async () => {
        let applied = 0;
        const preview = { changes: [{ code: 'target', phase: 1, from: '16', to: '8' }], refusals: [], locks: [], opened: 1 };
        const { container } = render(DirectionSettings, {
            props: {
                config,
                directionState: 'running',
                opened: 1,
                onPreview: async () => preview,
                onApply: () => (applied += 1)
            }
        });
        await fireEvent.click(container.querySelector('.actions button.primary'));
        expect(applied).toBe(0);
        const confirm = container.querySelector('.confirm');
        expect(confirm).toBeTruthy();
        expect(confirm.textContent).toContain('switch');
    });

    test('en préparation, enregistrer applique directement', async () => {
        let applied = 0;
        const { container } = render(DirectionSettings, {
            props: { config, directionState: 'draft', onPreview: async () => ({ changes: [], refusals: [] }), onApply: () => (applied += 1) }
        });
        await fireEvent.click(container.querySelector('.actions button.primary'));
        expect(applied).toBe(1);
        expect(container.querySelector('.confirm')).toBeNull();
    });
});
