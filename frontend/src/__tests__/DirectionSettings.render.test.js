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

/*
 * Les têtes de série (#394) : l'option est VISIBLE mais ÉTEINTE, et le défaut est une décision
 * — l'étude du moteur conclut « pas de têtes de série protégées ». Un test le tient, parce
 * qu'un défaut décidé se perd exactement comme un défaut oublié.
 */
describe('les têtes de série', () => {
    const bracket = {
        name: 'Open de Lyon',
        tables: { count: 8 },
        phases: [{ kind: 'bracket', length: 5 }]
    };

    test('l’option est là, éteinte, et dit pourquoi', () => {
        const { container } = render(DirectionSettings, { props: { config: bracket, directionState: 'draft' } });
        const boxes = [...container.querySelectorAll('input[type="checkbox"]')];
        const seeding = boxes.find((b) => (b.closest('label')?.textContent || '').includes('Seeding'));
        expect(seeding).toBeTruthy();
        expect(seeding.checked).toBe(false);
        // La raison est dans l'infobulle, pas dans une leçon à l'écran.
        expect(seeding.closest('label').getAttribute('title')).toContain('no protected seeds');
    });

    test('cochée, elle écrit le placement par cote dans la configuration', async () => {
        const config = { ...bracket, phases: [{ kind: 'bracket', length: 5 }] };
        const { container } = render(DirectionSettings, { props: { config, directionState: 'draft' } });
        const boxes = [...container.querySelectorAll('input[type="checkbox"]')];
        const seeding = boxes.find((b) => (b.closest('label')?.textContent || '').includes('Seeding'));
        await fireEvent.click(seeding);
        expect(config.phases[0].seeding).toBe('rating');
        await fireEvent.click(seeding);
        expect(config.phases[0].seeding).toBeUndefined();
    });
});
