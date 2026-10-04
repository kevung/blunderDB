/**
 * shimProcess.js — lance, tue (SIGKILL) et relance le shim sur une base du scratchpad.
 *
 * Le binaire est construit une fois (`go build -tags simulation`) dans SIM_DIR, GOTMPDIR hors de
 * /tmp. SIGKILL reproduit une fermeture brutale du processus : rien n'est fermé proprement, le
 * verrou d'écriture et le WAL restent tels quels, et la relance rejoue le journal comme le GUI.
 */
import { spawn, execFileSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
export const REPO = path.resolve(here, '../../../../..');
export const SIM_DIR = process.env.SIM_DIR || path.join(os.tmpdir(), 'blunderdb-sim');
const BIN = path.join(SIM_DIR, 'shim');

/**
 * Construit un outil `//go:build simulation` de `outils/_<nom>` vers SIM_DIR/<nom>. Le préfixe `_`
 * sort ces paquets de `./...` : `go vet` et la CI ne les voient pas, `go build` sur leur chemin
 * explicite les construit.
 */
export function buildTool(name) {
    fs.mkdirSync(path.join(SIM_DIR, 'gotmp'), { recursive: true });
    const bin = path.join(SIM_DIR, name);
    const src = path.join(here, '..', '_' + name);
    const stale = !fs.existsSync(bin) || fs.statSync(path.join(src, 'main.go')).mtimeMs > fs.statSync(bin).mtimeMs;
    if (stale) {
        execFileSync('go', ['build', '-tags', 'simulation', '-o', bin, './' + path.relative(REPO, src)], {
            cwd: REPO,
            env: { ...process.env, GOTMPDIR: path.join(SIM_DIR, 'gotmp') },
            stdio: 'inherit'
        });
    }
    return bin;
}

export const buildShim = () => buildTool('shim');

export class Shim {
    /**
     * @param {{ name: string, port: number, fresh?: boolean }} o — name nomme base et dossier de
     *   sortie ; fresh (vrai par défaut) repart d'une base neuve, faux reprend la base laissée là.
     */
    constructor({ name, port, fresh = true }) {
        this.port = port;
        this.url = `http://127.0.0.1:${port}`;
        this.dir = path.join(SIM_DIR, name);
        if (fresh) fs.rmSync(this.dir, { recursive: true, force: true });
        fs.mkdirSync(this.dir, { recursive: true });
        this.dbPath = path.join(this.dir, `${name}.db`);
        this.outDir = path.join(this.dir, 'sortie');
        fs.mkdirSync(this.outDir, { recursive: true });
        this.log = path.join(this.dir, 'shim.log');
        this.proc = null;
    }

    async start() {
        buildShim();
        const out = fs.openSync(this.log, 'a');
        this.proc = spawn(BIN, ['-db', this.dbPath, '-port', String(this.port), '-out', this.outDir], { stdio: ['ignore', out, out] });
        for (let i = 0; i < 100; i++) {
            try {
                const r = await fetch(`${this.url}/methods`);
                if (r.ok) return this;
            } catch {
                /* pas encore à l'écoute */
            }
            await new Promise((r) => setTimeout(r, 100));
        }
        throw new Error(`shim muet sur ${this.url}, voir ${this.log}`);
    }

    /** Fermeture brutale : SIGKILL, sans Close, comme une coupure de courant du portable. */
    async kill() {
        if (!this.proc) return;
        const p = this.proc;
        this.proc = null;
        await new Promise((res) => {
            p.once('exit', res);
            p.kill('SIGKILL');
        });
    }

    async restart() {
        await this.kill();
        return this.start();
    }

    /** Tue le shim et efface base et sorties (disque /home presque plein). */
    async dispose({ keep = false } = {}) {
        await this.kill();
        if (!keep) fs.rmSync(this.dir, { recursive: true, force: true });
    }
}
