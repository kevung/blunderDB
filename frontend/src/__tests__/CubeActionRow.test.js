/**
 * CubeActionRow.test.js — T2.5 : la rangée [D] [T] [P] [R].
 *
 * Ce qui se vérifie ici est la CIBLE : quatre boutons, ce qu'ils rendent, et
 * surtout ce qui s'allume. L'équivalence avec les touches est mesurée ailleurs
 * (transcriptionKeys.cubeMouse.test.js), parce qu'elle se compte en gestes et
 * non en pixels ; ce fichier tient l'autre moitié — la rangée dit de qui est le
 * tour, et n'offre jamais une cible qui ne répondrait à rien.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

import CubeActionRow from '../components/CubeActionRow.svelte';
import { COMMAND } from '../services/transcriptionKeys.js';

const buttons = () => [...document.querySelectorAll('button')];
const labels = () => buttons().map((b) => b.textContent.trim());

afterEach(cleanup);

describe('la rangée du videau', () => {
    test('quatre boutons : doubler, prendre, passer, abandonner', () => {
        render(CubeActionRow, { props: { canAct: true } });
        expect(buttons()).toHaveLength(4);
        expect(labels()).toEqual(['Double', 'Take', 'Pass', 'Resign']);
    });

    // Le camp au trait annonce : le double et l'abandon s'allument, les deux
    // réponses restent éteintes — il n'y a pas d'offre à répondre.
    test('camp au trait : [D] et [R] seuls actifs', () => {
        render(CubeActionRow, { props: { canAct: true, canAnswer: false } });
        expect(buttons().map((b) => b.disabled)).toEqual([false, true, true, false]);
    });

    // Le camp d'en face répond : les deux réponses, et rien d'autre. Le
    // clavier, lui, prend toujours `d` — un bouton éteint n'est pas un refus,
    // c'est une cible qu'on n'offre pas (ADR-0044).
    test('réponse attendue : [T] et [P] seuls actifs', () => {
        render(CubeActionRow, { props: { canAct: false, canAnswer: true } });
        expect(buttons().map((b) => b.disabled)).toEqual([true, false, false, true]);
    });

    test('chaque bouton rend son geste, une seule fois', async () => {
        const onGesture = vi.fn();
        const onResign = vi.fn();
        render(CubeActionRow, { props: { canAct: true, canAnswer: true, onGesture, onResign } });
        const [d, t, p, r] = buttons();

        await fireEvent.click(d);
        expect(onGesture).toHaveBeenCalledExactlyOnceWith(COMMAND.DOUBLE);
        onGesture.mockClear();

        await fireEvent.click(t);
        expect(onGesture).toHaveBeenCalledExactlyOnceWith(COMMAND.TAKE);
        onGesture.mockClear();

        await fireEvent.click(p);
        expect(onGesture).toHaveBeenCalledExactlyOnceWith(COMMAND.PASS);
        onGesture.mockClear();

        // [R] n'enregistre rien : il annonce, et le niveau vient ensuite.
        await fireEvent.click(r);
        expect(onResign).toHaveBeenCalledTimes(1);
        expect(onGesture).not.toHaveBeenCalled();
    });

    // Un vrai clic, et non un `fireEvent` : c'est le NAVIGATEUR qui refuse le
    // clic d'un bouton éteint, et un événement synthétique passerait outre.
    test('un bouton éteint ne rend rien', () => {
        const onGesture = vi.fn();
        render(CubeActionRow, { props: { canAct: true, canAnswer: false, onGesture } });
        buttons()[1].click();
        expect(onGesture).not.toHaveBeenCalled();
    });
});

describe('la résignation demande son niveau', () => {
    // Trois niveaux plus l'abandon, à la place des quatre cases : le geste sert
    // une fois par match, et trois boutons de plus à demeure auraient coûté la
    // place de ceux qui servent à chaque tour.
    test('la rangée devient les trois niveaux et « Annuler »', () => {
        render(CubeActionRow, { props: { resigning: true } });
        expect(labels()).toEqual(['Single', 'Gammon', 'Backgammon', 'Cancel']);
    });

    test('chaque niveau rend son chiffre', async () => {
        const onLevel = vi.fn();
        render(CubeActionRow, { props: { resigning: true, onLevel } });
        const cells = buttons();
        for (const level of [1, 2, 3]) {
            onLevel.mockClear();
            await fireEvent.click(cells[level - 1]);
            expect(onLevel).toHaveBeenCalledExactlyOnceWith(level);
        }
    });

    test('« Annuler » double Échap et n’enregistre aucun niveau', async () => {
        const onLevel = vi.fn();
        const onCancelResign = vi.fn();
        render(CubeActionRow, { props: { resigning: true, onLevel, onCancelResign } });
        await fireEvent.click(buttons()[3]);
        expect(onCancelResign).toHaveBeenCalledTimes(1);
        expect(onLevel).not.toHaveBeenCalled();
    });

    // Aucun des quatre gestes ordinaires ne reste offert pendant l'attente du
    // niveau : entre l'annonce et son chiffre, rien d'autre ne doit passer —
    // c'est la même règle que la phase RESIGN de la machine à touches.
    test('les gestes ordinaires disparaissent pendant l’attente', () => {
        render(CubeActionRow, { props: { resigning: true, canAct: true, canAnswer: true } });
        expect(labels()).not.toContain('Double');
        expect(labels()).not.toContain('Take');
    });
});
