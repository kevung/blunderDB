/**
 * Un geste du directeur produit un retour visible (barre d'état, aria-live) ; le forfait et le
 * retrait se confirment par un bouton à leur nom, non rouge ; la page murale se nomme.
 */
import { test, expect, vi, afterEach, beforeEach } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';
import { answerConfirm } from './confirmHelper.js';

vi.mock('../../wailsjs/go/database/Database.js', async (importOriginal) => {
    const orig = await importOriginal();
    return Object.fromEntries(Object.keys(/** @type {object} */ (orig)).map((k) => [k, vi.fn().mockResolvedValue(null)]));
});
vi.mock('../../wailsjs/go/gui/App.js', async (importOriginal) => ({ ...(await importOriginal()), OpenLocalPage: vi.fn().mockResolvedValue(undefined) }));
vi.mock('../../wailsjs/runtime/runtime.js', async (importOriginal) => ({ ...(await importOriginal()), BrowserOpenURL: vi.fn() }));

import DirectionView from '../components/direction/DirectionView.svelte';
import ResultCard from '../components/direction/ResultCard.svelte';
import PlayersView from '../components/direction/PlayersView.svelte';
import { directionStore, openDirectionIdStore } from '../stores/directionStore.js';
import { confirmModalStore, resolveConfirm } from '../services/confirmService.js';
import { statusBarTextStore } from '../stores/uiStore.js';
import { AddParticipant, WriteDirectionPage, CloseDirection, Standings } from '../../wailsjs/go/database/Database.js';
import { OpenLocalPage } from '../../wailsjs/go/gui/App.js';
import { t } from '../i18n';

const tr = (/** @type {string} */ k, /** @type {any} */ p) => get(t)(k, p);

const baseView = { tournamentId: 1, state: 'draft', config: { phases: [], tables: { count: 0 } }, proposals: [], running: [], warnings: [], players: [], outputDir: '/tmp/out' };

beforeEach(() => statusBarTextStore.set(''));
afterEach(() => {
    cleanup();
    directionStore.set(null);
    openDirectionIdStore.set(null);
    vi.clearAllMocks();
});

/** @param {any} view */
function mountView(view) {
    openDirectionIdStore.set(1);
    render(DirectionView);
    directionStore.set(view);
    return tick();
}

test('inscrire dit le nom et le nombre d’inscrits', async () => {
    await mountView(baseView);
    vi.mocked(AddParticipant).mockResolvedValue(/** @type {any} */ ({ ...baseView, players: [{ id: 'p1', name: 'Sophie Martin', state: 'free', wins: 0, losses: 0 }] }));
    await fireEvent.click(screen.getByTestId('direction-tab-players'));
    const form = screen.getByTestId('direction-player-entry');
    await fireEvent.input(/** @type {HTMLElement} */ (form.querySelector('input')), { target: { value: 'Sophie Martin' } });
    await fireEvent.submit(form);
    await vi.waitFor(() => expect(get(statusBarTextStore)).toEqual({ i18nKey: 'direction.feedback.registered', i18nParams: { name: 'Sophie Martin', count: 1 } }));
});

test('un retardataire arrivé en cours de tournoi a son propre retour', async () => {
    await mountView({ ...baseView, state: 'running' });
    vi.mocked(AddParticipant).mockResolvedValue(/** @type {any} */ ({ ...baseView, state: 'running', players: [{ id: 'p1', name: 'Luc', state: 'free', wins: 0, losses: 0 }] }));
    await fireEvent.click(screen.getByTestId('direction-tab-players'));
    const form = screen.getByTestId('direction-player-entry');
    await fireEvent.input(/** @type {HTMLElement} */ (form.querySelector('input')), { target: { value: 'Luc' } });
    await fireEvent.submit(form);
    await vi.waitFor(() => expect(get(statusBarTextStore)).toMatchObject({ i18nKey: 'direction.feedback.registeredLate' }));
});

test('l’en-tête nomme la page murale et l’écriture donne le chemin', async () => {
    await mountView(baseView);
    vi.mocked(WriteDirectionPage).mockResolvedValue('/tmp/out/index.html');
    const btn = screen.getByTestId('direction-open-page');
    expect(btn.textContent?.trim()).toBe(tr('direction.display.page'));
    await fireEvent.click(btn);
    await vi.waitFor(() => expect(get(statusBarTextStore)).toEqual({ i18nKey: 'direction.display.written', i18nParams: { path: '/tmp/out/index.html' } }));
    // Wails refuse le schéma file:// dans BrowserOpenURL : la page s'ouvre par le binding Go.
    expect(OpenLocalPage).toHaveBeenCalledWith('/tmp/out/index.html');
});

test('clore le tournoi donne un retour', async () => {
    vi.mocked(Standings).mockResolvedValue(/** @type {any} */ ({ rows: [], finished: false }));
    await mountView({ ...baseView, state: 'running' });
    vi.mocked(CloseDirection).mockResolvedValue(/** @type {any} */ ({ ...baseView, state: 'finished' }));
    await fireEvent.click(screen.getByTestId('direction-tab-standings'));
    await fireEvent.click(await screen.findByTestId('direction-standings-close'));
    await vi.waitFor(() => expect(get(statusBarTextStore)).toMatchObject({ i18nKey: 'direction.feedback.finished' }));
});

test('retirer se confirme par « Retirer », sans bouton rouge', async () => {
    render(PlayersView, { props: { rows: [{ id: 'p1', name: 'Alice', state: 'free', wins: 0, losses: 0 }], started: true } });
    await fireEvent.click(screen.getByTestId('direction-player-withdraw-now'));
    await vi.waitFor(() => expect(get(confirmModalStore)).not.toBeNull());
    expect(get(confirmModalStore)).toMatchObject({ confirmLabel: tr('direction.players.withdrawDo'), tone: 'primary' });
    resolveConfirm(false);
});

test('le forfait propose de retirer le perdant dans la même fiche', async () => {
    const onForfeit = vi.fn();
    const cell = { table: 3, length: 7, matchId: 'm1', a: 'pa', b: 'pb', aName: 'Alice', bName: 'Bruno' };
    render(ResultCard, { props: { cell, canWithdraw: true, onForfeit } });
    await fireEvent.click(screen.getByTestId('direction-result-more'));
    await fireEvent.click(screen.getByTestId('direction-result-forfeit-a'));
    await vi.waitFor(() => expect(get(confirmModalStore)).not.toBeNull());
    const labels = get(confirmModalStore).choices.map((/** @type {any} */ c) => c.label);
    expect(labels).toEqual([tr('direction.result.forfeitDo'), tr('direction.result.forfeitWithdraw', { loser: 'Alice' })]);
    resolveConfirm('forfeitWithdraw');
    await vi.waitFor(() => expect(onForfeit).toHaveBeenCalledWith('m1', 'pb', '', 'pa'));
});

test('le forfait seul se confirme par « Déclarer forfait », non rouge', async () => {
    const cell = { table: 3, length: 7, matchId: 'm1', a: 'pa', b: 'pb', aName: 'Alice', bName: 'Bruno' };
    render(ResultCard, { props: { cell, onForfeit: vi.fn() } });
    await fireEvent.click(screen.getByTestId('direction-result-more'));
    await fireEvent.click(screen.getByTestId('direction-result-forfeit-a'));
    await vi.waitFor(() => expect(get(confirmModalStore)).not.toBeNull());
    expect(get(confirmModalStore)).toMatchObject({ confirmLabel: tr('direction.result.forfeitDo'), tone: 'primary' });
    await answerConfirm(false);
});

test('une page écrite mais non ouverte ne se dit pas « écrite »', async () => {
    await mountView(baseView);
    vi.mocked(WriteDirectionPage).mockResolvedValue('/tmp/out/index.html');
    vi.mocked(OpenLocalPage).mockRejectedValueOnce(new Error('xdg-open introuvable'));
    await fireEvent.click(screen.getByTestId('direction-open-page'));
    await vi.waitFor(() => expect(get(statusBarTextStore)).toEqual({ i18nKey: 'direction.display.openFailed', i18nParams: { path: '/tmp/out/index.html' } }));
});
