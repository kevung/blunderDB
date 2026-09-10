/**
 * transcription-budgets.spec.js — les budgets de gestes d'ux.md §§ 4.1–4.3,
 * comptés sur l'application réelle (T3.5).
 *
 * ## Ce que ces specs ajoutent à ce qui existe déjà
 *
 * La machine à touches est pure et elle est déjà tenue ligne à ligne, au niveau
 * unitaire : `transcriptionKeys.turn.test.js` compte les touches d'un tour de
 * pions, `.cube.test.js` celles du videau (§4.2), `.correction.test.js` celles
 * des sept lignes de §4.3, `.mouse.test.js` le clic du triangle,
 * `transcriptionFilter.test.js` le rang douze filtré. Rien de tout cela n'est
 * refait ici.
 *
 * Ce que ces tests-là ne peuvent pas voir, et qui est le sujet de ce fichier :
 * entre la touche et le geste il y a une application — un onglet, un panneau
 * qui doit avoir le focus sans qu'on le lui donne, une garde de clavier
 * (`panelKeyGuard`), un dispatcher global qui pourrait prendre la touche
 * d'abord, une liste de candidats qui arrive après un aller-retour. Un budget
 * de trois touches se perd très bien en un clic d'armement que la machine pure
 * ne verra jamais. C'est ce clic-là que ces specs comptent, sur le vrai
 * navigateur, avec `countGestures`.
 *
 * ## Ce qui est faux ici, et assumé
 *
 * Le moteur Go, remplacé par le document figé de `helpers/transcriptionDraft.js`
 * (voir son en-tête : deux faits seulement y sont dérivés, le Cursor et la
 * réponse attendue après un double). Ce que la suite de gestes produit comme
 * document est tenu en Go, Action par Action ; ce qui est tenu ici est
 * COMBIEN de gestes l'utilisateur fait pour l'émettre, et LAQUELLE part.
 *
 * ## Les lignes des tableaux qui ne sont pas ici
 *
 * - §4.1 « coup loin dans la liste, filtré par clic sur le point de départ » et
 *   « coup joué au plateau » : les deux passent par un clic sur un point du
 *   damier, qui est dessiné par two.js et n'a pas de cible DOM — il faudrait
 *   viser une coordonnée dans un canvas, ce qui donne une spec instable pour
 *   une mesure que `transcriptionFilter.test.js` fait déjà exactement (elle
 *   compte les clics et les touches, et vérifie le budget de trois secondes).
 *   Mieux vaut ne pas la livrer instable.
 * - §4.2 « videau à la souris » : la cible souris du videau est T2.5, en cours
 *   d'écriture au moment où ce fichier est posé. La ligne sera à ajouter ici
 *   quand elle existera ; le reste du tableau est couvert.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { countGestures } from './helpers/gestureCount.js';
import { installTranscriptionEngine, sentKinds, sentGestures, resetGestures, ACTION_COUNT } from './helpers/transcriptionDraft.js';

/** Ouvre l'onglet et le brouillon : le décor, jamais compté. */
async function openDraft(page, opts = {}) {
    await installWailsMock(page, openLibraryMock());
    await installTranscriptionEngine(page, opts);
    await page.goto('/');
    await page.locator('[data-testid="tab-transcription"]').click();
    await page.locator('#transcriptionPanel tbody tr').first().click();
    await expect(page.locator('#transcriptionPanel .draft-bar')).toBeVisible();
    await resetGestures(page);
}

const panel = '#transcriptionPanel';
const candidates = `${panel} table.checker-table tbody tr`;

/** Les dix-sept candidats sont là : le moteur a répondu au jet. */
async function candidatesListed(page) {
    await expect(page.locator(candidates)).toHaveCount(17);
}

/**
 * Le rang (1 = le meilleur) de la ligne sélectionnée dans la liste montrée.
 *
 * Le rang et non la notation : la colonne affiche `moveLabel`, qui réordonne
 * les pas d'un coup pour les lire du point le plus haut au plus bas (« 13/10
 * 24/23 » s'écrit « 24/23 13/10 »). Comparer des libellés reviendrait à
 * recopier cette règle-là dans la spec ; le budget, lui, parle de rangs.
 */
async function selectedRank(page) {
    const rows = page.locator(candidates);
    const classes = await rows.evaluateAll((els) => els.map((el) => el.className));
    return classes.findIndex((c) => c.split(/\s+/).includes('selected')) + 1;
}

test.describe('ux.md §4.1 — un tour de pions', () => {
    test.beforeEach(async ({ page }) => {
        await openDraft(page);
    });

    // « meilleur coup joué | 3 1 puis le jet suivant | 2 K = 0,56 s ». Deux
    // depuis ADR-0048 décision 1 : le Cursor est en bout de document, donc le
    // chiffre y VALIDE avant d'ouvrir le jet suivant, et la validation de ce
    // tour-ci est portée par la première touche du tour d'après.
    test('le meilleur coup joué coûte deux touches', async ({ page }) => {
        const count = await countGestures(page, async (g) => {
            await g.press('Digit3');
            await g.press('Digit1');
            await candidatesListed(page);
        });
        expect(count.keys).toBe(2);
        expect(count.clicks).toBe(0);
        await expect.poll(() => sentKinds(page)).toEqual(['enter_die', 'enter_die', 'select_candidate']);

        // Et la troisième frappe appartient au tour SUIVANT : elle valide
        // celui-ci en passant.
        const next = await countGestures(page, (g) => g.press('Digit6'));
        expect(next.keys).toBe(1);
        await expect.poll(() => sentKinds(page)).toEqual(['enter_die', 'enter_die', 'select_candidate', 'validate', 'enter_die']);
    });

    // La sortie que la règle laisse ouverte : le dernier coup d'une partie n'a
    // pas de tour suivant pour porter sa validation.
    test('le dernier coup d’une partie coûte trois touches, Entrée comprise', async ({ page }) => {
        const count = await countGestures(page, async (g) => {
            await g.press('Digit3');
            await g.press('Digit1');
            await candidatesListed(page);
            await g.press('Enter');
        });
        expect(count.keys).toBe(3);
        await expect.poll(() => sentKinds(page)).toEqual(['enter_die', 'enter_die', 'select_candidate', 'validate']);
    });

    // « n-ième coup, n ≤ 5 | 3 1 j×(n−1) puis le jet suivant | (n+1) K ». La
    // validation est portée par le premier chiffre du tour suivant, elle
    // n'appartient pas à ce tour-ci : le budget est donc n+1 pour DÉSIGNER, et
    // le chiffre suivant est déjà celui du tour d'après.
    test('le n-ième coup coûte n+1 touches, validation portée par le tour suivant', async ({ page }) => {
        const n = 3;
        const count = await countGestures(page, async (g) => {
            await g.press('Digit3');
            await g.press('Digit1');
            await candidatesListed(page);
            for (let i = 1; i < n; i += 1) await g.press('KeyJ');
        });
        expect(count.total).toBe(n + 1);
        expect(await selectedRank(page)).toBe(n);

        // Le chiffre suivant valide ce qui est choisi, et ouvre le jet d'après.
        const next = await countGestures(page, (g) => g.press('Digit6'));
        expect(next.keys).toBe(1);
        await expect.poll(() => sentKinds(page)).toEqual(['enter_die', 'enter_die', 'select_candidate', 'select_candidate', 'select_candidate', 'validate', 'enter_die']);
    });

    // « coup loin dans la liste (rang 12) | 3 1 j×11 | 13 K = 3,6 s ». C'est la
    // ligne qui a motivé le filtre du lot 2 : elle est ici pour que le chiffre
    // reste vrai, pas parce qu'il est bon.
    test('le rang douze coûte treize touches au clavier seul', async ({ page }) => {
        const count = await countGestures(page, async (g) => {
            await g.press('Digit3');
            await g.press('Digit1');
            await candidatesListed(page);
            for (let i = 0; i < 11; i += 1) await g.press('KeyJ');
        });
        expect(count.total).toBe(13);
        expect(count.clicks).toBe(0);
        expect(await selectedRank(page)).toBe(12);
    });

    // « dés au clavier | 2 K | référence ».
    test('les dés au clavier coûtent deux touches', async ({ page }) => {
        const count = await countGestures(page, async (g) => {
            await g.press('Digit3');
            await g.press('Digit1');
            await candidatesListed(page);
        });
        expect(count.keys).toBe(2);
        await expect.poll(() => sentKinds(page)).toEqual(['enter_die', 'enter_die', 'select_candidate']);
    });

    // « dés à la souris, triangle 21 | H P(28 px) B B | 1,21 s ». UN clic, et
    // les deux mêmes gestes que les deux touches : c'est ce que compte T3.5.
    test('les dés au triangle coûtent un clic, et rendent les deux mêmes gestes', async ({ page }) => {
        const count = await countGestures(page, async (g) => {
            await g.click(page.locator(`${panel} .dice-triangle button`).filter({ hasText: /^31$/ }));
            await candidatesListed(page);
        });
        expect(count.clicks).toBe(1);
        expect(count.keys).toBe(0);
        const gestures = await sentGestures(page);
        expect(gestures.map((x) => x.Kind)).toEqual(['enter_die', 'enter_die', 'select_candidate']);
        expect(gestures.slice(0, 2).map((x) => x.Die)).toEqual([3, 1]);
    });
});

test.describe('ux.md §4.2 — le videau', () => {
    // « double, prise | d t | 2 K ». Le `d` vaut d'un état quelconque : le
    // moteur valide le candidat en attente avant de doubler, ce qui est ce qui
    // garde le geste à deux touches.
    test('double puis prise coûtent deux touches', async ({ page }) => {
        await openDraft(page);
        const count = await countGestures(page, async (g) => {
            await g.press('KeyD');
            await expect.poll(() => sentKinds(page)).toEqual(['double']);
            await g.press('KeyT');
        });
        expect(count.keys).toBe(2);
        await expect.poll(() => sentKinds(page)).toEqual(['double', 'take']);
    });

    // « double, passe (partie finie, ouverture suivante) | d p | 0,56 s ».
    test('double puis passe coûtent deux touches', async ({ page }) => {
        await openDraft(page);
        const count = await countGestures(page, async (g) => {
            await g.press('KeyD');
            await expect.poll(() => sentKinds(page)).toEqual(['double']);
            await g.press('KeyP');
        });
        expect(count.keys).toBe(2);
        await expect.poll(() => sentKinds(page)).toEqual(['double', 'pass']);
    });

    // « redouble | d t | 0,56 s ». Le redouble ne se distingue du double par
    // AUCUN geste — c'est le document qui sait que le videau était déjà pris,
    // et le budget est le même parce que la frappe est la même. La spec le
    // vérifie depuis un videau déjà possédé, c'est-à-dire après une prise.
    test('le redouble coûte les deux mêmes touches', async ({ page }) => {
        await openDraft(page);
        await page.keyboard.press('KeyD');
        await page.keyboard.press('KeyT');
        await resetGestures(page);
        const count = await countGestures(page, async (g) => {
            await g.press('KeyD');
            await expect.poll(() => sentKinds(page)).toEqual(['double']);
            await g.press('KeyT');
        });
        expect(count.keys).toBe(2);
        await expect.poll(() => sentKinds(page)).toEqual(['double', 'take']);
    });

    // « résignation gammon | r 2 | 2 K ».
    test('la résignation au gammon coûte deux touches', async ({ page }) => {
        await openDraft(page);
        const count = await countGestures(page, async (g) => {
            await g.press('KeyR');
            await g.press('Digit2');
        });
        expect(count.keys).toBe(2);
        const gestures = await sentGestures(page);
        expect(gestures.map((x) => x.Kind)).toEqual(['resign']);
        expect(gestures[0].Level).toBe(2);
    });
});

test.describe('ux.md §4.3 — la correction', () => {
    test.beforeEach(async ({ page }) => {
        await openDraft(page);
    });

    /** Tape un jet et attend sa liste : l'état « jet corrigeable ». */
    async function typeRoll(page) {
        await page.keyboard.press('Digit3');
        await page.keyboard.press('Digit1');
        await candidatesListed(page);
        await resetGestures(page);
    }

    // « dé mal lu, vu aussitôt (jet corrigeable) | 4 1 | 2 K ». Rien n'est
    // validé au passage : c'est toute la raison d'être de l'état.
    // ADR-0048 décision 1 : en bout de document, reprendre un jet qu'on vient de
    // taper commence par l'effacer, puisque le chiffre y valide. C'est la seule
    // ligne de §4.3 que la décision coûte — sur une Action relue elle reste à
    // deux touches — et elle se paie ~12 fois par match contre ~150 tours gagnés.
    test('un dé mal lu se corrige en trois touches, sans rien valider', async ({ page }) => {
        await typeRoll(page);
        const count = await countGestures(page, async (g) => {
            await g.press('Backspace');
            await g.press('Digit4');
            await g.press('Digit1');
            await candidatesListed(page);
        });
        expect(count.keys).toBe(3);
        const kinds = await sentKinds(page);
        expect(kinds).toEqual(['clear_dice', 'enter_die', 'enter_die', 'select_candidate']);
        expect(kinds).not.toContain('validate');
    });

    // « candidat voisin, vu aussitôt | j | 0,28 s ».
    test('le candidat voisin coûte une touche', async ({ page }) => {
        await typeRoll(page);
        const count = await countGestures(page, (g) => g.press('KeyJ'));
        expect(count.keys).toBe(1);
        await expect.poll(() => sentKinds(page)).toEqual(['select_candidate']);
        expect(await selectedRank(page)).toBe(2);
    });

    // « erreur vue un tour plus tard, dés déjà tapés | Retour, h, j, l | 4 K ».
    test('une erreur vue un tour plus tard coûte quatre touches', async ({ page }) => {
        await typeRoll(page);
        const count = await countGestures(page, async (g) => {
            await g.press('Backspace');
            await g.press('KeyH');
            await candidatesListed(page);
            await g.press('KeyJ');
            await g.press('KeyL');
        });
        expect(count.keys).toBe(4);
        await expect.poll(() => sentKinds(page)).toEqual(['clear_dice', 'cursor_back', 'select_candidate', 'cursor_forward']);
    });

    // « erreur vue k tours plus tard | Retour, h×k, j/k×m, l×k | (1 + 2k + m) K ».
    test('une erreur vue k tours plus tard coûte 1 + 2k + m touches', async ({ page }) => {
        const k = 5;
        const m = 1;
        await typeRoll(page);
        const count = await countGestures(page, async (g) => {
            await g.press('Backspace');
            for (let i = 0; i < k; i += 1) await g.press('KeyH');
            await candidatesListed(page);
            for (let i = 0; i < m; i += 1) await g.press('KeyJ');
            for (let i = 0; i < k; i += 1) await g.press('KeyL');
        });
        expect(count.keys).toBe(1 + 2 * k + m);
        const kinds = await sentKinds(page);
        expect(kinds.filter((x) => x === 'cursor_back')).toHaveLength(k);
        expect(kinds.filter((x) => x === 'cursor_forward')).toHaveLength(k);
        expect(kinds.filter((x) => x === 'select_candidate')).toHaveLength(m);
    });

    // « coup oublié | h×k, i, 3 1 (j…), Entrée, l×k | (2k + 4 + m) K ».
    //
    // MESURÉ le 2026-09-07, et le document portait 2k + 3 : une insertion n'est
    // PAS validée par le déplacement du Cursor. `commitCorrection`
    // (transcript/apply.go) ne commet qu'une Entry de mode `EntryReplace` — une
    // correction en place, pas une insertion — et vérification faite sur le
    // moteur lui-même, `cursor_forward` laisse le document à sept Actions là où
    // `validate` le porte à huit. La touche qui manquait est Entrée ; ux.md
    // §4.3 la porte depuis.
    test('un coup oublié coûte 2k + 4 touches, Entrée comprise', async ({ page }) => {
        const k = 3;
        const count = await countGestures(page, async (g) => {
            for (let i = 0; i < k; i += 1) await g.press('KeyH');
            await g.press('KeyI');
            await g.press('Digit3');
            await g.press('Digit1');
            await candidatesListed(page);
            await g.press('Enter');
            for (let i = 0; i < k; i += 1) await g.press('KeyL');
        });
        // 2k + 4 : les k retours, `i`, les deux dés, Entrée, les k avances.
        expect(count.keys).toBe(2 * k + 4);
        const kinds = await sentKinds(page);
        expect(kinds.filter((x) => x === 'insert_before')).toHaveLength(1);
        expect(kinds.filter((x) => x === 'validate')).toHaveLength(1);
    });

    // « coup en double | h×k, x, l×k | (2k + 1) K ».
    test('un coup en double se supprime en 2k + 1 touches', async ({ page }) => {
        const k = 4;
        const count = await countGestures(page, async (g) => {
            for (let i = 0; i < k; i += 1) await g.press('KeyH');
            await g.press('KeyX');
            for (let i = 0; i < k; i += 1) await g.press('KeyL');
        });
        expect(count.keys).toBe(2 * k + 1);
        await expect.poll(() => sentKinds(page)).toEqual([...Array(k).fill('cursor_back'), 'delete', ...Array(k).fill('cursor_forward')]);
    });

    // « camp faux | h×k, s, l×k | (2k + 1) K ».
    test('un camp faux se change en 2k + 1 touches', async ({ page }) => {
        const k = 2;
        const count = await countGestures(page, async (g) => {
            for (let i = 0; i < k; i += 1) await g.press('KeyH');
            await g.press('KeyS');
            for (let i = 0; i < k; i += 1) await g.press('KeyL');
        });
        expect(count.keys).toBe(2 * k + 1);
        await expect.poll(() => sentKinds(page)).toEqual([...Array(k).fill('cursor_back'), 'flip_side', ...Array(k).fill('cursor_forward')]);
    });

    // « À la souris, un clic sur la cellule du Transcript remplace h×k
    // (P B B = 1,3 s) » — la dernière phrase de §4.3, et la seule ligne du
    // tableau qui se mesure en clics.
    test('un clic sur la cellule du Transcript remplace les k touches', async ({ page }) => {
        const target = 4;
        const count = await countGestures(page, async (g) => {
            await g.click(page.locator(`${panel} .transcript-col button.cell`).nth(target));
        });
        expect(count.clicks).toBe(1);
        expect(count.keys).toBe(0);
        // Un geste de l'utilisateur, autant de pas de Cursor que la distance :
        // c'est le moteur qui les compte, pas la main.
        await expect.poll(() => sentKinds(page)).toEqual(Array(ACTION_COUNT - target).fill('cursor_back'));
    });
});
