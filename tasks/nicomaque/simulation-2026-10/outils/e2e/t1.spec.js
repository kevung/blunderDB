/**
 * t1.spec.js — tournoi T1 de la simulation #380 : 16 joueurs, préréglage suisse 2 vies continu →
 * tableau à exemptions, joué PAR L'INTERFACE de l'inscription au classement final.
 *
 * Persona Sophie (P1, 1920×1080, souris + clavier, le moins de gestes). Puis Hélène (P4) rejoue
 * l'inscription, un lancement, trois résultats et la lecture du classement, au clavier seul à
 * 1366×768 puis à 150 %, sur un second tournoi de la même base.
 *
 * Le shim ne sert qu'à vérifier et à tenir la feuille papier : les ids (le directeur connaît ses
 * joueurs), le tirage du tableau et les exemptions que l'application a faits (recopiés de l'écran
 * par le directeur). Les vainqueurs sont décidés par la graine, jamais lus dans l'application.
 *
 * Sorties : feuilles/T1.json (feuille papier), $SIM_DIR/t1/meter.jsonl, $SIM_DIR/t1/t1-trace.json
 * (interruptions V0/V1, comparaison oracle), $SIM_DIR/t1/sortie (page murale, CSV).
 */
import { test, expect } from '@playwright/test';
import { installRealBackend, shimCall, shimPageWarnings } from './realBackend.js';
import { Shim } from './shimProcess.js';
import { Meter, VIEWPORTS, zoomViewport } from './meter.js';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { execFileSync } from 'node:child_process';

const here = path.dirname(fileURLToPath(import.meta.url));
const SIMROOT = path.resolve(here, '../..');
const SHEET = path.join(SIMROOT, 'feuilles', 'T1.json');
const PORT = Number(process.env.T1_SHIM_PORT || 7183);

const NAMES = [
    'Aurore Delcambre', 'Basile Fontanet', 'Capucine Lemarrec', 'Désiré Oudinot',
    'Elvire Castagnac', 'Fulbert Arsonval', 'Gaëlle Picquenot', 'Hippolyte Marsault',
    'Ingrid Vallespir', 'Joachim Turpault', 'Katell Brasseur', 'Lazare Quintrec',
    'Mahaut Signoret', 'Nestor Bellefond', 'Ombeline Royer', 'Prosper Galimard'
];

function mulberry32(seed) {
    let a = seed >>> 0;
    return () => {
        a = (a + 0x6d2b79f5) >>> 0;
        let t = a;
        t = Math.imul(t ^ (t >>> 15), t | 1);
        t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
        return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
}
const rnd = mulberry32(380001);
// PR d'entrée (plus bas = meilleur), entre 3,0 et 14,0, une décimale.
const PLAYERS = NAMES.map((name) => ({ name, rating: Math.round((3 + rnd() * 11) * 10) / 10 }));
const ratingOf = Object.fromEntries(PLAYERS.map((p) => [p.name, p.rating]));
/** Probabilité que A batte B : logistique sur l'écart de PR (4 points de PR ≈ 77 %). */
const pWin = (ra, rb) => 1 / (1 + Math.exp((ra - rb) * 0.3));

test('T1 : Sophie dirige 16 joueurs, Hélène rejoue au clavier', async ({ page }) => {
    test.setTimeout(40 * 60 * 1000);
    const shim = await new Shim({ name: 't1', port: PORT }).start();
    const trace = { assignments: [], wallChecks: [], notes: [], labels: [] };
    const sheet = {
        name: 'T1',
        players: PLAYERS.map((p) => ({ name: p.name, rating: p.rating })),
        phases: [{ kind: 'swiss_lives', lives: 2 }, { kind: 'lives_bracket' }],
        events: []
    };
    const saveSheet = () => fs.writeFileSync(SHEET, JSON.stringify(sheet, null, 2) + '\n');
    const saveTrace = () => fs.writeFileSync(path.join(shim.dir, 't1-trace.json'), JSON.stringify(trace, null, 2));
    try {
        await page.setViewportSize(VIEWPORTS.hd);
        await page.context().grantPermissions(['clipboard-read', 'clipboard-write']);
        await installRealBackend(page, { shimUrl: shim.url, dbPath: shim.dbPath, outDir: shim.outDir });
        // realBackend.js rend UIScale = 1 (un facteur) là où Config attend un pourcentage : uiScaleStore
        // le borne à 50 % et toute l'interface est mesurée à moitié taille. On rend 100 ici (écart du
        // harnais, noté au rapport, realBackend.js n'est pas à cette spec).
        if (!process.env.T1_KEEP_HALF_SCALE)
            await page.addInitScript(() => {
                const c = window.go.main.Config;
                window.go.main.Config = new Proxy(c, { get: (t, p) => (p === 'GetUIScale' ? () => Promise.resolve(100) : t[p]) });
            });
        const m = new Meter(page, { persona: 'P1', tournament: 'T1', viewport: VIEWPORTS.hd, file: path.join(shim.dir, 'meter.jsonl'), feedbackMs: 800 });
        await page.goto('/');

        // ---------- Entrée ----------
        await m.act({ action: 'ouvrir Tournois', mustPass: true }, async (g) => g.click('[data-testid="tab-tournaments"]'));
        await m.act({ action: 'créer le tournoi', mustPass: true }, async (g) => {
            await g.click('#tournamentPanel [data-testid="panel-new"]');
            await g.type('Open du club T1');
            await g.press('Enter');
        });
        await expect(page.locator('#tournamentPanel tbody tr')).toHaveCount(1);
        await m.act({ action: 'diriger (préréglage suisse+tableau)', mustPass: true }, async (g) => {
            await g.dblclick(page.locator('#tournamentPanel tbody tr').first());
            await g.click('[data-testid="tournament-direct"]');
        });
        await expect(page.locator('[data-testid="direction-tab-players"]')).toBeVisible({ timeout: 20000 });
        const tid = (await shimCall(shim.url, 'ListDirections'))[0].tournamentId;
        const v0 = await shimCall(shim.url, 'GetDirection', tid);
        trace.config = v0.config;
        sheet.phases = (v0.config.phases || []).map((ph) => {
            const o = { kind: ph.kind };
            for (const k of ['lives', 'target', 'group_size', 'qualifiers', 'entry', 'seeding']) if (ph[k] != null && ph[k] !== '' && ph[k] !== 0) o[k] = ph[k];
            return o;
        });

        // ---------- Inscription (Sophie : nom, Tab Tab, cote, Entrée) ----------
        await m.act({ action: 'onglet Joueurs', mustPass: true }, async (g) => g.click('[data-testid="direction-tab-players"]'));
        const nameField = '[data-testid="direction-player-entry"] input[type="text"] >> nth=0';
        for (const [i, p] of PLAYERS.entries()) {
            m.o.checkViewports = i === 0;
            await m.act({ action: 'inscrire un joueur (nom + cote)', mustPass: true }, async (g) => {
                // Le champ garde-t-il le focus après Entrée ? Sinon il faut recliquer.
                const focused = await page.evaluate(() => document.activeElement?.placeholder || '');
                if (i === 0 || !/nom/i.test(focused)) {
                    if (i > 0) g.problem('focus quitte le champ nom après inscription');
                    await g.click(nameField);
                }
                await g.type(p.name);
                await g.press('Tab');
                await g.press('Tab');
                await g.type(String(p.rating).replace('.', ','));
                await g.press('Enter');
            });
        }
        const parts = await shimCall(shim.url, 'Participants', tid);
        expect(parts.length).toBe(16);
        const idOf = Object.fromEntries(parts.map((r) => [r.name, r.id]));
        const nameOf = Object.fromEntries(parts.map((r) => [r.id, r.name]));
        trace.ratingsStored = parts.map((r) => [r.name, r.rating]);
        sheet.players = PLAYERS.map((p) => ({ id: idOf[p.name], name: p.name, rating: p.rating }));
        saveSheet();

        // ---------- Page murale : Sophie la trouve dans l'en-tête ----------
        await m.act({ action: 'configurer la page murale (en-tête)' }, async (g) => {
            await g.click('[data-testid="direction-open-page"]');
        });
        await page.waitForTimeout(500);
        const pageFile = path.join(shim.outDir, 'tournoi.html');
        trace.wallWrittenAtSetup = fs.existsSync(pageFile);

        // ---------- Direction ----------
        await m.act({ action: 'onglet Direction', mustPass: true }, async (g) => g.click('[data-testid="direction-tab-direction"]'));
        const queue = page.locator('.proposals .queue li:not(.empty)');
        const busy = page.locator('[role="gridcell"].busy');
        let seenEvents = 0;
        const counts = {};
        const once = (k) => (counts[k] = (counts[k] || 0) + 1) === 1;

        /** Ce que l'application a fait sans que le directeur le décide : exemptions, tirage. */
        const noteEngineEvents = async () => {
            const hist = await shimCall(shim.url, 'History', tid, '', '');
            const fresh = hist.slice(seenEvents);
            seenEvents = hist.length;
            for (const e of fresh) trace.notes.push({ kind: e.kind, raw: JSON.stringify(e).slice(0, 600) });
            return fresh;
        };
        /** Contrôle P5 : chaque joueur affecté lit-il sa table dans le fichier écrit ? */
        const checkWall = async (assigned) => {
            await page.waitForTimeout(150);
            const html = fs.existsSync(pageFile) ? fs.readFileSync(pageFile, 'utf8') : '';
            if (!trace.wallSample && html) {
                trace.wallSample = true;
                fs.writeFileSync(path.join(shim.dir, 'tournoi-premiere-ronde.html'), html);
            }
            for (const a of assigned) {
                const r = wallHasSeat(html, a.name, a.table);
                trace.wallChecks.push({ ...a, ...r });
            }
        };

        page.on('pageerror', (e) => trace.notes.push({ pageerror: String(e.message).slice(0, 300) }));
        page.on('console', (msg) => msg.type() === 'error' && trace.notes.push({ console: msg.text().slice(0, 300) }));
        for (let iter = 0; iter < (process.env.T1_P4_ONLY ? 0 : 400); iter++) {
            const view = await shimCall(shim.url, 'GetDirection', tid);
            if (view.finished) break;
            const nBusy = await busy.count();
            // Sophie saisit d'abord les résultats des tables occupées.
            if (nBusy) {
                const cell = busy.first();
                const table = Number((await cell.getAttribute('data-testid'))?.replace(/\D/g, '') || 0);
                const key = `result-${view.phase}`;
                m.o.checkViewports = once(key);
                let rec = null;
                const r = await m.act({ action: `saisir un résultat (phase ${view.phase + 1})` }, async (g) => {
                    await g.click(cell);
                    const card = page.locator('[data-testid="direction-result-card"]');
                    await expect(card).toBeVisible();
                    const an = (await card.locator('[data-testid="direction-result-winner-a"]').innerText()).trim();
                    const bn = (await card.locator('[data-testid="direction-result-winner-b"]').innerText()).trim();
                    const aWins = rnd() < pWin(ratingOf[an], ratingOf[bn]);
                    rec = { an, bn, w: aWins ? an : bn, table };
                    await g.click(`[data-testid="direction-result-winner-${aWins ? 'a' : 'b'}"]`);
                    await expect(card).toBeHidden();
                });
                if (!r.ok) {
                    trace.notes.push({ blocked: 'résultat', error: r.error });
                    break;
                }
                sheet.events.push({ type: 'match', phase: view.phase, a: idOf[rec.an], b: idOf[rec.bn], winner: idOf[rec.w] });
                saveSheet();
                await noteEngineEvents();
                continue;
            }
            const n = await queue.count();
            if (!n) {
                trace.notes.push({ stuck: true, proposals: view.proposals, state: view.state });
                break;
            }
            const props = view.proposals || [];
            const head = props[0] || {};
            const label = (await queue.first().locator('.what').innerText()).trim();
            trace.labels.push({ phase: view.phase, kind: head.kind, label, n });
            const matches = props.filter((p) => p.kind === 'match' || p.kind === 'start');
            const launchAll = await page.locator('[data-testid="direction-proposals-all"]').count();
            const before = await shimCall(shim.url, 'GetDirection', tid);
            const runningBefore = new Set((before.running || []).map((x) => x.id));
            if (launchAll && props.length > 1 && props.every((p) => !/draw|phase/i.test(p.kind || ''))) {
                m.o.checkViewports = once('all');
                await m.act({ action: 'lancer toute la file (Tout lancer + Confirmer)' }, async (g) => {
                    await g.click('[data-testid="direction-proposals-all"]');
                    await g.click('[data-testid="direction-proposals-confirm-all"]');
                });
            } else {
                const kindKey = `go-${head.kind}`;
                m.o.checkViewports = once(kindKey);
                await m.act({ action: `lancer la tête de file (${head.kind || label})` }, async (g) => g.click(queue.first().locator('.go')));
            }
            await page.waitForTimeout(250);
            const after = await shimCall(shim.url, 'GetDirection', tid);
            const fresh = await noteEngineEvents();
            // La feuille : exemptions suisses, passage de phase, tirage recopié de l'onglet Tableaux.
            for (const e of fresh) {
                if (/bye/i.test(e.kind || '')) {
                    const pid = findPlayer(e, nameOf);
                    if (pid) sheet.events.push({ type: 'bye', player: pid, phase: before.phase });
                    if (pid) trace.assignments.push({ kind: 'bye', name: nameOf[pid], table: 0 });
                }
            }
            if (after.phase > before.phase) sheet.events.push({ type: 'next_phase' });
            if (fresh.some((e) => e.kind === 'draw')) {
                await m.act({ action: 'lire le tirage (onglet Tableaux)' }, async (g) => g.click('[data-testid="direction-tab-brackets"]'));
                const br = await shimCall(shim.url, 'Brackets', tid);
                const ph = br.find((b) => b.index === after.phase) || br[after.phase];
                const sec = ph.sections[0];
                const r1 = Math.min(...sec.matches.map((x) => x.round));
                const first = sec.matches.filter((x) => x.round === r1);
                trace.drawKeys = first.map((x) => [x.key, x.a, x.b]);
                const slots = first.flatMap((x) => [x.a && nameOf[x.a] ? x.a : null, x.b && nameOf[x.b] ? x.b : null]);
                sheet.events.push({ type: 'draw', phase: after.phase, slots });
                // « Est-ce que je joue ? » : l'exempté du 1er tour lit-il son exemption au mur ?
                const html = fs.existsSync(pageFile) ? fs.readFileSync(pageFile, 'utf8') : '';
                fs.writeFileSync(path.join(shim.dir, 'tournoi-apres-tirage.html'), html);
                trace.byeChecks = first
                    .filter((x) => (x.a === 'BYE') !== (x.b === 'BYE'))
                    .map((x) => {
                        const name = nameOf[x.a === 'BYE' ? x.b : x.a];
                        return { name, ...wallHasSeat(html, name, 0) };
                    });
                await m.act({ action: 'revenir à la Direction' }, async (g) => g.click('[data-testid="direction-tab-direction"]'));
            }
            const assigned = [];
            for (const mt of after.running || []) {
                if (runningBefore.has(mt.id)) continue;
                for (const pid of [mt.a, mt.b]) if (nameOf[pid]) assigned.push({ kind: 'match', name: nameOf[pid], table: mt.table, phase: after.phase });
            }
            trace.assignments.push(...assigned);
            await checkWall(assigned);
            saveSheet();
            saveTrace();
        }

        // ---------- Classement ----------
        await m.act({ action: 'onglet Classement', mustPass: true }, async (g) => g.click('[data-testid="direction-tab-standings"]'));
        const shownStandings = await page.locator('.standings table tbody tr, [data-testid^="direction-standings-row"]').allInnerTexts().catch(() => []);
        trace.standingsScreen = shownStandings;
        await m.act({ action: 'copier le classement CSV' }, async (g) => g.click('[data-testid="direction-standings-csv"]'));
        await page.waitForTimeout(300);
        trace.csvStatus = await page.locator('.status-bar, [data-testid="status-bar"]').first().innerText().catch(() => '');
        trace.clipboard = await page.evaluate(() => navigator.clipboard.readText()).catch((e) => 'ERR ' + e.message);
        await m.act({ action: 'enregistrer le classement (fichier)' }, async (g) => g.click('[data-testid="direction-standings-save"]'));
        await page.waitForTimeout(500);
        trace.saveStatus = await page.locator('.status-bar, [data-testid="status-bar"]').first().innerText().catch(() => '');
        trace.savedCalls = await page.evaluate(() => (window.__calls || []).filter((c) => /csv|save/i.test(c.name)).map((c) => ({ name: c.name, args: JSON.stringify(c.args).slice(0, 3000) })));
        trace.outFiles = fs.readdirSync(shim.outDir);
        const final = await shimCall(shim.url, 'GetDirection', tid);
        trace.appRanking = final.ranking;
        trace.finished = final.finished;
        trace.warnings = final.warnings;
        trace.pageWarnings = await shimPageWarnings(shim.url);
        saveSheet();
        // Oracle.
        try {
            const out = execFileSync('python3', [path.join(SIMROOT, 'outils/oracle/oracle.py'), SHEET], { encoding: 'utf8' });
            trace.oracle = JSON.parse(out);
        } catch (e) {
            trace.oracle = { error: String(e.stdout || e.message).slice(0, 2000) };
        }
        saveTrace();

        // ---------- Hélène (P4) : clavier seul, second tournoi ----------
        await helene(page, m, shim, trace, saveTrace);
        saveTrace();
    } finally {
        saveTrace();
        await shim.dispose({ keep: !!process.env.SIM_KEEP });
    }
});

async function helene(page, m, shim, trace, saveTrace) {
    const kb = makeKb(page);
    const lap = VIEWPORTS.laptop;
    await page.setViewportSize(lap);
    m.setPersona('P4', { input: 'keyboard', viewport: lap, checkViewports: true });
    m.o.tournament = 'T1-P4';
    await page.keyboard.press('Escape');
    await m.act({ action: 'P4 ouvrir Tournois' }, async (g) => g.click('[data-testid="tab-tournaments"]'));
    await m.act({ action: 'P4 créer le tournoi' }, async (g) => {
        await g.click('#tournamentPanel [data-testid="panel-new"]');
        await g.type('T1 rejoué clavier');
        await g.press('Enter');
    });
    const row = page.locator('#tournamentPanel tbody tr', { hasText: 'T1 rejoué clavier' });
    await m.act({ action: 'P4 ouvrir le tournoi (ligne) : essai clavier' }, async (g) => {
        // La ligne n'a ni tabindex ni touche : Tab atteint au mieux son bouton ✎ (renommer).
        const reached = await g.tabTo(row.locator('td').first());
        if (!reached) g.problem('ligne de tournoi injoignable au clavier (ni tabindex ni Entrée) : repli double-clic souris');
        await page.keyboard.press('Escape');
    });
    await m.act({ action: 'P4 ouvrir le tournoi (repli : double-clic souris)' }, async (g) => {
        g.problem('repli souris imposé : PanelTable n’active une ligne qu’au double-clic');
        await g.dblclick(row);
    });
    await m.act({ action: 'P4 diriger' }, async (g) => g.click('[data-testid="tournament-direct"]'));
    await page.waitForTimeout(2500);
    const diag = async (tag) => {
        await page.screenshot({ path: path.join(shim.dir, `p4-${tag}.png`) });
        return page.evaluate(() => ({
            active: document.activeElement?.outerHTML?.slice(0, 160),
            tabs: [...document.querySelectorAll('[data-testid^="direction-tab-"]')].map((e) => e.getAttribute('data-testid') + (e.offsetParent ? '' : '(caché)')),
            panes: [...document.querySelectorAll('[data-testid^="direction-pane-"]')].map((e) => e.getAttribute('data-testid') + (e.offsetParent ? '' : '(caché)')),
            dialogs: window.__dialogs,
            heading: document.querySelector('.direction-view h2, .direction-view header')?.textContent?.slice(0, 120)
        }));
    };
    trace.p4AfterDirect = await diag('apres-diriger');
    let opened = await page.locator('[data-testid="direction-tab-players"]').isVisible().catch(() => false);
    if (!opened) {
        await m.act({ action: 'P4 rouvrir la direction (Ouvrir la direction)' }, async (g) => {
            g.problem('la Direction ouverte au clavier par « Diriger » n’est pas affichée : nouvel essai par « Ouvrir la direction »');
            await g.click('[data-testid="tournament-direction-toggle"]');
        });
        await page.waitForTimeout(2500);
        trace.p4AfterToggle = await diag('apres-ouvrir');
        opened = await page.locator('[data-testid="direction-tab-players"]').isVisible().catch(() => false);
    }
    const dirs = await shimCall(shim.url, 'ListDirections');
    trace.heleneDirections = dirs.map((d) => d.tournamentId);
    if (!opened || dirs.length < 2) {
        trace.notes.push({ helene: 'Direction non ouverte au clavier', opened, dirs: dirs.length });
        return;
    }
    const tid = dirs.map((d) => d.tournamentId).sort((a, b) => b - a)[0];
    // Diagnostic : où va le focus aux premiers Tab depuis le panneau ouvert ?
    trace.p4FirstTabs = [];
    for (let i = 0; i < 4; i++) {
        await page.keyboard.press('Tab');
        trace.p4FirstTabs.push(await page.evaluate(() => {
            const a = document.activeElement;
            const modal = [...document.querySelectorAll('[role="dialog"],[aria-modal="true"],.modal,.backdrop,.overlay')].filter((e) => e.offsetParent).map((e) => e.className + '|' + (e.getAttribute('data-testid') || '') + '|' + e.textContent.slice(0, 80));
            return { active: a === document.body ? 'body' : a?.outerHTML?.slice(0, 120), modal };
        }));
    }
    await page.screenshot({ path: path.join(shim.dir, 'p4-apres-tab.png') });
    await m.act({ action: 'P4 onglet Joueurs' }, async (g) => kb(g, '[data-testid="direction-tab-players"]'));
    const nameField = page.locator('[data-testid="direction-player-entry"] input[type="text"]').first();
    for (const [i, p] of PLAYERS.slice(0, 8).entries()) {
        m.o.checkViewports = i === 0;
        await m.act({ action: 'P4 inscrire un joueur' }, async (g) => {
            const onName = await nameField.evaluate((el) => el === document.activeElement);
            if (!onName) await g.tabTo(nameField);
            await g.type(p.name);
            await g.press('Tab');
            await g.press('Tab');
            await g.type(String(p.rating).replace('.', ','));
            await g.press('Enter');
        });
    }
    trace.heleneParticipants = (await shimCall(shim.url, 'Participants', tid)).length;
    await m.act({ action: 'P4 onglet Direction' }, async (g) => kb(g, '[data-testid="direction-tab-direction"]'));
    const queue = page.locator('.proposals .queue li:not(.empty)');
    const busy = page.locator('[role="gridcell"].busy');
    const ensureDirectionPane = async (tag) => {
        if (await page.locator('.proposals').isVisible().catch(() => false)) return;
        await m.act({ action: `${tag} revenir à l’onglet Direction` }, async (g) => {
            g.problem('la Direction est revenue sur un autre onglet (Réglages) : il faut reprendre l’onglet Direction');
            await kb(g, '[data-testid="direction-tab-direction"]');
        });
    };
    const launchLoop = async (tag, maxIter) => {
        for (let i = 0; i < maxIter && !(await busy.count()); i++) {
            await ensureDirectionPane(tag);
            if (!(await queue.count())) break;
            m.o.checkViewports = i === 0;
            if (i === 0) {
                await m.act({ action: `${tag} lancer la tête de file (Tab jusqu’à « Lancer »)` }, async (g) => kb(g, queue.first().locator('.go')));
                // Le réflexe suivant d'Hélène : Tab pour repartir. Mesuré tel quel (focus sur body).
                await page.waitForTimeout(300);
                await m.act({ action: `${tag} repartir au clavier après « Lancer » (réflexe Tab)` }, async (g) => kb(g, '[data-testid="direction-tab-direction"]'));
                await page.waitForTimeout(300);
            } else {
                // Le raccourci documenté : K/J sélectionne dans la file (et y met le focus), Entrée lance.
                await m.act({ action: `${tag} lancer la tête de file (K, Entrée)` }, async (g) => {
                    const onBody = await page.evaluate(() => document.activeElement === document.body);
                    if (onBody) g.problem('focus sur body avant K');
                    await g.press('k');
                    await g.press('Enter');
                });
            }
            await page.waitForTimeout(400);
        }
    };
    await launchLoop('P4', 8);
    let results = 0;
    await ensureDirectionPane('P4');
    for (let k = 0; k < 3 && (await busy.count()); k++) {
        await ensureDirectionPane('P4');
        m.o.checkViewports = k === 0;
        const r = await m.act({ action: 'P4 saisir un résultat (Tab, Entrée, ←, Entrée)' }, async (g) => {
            await kb(g, busy.first());
            const card = page.locator('[data-testid="direction-result-card"]');
            await expect(card).toBeVisible();
            await g.press('ArrowLeft');
            await g.press('Enter');
            await expect(card).toBeHidden();
        });
        if (r.ok) results++;
    }
    trace.heleneResults = results;
    await m.act({ action: 'P4 lire le classement' }, async (g) => kb(g, '[data-testid="direction-tab-standings"]'));
    // Zoom 150 % : mêmes gestes clés.
    const z = zoomViewport(lap, 1.5);
    await page.setViewportSize(z);
    m.setPersona('P4', { input: 'keyboard', viewport: z, checkViewports: true });
    m.o.tournament = 'T1-P4@150';
    await m.act({ action: 'P4@150 onglet Direction' }, async (g) => kb(g, '[data-testid="direction-tab-direction"]'));
    await launchLoop('P4@150', 6);
    if (await busy.count()) {
        await m.act({ action: 'P4@150 saisir un résultat' }, async (g) => {
            await kb(g, busy.first());
            await g.press('ArrowRight');
            await g.press('Enter');
        });
    }
    await m.act({ action: 'P4@150 onglet Joueurs, inscrire' }, async (g) => {
        await kb(g, '[data-testid="direction-tab-players"]');
        await g.tabTo(nameField);
        await g.type('Quentin Ravelet');
        await g.press('Enter');
    });
    await m.act({ action: 'P4@150 lire le classement' }, async (g) => kb(g, '[data-testid="direction-tab-standings"]'));
    trace.heleneStandingsBox = await page.locator('[data-testid="direction-standings-csv"]').boundingBox().catch(() => null);
    saveTrace();
}


/**
 * Clavier seul, comme Hélène : Maj+Tab d'abord (la cible est souvent juste avant le focus : les
 * onglets de la Direction précèdent son panneau), puis Tab vers l'avant (pilote du mesureur).
 */
function makeKb(page) {
    return async (g, target, key = 'Enter') => {
        const loc = typeof target === 'string' ? page.locator(target) : target;
        // Focus perdu sur body (après « Lancer », après un vainqueur) : le premier réflexe, Tab,
        // est pris par keyboardService (isFocusOnBoard → onglet Recherche) et la Direction disparaît.
        if (await page.evaluate(() => !document.activeElement || document.activeElement === document.body)) {
            g.record.keys += 1;
            g.record.tabs += 1;
            await page.keyboard.press('Tab');
            await page.waitForTimeout(300);
            const gone = !(await page.locator('[data-testid="direction-tab-direction"]').isVisible().catch(() => false));
            g.problem(`focus sur body : Tab détourné (keyboardService, isFocusOnBoard)${gone ? ' → onglet Recherche, Direction masquée' : ''}`);
            if (gone) {
                g.record.keys += 1;
                await page.keyboard.press('Control+y');
                await page.waitForTimeout(800);
                const back = await page.locator('[data-testid="direction-tab-direction"]').isVisible().catch(() => false);
                g.problem(`retour par Ctrl+Y (onglet Tournois)${back ? ', Direction revenue' : ', Direction toujours absente'}`);
            }
            if (await page.evaluate(() => document.activeElement === document.body)) {
                // J donne le focus à la file des propositions (raccourci de la Direction).
                g.record.keys += 1;
                await page.keyboard.press('j');
                await page.waitForTimeout(200);
                g.problem('J pour remettre le focus dans la Direction (file des propositions)');
            }
        }
        const h = await loc.elementHandle({ timeout: 5000 }).catch(() => null);
        if (!h) {
            g.problem(`cible absente au clavier : ${typeof target === 'string' ? target : 'locator'}`);
            await page.screenshot({ path: path.join(process.env.SIM_DIR || '/tmp', 't1', `p4-absent-${g.record.seq}.png`) }).catch(() => {});
            return false;
        }
        const on = () => h.evaluate((el) => el === document.activeElement || el.contains(document.activeElement));
        for (let i = 0; i < 30; i++) {
            if (await on()) {
                const vis = await page.evaluate(() => {
                    const a = document.activeElement;
                    const cs = getComputedStyle(a);
                    return a.matches(':focus-visible') && (cs.outlineStyle !== 'none' || cs.boxShadow !== 'none');
                });
                if (!vis) g.problem('focus invisible (Maj+Tab)');
                g.record.keys += 1;
                await page.keyboard.press(key);
                return true;
            }
            g.record.tabs += 1;
            g.record.keys += 1;
            await page.keyboard.press('Shift+Tab');
            g.record.tabOrder.push('⇧' + (await page.evaluate(() => {
                const a = document.activeElement;
                if (!a || a === document.body) return 'body';
                const tid = a.closest('[data-testid]')?.getAttribute('data-testid');
                return a.tagName.toLowerCase() + (tid ? '[' + tid + ']' : '') + ' ' + (a.textContent || a.placeholder || '').trim().slice(0, 20);
            })));
        }
        await g.click(loc);
        return true;
    };
}

/** Le nom du joueur est-il dans le même bloc (tr/li/div de ligne) qu'un numéro de table ? */
function wallHasSeat(html, name, table) {
    if (!html) return { written: false, ok: false };
    const esc = (s) => s.replace(/&/g, '&amp;').replace(/'/g, '&#39;');
    const hit = [name, esc(name)].map((n) => html.indexOf(n)).find((i) => i >= 0);
    if (hit == null) return { written: true, nameFound: false, ok: false };
    if (!table) return { written: true, nameFound: true, ok: true };
    // bloc englobant : de la dernière ouverture <tr|<li|<article avant le nom à sa fermeture.
    const start = Math.max(html.lastIndexOf('<tr', hit), html.lastIndexOf('<li', hit), html.lastIndexOf('<article', hit));
    const endTag = html.slice(start, start + 8).match(/^<(tr|li|article)/)?.[1] || 'tr';
    const end = html.indexOf(`</${endTag}>`, hit);
    const block = html.slice(start, end > 0 ? end : hit + 400).replace(/<[^>]+>/g, ' ');
    const ok = new RegExp(`(^|\\D)${table}(\\D|$)`).test(block);
    return { written: true, nameFound: true, ok, block: block.replace(/\s+/g, ' ').trim().slice(0, 160) };
}

function findKey(o, key) {
    if (!o || typeof o !== 'object') return undefined;
    if (key in o) return o[key];
    for (const v of Object.values(o)) {
        const r = findKey(typeof v === 'string' && v.startsWith('{') ? safeParse(v) : v, key);
        if (r !== undefined) return r;
    }
    return undefined;
}
function safeParse(s) {
    try {
        return JSON.parse(s);
    } catch {
        return null;
    }
}
function findPlayer(e, nameOf) {
    const s = JSON.stringify(e);
    for (const id of Object.keys(nameOf)) if (s.includes(`"${id}"`)) return id;
    return null;
}
