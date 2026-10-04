/**
 * t2.spec.js — T2 : 32 joueurs, suisse 2 vies continu puis tableau à exemptions, dirigé de bout en
 * bout PAR L'INTERFACE par Yanis (P2, bénévole novice, 1920×1080), avec ses erreurs et ses
 * hésitations, et les incidents obligatoires : retardataire (I2), forfait (I1), résultat saisi
 * par erreur découvert plus tard puis corrigé (I3), fermeture brutale du processus en pleine
 * ronde puis reprise (I5).
 *
 * Le shim n'est appelé directement que pour vérifier (ce que l'écran devrait montrer, la feuille
 * papier, l'instantané avant/après SIGKILL) : aucun geste du directeur ne passe par lui.
 *
 * Sorties : feuilles/T2.json (feuille papier : ce que Yanis VOULAIT), $SIM_DIR/t2/meter.jsonl,
 * $SIM_DIR/t2/t2-result.json (mesures agrégées, interruptions, reprise, oracle).
 */
import { test, expect } from '@playwright/test';
import { installRealBackend, shimCall, shimPageWarnings } from './realBackend.js';
import { Shim, SIM_DIR, buildTool } from './shimProcess.js';
import { Meter, VIEWPORTS } from './meter.js';
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const SIMROOT = path.resolve(here, '../..');
const SHEET = path.join(SIMROOT, 'feuilles', 'T2.json');
const ORACLE = path.join(SIMROOT, 'outils', 'oracle', 'oracle.py');

// 32 noms fictifs (sans apostrophe : la page murale échappe ' en &#39;).
const NAMES = [
    'Adèle Fournier', 'Bruno Lacaze', 'Chloé Marchetti', 'Denis Rocher', 'Élodie Brunet', 'Fabien Ollier',
    'Gisèle Taupin', 'Hugo Ferrand', 'Inès Delorme', 'Jules Barrault', 'Karine Vidal', 'Lucas Perrot',
    'Manon Girard', 'Nathan Coste', 'Océane Joly', 'Pascal Renaud', 'Quentin Morel', 'Rose Aubert',
    'Samuel Leroux', 'Tatiana Klein', 'Ugo Benard', 'Valérie Mace', 'William Royer', 'Xavier Pons',
    'Yasmine Hadad', 'Zoé Carlier', 'Arthur Lemoine', 'Bérénice Gomez', 'Cyril Dumas', 'Diane Leclerc',
    'Étienne Robin', 'Florence Guyot'
];
const LATE = 'Gaël Tessier';

function mulberry32(a) {
    return () => {
        a |= 0;
        a = (a + 0x6d2b79f5) | 0;
        let t = Math.imul(a ^ (a >>> 15), 1 | a);
        t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
        return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
}
const rng = mulberry32(20261002);
const RATING = {};
for (const n of [...NAMES, LATE]) RATING[n] = Math.round((3 + rng() * 9) * 10) / 10; // PR 3,0 à 12,0

/** Le vainqueur voulu, tiré de la cote (PR : plus bas = meilleur). */
const intended = (pa, pb) => (rng() < 1 / (1 + Math.exp((pa - pb) / 2.5)) ? 'a' : 'b');

test('T2 : Yanis dirige 32 joueurs, avec incidents et fermeture brutale', async ({ page }) => {
    test.setTimeout(60 * 60 * 1000);
    let shim = await new Shim({ name: 't2', port: Number(process.env.T2_SHIM_PORT || 7184) }).start();
    const OUT = {
        wasted: [],
        reshuffles: [],
        interruptions: { V0: 0, V1: 0, V1_beforePage: 0, V1_missing: [], askPlay: { V0: 0, V1: 0, events: [] }, afterResume: null },
        resume: null,
        oracle: null,
        notes: []
    };
    const STEP = { v: '' };
    const api = (m, ...a) => shimCall(shim.url, m, ...a);
    const meterFile = path.join(shim.dir, 'meter.jsonl');
    try {
        await page.setViewportSize(VIEWPORTS.hd);
        await installRealBackend(page, { shimUrl: shim.url, dbPath: shim.dbPath, outDir: shim.outDir });
        const m = new Meter(page, { persona: 'P2', tournament: 'T2', viewport: VIEWPORTS.hd, file: meterFile });
        const seen = {};
        /** Sonde les deux viewports aux deux premières occurrences d'une action, pas aux suivantes. */
        const act = async (cat, detail, fn, extra = {}) => {
            seen[cat] = (seen[cat] || 0) + 1;
            m.o.checkViewports = seen[cat] <= 2;
            m.o.feedbackMs = seen[cat] <= 3 ? 1500 : 700;
            const r = await m.act({ action: `${cat} : ${detail}`, ...extra }, fn);
            if (!r.ok) {
                console.log('ÉCHEC', cat, detail, r.error, STEP.v);
                OUT.notes.push(`échec « ${cat} » (${detail}) à l'étape ${STEP.v} : ${r.error}`);
                await page.screenshot({ path: path.join(SIM_DIR, `t2-echec-${r.seq}.png`) }).catch(() => {});
            }
            return r;
        };
        await page.goto('/');

        // ── Entrée ────────────────────────────────────────────────────────
        await act('ouvrir Tournois', 'onglet', async (g) => g.click('[data-testid="tab-tournaments"]'), { mustPass: true });
        await act('créer le tournoi', 'Club du jeudi', async (g) => {
            await g.click('#tournamentPanel [data-testid="panel-new"]');
            await g.type('Club du jeudi T2');
            await g.press('Enter');
        }, { mustPass: true });
        await act('diriger', 'ouvrir puis Diriger', async (g) => {
            const row = page.locator('#tournamentPanel tbody tr').first();
            // Yanis clique une fois sur la ligne, comme partout ailleurs : rien ne s'ouvre (C-H4).
            await g.hesitate('un clic sur la ligne du tournoi ne l’ouvre pas', async () => {
                await g.click(row);
                await page.waitForTimeout(600);
            });
            await g.dblclick(row);
            await g.click('[data-testid="tournament-direct"]');
            await expect(page.locator('[data-testid="direction-tab-players"]')).toBeVisible({ timeout: 20000 });
        }, { mustPass: true });
        const tid = (await api('ListDirections'))[0].tournamentId;
        const view = async () => {
            const x = await api('GetDirection', tid);
            x.running ||= [];
            x.proposals ||= [];
            x.players ||= [];
            return x;
        };
        let v = await view();
        const TABLES = v.config.tables.count;

        // Yanis vérifie le format avant d'inscrire : il lit Réglages, le préréglage lui convient.
        await act('lire les réglages', 'format suisse + tableau, 8 tables', async (g) => {
            await g.click('[data-testid="direction-tab-settings"]');
            await page.waitForTimeout(300);
        });

        // ── Inscriptions ─────────────────────────────────────────────────
        await act('onglet Joueurs', 'avant les inscriptions', async (g) => g.click('[data-testid="direction-tab-players"]'), { mustPass: true });
        const nameInput = page.locator('[data-testid="direction-player-entry"] input[type="text"]').first();
        const register = async (name, cat) =>
            act(cat, name, async (g) => {
                if (!(await nameInput.evaluate((el) => el === document.activeElement))) await g.click(nameInput);
                await g.type(name);
                await g.press('Tab');
                await g.press('Tab');
                await g.type(String(RATING[name]).replace('.', ','));
                await g.press('Enter');
                await expect(page.locator('td.name').filter({ hasText: name }).first()).toBeAttached();
            });
        for (const n of NAMES) await register(n, 'inscrire un joueur');
        v = await view();
        expect(v.players.length).toBe(32);
        const nameOf = (id) => (v.players.find((p) => p.id === id) || {}).name || id;
        const idOf = (name) => v.players.find((p) => p.name === name)?.id;
        // La cote saisie « 7,4 » est-elle lue comme 7,4 ?
        const badRating = v.players.filter((p) => Math.abs((p.rating || 0) - RATING[p.name]) > 0.05);
        if (badRating.length) OUT.notes.push(`cotes mal lues : ${badRating.map((p) => `${p.name} ${p.rating}`).join(', ')}`);

        const sheet = {
            name: 'T2 — Club du jeudi',
            persona: 'Yanis (P2)',
            players: v.players.map((p) => ({ id: p.id, name: p.name, rating: RATING[p.name] })),
            phases: [
                { kind: 'swiss_lives', lives: 2, target: v.config.phases[0].target, entry: 'survivors' },
                { kind: 'lives_bracket', entry: 'survivors' }
            ],
            events: [],
            incidents: []
        };
        const saveSheet = () => fs.writeFileSync(SHEET, JSON.stringify(sheet, null, 1));

        await act('onglet Direction', 'commencer', async (g) => g.click('[data-testid="direction-tab-direction"]'), { mustPass: true });

        // ── Page murale et interruptions ─────────────────────────────────
        let wallOn = false;
        const pageHtml = () => {
            const f = path.join(shim.outDir, 'tournoi.html');
            return fs.existsSync(f) ? fs.readFileSync(f, 'utf8') : '';
        };
        const statusOnPage = (html, name) => {
            const block = html.match(/<section class="statuts">([\s\S]*?)<\/section>/)?.[1] || '';
            return [...block.matchAll(/<li[^>]*>([^<]*)<\/li>/g)].some((li) => li[1].startsWith(`${name} — `));
        };
        const seatedOnPage = (html, name, table) => {
            const re = new RegExp(`<span class="num">Table ${table}</span>([^<]*)`, 'g');
            let mm;
            while ((mm = re.exec(html))) if (mm[1].includes(name)) return true;
            return false;
        };
        /** Une affectation : chaque joueur du match cherche sa table (modèle P5). */
        const assign = (match) => {
            const html = wallOn ? pageHtml() : '';
            for (const id of [match.a, match.b]) {
                OUT.interruptions.V0 += 1;
                if (!wallOn) {
                    OUT.interruptions.V1 += 1;
                    OUT.interruptions.V1_beforePage += 1;
                } else if (!seatedOnPage(html, nameOf(id), match.table)) {
                    OUT.interruptions.V1 += 1;
                    OUT.interruptions.V1_missing.push(`${nameOf(id)} table ${match.table} (${match.id})`);
                }
            }
        };
        /** « Est-ce que je joue ? » : exemption, retrait, élimination. */
        const askPlay = (what, id) => {
            OUT.interruptions.askPlay.V0 += 1;
            const html = wallOn ? pageHtml() : '';
            const name = nameOf(id);
            // La réponse est la ligne « Nom — statut » du bloc « Est-ce que je joue ? » de la page.
            const told = wallOn && statusOnPage(html, name);
            if (!told) OUT.interruptions.askPlay.V1 += 1;
            OUT.interruptions.askPlay.events.push(`${what} ${name}${told ? ' (lu sur la page)' : ''}`);
        };

        // ── Boucle de direction ──────────────────────────────────────────
        const queue = page.locator('.proposals .queue li:not(.empty)');
        let launches = 0;
        let results = 0;
        let lastLives = Object.fromEntries(v.players.map((p) => [p.id, 2]));
        const wrong = { pending: null }; // I3 : saisie erronée en attente de découverte
        let resumed = false;
        let withdrew = false;
        let lateDone = false;
        let quickWrongDone = false;
        let wrongTableDone = false;
        const confirmModal = () => page.getByRole('button', { name: 'Supprimer', exact: true });

        const trackEliminations = async () => {
            v = await view();
            const parts = await api('Participants', tid);
            for (const p of parts) {
                const was = lastLives[p.id];
                if (was > 0 && p.lives === 0 && p.state !== 'withdrawn' && v.phase === 0) askPlay('éliminé', p.id);
                lastLives[p.id] = p.lives;
            }
        };

        const discoverWallPage = async () => {
            // 16 joueurs viennent de demander où ils jouent : Yanis cherche un affichage.
            await act('page murale', 'trouver et configurer', async (g) => {
                await g.hesitate('onglet « Matchs » : une liste, pas un affichage pour la salle', async () => {
                    await g.click('[data-testid="direction-tab-slots"]');
                    await page.waitForTimeout(400);
                });
                await g.hesitate('onglet « Arbres » : vide pendant la suisse', async () => {
                    await g.click('[data-testid="direction-tab-brackets"]');
                    await page.waitForTimeout(400);
                });
                await g.click('[data-testid="direction-tab-settings"]');
                await g.click('[data-testid="direction-settings-output"]');
                await page.waitForTimeout(500);
                await g.click(page.getByRole('button', { name: 'Ouvrir dans le navigateur' }).last());
                await page.waitForTimeout(500);
                await g.click('[data-testid="direction-tab-direction"]');
            });
            v = await view();
            wallOn = !!v.outputDir && !!pageHtml();
            OUT.notes.push(`page murale : dossier ${v.outputDir ? 'choisi' : 'absent'}, fichier ${pageHtml() ? 'écrit' : 'absent'}, ouverte ${await page.evaluate(() => window.__opened.length)} fois`);
            // Rattrapage : les matchs déjà lancés sont maintenant lisibles.
        };

        const launchHead = async () => {
            const head = (await queue.first().innerText()).split('\n')[0].trim();
            const before = new Set((await view()).running.map((r) => r.id));
            const vb = await view();
            await act(/^Exemption/.test(head) ? 'donner une exemption' : /^Tirage/.test(head) ? 'lancer le tirage' : /^Phase suivante/.test(head) ? 'phase suivante' : /^Clore/.test(head) ? 'clore le tournoi' : /^Annuler/.test(head) ? 'réparer (annuler un match)' : 'lancer un match', head, async (g) => {
                await g.click(queue.first().locator('button').first());
                await page.waitForTimeout(250);
                if (await confirmModal().isVisible().catch(() => false)) {
                    g.problem('confirmation par un bouton « Supprimer »');
                    await g.click(confirmModal());
                }
                await page.waitForTimeout(250);
            });
            v = await view();
            const started = v.running.filter((r) => !before.has(r.id));
            if (/^Exemption/.test(head)) {
                const act0 = vb.proposals.find((p) => p.kind === 'bye');
                if (act0) {
                    sheet.events.push({ type: 'bye', player: act0.a || act0.player });
                    askPlay('exempté', act0.a || act0.player);
                }
            } else if (/^Phase suivante/.test(head)) {
                sheet.events.push({ type: 'next_phase' });
            } else if (/^Tirage/.test(head)) {
                const br = await api('Brackets', tid);
                const ph = br.find((b) => b.current) || br[br.length - 1];
                const ms = ph.sections[0].matches;
                const r0 = Math.min(...ms.map((x) => x.round));
                const slots = [];
                for (const x of ms.filter((x) => x.round === r0)) slots.push(x.a && x.a !== 'BYE' ? x.a : null, x.b && x.b !== 'BYE' ? x.b : null);
                sheet.events.push({ type: 'draw', slots });
                for (const x of ms.filter((x) => x.round === r0 && (!x.a || !x.b || x.a === 'BYE' || x.b === 'BYE'))) askPlay('exempté au tableau', x.a && x.a !== 'BYE' ? x.a : x.b);
            } else if (/^Annuler/.test(head)) {
                const c = vb.proposals.find((p) => p.kind === 'cancel_match');
                if (c) sheet.events.push({ type: 'cancel', match: c.match || c.matchId });
                OUT.notes.push(`réparation proposée et confirmée : ${head}`);
            }
            for (const s of started) {
                launches += 1;
                const expectTxt = `${nameOf(s.a)} – ${nameOf(s.b)}`;
                if (!head.includes(nameOf(s.a)) || !head.includes(nameOf(s.b))) OUT.reshuffles.push(`annoncé « ${head} », lancé « ${expectTxt} »`);
                assign(s);
            }
        };

        const enterResult = async (match, opts = {}) => {
            const want = intended(RATING[nameOf(match.a)] ?? 7, RATING[nameOf(match.b)] ?? 7);
            const wantId = want === 'a' ? match.a : match.b;
            const typedSide = opts.typeWrong || opts.quickFix ? (want === 'a' ? 'b' : 'a') : want;
            const cell = page.locator(`[data-testid="direction-table-${match.table}"]`);
            await act(opts.cat || 'saisir un résultat', `table ${match.table} : ${nameOf(wantId)} gagne${opts.typeWrong ? ' (tape le perdant)' : ''}`, async (g) => {
                if (opts.wrongTable) {
                    await g.hesitate(`fiche de la table ${opts.wrongTable.table} ouverte au lieu de la ${match.table}`, async () => {
                        await g.click(`[data-testid="direction-table-${opts.wrongTable.table}"]`);
                        await expect(page.locator('[data-testid="direction-result-card"]')).toBeVisible();
                        await g.press('Escape');
                        await page.waitForTimeout(200);
                        if (await page.locator('[data-testid="direction-result-card"]').isVisible()) {
                            g.problem('Échap ne referme pas la fiche');
                            await g.click('[data-testid="direction-result-card"] button.close');
                        }
                    });
                }
                await g.click(cell);
                await expect(page.locator('[data-testid="direction-result-card"]')).toBeVisible();
                const btnA = await page.locator('[data-testid="direction-result-winner-a"]').textContent();
                // La fiche range les joueurs dans l'ordre du match ; on clique le nom voulu, pas une position.
                const sideFor = (who) => (btnA.trim() === nameOf(who === 'a' ? match.a : match.b) ? 'a' : 'b');
                await g.click(`[data-testid="direction-result-winner-${sideFor(typedSide)}"]`);
                await expect(page.locator('[data-testid="direction-last"]')).toBeVisible();
                if (opts.quickFix) {
                    // Il lit le bandeau « X bat Y » : ce n'est pas ce qu'il voulait.
                    await g.click('[data-testid="direction-last-correct"]');
                    const a = page.locator('[data-testid="direction-last-winner-a"]');
                    const pick = (await a.textContent()).trim() === nameOf(wantId) ? 'a' : 'b';
                    await g.click(`[data-testid="direction-last-winner-${pick}"]`);
                    await page.waitForTimeout(300);
                }
            });
            results += 1;
            sheet.events.push({ type: 'match', id: match.id, phase: match.phase, a: match.a, b: match.b, winner: wantId });
            if (opts.typeWrong) {
                wrong.pending = { match, wantId, typed: want === 'a' ? match.b : match.a, at: results };
                sheet.incidents.push({ incident: 'I3', match: match.id, voulu: wantId, saisi: wrong.pending.typed, saisi_apres_resultats: results });
            }
            {
                const h = (await api('History', tid, '', match.id)).filter((e) => e.matchId === match.id && /result/.test(e.kind));
                const appW = h.at(-1)?.winner;
                const expectW = opts.typeWrong ? (want === 'a' ? match.b : match.a) : wantId;
                if (appW !== expectW) OUT.notes.push(`saisie divergente ${match.id} : voulu ${expectW}, app ${appW}`);
            }
            if (opts.quickFix) sheet.incidents.push({ incident: 'mauvais vainqueur corrigé aussitôt', match: match.id, voulu: wantId });
            await trackEliminations();
            saveSheet();
        };

        const fixWrong = async () => {
            const w = wrong.pending;
            const wName = nameOf(w.wantId);
            w.tries = (w.tries || 0) + 1;
            const rr = await act('corriger un ancien résultat', `${nameOf(w.match.a)} – ${nameOf(w.match.b)} : ${wName} avait gagné`, async (g) => {
                STEP.v = 'bandeau Corriger';
                await g.hesitate('« Corriger » du bandeau : il porte sur le dernier match, pas celui-là', async () => {
                    await g.click('[data-testid="direction-last-correct"]');
                    await page.waitForTimeout(300);
                    await g.click('[data-testid="direction-last-correct"]');
                });
                STEP.v = 'onglet Classement';
                await g.hesitate('onglet Classement : on y voit l’erreur, on ne peut pas la corriger', async () => {
                    await g.click('[data-testid="direction-tab-standings"]');
                    await page.waitForTimeout(400);
                });
                STEP.v = 'onglet Historique';
                await g.click('[data-testid="direction-tab-history"]');
                STEP.v = 'filtre Historique';
                await g.click('[data-testid="direction-history-filter"]');
                await g.type(wName.split(' ')[0]);
                await page.waitForTimeout(400);
                const line = page.locator('li[data-testid^="direction-history-"]').filter({ hasText: `${nameOf(w.typed)} gagne` }).filter({ hasText: wName }).first();
                const alt = page.locator('li[data-testid^="direction-history-"]').filter({ has: page.locator('[data-testid="direction-history-correct"]') }).filter({ hasText: nameOf(w.typed) }).first();
                const target = (await line.count()) ? line : alt;
                STEP.v = `ligne à corriger (${await line.count()} exactes, ${await alt.count()} approchées)`;
                await g.click(target.locator('[data-testid="direction-history-correct"]'));
                STEP.v = 'choix du vainqueur';
                const a = page.locator('[data-testid="direction-history-winner-a"]');
                const pick = (await a.textContent()).trim() === wName ? 'a' : 'b';
                await g.click(`[data-testid="direction-history-winner-${pick}"]`);
                await page.waitForTimeout(400);
                await g.click('[data-testid="direction-tab-direction"]');
            });
            if (!rr.ok) {
                await page.keyboard.press('Escape');
                if (w.tries < 3) {
                    w.at += 3;
                    return;
                }
                OUT.notes.push('I3 : correction impossible par l’interface après trois essais');
            }
            const hist = await api('History', tid, '', '');
            const corr = hist.filter((e) => e.kind === 'result_corrected');
            OUT.notes.push(`I3 : ${corr.length} correction(s) au journal ; dernière : ${corr.at(-1)?.winnerName || '—'}`);
            v = await view();
            OUT.notes.push(`I3 : avertissements après correction : ${JSON.stringify(v.warnings || [])}`);
            wrong.pending = null;
            await trackEliminations();
        };

        const lateArrival = async () => {
            await act('retardataire', `${LATE} arrive après le lancement`, async (g) => {
                await g.click('[data-testid="direction-tab-players"]');
                await g.click(nameInput);
                await g.type(LATE);
                await g.press('Tab');
                await g.press('Tab');
                await g.type(String(RATING[LATE]).replace('.', ','));
                if (await page.locator('[data-testid="direction-player-slot"]').isVisible().catch(() => false)) g.problem('un choix « Entre à » s’affiche');
                await g.press('Enter');
                await expect(page.locator('td.name').filter({ hasText: LATE }).first()).toBeAttached();
                await g.click('[data-testid="direction-tab-direction"]');
            });
            v = await view();
            const p = v.players.find((x) => x.name === LATE);
            sheet.events.push({ type: 'late', player: { id: p.id, name: LATE, rating: RATING[LATE] } });
            lastLives[p.id] = 2;
            OUT.notes.push(`I2 : infos après l'arrivée : ${JSON.stringify(v.infos || [])}`);
            saveSheet();
        };

        const withdrawPlaying = async () => {
            const mm = v.running.find((r) => r.phase === 0) || v.running[0];
            const leaver = mm.b;
            const stays = mm.a;
            await act('forfait et retrait', `${nameOf(leaver)} s’en va en plein match (table ${mm.table})`, async (g) => {
                // Il part de la table : la fiche du match, puis « ⋯ » qui montre « Forfait de ».
                await g.click(`[data-testid="direction-table-${mm.table}"]`);
                await g.click('[data-testid="direction-result-more"]');
                const btnA = (await page.locator('[data-testid="direction-result-winner-a"]').textContent()).trim();
                const side = btnA === nameOf(leaver) ? 'a' : 'b';
                await g.click(`[data-testid="direction-result-forfeit-${side}"]`);
                // La fiche propose forfait seul ou forfait et retrait du perdant, nommés.
                const both = page.getByRole('button', { name: `Déclarer forfait et retirer ${nameOf(leaver)}`, exact: true });
                await both.waitFor({ state: 'visible', timeout: 5000 }).catch(() => {});
                if (await both.isVisible()) await g.click(both);
                else {
                    g.problem('forfait et retrait non proposés ensemble : repli Joueurs → Retirer');
                    if (await confirmModal().isVisible().catch(() => false)) await g.click(confirmModal());
                    await g.click('[data-testid="direction-tab-players"]');
                    await g.click('[data-testid="direction-player-filter"]');
                    await g.type(nameOf(leaver).split(' ')[0]);
                    await g.click(`[data-testid="direction-player-${leaver}"] [data-testid="direction-player-withdraw-now"]`);
                    await page.waitForTimeout(250);
                    if (await confirmModal().isVisible().catch(() => false)) await g.click(confirmModal());
                    await g.click('[data-testid="direction-player-filter"]');
                    await g.press('ControlOrMeta+a');
                    await g.press('Backspace');
                }
                await page.waitForTimeout(300);
                await g.click('[data-testid="direction-tab-direction"]');
            });
            results += 1;
            sheet.events.push({ type: 'match', id: mm.id, phase: mm.phase, a: mm.a, b: mm.b, winner: stays, forfeit: true });
            sheet.events.push({ type: 'withdraw', player: leaver });
            sheet.incidents.push({ incident: 'I1', player: leaver, match: mm.id });
            askPlay('retiré', leaver);
            v = await view();
            const pw = v.players.find((p) => p.id === leaver);
            OUT.notes.push(`I1 : état de ${nameOf(leaver)} après : ${JSON.stringify(pw)}`);
            await trackEliminations();
            saveSheet();
        };

        const snapshot = async () => {
            const vv = await view();
            const grid = await api('TableGrid', tid);
            const st = await api('Standings', tid);
            const parts = await api('Participants', tid);
            const ui = (await page.locator('.direction-view').innerText()).replace(/\d{2}:\d{2}/g, 'HH:MM').replace(/\d+ min/g, 'N min');
            const strip = (o) => JSON.parse(JSON.stringify(o, (k, val) => (['elapsed', 'elapsedSeconds', 'minutes', 'now', 'estimatedEnd', 'since'].includes(k) ? undefined : val)));
            return {
                api: strip({ state: vv.state, phase: vv.phase, ranking: vv.ranking, running: vv.running, proposals: vv.proposals, players: vv.players, eventCount: vv.eventCount, finished: vv.finished, outputDir: vv.outputDir }),
                grid: strip(grid),
                standings: strip(st),
                participants: strip(parts),
                ui,
                warnings: vv.warnings || []
            };
        };

        const crashAndResume = async () => {
            const before = await snapshot();
            const htmlBefore = pageHtml();
            await shim.kill(); // SIGKILL juste après la saisie, matchs en cours
            // Yanis : l'écran ne répond plus ; il « relance l'application ».
            const afterKill = await act('après la coupure', 'cliquer une table pendant que le processus est mort', async (g) => {
                await g.click(page.locator('[role="gridcell"].busy').first());
                await page.waitForTimeout(800);
                if (await page.locator('[data-testid="direction-result-card"]').isVisible()) await g.press('Escape');
            });
            shim = await new Shim({ name: 't2', port: shim.port, fresh: false }).start();
            await page.reload();
            await act('reprise', 'retrouver la direction après relance', async (g) => {
                const tab = page.locator('[data-testid="direction-tab-direction"]');
                // isVisible n'attend pas : la Direction rouverte par la session arrive après le
                // rechargement, et un clic sur « Ouvrir la direction » pendant ce temps la refermerait.
                const auto = await tab.waitFor({ state: 'visible', timeout: 20000 }).then(() => true, () => false);
                OUT.notes.push(`reprise : Direction ${auto ? 'rouverte seule par la session' : 'à rouvrir à la main'}`);
                if (!auto) {
                    await g.click('[data-testid="tab-tournaments"]');
                    const row = page.locator('#tournamentPanel tbody tr').first();
                    await g.dblclick(row);
                    const open = page.locator('[data-testid="tournament-direction-toggle"]');
                    const direct = page.locator('[data-testid="tournament-direct"]');
                    if (await open.isVisible().catch(() => false)) await g.click(open);
                    else await g.click(direct);
                    await expect(tab).toBeVisible({ timeout: 20000 });
                }
                await g.click(tab);
                await page.waitForTimeout(800);
            });
            const after = await snapshot();
            const same = (k) => JSON.stringify(before[k]) === JSON.stringify(after[k]);
            const diffs = ['api', 'grid', 'standings', 'participants', 'ui'].filter((k) => !same(k));
            const firstDiff = {};
            for (const k of diffs) {
                const a = JSON.stringify(before[k]);
                const b = JSON.stringify(after[k]);
                let i = 0;
                while (i < a.length && a[i] === b[i]) i++;
                firstDiff[k] = { before: a.slice(Math.max(0, i - 80), i + 120), after: b.slice(Math.max(0, i - 80), i + 120) };
            }
            const pageWarn = await shimPageWarnings(shim.url);
            const verify = spawnSync(buildTool('t2-verify'), ['tournament', 'verify', '--db', shim.dbPath, '--id', String(tid)], { encoding: 'utf8', timeout: 60000 });
            // La page murale juste après la reprise : chaque match en cours y est-il, à sa table ?
            v = await view();
            const html = pageHtml();
            const missing = [];
            for (const r of v.running) for (const id of [r.a, r.b]) if (!seatedOnPage(html, nameOf(id), r.table)) missing.push(`${nameOf(id)} table ${r.table}`);
            OUT.interruptions.afterResume = { running: v.running.length, missingOnPage: missing, pageRewritten: html !== htmlBefore, pageWarnings: pageWarn };
            OUT.resume = {
                running: before.api.running.length,
                eventCount: [before.api.eventCount, after.api.eventCount],
                identical: diffs.length === 0,
                diffs,
                firstDiff,
                warningsBefore: before.warnings,
                warningsAfter: after.warnings,
                pageWarnings: pageWarn,
                verify: { status: verify.status, out: (verify.stdout || '').slice(-600), err: (verify.stderr || '').slice(-600) },
                afterKillGesture: { feedback: afterKill.feedback, problems: afterKill.problems, error: afterKill.error }
            };
            console.log('REPRISE', JSON.stringify(OUT.resume).slice(0, 2500));
        };

        let guard = 0;
        while (guard++ < 400) {
            v = await view();
            if (v.finished) break;
            await page.waitForTimeout(150);
            const heads = await queue.count();
            const headTxt = heads ? (await queue.first().innerText()).split('\n')[0] : '';
            const headIsMatch = heads && / – /.test(headTxt) && !/^(Annuler|Exemption|Tirage|Phase)/.test(headTxt);
            const free = v.running.length < TABLES;

            if (!(await page.locator('[data-testid="direction-pane-direction"]').isVisible().catch(() => false))) {
                await act('revenir à Direction', 'la file et les tables ne sont pas à l’écran', async (g) => g.click('[data-testid="direction-tab-direction"]'));
                continue;
            }
            const lastRecs = m.records.slice(-3);
            if (lastRecs.length === 3 && lastRecs.every((r) => !r.ok)) {
                OUT.notes.push(`arrêt : trois gestes de suite en échec (${lastRecs.map((r) => r.action).join(' | ')})`);
                break;
            }
            // Incidents, déclenchés par l'avancée du tournoi.
            if (!wallOn && launches >= TABLES && results === 0) {
                await discoverWallPage();
                // Ceux qui jouent déjà : la page les montre maintenant, mais ils ont déjà demandé.
                continue;
            }
            if (!lateDone && results >= 3) {
                lateDone = true;
                await lateArrival();
                continue;
            }
            if (wrong.pending && results >= wrong.pending.at + 8) {
                await fixWrong();
                continue;
            }
            if (!resumed && results >= 26 && v.running.length >= 3 && v.phase === 0) {
                // Saisie, puis la coupure aussitôt après.
                const mm = v.running[Math.floor(rng() * v.running.length)];
                await enterResult(mm, { cat: 'saisir un résultat' });
                resumed = true;
                await crashAndResume();
                continue;
            }
            if (!withdrew && resumed && results >= 32 && v.running.length >= 1 && v.phase === 0) {
                withdrew = true;
                await withdrawPlaying();
                continue;
            }

            if (heads && (!headIsMatch || free)) {
                await launchHead();
                continue;
            }
            if (v.running.length) {
                const mm = v.running[Math.floor(rng() * v.running.length)];
                const opts = {};
                if (!quickWrongDone && results === 5) {
                    quickWrongDone = true;
                    opts.typeWrong = false;
                    opts.quickFix = true;
                }
                if (!wrongTableDone && results === 9 && v.running.length >= 2) {
                    wrongTableDone = true;
                    opts.wrongTable = v.running.find((r) => r.id !== mm.id);
                }
                if (!wrong.pending && !sheet.incidents.some((i) => i.incident === 'I3') && results >= 13 && v.phase === 0) {
                    // Il ne remarque rien : seulement si le perdant saisi garde une vie (sinon il le verrait éliminé).
                    const parts = await api('Participants', tid);
                    const lv = Object.fromEntries(parts.map((p) => [p.id, p.lives]));
                    if (lv[mm.a] === 2 && lv[mm.b] === 2) opts.typeWrong = true;
                }
                await enterResult(mm, opts);
                continue;
            }
            OUT.notes.push(`bloqué : file vide, aucun match en cours, état ${v.state}, phase ${v.phase}`);
            break;
        }
        // Mauvais vainqueur rapide : le bandeau doit avoir été corrigé — on le relit dans la feuille.
        v = await view();
        OUT.final = { finished: v.finished, state: v.state, warnings: v.warnings || [], ranking: v.ranking, results, launches };
        await act('lire le classement', 'onglet Classement', async (g) => g.click('[data-testid="direction-tab-standings"]'));
        OUT.uiStandings = (await page.locator('.direction-view').innerText()).slice(0, 4000);
        saveSheet();

        // ── Oracle ───────────────────────────────────────────────────────
        const or = spawnSync('python3', [ORACLE, SHEET], { encoding: 'utf8' });
        let oracle = null;
        try {
            oracle = JSON.parse(or.stdout);
        } catch {
            OUT.oracle = { error: (or.stderr || or.stdout).slice(-1500) };
        }
        if (oracle) {
            const o = Object.fromEntries(oracle.final.map((r) => [r.player, r.rank]));
            const a = Object.fromEntries((v.ranking || []).map((r) => [r.player, r.rank]));
            const ids = [...new Set([...Object.keys(o), ...Object.keys(a)])];
            const diff = ids.filter((id) => o[id] !== a[id]).map((id) => `${nameOf(id)} : oracle ${o[id] ?? '—'}, app ${a[id] ?? '—'}`);
            OUT.oracle = { identical: diff.length === 0, compared: ids.length, diff, warnings: oracle.warnings || [], final: oracle.final.map((r) => `${r.rank} ${r.name || r.player}`) };
        }
        console.log('ORACLE', JSON.stringify(OUT.oracle).slice(0, 2500));
        // Vérification CLI finale, processus arrêté proprement après le dernier geste.
        await shim.kill();
        const vf = spawnSync(buildTool('t2-verify'), ['tournament', 'verify', '--db', shim.dbPath, '--id', String(tid)], { encoding: 'utf8', timeout: 60000 });
        OUT.verifyFinal = { status: vf.status, out: (vf.stdout || '').slice(-500), err: (vf.stderr || '').slice(-500) };
    } finally {
        fs.writeFileSync(path.join(SIM_DIR, 't2-result.json'), JSON.stringify(OUT, null, 1));
        if (fs.existsSync(meterFile)) fs.copyFileSync(meterFile, path.join(SIM_DIR, 't2-meter.jsonl'));
        await shim.dispose({ keep: !!process.env.SIM_KEEP });
    }
});
