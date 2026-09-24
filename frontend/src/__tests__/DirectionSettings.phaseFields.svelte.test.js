/**
 * Les réglages de phase qui n'avaient pas de champ (#455, D8.2) : consolante, réconciliation,
 * recharge pour un tableau ; taille des poules et qualifiés pour un tournoi toutes rondes ; le
 * rythme prévu (minutes par point) pour le tournoi. Chacun est figé avec sa raison dès que le
 * tirage de sa phase est fait.
 *
 * Un fichier `.svelte.test.js` : la configuration est un `$state`, comme dans la vue, sans quoi
 * un champ qui dépend d'un autre (la recharge de la réconciliation) ne se montrerait jamais.
 */
/* global $state -- rune compilée par le greffon Svelte dans un fichier .svelte.test.js */
import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

import DirectionSettings from '../components/direction/DirectionSettings.svelte';

afterEach(cleanup);

/** @param {object[]} phases */
const configOf = (phases) => ({ name: 'Open', tables: { count: 8 }, phases });
const q = (/** @type {Element} */ c, /** @type {string} */ id) => /** @type {HTMLInputElement | null} */ (c.querySelector(`[data-testid="${id}"]`));

describe('un tableau', () => {
    test('la consolante se coche ; la réconciliation suit la consolante, la recharge la réconciliation', async () => {
        const config = $state(configOf([{ kind: 'bracket', length: 9 }]));
        const { container } = render(DirectionSettings, { props: { config, directionState: 'draft' } });
        expect(q(container, 'direction-settings-reconciliation-1')).toBeNull();
        await fireEvent.click(/** @type {Element} */ (q(container, 'direction-settings-consolation-1')));
        await tick();
        expect(config.phases[0].consolation).toBe(true);
        expect(q(container, 'direction-settings-recharge-1')).toBeNull();
        await fireEvent.click(/** @type {Element} */ (q(container, 'direction-settings-reconciliation-1')));
        await tick();
        expect(config.phases[0].reconciliation).toBe(true);
        expect(q(container, 'direction-settings-recharge-1')).toBeTruthy();
    });

    test('décocher la consolante décoche ce qui en dépend', async () => {
        const config = $state(configOf([{ kind: 'bracket', length: 9, consolation: true, reconciliation: true, recharge: true }]));
        const { container } = render(DirectionSettings, { props: { config, directionState: 'draft' } });
        await fireEvent.click(/** @type {Element} */ (q(container, 'direction-settings-consolation-1')));
        await tick();
        expect(config.phases[0]).toMatchObject({ consolation: false, reconciliation: false, recharge: false });
    });

    test('après le tirage, les cases sont grisées avec leur raison', () => {
        const config = $state(configOf([{ kind: 'bracket', length: 9, consolation: true }]));
        const { container } = render(DirectionSettings, {
            props: { config, directionState: 'running', opened: 1, locks: [{ phase: 1, kind: 'bracket', locked: true, reason: 'drawn' }] }
        });
        expect(q(container, 'direction-settings-consolation-1')?.disabled).toBe(true);
        expect(q(container, 'direction-settings-reconciliation-1')?.disabled).toBe(true);
        expect(q(container, 'direction-settings-consolation-1-frozen')?.textContent).toMatch(/draw/i);
    });

    test('une consolante sans barème à elle le dit', async () => {
        const config = $state(configOf([{ kind: 'bracket', length: 9 }]));
        const { container } = render(DirectionSettings, { props: { config, directionState: 'draft' } });
        expect(q(container, 'direction-settings-conso-scale-hint')).toBeNull();
        await fireEvent.click(/** @type {Element} */ (q(container, 'direction-settings-consolation-1')));
        await tick();
        expect(q(container, 'direction-settings-conso-scale-hint')).toBeTruthy();
    });
});

describe('un tournoi toutes rondes', () => {
    test('la taille des poules et les qualifiés ont leur champ', async () => {
        const config = $state(configOf([{ kind: 'round_robin', length: 5 }]));
        const { container } = render(DirectionSettings, { props: { config, directionState: 'draft' } });
        const size = /** @type {HTMLInputElement} */ (q(container, 'direction-settings-group-size-1'));
        const qual = /** @type {HTMLInputElement} */ (q(container, 'direction-settings-qualifiers-1'));
        await fireEvent.input(size, { target: { value: '5' } });
        await fireEvent.input(qual, { target: { value: '1' } });
        expect(config.phases[0]).toMatchObject({ group_size: 5, qualifiers: 1 });
        expect(q(container, 'direction-settings-consolation-1')).toBeNull();
    });
});

describe('le tournoi', () => {
    test('le rythme prévu (minutes par point) a son champ', async () => {
        const config = $state(configOf([{ kind: 'bracket', length: 9 }]));
        const { container } = render(DirectionSettings, { props: { config, directionState: 'draft' } });
        await fireEvent.input(/** @type {Element} */ (q(container, 'direction-settings-min-per-point')), { target: { value: '6.5' } });
        expect(config.min_per_point).toBe(6.5);
    });
});
