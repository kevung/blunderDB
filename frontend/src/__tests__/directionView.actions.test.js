/**
 * DirectionView.svelte, vue montée en entier : chaque geste du directeur (file, appariement
 * manuel, saisie de résultat, table, annulation, inscription, retrait, clôture, remarque) appelle
 * le bon binding avec les bons arguments, et ce qui s'affiche suit la vue que le service rend.
 */
import { must } from './helpers/must.js';
import { test, expect, vi, afterEach, beforeEach, describe } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';
import { answerConfirm } from './confirmHelper.js';

vi.mock('../../wailsjs/go/database/Database.js', async (importOriginal) => {
    const orig = await importOriginal();
    return Object.fromEntries(Object.keys(/** @type {object} */ (orig)).map((k) => [k, vi.fn().mockResolvedValue(null)]));
});

import * as DB from '../../wailsjs/go/database/Database.js';
import DirectionView from '../components/direction/DirectionView.svelte';
import { directionStore, openDirectionIdStore } from '../stores/directionStore.js';
import { resolveConfirm } from '../services/confirmService.js';
import { statusBarTextStore } from '../stores/uiStore.js';
import { t } from '../i18n';
import { HoldLibrary } from '../stores/libraryHold.js';

const tr = (/** @type {string} */ k, /** @type {any} */ p = undefined) => get(t)(k, p);
const players = [
    { id: 'pa', name: 'Alice', state: 'free', wins: 0, losses: 0 },
    { id: 'pb', name: 'Bruno', state: 'free', wins: 0, losses: 0 }
];
const base = { tournamentId: 1, state: 'running', eventCount: 1, config: { phases: [], tables: { count: 4 } }, proposals: [], running: [], warnings: [], players, outputDir: '' };
const startAB = { kind: 'start_match', a: 'pa', b: 'pb', length: 7, table: 1 };
const startCD = { kind: 'start_match', a: 'pc', b: 'pd', length: 7, table: 2 };
const cell = { table: 3, length: 7, matchId: 'm1', a: 'pa', b: 'pb', aName: 'Alice', bName: 'Bruno' };

/** @param {any} [view] */
async function mount(view = base) {
    openDirectionIdStore.set(1);
    render(DirectionView);
    directionStore.set(view);
    await flush();
}

async function goTab(/** @type {string} */ id) {
    await fireEvent.click(screen.getByTestId(`direction-tab-${id}`));
    await flush();
}

async function flush() {
    for (let i = 0; i < 6; i++) {
        await Promise.resolve();
        await tick();
    }
}

const status = () => /** @type {any} */ (get(statusBarTextStore));
const byText = (/** @type {Element} */ root, /** @type {string} */ label) => /** @type {HTMLElement} */ (must([...root.querySelectorAll('button')].find((b) => b.textContent?.trim() === label)));

beforeEach(() => {
    statusBarTextStore.set('');
    for (const f of Object.values(DB)) vi.mocked(f).mockReset().mockResolvedValue(null);
});
afterEach(() => {
    cleanup();
    resolveConfirm(false);
    directionStore.set(null);
    openDirectionIdStore.set(null);
});

describe('la file', () => {
    test('lancer une proposition envoie l’action et dit le match par ses noms', async () => {
        await mount({ ...base, proposals: [startAB] });
        vi.mocked(DB.ConfirmProposal).mockResolvedValue(/** @type {any} */ ({ ...base, eventCount: 2, proposals: [] }));
        const queue = screen.getByTestId('direction-queue');
        await fireEvent.click(byText(queue, tr('direction.proposals.launch')));
        await flush();
        expect(DB.ConfirmProposal).toHaveBeenCalledWith(1, JSON.stringify(startAB));
        expect(status()).toMatchObject({ i18nKey: 'direction.feedback.matchLaunched', i18nParams: { a: 'Alice', b: 'Bruno' } });
        expect(screen.getByTestId('direction-queue').querySelectorAll('li:not(.empty)')).toHaveLength(0);
    });

    test('un lancement refusé le dit et laisse la file telle quelle', async () => {
        await mount({ ...base, proposals: [startAB] });
        vi.mocked(DB.ConfirmProposal).mockRejectedValue(new Error('no'));
        await fireEvent.click(byText(screen.getByTestId('direction-queue'), tr('direction.proposals.launch')));
        await flush();
        expect(status()).toMatchObject({ i18nKey: 'direction.proposals.errorConfirm' });
        expect(screen.getByTestId('direction-queue').querySelectorAll('li:not(.empty)')).toHaveLength(1);
    });

    test('« tout lancer » se confirme puis lance toute la file en un appel', async () => {
        await mount({ ...base, proposals: [startAB, startCD] });
        vi.mocked(DB.ConfirmAllProposals).mockResolvedValue(/** @type {any} */ ({ ...base, eventCount: 3 }));
        await fireEvent.click(screen.getByTestId('direction-proposals-all'));
        expect(DB.ConfirmAllProposals).not.toHaveBeenCalled();
        await fireEvent.click(screen.getByTestId('direction-proposals-confirm-all'));
        await flush();
        expect(DB.ConfirmAllProposals).toHaveBeenCalledWith(1);
        expect(status()).toMatchObject({ i18nKey: 'direction.feedback.allLaunched' });
    });

    test('« tout lancer » annulé ne lance rien', async () => {
        await mount({ ...base, proposals: [startAB, startCD] });
        await fireEvent.click(screen.getByTestId('direction-proposals-all'));
        await fireEvent.click(byText(screen.getByTestId('direction-proposals-confirm'), tr('common.cancel')));
        await flush();
        expect(DB.ConfirmAllProposals).not.toHaveBeenCalled();
        expect(screen.queryByTestId('direction-proposals-confirm')).toBeNull();
    });
});

describe('l’appariement manuel', () => {
    /** @returns {Promise<HTMLElement>} */
    async function openManual() {
        vi.mocked(DB.FreeParticipants).mockResolvedValue(/** @type {any} */ (players));
        await mount();
        await fireEvent.click(screen.getByTestId('direction-manual-open'));
        return /** @type {HTMLElement} */ (must(document.querySelector('.manual')));
    }

    test('deux joueurs, une longueur : StartMatchManually les reçoit', async () => {
        const form = await openManual();
        const [a, b] = /** @type {HTMLSelectElement[]} */ ([...form.querySelectorAll('select')]);
        await fireEvent.change(a, { target: { value: 'pa' } });
        await fireEvent.change(b, { target: { value: 'pb' } });
        await fireEvent.input(must(form.querySelector('input')), { target: { value: '9' } });
        await fireEvent.click(byText(form, tr('direction.proposals.launch')));
        await flush();
        expect(DB.StartMatchManually).toHaveBeenCalledWith(1, 'pa', 'pb', 9, 0);
    });

    test('un joueur contre lui-même est refusé avant le service', async () => {
        const form = await openManual();
        const [a, b] = /** @type {HTMLSelectElement[]} */ ([...form.querySelectorAll('select')]);
        await fireEvent.change(a, { target: { value: 'pa' } });
        await fireEvent.change(b, { target: { value: 'pa' } });
        await fireEvent.click(byText(form, tr('direction.proposals.launch')));
        await flush();
        expect(DB.StartMatchManually).not.toHaveBeenCalled();
    });

    test('un appariement refusé par le service le dit', async () => {
        const form = await openManual();
        vi.mocked(DB.StartMatchManually).mockRejectedValue(new Error('busy'));
        const [a, b] = /** @type {HTMLSelectElement[]} */ ([...form.querySelectorAll('select')]);
        await fireEvent.change(a, { target: { value: 'pa' } });
        await fireEvent.change(b, { target: { value: 'pb' } });
        await fireEvent.click(byText(form, tr('direction.proposals.launch')));
        await flush();
        expect(status()).toMatchObject({ i18nKey: 'direction.proposals.errorManual' });
    });
});

describe('la fiche de résultat', () => {
    async function openCard() {
        vi.mocked(DB.TableGrid).mockResolvedValue(/** @type {any} */ ([cell]));
        await mount({ ...base, running: [{ matchId: 'm1' }] });
        await fireEvent.click(screen.getByTestId('direction-table-3'));
        await flush();
        return screen.getByTestId('direction-result-card');
    }

    test('le score et le vainqueur partent tels quels', async () => {
        await openCard();
        await fireEvent.input(screen.getByTestId('direction-result-score-a'), { target: { value: '7' } });
        await fireEvent.input(screen.getByTestId('direction-result-score-b'), { target: { value: '3' } });
        await fireEvent.click(screen.getByTestId('direction-result-winner-a'));
        await flush();
        expect(DB.EnterResult).toHaveBeenCalledWith(1, 'm1', 'pa', 7, 3, '');
    });

    test('un résultat non envoyé (la bibliothèque a changé) laisse la fiche ouverte et le dit', async () => {
        await openCard();
        vi.mocked(HoldLibrary).mockRejectedValueOnce(new Error('library replaced'));
        await fireEvent.click(screen.getByTestId('direction-result-winner-b'));
        await flush();
        expect(DB.EnterResult).not.toHaveBeenCalled();
        expect(status()).toMatchObject({ i18nKey: 'libraryGroups.writeDropped' });
        expect(screen.getByTestId('direction-result-card')).toBeTruthy();
    });

    test('un résultat refusé reste sur la fiche et le dit', async () => {
        await openCard();
        vi.mocked(DB.EnterResult).mockRejectedValue(new Error('no'));
        await fireEvent.click(screen.getByTestId('direction-result-winner-b'));
        await flush();
        expect(DB.EnterResult).toHaveBeenCalledWith(1, 'm1', 'pb', 0, 0, '');
        expect(status()).toMatchObject({ i18nKey: 'direction.result.error' });
        expect(screen.getByTestId('direction-result-error')).toBeTruthy();
    });

    test('changer de table envoie le numéro ; un refus se lit avec sa raison', async () => {
        await openCard();
        vi.mocked(DB.MoveMatchToTable).mockRejectedValue(new Error('direction: table 5 out of service'));
        await fireEvent.click(screen.getByTestId('direction-result-more'));
        await fireEvent.input(screen.getByTestId('direction-result-move-table'), { target: { value: '5' } });
        await fireEvent.click(screen.getByTestId('direction-result-move'));
        await flush();
        expect(DB.MoveMatchToTable).toHaveBeenCalledWith(1, 'm1', 5);
        expect(status()).toMatchObject({ i18nKey: 'direction.table.moveRefused', i18nParams: { reason: 'table 5 out of service' } });
    });

    test('annuler un match se confirme ; refusé, rien n’est annulé', async () => {
        await openCard();
        await fireEvent.click(screen.getByTestId('direction-result-more'));
        await fireEvent.click(screen.getByTestId('direction-result-cancel'));
        await answerConfirm(false);
        expect(DB.CancelMatch).not.toHaveBeenCalled();
        await fireEvent.click(screen.getByTestId('direction-result-cancel'));
        await answerConfirm(true);
        await vi.waitFor(() => expect(DB.CancelMatch).toHaveBeenCalledWith(1, 'm1'));
    });

    test('Échap ferme la fiche sans rien envoyer', async () => {
        const card = await openCard();
        await fireEvent.keyDown(card, { key: 'Escape' });
        await flush();
        expect(screen.queryByTestId('direction-result-card')).toBeNull();
        expect(DB.EnterResult).not.toHaveBeenCalled();
    });
});

describe('les joueurs', () => {
    test('une inscription envoie nom, club et classement ; la liste suit la vue rendue', async () => {
        await mount({ ...base, state: 'draft', players: [] });
        vi.mocked(DB.AddParticipant).mockResolvedValue(/** @type {any} */ ({ ...base, state: 'draft', eventCount: 2, players: [players[0]] }));
        vi.mocked(DB.Participants).mockResolvedValue(/** @type {any} */ ([{ id: 'pa', name: 'Alice', state: 'free' }]));
        const form = screen.getByTestId('direction-player-entry');
        const [name, club, rating] = [...form.querySelectorAll('input[type="text"]')];
        await fireEvent.input(name, { target: { value: ' Alice ' } });
        await fireEvent.input(club, { target: { value: 'Lyon' } });
        await fireEvent.input(rating, { target: { value: '1650' } });
        await fireEvent.submit(form);
        await flush();
        expect(DB.AddParticipant).toHaveBeenCalledWith(1, 'Alice', 'Lyon', 1650);
        expect(screen.getByTestId('direction-player-pa')).toBeTruthy();
    });

    test('un nom vide n’inscrit personne', async () => {
        await mount({ ...base, state: 'draft', players: [] });
        const form = screen.getByTestId('direction-player-entry');
        await fireEvent.input(must(form.querySelector('input')), { target: { value: '   ' } });
        await fireEvent.submit(form);
        await flush();
        expect(DB.AddParticipant).not.toHaveBeenCalled();
    });

    test('retirer se confirme par un bouton à son nom ; refusé, personne n’est retiré', async () => {
        vi.mocked(DB.Participants).mockResolvedValue(/** @type {any} */ ([{ id: 'pa', name: 'Alice', state: 'free' }]));
        await mount();
        await goTab('players');
        const row = screen.getByTestId('direction-player-pa');
        await fireEvent.click(must(row.querySelector('[data-testid="direction-player-withdraw-now"]')));
        await answerConfirm(false);
        expect(DB.WithdrawParticipant).not.toHaveBeenCalled();
        await fireEvent.click(must(row.querySelector('[data-testid="direction-player-withdraw-now"]')));
        await answerConfirm(true);
        await vi.waitFor(() => expect(DB.WithdrawParticipant).toHaveBeenCalledWith(1, 'pa', false));
    });

    test('réintégrer un joueur retiré appelle le service', async () => {
        vi.mocked(DB.Participants).mockResolvedValue(/** @type {any} */ ([{ id: 'pa', name: 'Alice', state: 'withdrawn' }]));
        await mount();
        await goTab('players');
        await fireEvent.click(screen.getByTestId('direction-player-reinstate'));
        await flush();
        expect(DB.ReinstateParticipant).toHaveBeenCalledWith(1, 'pa');
    });

    test('une erreur du service sur un joueur se dit dans la barre d’état', async () => {
        vi.mocked(DB.Participants).mockResolvedValue(/** @type {any} */ ([{ id: 'pa', name: 'Alice', state: 'withdrawn' }]));
        vi.mocked(DB.ReinstateParticipant).mockRejectedValue(new Error('no'));
        await mount();
        await goTab('players');
        await fireEvent.click(screen.getByTestId('direction-player-reinstate'));
        await flush();
        expect(status()).toMatchObject({ i18nKey: 'direction.players.error' });
    });
});

describe('classement et historique', () => {
    test('clore sans match en cours part tout de suite et le dit', async () => {
        vi.mocked(DB.Standings).mockResolvedValue(/** @type {any} */ ({ sections: [], finished: false }));
        vi.mocked(DB.CloseDirection).mockResolvedValue(/** @type {any} */ ({ ...base, state: 'finished', eventCount: 9 }));
        await mount();
        await goTab('standings');
        await fireEvent.click(screen.getByTestId('direction-standings-close'));
        await flush();
        expect(DB.CloseDirection).toHaveBeenCalledWith(1);
        expect(status()).toMatchObject({ i18nKey: 'direction.feedback.finished' });
    });

    test('clore avec des matchs en cours demande d’abord ; annuler ne clôt rien', async () => {
        vi.mocked(DB.Standings).mockResolvedValue(/** @type {any} */ ({ sections: [], finished: false }));
        await mount({ ...base, running: [{ matchId: 'm1' }] });
        await goTab('standings');
        await fireEvent.click(screen.getByTestId('direction-standings-close'));
        await fireEvent.click(screen.getByTestId('direction-standings-confirm-cancel'));
        await flush();
        expect(DB.CloseDirection).not.toHaveBeenCalled();
        await fireEvent.click(screen.getByTestId('direction-standings-close'));
        await fireEvent.click(screen.getByTestId('direction-standings-confirm-go'));
        await flush();
        expect(DB.CloseDirection).toHaveBeenCalledWith(1);
    });

    test('une remarque de l’historique part rognée', async () => {
        await mount();
        await goTab('history');
        const form = /** @type {HTMLFormElement} */ (must(screen.getByTestId('direction-pane-history').querySelector('form.note')));
        await fireEvent.input(must(form.querySelector('input')), { target: { value: '  pause café  ' } });
        await fireEvent.submit(form);
        await flush();
        expect(DB.AddDirectionNote).toHaveBeenCalledWith(1, 'pause café');
    });
});
