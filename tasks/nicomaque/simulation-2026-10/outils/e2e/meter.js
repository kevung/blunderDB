/**
 * meter.js — mesure de l'ergonomie geste par geste, sur la vraie Direction.
 *
 * Chaque action du directeur passe par `meter.act(meta, async (g) => { … })` ; le pilote `g` compte
 * ce qu'il fait et l'action est écrite en une ligne JSONL dans METER_FILE (ou `opts.file`).
 *
 * Ce qu'une ligne porte :
 *   clicks, keys          — clics et frappes (un `g.type` de 6 lettres = 6 frappes)
 *   wheel                 — crans de molette demandés par la spec, et pixels par conteneur
 *   autoScroll            — pixels qu'il a fallu défiler pour atteindre une cible : Playwright le fait
 *                           en silence, la personne doit le faire à la molette (≈ 100 px par cran) ;
 *                           `hiddenOnly` quand le conteneur défilé est en overflow hidden/clip, donc
 *                           inatteignable à la molette ou à la barre
 *   targets[]             — pour chaque cible cliquée : hors écran / masquée (elementFromPoint au
 *                           centre, avec le masque nommé) au viewport du persona, à 1366×768 et à
 *                           1920×1080 ; taille < 24 px ; texte tronqué
 *   tabs, tabOrder        — mode clavier seul : Tab nécessaires et ordre traversé, focus visible
 *   focusAfter, focusLost — l'élément actif après l'action ; perdu = body
 *   feedback              — un retour visuel apparu dans FEEDBACK_MS (statut, alerte, aria-live,
 *                           bandeau, toast, barre d'état), ou null
 *   wasted                — écrans ouverts pour rien (hésitations du novice), déclarés par la spec
 *   ecart                 — texte libre : ce que le directeur voulait, ce que la page proposait
 *
 * Avant chaque action, tous les conteneurs défilés sont remis à scrollTop = 0 : la mesure de 2026-09
 * a appris qu'un défilement hérité d'une action précédente masque le coût réel de la suivante.
 *
 * Le zoom 150 % est simulé par le viewport CSS divisé par 1,5 (`zoomViewport`) : c'est ce que fait
 * Ctrl+ dans un navigateur pour la mise en page. Le facteur d'échelle (deviceScaleFactor) ne change
 * que la densité de pixels et ne déplace rien : il ne mesurerait pas ce qu'on cherche.
 */
import fs from 'node:fs';

export const VIEWPORTS = {
    hd: { width: 1920, height: 1080 },
    laptop: { width: 1366, height: 768 }
};
const FEEDBACK_SELECTORS =
    '[role="status"],[role="alert"],[aria-live],.toast,.banner,.status-bar,.statusbar,[class*="toast"],[class*="error"],[data-testid*="error"],[data-testid*="confirm"],[data-testid="direction-last"],[data-testid="direction-warnings"]';
const MIN_TARGET = 24;

/** Viewport équivalent à un zoom navigateur de `factor` sur `base`. */
export function zoomViewport(base, factor = 1.5) {
    return { width: Math.round(base.width / factor), height: Math.round(base.height / factor) };
}

export class Meter {
    /**
     * @param {import('@playwright/test').Page} page
     * @param {{ persona: string, tournament: string, viewport: {width:number,height:number}, input?: 'mouse'|'keyboard', file?: string, feedbackMs?: number, checkViewports?: boolean }} opts
     */
    constructor(page, opts) {
        this.page = page;
        this.o = { input: 'mouse', checkViewports: true, feedbackMs: Number(process.env.METER_FEEDBACK_MS || 1500), ...opts };
        this.file = opts.file || process.env.METER_FILE;
        this.seq = 0;
        this.records = [];
    }

    /** Change de persona en cours de tournoi (un remplaçant prend la salle). */
    setPersona(persona, patch = {}) {
        this.o = { ...this.o, persona, ...patch };
    }

    async resetScroll() {
        await this.page.evaluate(() => {
            for (const el of document.querySelectorAll('*')) if (el.scrollTop > 0 || el.scrollLeft > 0) el.scrollTo(0, 0);
            window.scrollTo(0, 0);
        });
    }

    /** Décrit une cible au viewport courant, sans la toucher. */
    async probe(locator) {
        const h = await locator.elementHandle({ timeout: 5000 }).catch(() => null);
        if (!h) return { found: false };
        return h.evaluate((el, min) => {
            const name = (n) => {
                if (!n || n === document.body) return 'body';
                const tid = n.closest('[data-testid]')?.getAttribute('data-testid');
                return `${n.tagName.toLowerCase()}${n.id ? '#' + n.id : ''}${n.classList[0] ? '.' + n.classList[0] : ''}${tid ? '[' + tid + ']' : ''}`;
            };
            const r = el.getBoundingClientRect();
            const vw = window.innerWidth;
            const vh = window.innerHeight;
            const cx = r.left + r.width / 2;
            const cy = r.top + r.height / 2;
            const visibleBox = r.width > 0 && r.height > 0;
            const inViewport = visibleBox && cx >= 0 && cy >= 0 && cx < vw && cy < vh;
            let coveredBy = null;
            if (inViewport) {
                const hit = document.elementFromPoint(cx, cy);
                if (hit && hit !== el && !el.contains(hit) && !hit.contains(el)) coveredBy = name(hit);
            }
            // Le premier ancêtre qui défile, et s'il défile à la main.
            const scrollers = [];
            for (let p = el.parentElement; p; p = p.parentElement) {
                if (p.scrollHeight > p.clientHeight + 1 || p.scrollWidth > p.clientWidth + 1) {
                    const s = getComputedStyle(p);
                    scrollers.push({ name: name(p), overflowY: s.overflowY, overflowX: s.overflowX });
                }
            }
            const text = (el.innerText || el.value || '').trim().slice(0, 60);
            return {
                found: true,
                name: name(el),
                text,
                w: Math.round(r.width),
                h: Math.round(r.height),
                small: visibleBox && (r.width < min || r.height < min),
                truncated: el.scrollWidth > el.clientWidth + 1 && getComputedStyle(el).overflowX !== 'visible',
                inViewport,
                offscreen: !inViewport,
                coveredBy,
                scrollers
            };
        }, MIN_TARGET);
    }

    /** La même cible vue à 1366×768 et à 1920×1080, puis retour au viewport du persona. */
    async probeViewports(locator) {
        const out = {};
        const back = this.page.viewportSize();
        for (const [k, vp] of Object.entries(VIEWPORTS)) {
            await this.page.setViewportSize(vp);
            await this.page.evaluate(() => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))));
            await this.resetScroll();
            const p = await this.probe(locator);
            out[k] = { offscreen: p.offscreen ?? null, coveredBy: p.coveredBy ?? null };
        }
        await this.page.setViewportSize(back);
        await this.page.evaluate(() => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))));
        return out;
    }

    async scrollState() {
        return this.page.evaluate(() => {
            const m = {};
            let i = 0;
            for (const el of document.querySelectorAll('*')) {
                if (el.scrollHeight > el.clientHeight + 1 || el.scrollWidth > el.clientWidth + 1) {
                    if (!el.__meterId) el.__meterId = `s${++i}-${Math.random().toString(36).slice(2, 6)}`;
                    const tid = el.closest('[data-testid]')?.getAttribute('data-testid');
                    const s = getComputedStyle(el);
                    m[el.__meterId] = {
                        name: `${el.tagName.toLowerCase()}${el.id ? '#' + el.id : ''}${el.classList[0] ? '.' + el.classList[0] : ''}${tid ? '[' + tid + ']' : ''}`,
                        top: el.scrollTop,
                        left: el.scrollLeft,
                        hidden: ['hidden', 'clip'].includes(s.overflowY) && ['hidden', 'clip', 'visible'].includes(s.overflowX)
                    };
                }
            }
            m.__window = { name: 'window', top: window.scrollY, left: window.scrollX, hidden: false };
            return m;
        });
    }

    async armFeedback() {
        await this.page.evaluate((sel) => {
            window.__meterFeedback = null;
            window.__meterObs?.disconnect();
            const seen = (n) => {
                const el = n.nodeType === 1 ? n : n.parentElement;
                if (!el) return null;
                const hit = el.matches?.(sel) ? el : el.closest?.(sel);
                return hit && (hit.innerText || '').trim() ? hit : null;
            };
            window.__meterObs = new MutationObserver((muts) => {
                if (window.__meterFeedback) return;
                for (const m of muts) {
                    const nodes = m.type === 'childList' ? [...m.addedNodes] : [m.target];
                    for (const n of nodes) {
                        const hit = seen(n);
                        if (hit) {
                            const tid = hit.closest('[data-testid]')?.getAttribute('data-testid');
                            window.__meterFeedback = { what: tid || hit.getAttribute('role') || hit.className || hit.tagName, text: (hit.innerText || '').trim().slice(0, 80) };
                            return;
                        }
                    }
                }
            });
            window.__meterObs.observe(document.body, { childList: true, subtree: true, characterData: true });
        }, FEEDBACK_SELECTORS);
    }

    async collectFeedback() {
        const deadline = Date.now() + this.o.feedbackMs;
        let fb = null;
        while (Date.now() < deadline) {
            fb = await this.page.evaluate(() => window.__meterFeedback);
            if (fb) break;
            await this.page.waitForTimeout(100);
        }
        await this.page.evaluate(() => window.__meterObs?.disconnect());
        return fb;
    }

    async focusInfo() {
        return this.page.evaluate(() => {
            const a = document.activeElement;
            if (!a || a === document.body || a === document.documentElement) return { lost: true, name: 'body' };
            const s = getComputedStyle(a);
            const visible = (s.outlineStyle !== 'none' && parseFloat(s.outlineWidth) > 0) || s.boxShadow !== 'none';
            const tid = a.closest('[data-testid]')?.getAttribute('data-testid');
            return { lost: false, name: `${a.tagName.toLowerCase()}${a.classList[0] ? '.' + a.classList[0] : ''}${tid ? '[' + tid + ']' : ''}`, focusVisible: visible };
        });
    }

    /**
     * Une action du directeur.
     * @param {{ action: string, persona?: string, ecart?: string, attendu?: string }} meta
     * @param {(g: Driver) => Promise<void>} run
     */
    async act(meta, run) {
        const page = this.page;
        const rec = {
            seq: ++this.seq,
            tournament: this.o.tournament,
            persona: meta.persona || this.o.persona,
            input: this.o.input,
            viewport: page.viewportSize(),
            action: meta.action,
            clicks: 0,
            keys: 0,
            tabs: 0,
            wheel: { notches: 0, px: {} },
            autoScroll: { px: 0, notches: 0, hiddenOnly: false, containers: {} },
            targets: [],
            tabOrder: [],
            wasted: 0,
            problems: [],
            ecart: meta.ecart || null,
            ok: true,
            error: null
        };
        await this.resetScroll();
        await this.armFeedback();
        const t0 = Date.now();
        const meter = this;

        const before = async () => meter.scrollState();
        const diff = (a, b, into, wheelMode) => {
            for (const [k, v] of Object.entries(b)) {
                const p = a[k];
                if (!p) continue;
                const dpx = Math.abs(v.top - p.top) + Math.abs(v.left - p.left);
                if (!dpx) continue;
                if (wheelMode) {
                    into.px[v.name] = (into.px[v.name] || 0) + dpx;
                } else {
                    into.px += dpx;
                    into.containers[v.name] = (into.containers[v.name] || 0) + dpx;
                    if (v.hidden) into.hiddenOnly = true;
                }
            }
        };

        /** @typedef {ReturnType<typeof makeDriver>} Driver */
        const makeDriver = () => ({
            /** Clic souris, ou en clavier seul : Tab jusqu'à la cible puis Entrée. */
            click: async (target, opts = {}) => {
                const loc = typeof target === 'string' ? page.locator(target) : target;
                const p = await meter.probe(loc);
                const t = { ...p };
                if (meter.o.checkViewports && !opts.noViewportCheck) t.viewports = await meter.probeViewports(loc);
                if (p.small) rec.problems.push(`cible ${p.w}×${p.h} px < 24 : ${p.name}`);
                if (p.truncated) rec.problems.push(`texte tronqué : ${p.name} « ${p.text} »`);
                if (p.coveredBy) rec.problems.push(`cible masquée par ${p.coveredBy} : ${p.name}`);
                if (t.viewports) {
                    for (const [k, v] of Object.entries(t.viewports)) {
                        if (v.coveredBy) rec.problems.push(`${k} : masquée par ${v.coveredBy}`);
                    }
                }
                rec.targets.push(t);
                if (meter.o.input === 'keyboard' && !opts.mouse) {
                    const reached = await driver.tabTo(loc, opts.maxTabs);
                    if (reached) {
                        rec.keys += 1;
                        await page.keyboard.press(opts.key || 'Enter');
                        return;
                    }
                    rec.problems.push(`injoignable au clavier : ${p.name}`);
                }
                const s0 = await before();
                rec.clicks += 1;
                await loc.click(opts.clickOptions);
                diff(s0, await before(), rec.autoScroll, false);
            },
            dblclick: async (target) => {
                const loc = typeof target === 'string' ? page.locator(target) : target;
                rec.targets.push(await meter.probe(loc));
                const s0 = await before();
                rec.clicks += 2;
                await loc.dblclick();
                diff(s0, await before(), rec.autoScroll, false);
            },
            press: async (key) => {
                rec.keys += 1;
                await page.keyboard.press(key);
            },
            /** Frappe réelle : chaque caractère compte. */
            type: async (text, target) => {
                if (target) await driver.click(target);
                rec.keys += [...text].length;
                await page.keyboard.type(text);
            },
            /** Remplace le contenu d'un champ : Ctrl+A (1) + la frappe. */
            fill: async (target, text) => {
                await driver.click(target);
                rec.keys += 1 + [...text].length;
                await page.keyboard.press('ControlOrMeta+a');
                await page.keyboard.type(text);
            },
            select: async (target, value) => {
                await driver.click(target, { noViewportCheck: true, mouse: true });
                rec.clicks += 1;
                await (typeof target === 'string' ? page.locator(target) : target).selectOption(value);
            },
            /** Molette au-dessus d'un conteneur : `notches` crans de 100 px. */
            wheel: async (target, notches) => {
                const loc = typeof target === 'string' ? page.locator(target) : target;
                const box = await loc.boundingBox();
                if (box) await page.mouse.move(box.x + box.width / 2, box.y + Math.min(box.height / 2, 200));
                const s0 = await before();
                for (let i = 0; i < Math.abs(notches); i++) await page.mouse.wheel(0, Math.sign(notches) * 100);
                await page.waitForTimeout(150);
                rec.wheel.notches += Math.abs(notches);
                diff(s0, await before(), rec.wheel, true);
            },
            /** Tab jusqu'à la cible ; relève l'ordre traversé et la visibilité du focus. */
            tabTo: async (target, max = 80) => {
                const loc = typeof target === 'string' ? page.locator(target) : target;
                const h = await loc.elementHandle({ timeout: 5000 }).catch(() => null);
                if (!h) return false;
                for (let i = 0; i < max; i++) {
                    if (await h.evaluate((el) => el === document.activeElement || el.contains(document.activeElement))) {
                        const f = await meter.focusInfo();
                        if (!f.focusVisible) rec.problems.push(`focus invisible sur ${f.name}`);
                        return true;
                    }
                    rec.tabs += 1;
                    rec.keys += 1;
                    await page.keyboard.press('Tab');
                    const f = await meter.focusInfo();
                    rec.tabOrder.push(f.name);
                }
                return false;
            },
            /** Un écran ouvert pour rien (le novice cherche) : compté, et noté. */
            hesitate: async (why, fn) => {
                rec.wasted += 1;
                rec.problems.push(`hésitation : ${why}`);
                if (fn) await fn(driver);
            },
            problem: (text) => rec.problems.push(text),
            ecart: (text) => (rec.ecart = rec.ecart ? `${rec.ecart} ; ${text}` : text),
            record: rec
        });
        const driver = makeDriver();

        try {
            await run(driver);
        } catch (e) {
            rec.ok = false;
            rec.error = String(e?.message || e)
                .split('\n')[0]
                .slice(0, 300);
        }
        rec.ms = Date.now() - t0;
        rec.feedback = await this.collectFeedback();
        const f = await this.focusInfo();
        rec.focusAfter = f.name;
        rec.focusLost = f.lost;
        rec.autoScroll.notches = Math.ceil(rec.autoScroll.px / 100);
        if (rec.autoScroll.hiddenOnly) rec.problems.push('cible atteinte seulement en défilant un conteneur overflow hidden');
        this.records.push(rec);
        if (this.file) fs.appendFileSync(this.file, JSON.stringify(rec) + '\n');
        if (!rec.ok && meta.mustPass) throw new Error(`${meta.action}: ${rec.error}`);
        return rec;
    }
}

/** Agrège un fichier JSONL par tournoi et persona : gestes, défilements, problèmes, échecs. */
export function summarize(file) {
    const rows = fs
        .readFileSync(file, 'utf8')
        .split('\n')
        .filter(Boolean)
        .map((l) => JSON.parse(l));
    const by = {};
    for (const r of rows) {
        const k = `${r.tournament}/${r.persona}`;
        const s = (by[k] ??= { actions: 0, clicks: 0, keys: 0, tabs: 0, wheelNotches: 0, autoNotches: 0, wasted: 0, focusLost: 0, noFeedback: 0, failed: 0, problems: 0 });
        s.actions += 1;
        s.clicks += r.clicks;
        s.keys += r.keys;
        s.tabs += r.tabs;
        s.wheelNotches += r.wheel.notches;
        s.autoNotches += r.autoScroll.notches;
        s.wasted += r.wasted;
        s.focusLost += r.focusLost ? 1 : 0;
        s.noFeedback += r.feedback ? 0 : 1;
        s.failed += r.ok ? 0 : 1;
        s.problems += r.problems.length;
    }
    return by;
}
