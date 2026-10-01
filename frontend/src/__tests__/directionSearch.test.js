// @ts-nocheck — entrées d'index partielles : le type généré par Wails est une classe.
/**
 * La recherche rapide de la Direction : la source `direction` de la palette (ce qu'elle liste,
 * comment elle classe, où mène un choix) et la touche `/` qui l'ouvre.
 */

import { describe, test, expect, afterEach, beforeEach } from 'vitest';
import { get } from 'svelte/store';

import { buildPaletteItems, rankPaletteItems, runPaletteItem, openDirectionSearch, toggleCommandPalette } from '../services/commandPalette.js';
import { directionSearchKey } from '../services/directionKeys.js';
import { openDirectionIdStore, directionJumpStore } from '../stores/directionStore.js';
import { activeTabStore, commandPaletteOpenStore, commandPaletteScopeStore } from '../stores/uiStore.js';
import { translate } from '../i18n';

import fr from '../i18n/locales/fr.json';
import en from '../i18n/locales/en.json';
import de from '../i18n/locales/de.json';
import el from '../i18n/locales/el.json';
import es from '../i18n/locales/es.json';
import fi from '../i18n/locales/fi.json';
import it from '../i18n/locales/it.json';
import ja from '../i18n/locales/ja.json';
import ru from '../i18n/locales/ru.json';

const INDEX = [
    { kind: 'epreuve', tournamentId: 1, epreuve: 'Principal', name: 'Principal' },
    { kind: 'epreuve', tournamentId: 2, epreuve: 'Consolante', name: 'Consolante' },
    { kind: 'player', tournamentId: 1, epreuve: 'Principal', name: 'Alice Martin', playerId: 'P1', club: 'Lyon', state: 'playing', table: 4, opponent: 'Bob Durand' },
    { kind: 'player', tournamentId: 2, epreuve: 'Consolante', name: 'Alain Petit', playerId: 'P9', state: 'free' },
    { kind: 'table', tournamentId: 1, epreuve: 'Principal', name: 'Alice Martin – Bob Durand', table: 4, state: 'running' },
    { kind: 'table', tournamentId: 0, table: 5, state: 'free' },
    { kind: 'table', tournamentId: 0, table: 14, state: 'free' },
    { kind: 'match', tournamentId: 1, epreuve: 'Principal', name: 'Alice Martin – Bob Durand', table: 4, matchId: 'M1', state: 'running' }
];

const items = buildPaletteItems({ translate, direction: INDEX });
const first = (query) => rankPaletteItems(items, query)[0]?.item;

beforeEach(() => {
    activeTabStore.set('tournaments');
    openDirectionIdStore.set(1);
});
afterEach(() => {
    openDirectionIdStore.set(null);
    directionJumpStore.set(null);
    commandPaletteOpenStore.set(false);
    commandPaletteScopeStore.set(null);
    document.body.innerHTML = '';
});

describe('source direction', () => {
    test('un joueur se trouve par son nom ou son club ; ceux à une table passent avant les libres', () => {
        expect(first('alice').id).toBe('direction:player:1:P1');
        expect(first('lyon').id).toBe('direction:player:1:P1');
        const ids = rankPaletteItems(items, 'al').map((r) => r.item.id);
        expect(ids.indexOf('direction:player:1:P1')).toBeLessThan(ids.indexOf('direction:player:2:P9'));
    });

    test('un numéro de table, tapé entier, met la table en tête (4, t4), pas la 14', () => {
        expect(first('4').id).toBe('direction:table:4');
        expect(first('t4').id).toBe('direction:table:4');
        expect(first('t5').id).toBe('direction:table:5');
    });

    test("une épreuve s'atteint par son nom", () => {
        expect(first('consol').id).toBe('direction:epreuve:2');
    });

    test('le détail d’un joueur dit son état, sa table, son adversaire et son épreuve', () => {
        const alice = items.find((i) => i.id === 'direction:player:1:P1');
        expect(alice.detail).toContain('Bob Durand');
        expect(alice.detail).toContain('Principal');
        expect(alice.detail).toMatch(/4/);
    });

    test('portée direction : rien d’autre que la salle', () => {
        const scoped = buildPaletteItems({ translate, direction: INDEX, scope: 'direction' });
        expect(scoped.every((i) => i.kind === 'direction')).toBe(true);
        expect(scoped).toHaveLength(INDEX.length);
        expect(buildPaletteItems({ translate, direction: INDEX }).some((i) => i.kind === 'command')).toBe(true);
    });

    test('sans Direction, la palette ne gagne aucune entrée', () => {
        expect(buildPaletteItems({ translate }).some((i) => i.kind === 'direction')).toBe(false);
    });
});

describe('choisir une entrée', () => {
    test('un joueur à une table mène à la fiche de cette table, dans son épreuve', async () => {
        activeTabStore.set('board');
        await runPaletteItem(items.find((i) => i.id === 'direction:player:1:P1'));
        expect(get(activeTabStore)).toBe('tournaments');
        expect(get(directionJumpStore)).toMatchObject({ kind: 'table', tournamentId: 1, table: 4, open: true });
    });

    test('un joueur libre mène à sa liste, filtrée sur son nom', async () => {
        await runPaletteItem(items.find((i) => i.id === 'direction:player:2:P9'));
        expect(get(directionJumpStore)).toMatchObject({ kind: 'player', tournamentId: 2, name: 'Alain Petit' });
    });

    test("une table libre se montre sans ouvrir de fiche ; une occupée l'ouvre", async () => {
        await runPaletteItem(items.find((i) => i.id === 'direction:table:5'));
        expect(get(directionJumpStore)).toMatchObject({ kind: 'table', table: 5, open: false });
        await runPaletteItem(items.find((i) => i.id === 'direction:table:4'));
        expect(get(directionJumpStore)).toMatchObject({ kind: 'table', table: 4, open: true });
    });

    test('une épreuve bascule l’onglet', async () => {
        await runPaletteItem(items.find((i) => i.id === 'direction:epreuve:2'));
        expect(get(directionJumpStore)).toMatchObject({ kind: 'epreuve', tournamentId: 2 });
    });

    test('la même demande répétée se rejoue', async () => {
        const it = items.find((i) => i.id === 'direction:table:4');
        await runPaletteItem(it);
        const a = get(directionJumpStore).seq;
        await runPaletteItem(it);
        expect(get(directionJumpStore).seq).toBeGreaterThan(a);
    });
});

describe('ouverture', () => {
    test('/ ouvre la palette sur la salle ; Ctrl+Maj+P la rouvre sur tout', () => {
        openDirectionSearch();
        expect(get(commandPaletteOpenStore)).toBe(true);
        expect(get(commandPaletteScopeStore)).toBe('direction');
        commandPaletteOpenStore.set(false);
        toggleCommandPalette();
        expect(get(commandPaletteScopeStore)).toBeNull();
    });

    test('sans Direction ouverte, / n’ouvre rien', () => {
        openDirectionIdStore.set(null);
        openDirectionSearch();
        expect(get(commandPaletteOpenStore)).toBe(false);
    });
});

describe('la touche /', () => {
    const slash = (init = {}, target = document.body) => {
        const ev = new KeyboardEvent('keydown', { key: '/', bubbles: true, ...init });
        Object.defineProperty(ev, 'target', { value: target });
        return ev;
    };

    test('sur la page Direction, nue ou avec Maj (AZERTY)', () => {
        expect(directionSearchKey(slash())).toBe(true);
        expect(directionSearchKey(slash({ shiftKey: true }))).toBe(true);
    });

    test('ni Ctrl, ni Alt, ni Méta ; ni une autre touche', () => {
        expect(directionSearchKey(slash({ ctrlKey: true }))).toBe(false);
        expect(directionSearchKey(slash({ altKey: true }))).toBe(false);
        expect(directionSearchKey(slash({ metaKey: true }))).toBe(false);
        expect(directionSearchKey(new KeyboardEvent('keydown', { key: 'a' }))).toBe(false);
    });

    test('jamais dans un champ de saisie : on y tape une barre oblique', () => {
        const input = document.createElement('input');
        document.body.append(input);
        input.focus();
        expect(directionSearchKey(slash({}, input))).toBe(false);
    });

    test('pas hors de la page Direction, ni sous une modale ou un menu', () => {
        activeTabStore.set('board');
        expect(directionSearchKey(slash())).toBe(false);
        activeTabStore.set('tournaments');
        const dialog = document.createElement('div');
        dialog.setAttribute('aria-modal', 'true');
        document.body.append(dialog);
        expect(directionSearchKey(slash())).toBe(false);
    });
});

describe('textes de la palette de la Direction, dans les neuf langues', () => {
    const LOCALES = { fr, en, de, el, es, fi, it, ja, ru };
    for (const [name, loc] of Object.entries(LOCALES)) {
        test(name, () => {
            for (const k of ['kindPlayer', 'kindTable', 'kindEpreuve', 'kindRunning', 'placeholderDirection', 'directionTable', 'directionVs']) {
                expect(loc.palette[k], `palette.${k}`).toBeTruthy();
            }
            expect(loc.palette.directionTable).toContain('{n}');
            expect(loc.palette.directionVs).toContain('{name}');
        });
    }
});
