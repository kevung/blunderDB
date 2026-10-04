/**
 * t3.spec.js — T3 : une Rencontre (ADR-0056) à deux épreuves sur les mêmes tables, jouée par
 * l'interface par Léa (P3, directrice-joueuse, 1366×768, dock ouvert).
 *
 *   épreuve A « Open A »   16 joueurs, préréglage « poules » (poules de 4, 2 qualifiés → tableau)
 *   épreuve B « Suisse B »  8 joueurs, suisse 2 vies → tableau (3 joueurs aussi inscrits en A)
 *
 * Incident N26 : un qualifié de poule se retire juste avant le tableau. Léa quitte la page trois
 * fois (autre onglet de l'app, rechargement) et y revient : l'état retrouvé est relevé, le retour
 * mesuré. La page murale de la Rencontre est configurée ; après chaque affectation, le fichier
 * écrit est lu pour savoir si le joueur y lit sa table et son épreuve (modèle P5, V0/V1).
 *
 * Sorties dans $SIM_DIR/t3 : meter.jsonl, t3-raw.json (ce que Léa a noté : lancements, résultats,
 * retrait, tirages relevés, interruptions, contrôles de page murale), standings-*.json, dumps.
 * La feuille oracle est construite ensuite par t3-feuille.py.
 */
import { test, expect } from '@playwright/test';
import { installRealBackend, shimCall } from './realBackend.js';
import { Shim } from './shimProcess.js';
import { Meter, VIEWPORTS } from './meter.js';
import { A_PLAYERS, B_PLAYERS, rng, pWin } from './t3-lib.js';
import path from 'node:path';
import fs from 'node:fs';

const A = 'Open A';
const B = 'Suisse B';
const PR = Object.fromEntries([...A_PLAYERS, ...B_PLAYERS]);

test('T3 : Rencontre poules + suisse, Léa (P3)', async ({ page }) => {
    const shim = await new Shim({ name: 't3', port: 7185 }).start();
    const raw = { events: [], launches: [], wall: [], collisions: [], interruptions: [], notes: [], draws: {}, n26: {} };
    const note = (s) => {
        raw.notes.push(s);
        console.log('NOTE', s);
    };
    const dump = async (tag) => {
        fs.writeFileSync(path.join(shim.dir, `${tag}.txt`), await page.locator('body').innerText());
        await page.screenshot({ path: path.join(shim.dir, `${tag}.png`) });
    };
    const roll = rng(20261004);
    const tid = {};
    const occupied = new Map(); // table → {ep, a, b}
    try {
        await page.setViewportSize(VIEWPORTS.laptop);
        await installRealBackend(page, { shimUrl: shim.url, dbPath: shim.dbPath, outDir: shim.outDir });
        const m = new Meter(page, { persona: 'P3', tournament: 'T3', viewport: VIEWPORTS.laptop, file: path.join(shim.dir, 'meter.jsonl'), feedbackMs: 600 });
        await page.goto('/');

        // ---------- Mise en place des deux épreuves ----------
        await m.act({ action: 'ouvrir Tournois', mustPass: true }, async (g) => g.click('[data-testid="tab-tournaments"]'));
        for (const [tname, players] of [[A, A_PLAYERS], [B, B_PLAYERS]]) {
            await m.act({ action: `créer le tournoi ${tname}`, mustPass: true }, async (g) => {
                await g.click('#tournamentPanel [data-testid="panel-new"]');
                await g.type(tname);
                await g.press('Enter');
            });
            await m.act({ action: `diriger ${tname}`, mustPass: true }, async (g) => {
                await g.dblclick(page.locator('#tournamentPanel tbody tr', { hasText: tname }).first());
                await g.click('[data-testid="tournament-direct"]');
            });
            await expect(page.locator('[data-testid="direction-tab-players"]')).toBeVisible({ timeout: 20000 });
            if (tname === A) {
                await m.act({ action: 'A : format « Poules puis tableau »', mustPass: true }, async (g) => {
                    await g.click('[data-testid="direction-tab-settings"]');
                    await g.click('[data-testid="direction-format-poules"]');
                    await g.click('[data-testid="direction-settings-apply"]');
                });
            } else {
                // Léa veut un suisse 2 vies à 8 : le préréglage vise 16 (Σ vies = 16 ≤ 16 → la
                // suisse serait sautée, C-H5). Le champ s'appelle « Bascule ».
                await m.act(
                    { action: 'B : bascule du suisse à 8 (sinon suisse sautée)', mustPass: true, ecart: 'voulait un suisse 2 vies à 8 joueurs ; le préréglage vise 16 et sauterait la suisse sans un mot ; a dû trouver le champ « Bascule » et y mettre 8' },
                    async (g) => {
                        await g.click('[data-testid="direction-tab-settings"]');
                        await g.fill('label:has-text("Bascule") input', '8');
                        await g.click('[data-testid="direction-settings-apply"]');
                    }
                );
            }
            await m.act({ action: `${tname} : onglet Joueurs` }, async (g) => g.click('[data-testid="direction-tab-players"]'));
            let first = true;
            for (const [n, pr] of players) {
                m.o.checkViewports = first;
                await m.act({ action: 'inscrire un joueur (nom, Tab Tab, cote, Entrée)', mustPass: true }, async (g) => {
                    await g.click('[data-testid="direction-player-entry"] input[type="text"] >> nth=0');
                    await g.type(n);
                    await g.press('Tab');
                    await g.press('Tab');
                    await g.type(String(pr));
                    await g.press('Enter');
                });
                first = false;
            }
            m.o.checkViewports = true;
            await m.act({ action: `${tname} : Réglages (Rencontre)` }, async (g) => g.click('[data-testid="direction-tab-settings"]'));
            if (tname === A) {
                await m.act({ action: 'créer la Rencontre (nom, 8 tables)', mustPass: true }, async (g) => {
                    await g.click('[data-testid="rencontre-name"]');
                    await g.type('Rencontre du club');
                    await g.fill('[data-testid="rencontre-tables"]', '8');
                    await g.click('[data-testid="rencontre-create"]');
                });
            } else {
                await m.act({ action: 'rattacher B à la Rencontre', mustPass: true }, async (g) => g.click('[data-testid="rencontre-attach"]'));
            }
            await m.act({ action: 'confirmer le rattachement', mustPass: true }, async (g) => g.click('[data-testid="rencontre-attach-yes"]'));
            if (tname === A) {
                await m.act({ action: 'page murale : choisir le dossier de la Rencontre', mustPass: true }, async (g) => {
                    await g.click('[data-testid="rencontre-output"]');
                    await expect(page.locator('[data-testid="rencontre-open-page"]')).toBeVisible();
                });
                await m.act({ action: 'page murale : ouvrir dans le navigateur' }, async (g) => g.click('[data-testid="rencontre-open-page"]'));
                await dump('setup-rencontre');
                await m.act({ action: 'fermer la Direction, revenir à la liste', mustPass: true }, async (g) => {
                    await g.click('[data-testid="direction-close"]');
                    await g.click('#tournamentPanel .back-btn');
                });
            }
        }
        for (const d of await shimCall(shim.url, 'ListDirections')) {
            const ts = await shimCall(shim.url, 'GetDirection', d.tournamentId).catch(() => null);
            tid[ts?.config?.name || (d.tournamentId === 1 ? A : B)] = d.tournamentId;
        }
        note(`ids : ${JSON.stringify(tid)}`);
        await expect(page.locator('[data-testid="epreuve-tabs"]')).toBeVisible();
        await m.act({ action: 'ouvrir « Toutes les tables » (salle)', mustPass: true }, async (g) => g.click('[data-testid="epreuve-tab-hall"]'));
        await dump('hall-start');

        // ---------- Outils de la boucle ----------
        const wallFile = path.join(shim.outDir, 'index.html');
        const wallText = () => {
            if (!fs.existsSync(wallFile)) return '';
            return fs
                .readFileSync(wallFile, 'utf8')
                .replace(/<style[\s\S]*?<\/style>|<script[\s\S]*?<\/script>/g, '')
                .replace(/<[^>]+>/g, ' ')
                .replace(/&nbsp;/g, ' ')
                .replace(/&#39;/g, "'")
                .replace(/&amp;/g, '&')
                .replace(/\s+/g, ' ');
        };
        const wallSegment = (txt, table) => {
            const i = txt.indexOf(`Table ${table} `);
            if (i < 0) return '';
            const j = txt.indexOf(`Table ${table + 1} `, i + 1);
            return txt.slice(i, j < 0 ? i + 400 : j);
        };
        const checkWall = (ep, a, b, table, kind) => {
            const txt = wallText();
            const seg = wallSegment(txt, table);
            const rec = { ep, a, b, table, kind, onPage: !!seg && seg.includes(ep) && seg.includes(a) && seg.includes(b), aEp: seg.includes(ep) };
            // Deux matchs sur la même ligne de table ?
            for (let t = 1; t <= 8; t++) if ((wallSegment(txt, t).match(/ contre /g) || []).length > 1) raw.collisions.push({ where: 'page murale', table: t, seg: wallSegment(txt, t) });
            raw.wall.push(rec);
            return rec;
        };
        const hallTab = () => page.locator('[data-testid="epreuve-tab-hall"]');
        const ensureHall = async (g) => {
            if ((await hallTab().getAttribute('aria-pressed')) !== 'true') await g.click('[data-testid="epreuve-tab-hall"]');
        };
        const hallItems = () => page.locator('[data-testid="hall-queue"] li:has(.go)');
        const busyCells = () => page.locator('[data-testid="direction-hall"] [role="gridcell"].busy');
        const drawOf = async (ep) => {
            const br = await shimCall(shim.url, 'Brackets', tid[ep]).catch((e) => ({ error: String(e) }));
            raw.draws[`${ep}#${raw.events.length}`] = br;
            // « Est-ce que je joue ? » : ce que le mur répond à ce moment, par épreuve.
            await page.waitForTimeout(300);
            const html = fs.existsSync(wallFile) ? fs.readFileSync(wallFile, 'utf8') : '';
            const blocks = [...html.matchAll(/<section class="statuts"><h2>([^<]*)<\/h2><ul>([\s\S]*?)<\/ul><\/section>/g)];
            raw.statusBlocks = raw.statusBlocks || [];
            raw.statusBlocks.push({
                ep,
                at: raw.events.length,
                blocks: blocks.map((b) => ({ head: b[1], items: [...b[2].matchAll(/<li[^>]*>([^<]*)<\/li>/g)].map((x) => x[1].replace(/&#39;/g, "'")) }))
            });
            return br;
        };
        let checkedLaunch = 0;
        let checkedResult = 0;

        /** Lance tout ce que la salle propose (tables libres), épreuve par épreuve. Rend le nombre lancé. */
        const launchAll = async (only) => {
            let n = 0;
            for (let guard = 0; guard < 20; guard++) {
                let items = hallItems();
                if (only) items = page.locator(`[data-testid="hall-queue"] li[data-testid="hall-proposal-${tid[only]}"]:has(.go)`);
                if (!(await items.count())) break;
                const li = items.first();
                const group = await li.getAttribute('data-testid');
                const ep = Number(group.split('-').pop()) === tid[A] ? A : B;
                // La salle mêle les épreuves : chaque ligne commence par le nom de la sienne.
                const label = (await li.innerText())
                    .replace(/\s*Lancer\s*$/, '')
                    .replace(/\n/g, ' · ')
                    .replace(new RegExp(`^${ep} · `), '');
                if (/Phase suivante/.test(label) && ep === A && !raw.n26.done) return { n, stop: 'n26' };
                m.o.checkViewports = checkedLaunch++ < 3;
                const r = await m.act({ action: 'salle : Lancer la proposition en tête' }, async (g) => g.click(li.locator('.go')));
                m.o.checkViewports = true;
                n++;
                raw.launches.push({ ep, label, seq: r.seq });
                const mm = label.match(/^(.+?) – (.+?) · /);
                if (mm) {
                    const [, a, b] = mm;
                    const tm = label.match(/table (\d+)/);
                    let table = tm ? Number(tm[1]) : 0;
                    if (!table) {
                        // Pas de table annoncée dans la proposition : lue sur la grille après lancement.
                        await page.waitForTimeout(250);
                        const c = busyCells().filter({ hasText: a }).filter({ hasText: b }).first();
                        // « 7 pts » n'est pas un numéro de table : seul le data-testid en porte un.
                        const tidAttr = (await c.getAttribute('data-testid').catch(() => '')) || '';
                        table = Number((tidAttr.match(/direction-table-(\d+)/) || [0, 0])[1]);
                        raw.launches[raw.launches.length - 1].tableUnannounced = true;
                    }
                    if (!table) raw.noTable = [...(raw.noTable || []), { ep, a, b }];
                    if (table && occupied.has(table)) raw.collisions.push({ where: 'lancement', table, was: occupied.get(table), now: { ep, a, b } });
                    if (table) occupied.set(table, { ep, a, b });
                    await page.waitForTimeout(250);
                    checkWall(ep, a, b, table, 'match');
                } else if (/^Phase suivante/.test(label)) {
                    raw.events.push({ ep, type: 'next_phase', label });
                    await page.waitForTimeout(300);
                    await drawOf(ep);
                } else if (/^Tirage/.test(label)) {
                    await page.waitForTimeout(300);
                    raw.events.push({ ep, type: 'draw', label, bracketsKey: `${ep}#${raw.events.length}` });
                    await drawOf(ep);
                } else if (/xempt/i.test(label)) {
                    raw.events.push({ ep, type: 'bye', label });
                } else {
                    note(`proposition non classée : ${ep} « ${label} »`);
                    raw.events.push({ ep, type: 'other', label });
                }
                await page.waitForTimeout(150);
            }
            return { n };
        };

        /** Saisit le résultat de chaque table occupée dans la salle. */
        const resultAll = async (max = 99) => {
            let n = 0;
            for (let guard = 0; guard < max; guard++) {
                const cells = busyCells();
                if (!(await cells.count())) break;
                const cell = cells.first();
                const chip = (await cell.locator('[data-testid="hall-event-chip"]').innerText().catch(() => '')).trim();
                const ep = chip === A ? A : B;
                m.o.checkViewports = checkedResult++ < 3;
                const r = await m.act({ action: 'salle : ouvrir la table, cliquer le vainqueur' }, async (g) => {
                    await g.click(cell);
                    const card = page.locator('[data-testid="direction-result-card"]');
                    await expect(card).toBeVisible();
                    const a = (await card.locator('[data-testid="direction-result-winner-a"]').innerText()).trim();
                    const b = (await card.locator('[data-testid="direction-result-winner-b"]').innerText()).trim();
                    const wa = roll() < pWin(PR[a] ?? 10, PR[b] ?? 10);
                    g.record.match = { ep, a, b, winner: wa ? a : b };
                    await g.click(card.locator(`[data-testid="direction-result-winner-${wa ? 'a' : 'b'}"]`));
                    await expect(card).toBeHidden();
                });
                m.o.checkViewports = true;
                if (!r.ok) {
                    note(`saisie impossible : ${r.error}`);
                    await dump(`result-fail-${r.seq}`);
                    break;
                }
                raw.events.push({ ep, type: 'match', ...r.match });
                for (const [t, o] of occupied) if (o.ep === ep && [o.a, o.b].sort().join() === [r.match.a, r.match.b].sort().join()) occupied.delete(t);
                n++;
            }
            return n;
        };

        /** État visible de la Direction (pour les retours). */
        const snapshot = async () =>
            page.evaluate(() => {
                const q = (s) => document.querySelector(s);
                const active = [...document.querySelectorAll('[data-testid^="epreuve-tab-"]')].find((b) => b.classList.contains('active'));
                const tab = [...document.querySelectorAll('[data-testid^="direction-tab-"]')].find((b) => b.classList.contains('active'));
                const pane = [...document.querySelectorAll('[data-testid^="direction-pane-"]')].find((p) => !p.hidden);
                return {
                    directionOpen: !!q('[data-testid="direction-tab-direction"]'),
                    epreuve: active?.getAttribute('data-testid') || null,
                    tab: tab?.getAttribute('data-testid') || null,
                    card: !!q('[data-testid="direction-result-card"]'),
                    pane: pane?.getAttribute('data-testid') || null,
                    scroll: pane ? pane.scrollTop : null,
                    filter: q('[data-testid="direction-player-filter"]')?.value ?? null
                };
            });

        /** Une sortie de Léa : avant, partir, revenir, relever, puis remettre l'état (mesuré). */
        const interruption = async (label, setup, leave, want) => {
            await setup();
            const before = await snapshot();
            const t0 = Date.now();
            await leave();
            const after = await snapshot();
            await dump(`interruption-${raw.interruptions.length + 1}`);
            const rec = await m.act({ action: `retour après ${label} : retrouver épreuve/onglet/fiche/défilement` }, async (g) => {
                if (!(await page.locator('[data-testid="direction-tab-direction"]').count())) {
                    if (!(await page.locator('#tournamentPanel').isVisible().catch(() => false))) await g.click('[data-testid="tab-tournaments"]');
                    if (await page.locator('#tournamentPanel .back-btn').count()) await g.click('#tournamentPanel .back-btn');
                    await g.dblclick(page.locator('#tournamentPanel tbody tr', { hasText: A }).first());
                    await g.click('[data-testid="tournament-direction-toggle"]');
                    await expect(page.locator('[data-testid="direction-tab-direction"]')).toBeVisible({ timeout: 20000 });
                }
                await want(g);
            });
            const restored = await snapshot();
            raw.interruptions.push({ label, before, after, restored, ms: Date.now() - t0, clicks: rec.clicks, keys: rec.keys, wheel: rec.wheel.notches, ok: rec.ok, error: rec.error });
            note(`interruption ${label} : avant ${JSON.stringify(before)} / au retour ${JSON.stringify(after)}`);
        };
        const goTab = async (name) => {
            const tab = page.locator('[data-testid="tab-' + name + '"]');
            await tab.click();
            await page.waitForTimeout(500);
            await page.locator('[data-testid="tab-tournaments"]').click();
            await page.waitForTimeout(800);
        };

        // ---------- Jeu : poules de A et suisse de B en parallèle ----------
        let tick = 0;
        let interruptionsDone = 0;
        for (; tick < 60; tick++) {
            const { n: launched, stop } = await launchAll();
            if (stop === 'n26' && !(await busyCells().count())) break;
            // Interruption 1 : une fiche de résultat ouverte, Léa part jouer (onglet Matchs).
            if (tick === 1 && interruptionsDone === 0) {
                interruptionsDone++;
                await interruption(
                    'onglet Matchs (fiche ouverte dans la salle)',
                    async () => {
                        await busyCells().first().click();
                        await expect(page.locator('[data-testid="direction-result-card"]')).toBeVisible();
                    },
                    async () => goTab('matches'),
                    async (g) => {
                        await ensureHall(g);
                        if (!(await page.locator('[data-testid="direction-result-card"]').count())) await g.click(busyCells().first());
                        await g.press('Escape');
                    }
                );
            }
            const entered = await resultAll(1 + Math.floor(roll() * 3));
            if (!launched && !entered && !(await busyCells().count())) break;
            // Va-et-vient : Léa consulte le classement de B, puis revient à la salle.
            if (tick === 3) {
                await m.act({ action: 'va-et-vient : épreuve B → Classement' }, async (g) => {
                    await g.click(`[data-testid="epreuve-tab-${tid[B]}"]`);
                    await g.click('[data-testid="direction-tab-standings"]');
                });
                await m.act({ action: 'va-et-vient : épreuve A → Arbres (poules)' }, async (g) => {
                    await g.click(`[data-testid="epreuve-tab-${tid[A]}"]`);
                    await g.click('[data-testid="direction-tab-brackets"]');
                });
                await dump('va-et-vient-arbres-A');
                await m.act({ action: 'va-et-vient : retour à la salle' }, async (g) => g.click('[data-testid="epreuve-tab-hall"]'));
            }
            // Interruption 2 : rechargement, Léa était dans les Joueurs de A, liste défilée.
            if (tick === 4 && interruptionsDone === 1) {
                interruptionsDone++;
                await interruption(
                    'rechargement de la page (Joueurs de A défilés, filtre saisi)',
                    async () => {
                        await page.locator(`[data-testid="epreuve-tab-${tid[A]}"]`).click();
                        await page.locator('[data-testid="direction-tab-players"]').click();
                        const pane = page.locator('[data-testid="direction-pane-players"]');
                        await pane.evaluate((el) => (el.scrollTop = 300));
                        await page.waitForTimeout(200);
                    },
                    async () => {
                        await page.reload();
                        await page.waitForTimeout(3000);
                    },
                    async (g) => {
                        await g.click(`[data-testid="epreuve-tab-${tid[A]}"]`);
                        await g.click('[data-testid="direction-tab-players"]');
                        await g.wheel('[data-testid="direction-pane-players"]', 3);
                    }
                );
                await m.act({ action: 'retour à la salle' }, async (g) => g.click('[data-testid="epreuve-tab-hall"]'));
            }
        }
        await dump('avant-n26');

        // ---------- N26 : un qualifié de poule se retire avant le tableau ----------
        const qa = page.locator(`[data-testid="hall-queue"] li[data-testid="hall-proposal-${tid[A]}"]:has(.go)`);
        raw.n26.queueBefore = (await page.locator(`[data-testid="hall-queue"] li[data-testid="hall-proposal-${tid[A]}"]`).allInnerTexts().catch(() => [])).join('\n');
        await m.act({ action: 'N26 : A → Classement (qui est qualifié ?)' }, async (g) => {
            await g.click(`[data-testid="epreuve-tab-${tid[A]}"]`);
            await g.click('[data-testid="direction-tab-standings"]');
        });
        await dump('n26-classement-poules');
        const st = await shimCall(shim.url, 'Standings', tid[A]).catch((e) => ({ error: String(e) }));
        fs.writeFileSync(path.join(shim.dir, 'standings-A-poules.json'), JSON.stringify(st, null, 1));
        // Le retiré : un qualifié de la poule B qui ne joue pas l'épreuve B (relevé à l'écran Arbres).
        await m.act({ action: 'N26 : A → Arbres (poules, qualifiés)' }, async (g) => g.click('[data-testid="direction-tab-brackets"]'));
        await dump('n26-arbres-poules');
        const br = await drawOf(A);
        raw.n26.bracketsBefore = br;
        const inB = new Set(B_PLAYERS.map((p) => p[0]));
        const qualifiedText = await page.locator('[data-testid="direction-pane-brackets"]').innerText({ timeout: 3000 }).catch(() => '');
        raw.n26.arbresText = qualifiedText.slice(0, 3000);
        // Choisi d'après le classement de l'app : premier qualifié (note +100) hors épreuve B.
        const rows = Array.isArray(st) ? st : st?.rows || st?.final || [];
        const flat = JSON.stringify(st);
        // Qualifié sûr : dans une poule, plus de victoires que le 3e, et pas inscrit en B.
        let quitter = null;
        for (const sec of (Array.isArray(br) && br[0]?.sections) || []) {
            if (quitter || sec.kind !== 'poule') continue;
            const nm = {};
            const wins = {};
            for (const mt of sec.matches || []) for (const k of ['a', 'b']) if (mt[k]) wins[mt[k]] = wins[mt[k]] || 0;
            for (const mt of sec.matches || []) {
                if (mt.a) nm[mt.a] = mt.aName;
                if (mt.b) nm[mt.b] = mt.bName;
                if (mt.winner) wins[mt.winner] = (wins[mt.winner] || 0) + 1;
            }
            const order = Object.entries(wins).sort((x, y) => y[1] - x[1]);
            const third = order[2]?.[1] ?? -1;
            const pick = order.slice(0, 2).find(([p, w]) => w > third && !inB.has(nm[p]));
            if (pick) {
                quitter = nm[pick[0]];
                raw.n26.pool = sec.name;
                raw.n26.poolWins = order.map(([p, w]) => [nm[p], w]);
            }
        }
        quitter = quitter || 'Benoît Carrel';
        void rows;
        void flat;
        raw.n26.withdrawn = quitter;
        raw.n26.wanted = 'repêcher le suivant non retiré de sa poule';
        await m.act({ action: `N26 : A → Joueurs, filtrer « ${quitter} »` }, async (g) => {
            await g.click('[data-testid="direction-tab-players"]');
            await g.fill('[data-testid="direction-player-filter"]', quitter.split(' ')[0]);
        });
        const row = page.locator('tr[data-testid^="direction-player-"]', { hasText: quitter }).first();
        raw.n26.rowButtons = await row.innerText();
        await m.act({ action: 'N26 : retirer maintenant (+ confirmer)' }, async (g) => {
            await g.click(row.locator('[data-testid="direction-player-withdraw-now"]'));
            await page.waitForTimeout(300);
            await dump('n26-confirm');
            const dlg = page.locator('[role="dialog"], [role="alertdialog"], .modal').filter({ hasText: quitter }).last();
            // Le bouton de confirmation d'un retrait porte le nom du geste (« Retirer »).
            const btn = page.locator('[role="dialog"][aria-label="Confirmation"] button').filter({ hasText: /^(Retirer|Supprimer)$/ }).first();
            raw.n26.confirmText = await dlg.innerText().catch(() => '');
            await g.click(btn);
        });
        raw.events.push({ ep: A, type: 'withdraw', player: quitter });
        await page.waitForTimeout(500);
        await m.act({ action: 'N26 : A → Direction (ce que propose la file)' }, async (g) => g.click('[data-testid="direction-tab-direction"]'));
        await dump('n26-file-apres-retrait');
        raw.n26.queueAfter = await page.locator('[data-testid="direction-pane-direction"] .proposals').innerText().catch(() => '');
        raw.n26.repechageOffered = /rep[êe]ch|suivant|remplac/i.test(raw.n26.queueAfter + (await page.locator('body').innerText()));
        // Ce que Léa voulait : le suivant de la poule prend la place. La file le propose ; elle
        // prend la première proposition de repêchage et note sur sa feuille qui elle a repêché.
        const rep = page.locator('[data-testid="direction-pane-direction"] .proposals .queue li:not(.empty)', { hasText: /Repêchage/ }).first();
        if (await rep.isVisible().catch(() => false)) {
            raw.n26.repechageLabel = (await rep.innerText()).split('\n')[0];
            raw.n26.picked = raw.n26.repechageLabel.split(' remplace ')[0].trim();
            await m.act({ action: `N26 : repêcher ${raw.n26.picked}` }, async (g) => g.click(rep.locator('.go')));
            await page.waitForTimeout(400);
            raw.events.push({ ep: A, type: 'repechage', pool: raw.n26.pool, player: raw.n26.picked });
            raw.n26.queueAfterRepechage = await page.locator('[data-testid="direction-pane-direction"] .proposals').innerText().catch(() => '');
        }
        raw.n26.done = true;
        await m.act({ action: 'N26 : retour à la salle' }, async (g) => g.click('[data-testid="epreuve-tab-hall"]'));

        // Interruption 3 : rechargement pendant le tableau, Léa était dans le Classement de B.
        await interruption(
            'rechargement (Classement de B)',
            async () => {
                await page.locator(`[data-testid="epreuve-tab-${tid[B]}"]`).click();
                await page.locator('[data-testid="direction-tab-standings"]').click();
            },
            async () => {
                await page.reload();
                await page.waitForTimeout(3000);
            },
            async (g) => {
                await g.click(`[data-testid="epreuve-tab-${tid[B]}"]`);
                await g.click('[data-testid="direction-tab-standings"]');
            }
        );
        await m.act({ action: 'retour à la salle' }, async (g) => g.click('[data-testid="epreuve-tab-hall"]'));

        // ---------- Jeu : tableau de A, fin de B ----------
        for (let k = 0; k < 60; k++) {
            const { n: launched } = await launchAll();
            const entered = await resultAll(1 + Math.floor(roll() * 3));
            if (!launched && !entered && !(await busyCells().count())) break;
            if (k === 2) {
                // Interruption supplémentaire : onglet Stats pendant le tableau.
                await interruption(
                    'onglet Stats (salle)',
                    async () => ensureHall({ click: (s) => page.locator(s).click() }),
                    async () => goTab('stats'),
                    async (g) => ensureHall(g)
                );
            }
        }
        await dump('fin-salle');

        // ---------- Classements ----------
        for (const ep of [A, B]) {
            await m.act({ action: `classement final : épreuve ${ep}` }, async (g) => {
                await g.click(`[data-testid="epreuve-tab-${tid[ep]}"]`);
                await g.click('[data-testid="direction-tab-standings"]');
            });
            await dump(`classement-${ep === A ? 'A' : 'B'}`);
            fs.writeFileSync(path.join(shim.dir, `standings-${ep === A ? 'A' : 'B'}.json`), JSON.stringify(await shimCall(shim.url, 'Standings', tid[ep]), null, 1));
            fs.writeFileSync(path.join(shim.dir, `history-${ep === A ? 'A' : 'B'}.json`), JSON.stringify(await shimCall(shim.url, 'History', tid[ep], '', ''), null, 1));
        }
        raw.tid = tid;
        raw.wallFinal = wallText();
    } finally {
        fs.writeFileSync(path.join(shim.dir, 't3-raw.json'), JSON.stringify(raw, null, 1));
        await shim.dispose({ keep: true });
    }
});
