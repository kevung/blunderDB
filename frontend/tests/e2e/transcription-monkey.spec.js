/**
 * Test « singe » du panneau de transcription, sur l'application réelle : des
 * séquences aléatoires reproductibles de frappes, de clics (plateau, cellules
 * du Transcript, boutons de la barre, liste des brouillons), de changements de
 * métadonnées et d'onglet, contre un moteur à état qui répond lentement et
 * dans le désordre (`helpers/transcriptionMonkey.js`).
 *
 * Le niveau e2e est le seul qui voit à la fois le dispatcher global (une
 * touche qui fuit), le démontage au changement d'onglet et le plateau.
 *
 * Invariants, vérifiés au repos (plus aucun appel en vol) :
 *   1. aucune exception non rattrapée ni `console.error` ;
 *   2. le panneau reste interactif : en fin de document, un chiffre est pris
 *      comme un dé ;
 *   3. aucune touche tapée dans le panneau ne change l'onglet ni la position
 *      parcourue, ni n'appelle le backend hors transcription ;
 *   4. le Cursor encadré est celui du document, et le document affiché est
 *      l'état courant du brouillon affiché, celui que l'utilisateur a ouvert ;
 *   5. aucun geste n'est reçu pour un brouillon autre que celui affiché.
 *
 * Rejouer une graine : `MONKEY_SEED=1234 npx playwright test transcription-monkey`
 * (plusieurs : `MONKEY_SEED=5,6,7`).
 * Allonger : `MONKEY_STEPS=400`.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installMonkeyEngine } from './helpers/transcriptionMonkey.js';

const panel = '#transcriptionPanel';

/** Méthodes du backend qu'un geste de transcription a le droit d'appeler. */
const TRANSCRIPTION_CALLS = new Set([
    'ListTranscriptions',
    'OpenTranscription',
    'CreateTranscription',
    'FinishTranscription',
    'AbandonTranscription',
    'MatchTranscriptionLosses',
    'ApplyTranscriptionGesture',
    'TranscriptionMAT',
    'PendingTranscriptionAnalysis',
    'SuggestTranscriptionMatFilename',
    'ExportTranscriptionMAT',
    'OpenExportMatDialog',
    'StartGammonNetMatchBatch',
    'GetGammonNetAnalysisPly',
    'GetGammonNetPruneK',
    'LegalMoves',
    'EvaluatePositionImmediate',
    'SaveSessionState',
    // Géométrie de la fenêtre, notée à son propre rythme, pas par une touche.
    'WindowGetSize',
    'SaveWindowDimensions'
]);

/** mulberry32 : la séquence d'une graine se rejoue à l'identique. */
function prng(seed) {
    let s = seed >>> 0;
    return () => {
        s = (s + 0x6d2b79f5) >>> 0;
        let t = s;
        t = Math.imul(t ^ (t >>> 15), t | 1);
        t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
        return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
}

const KEYS = [
    ...['1', '2', '3', '4', '5', '6'].flatMap((k) => [k, k, k]),
    '8',
    '/',
    ' ',
    'Delete',
    'Backspace',
    'ArrowLeft',
    'ArrowRight',
    'ArrowUp',
    'ArrowDown',
    'Home',
    'End',
    'Control+z',
    'Control+y',
    'Control+Shift+z',
    'Control+Enter',
    'Escape',
    'Tab',
    'Enter',
    'Enter',
    'd',
    't',
    'p',
    'r',
    'x',
    'i',
    'a',
    's',
    'j',
    'k',
    'h',
    'l'
];

/**
 * Touches que le panneau laisse passer à dessein : combos Ctrl, Tab, Espace
 * (la ligne de commande), et `p`, compte de pips global dès qu'un dé est tapé.
 */
const GLOBAL_KEYS = new Set(['Tab', ' ', 'p']);

/** Une touche que le panneau doit garder pour lui quand il a le focus. */
function panelOwned(key) {
    return !key.startsWith('Control+') && !GLOBAL_KEYS.has(key);
}

/**
 * Ouvre l'application, l'onglet Transcription, et branche l'instrumentation
 * qui lit le store affiché.
 */
async function boot(page, seed, errors) {
    page.on('pageerror', (err) => errors.push(`pageerror: ${err.message}`));
    page.on('console', (msg) => {
        if (msg.type() === 'error') errors.push(`console.error: ${msg.text()}`);
    });
    // Presse-papiers accordé : son refus est gardé par un test à part.
    await page.context().grantPermissions(['clipboard-read', 'clipboard-write']);
    // Tab puis Entrée peut ouvrir un autre onglet : ses lectures ne doivent pas rendre null.
    await installWailsMock(page, openLibraryMock({ database: { LoadMetadata: { user: '', description: '', dateOfCreation: '', database_version: '2.15.0' } } }));
    await installMonkeyEngine(page, { seed });
    await page.goto('/');
    await page.locator('[data-testid="tab-transcription"]').click();
    await expect(page.locator(`${panel} tbody tr`).first()).toBeVisible();
    await page.evaluate(async () => {
        const { transcriptionStore } = await import('/src/stores/transcriptionStore.js');
        transcriptionStore.subscribe((v) => (window.__monkeyDisplayed = v?.id ?? null));
    });
    // Le démarrage lit des réglages que le mock générique rend nuls : ce
    // bruit précède le singe et ne dit rien du panneau.
    errors.length = 0;
}

/** Attend le repos : plus aucun appel du moteur en vol, deux lectures d'affilée. */
async function settle(page) {
    await page.waitForFunction(
        () => {
            const w = /** @type {any} */ (window);
            // Une touche qui fuit agit hors du moteur (parcours, chargements) :
            // le repos exige aussi un journal d'appels immobile.
            const calls = w.__wailsCalls || [];
            const last = calls[calls.length - 1];
            if (w.__monkeyInflight !== 0 || last !== w.__monkeyLastCall) {
                w.__monkeyLastCall = last;
                w.__monkeyQuiet = 0;
                return false;
            }
            w.__monkeyQuiet = (w.__monkeyQuiet ?? 0) + 1;
            return w.__monkeyQuiet >= 6;
        },
        null,
        { polling: 40, timeout: 10000 }
    );
}

/** Ce que l'invariant 3 compare avant et après une touche. */
async function worldOf(page) {
    return page.evaluate(async () => {
        const get = (store) => {
            let v;
            store.subscribe((x) => (v = x))();
            return v;
        };
        const ui = await import('/src/stores/uiStore.js');
        const { transcriptionStore } = await import('/src/stores/transcriptionStore.js');
        const active = document.activeElement;
        const host = document.querySelector('#transcriptionPanel');
        return {
            tab: get(ui.activeTabStore),
            index: get(ui.currentPositionIndexStore),
            draft: get(transcriptionStore)?.id ?? null,
            focusInPanel: !!host && !!active && host.contains(active) && !active.matches('input, textarea, select, [contenteditable="true"]'),
            modal: !!document.querySelector('.modal-overlay'),
            // Le journal plafonne à 500 entrées : vidé ici, il se lit depuis 0.
            calls: ((window.__wailsCalls || []).length = 0)
        };
    });
}

/** Les méthodes appelées depuis la marque `from` du journal. */
async function callsSince(page, from) {
    return page.evaluate((n) => (window.__wailsCalls || []).slice(n).map((c) => c.method), from);
}

/** Invariants 4 et 5, au repos. Rend la liste des manquements. */
async function checkDocument(page) {
    return page.evaluate(async () => {
        const get = (store) => {
            let v;
            store.subscribe((x) => (v = x))();
            return v;
        };
        const { transcriptionStore } = await import('/src/stores/transcriptionStore.js');
        const ui = await import('/src/stores/uiStore.js');
        const out = [];
        const w = /** @type {any} */ (window);
        for (const v of w.__monkeyViolations.splice(0)) out.push(`geste ${v.gesture} envoyé au brouillon ${v.id} pendant que ${v.shown} est affiché`);
        const st = get(transcriptionStore);
        if (!st) return out;
        const truth = w.__monkeyState(st.id);
        if (!truth) {
            out.push(`le brouillon affiché ${st.id} est fermé côté moteur`);
            return out;
        }
        if (w.__monkeyIntent != null && w.__monkeyIntent !== st.id) out.push(`brouillon affiché ${st.id}, demandé ${w.__monkeyIntent}`);
        const ann = st.annotated;
        if (ann.cursor !== truth.annotated.cursor || ann.actions.length !== truth.annotated.actions.length) {
            out.push(`document périmé : affiché cursor=${ann.cursor}/${ann.actions.length}, moteur cursor=${truth.annotated.cursor}/${truth.annotated.actions.length}`);
        }
        if (get(ui.activeTabStore) === 'transcription') {
            const framed = [...document.querySelectorAll('#transcriptionPanel .cell.cursor:not([data-pending])')].map((el) => Number(el.getAttribute('data-index')));
            if (framed.length > 1 || framed.some((i) => i !== ann.cursor)) out.push(`Cursor encadré ${JSON.stringify(framed)}, document ${ann.cursor}`);
        }
        return out;
    });
}

/**
 * Un pas du singe. Rend sa description (pour la repro) ; un pas impossible
 * (cible absente) est un pas nul, jamais une erreur.
 */
async function step(page, rand) {
    const pick = (arr) => arr[Math.floor(rand() * arr.length)];
    const r = rand();
    const click = async (selector, label, opts = {}) => {
        const loc = page.locator(selector);
        const n = await loc.count();
        if (!n) return `${label}: absent`;
        const i = Math.floor(rand() * n);
        await loc
            .nth(i)
            .click({ timeout: 500, ...opts })
            .catch(() => {});
        return `${label}[${i}]`;
    };

    if (r < 0.55) {
        const key = pick(KEYS);
        const before = await worldOf(page);
        await page.keyboard.press(key);
        return { desc: `key ${key}`, key, before };
    }
    if (r < 0.63) return { desc: await click(`${panel} .cell[data-index], ${panel} .cell.hole`, 'cellule') };
    if (r < 0.66) return { desc: await click(`${panel} .cell[data-index]`, 'dblclic cellule', { clickCount: 2 }) };
    if (r < 0.72) {
        const box = await page
            .locator('#backgammon-board')
            .boundingBox()
            .catch(() => null);
        if (!box) return { desc: 'plateau: absent' };
        const x = Math.round(rand() * box.width);
        const y = Math.round(rand() * box.height);
        await page.mouse.click(box.x + x, box.y + y);
        return { desc: `plateau(${x},${y})` };
    }
    if (r < 0.8) return { desc: await click(`${panel} .draft-bar button`, 'barre') };
    if (r < 0.84) return { desc: await click(`${panel} tbody tr`, 'ligne brouillon') };
    if (r < 0.87) return { desc: await click('.modal-overlay button', 'modale') };
    if (r < 0.91) {
        const inputs = page.locator('[data-testid="transcription-metadata"] input');
        const n = await inputs.count();
        if (!n) return { desc: 'méta: fermées' };
        const i = Math.floor(rand() * n);
        const input = inputs.nth(i);
        const type = await input.getAttribute('type').catch(() => null);
        if (type === 'checkbox') await input.click({ timeout: 500 }).catch(() => {});
        else if (type === 'date') await input.fill('2026-09-30', { timeout: 500 }).catch(() => {});
        else {
            await input.fill(pick(['Open', '5', '0', '', 'xx', '13']), { timeout: 500 }).catch(() => {});
            await input.press(pick(['Enter', 'Tab']), { timeout: 500 }).catch(() => {});
        }
        return { desc: `méta[${i}]` };
    }
    if (r < 0.97) {
        const tab = pick(['matches', 'transcription', 'transcription']);
        await page
            .locator(`[data-testid="tab-${tab}"]`)
            .click({ timeout: 500 })
            .catch(() => {});
        return { desc: `onglet ${tab}` };
    }
    return { desc: await click(`${panel} .candidates tbody tr, ${panel} .plain-candidate, ${panel} .die`, 'candidat/dé') };
}

/** Invariant 3 : la touche, tapée dans le panneau, est restée dans le panneau. */
async function checkLeak(page, s) {
    if (!s.key || !panelOwned(s.key)) return [];
    const b = s.before;
    if (b.tab !== 'transcription' || b.draft == null || !b.focusInPanel || b.modal) return [];
    const after = await worldOf(page);
    const out = [];
    if (after.tab !== b.tab) out.push(`touche ${s.key} : onglet ${b.tab} → ${after.tab}`);
    if (after.index !== b.index) out.push(`touche ${s.key} : position parcourue ${b.index} → ${after.index}`);
    const foreign = (await callsSince(page, b.calls)).filter((m) => !TRANSCRIPTION_CALLS.has(m));
    if (foreign.length) out.push(`touche ${s.key} : appels hors transcription ${[...new Set(foreign)].join(', ')}`);
    return out;
}

/** Invariant 2 : de retour en fin de document, un chiffre est pris comme un dé. */
async function checkInteractive(page) {
    for (let i = 0; i < 3 && (await page.locator('.modal-overlay').count()); i += 1) await page.keyboard.press('Escape');
    await page.locator('[data-testid="tab-transcription"]').click();
    await settle(page);
    if (!(await page.locator(`${panel} .draft-bar`).count())) {
        await page.locator(`${panel} tbody tr`).first().click();
        await expect(page.locator(`${panel} .draft-bar`)).toBeVisible();
    }
    await settle(page);
    await page.locator(`${panel} .save-state`).click();
    await page.keyboard.press('Escape');
    const total = await page.evaluate(async () => {
        const get = (store) => {
            let v;
            store.subscribe((x) => (v = x))();
            return v;
        };
        const { transcriptionStore } = await import('/src/stores/transcriptionStore.js');
        return get(transcriptionStore)?.annotated?.actions?.length ?? 0;
    });
    for (let i = 0; i <= total; i += 1) await page.keyboard.press('ArrowRight');
    await settle(page);
    const expectsAnswer = await page.evaluate(async () => {
        const get = (store) => {
            let v;
            store.subscribe((x) => (v = x))();
            return v;
        };
        const { transcriptionStore } = await import('/src/stores/transcriptionStore.js');
        return get(transcriptionStore)?.annotated?.next?.expects === 'take';
    });
    if (expectsAnswer) {
        await page.keyboard.press('t');
        await settle(page);
    }
    const sent = await page.evaluate(() => window.__monkeyGestures.length);
    await page.keyboard.press('3');
    await settle(page);
    const gestures = await page.evaluate((n) => window.__monkeyGestures.slice(n), sent);
    const filled = await page.locator(`${panel} [data-testid="transcription-dice"] .die.filled`).count();
    const took = gestures.some((g) => g.kind === 'enter_die') || filled > 0;
    return took ? [] : [`fin de document : la frappe « 3 » n'est pas prise (gestes ${JSON.stringify(gestures)}, dés remplis ${filled})`];
}

/**
 * Joue une séquence et rend le premier manquement, avec la trace qui y mène.
 *
 * @param {import('@playwright/test').Page} page
 * @param {number} seed
 * @param {number} steps
 */
async function runMonkey(page, seed, steps) {
    const errors = [];
    await boot(page, seed, errors);
    const rand = prng(seed);
    const trace = [];
    const fail = (why) => `graine ${seed}, pas ${trace.length} : ${why}\n  trace : ${trace.join(' | ')}`;
    for (let i = 0; i < steps; i += 1) {
        const s = await step(page, rand);
        trace.push(s.desc);
        if (s.key || i % 8 === 7) {
            await settle(page);
            const bad = [...errors.splice(0), ...(await checkLeak(page, s)), ...(await checkDocument(page))];
            if (bad.length) return fail(bad.join(' ; '));
        }
    }
    await settle(page);
    const end = [...errors.splice(0), ...(await checkDocument(page)), ...(await checkInteractive(page)), ...errors.splice(0)];
    return end.length ? fail(end.join(' ; ')) : '';
}

// `MONKEY_SEED=7` rejoue une graine, `MONKEY_SEED=5,6,7` en balaie plusieurs.
const envSeeds = (process.env.MONKEY_SEED || '').split(',').map(Number).filter(Boolean);
const SEEDS = envSeeds.length ? envSeeds : [1, 2, 3, 4];
const STEPS = Number(process.env.MONKEY_STEPS || 120);

test.describe('transcription — singe', () => {
    for (const seed of SEEDS) {
        test(`graine ${seed} (${STEPS} pas)`, async ({ page }) => {
            test.setTimeout(60_000 + STEPS * 1_500);
            test.info().annotations.push({ type: 'seed', description: String(seed) });
            expect(await runMonkey(page, seed, STEPS)).toBe('');
        });
    }
});

/** Ouvre le premier brouillon de la liste et donne le focus au panneau. */
async function openFirstDraft(page) {
    await page.locator(`${panel} tbody tr`).first().click();
    await expect(page.locator(`${panel} .draft-bar`)).toBeVisible();
    await settle(page);
    await page.locator(`${panel} .save-state`).click();
}

// Repros réduites des manquements que le singe a trouvés : plus courtes à rejouer que sa graine.
test.describe('transcription — singe, manquements réduits', () => {
    test.beforeEach(() => test.setTimeout(60_000));

    // Brouillon ouvert, focus dans le panneau, fin de document, aucun dé tapé :
    // sans liste de candidats `k` n'est pas pris par la machine à touches et
    // remonte au dispatcher global, qui recule dans la bibliothèque (et
    // recharge analyse, commentaire, provenance). `j` passe par la même branche.
    test('k sans candidats ne parcourt pas la bibliothèque', async ({ page }) => {
        const errors = [];
        await boot(page, 1, errors);
        await openFirstDraft(page);
        const before = await worldOf(page);
        await page.keyboard.press('k');
        await settle(page);
        expect(before.focusInPanel).toBe(true);
        expect(await checkLeak(page, { key: 'k', before })).toEqual([]);
    });

    // Modale MAT, bouton Copier, presse-papiers refusé par le webview : le
    // refus est annoncé (console.error du logger) sans exception non rattrapée.
    test('Copier le MAT sans droit au presse-papiers ne lève rien', async ({ page }) => {
        const errors = [];
        await boot(page, 1, errors);
        await page.context().clearPermissions();
        await openFirstDraft(page);
        await page.locator(`${panel} .draft-bar .new-btn`).nth(2).click();
        await page.locator('.modal-overlay .mat-body button').click();
        await settle(page);
        expect(errors.filter((e) => e.startsWith('pageerror'))).toEqual([]);
    });
});
